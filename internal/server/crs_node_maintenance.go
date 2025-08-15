package server

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

// HandleNodeMaintenance migrates stopped VMs and templates from maintenance nodes
func (s *Server) HandleNodeMaintenance() error {
	logging.Debug("Checking for nodes in maintenance mode")

	// Get cluster resources to check node maintenance state (hastate field)
	resources, err := s.proxmox.GetClusterResources()
	if err != nil {
		return fmt.Errorf("failed to get cluster resources: %w", err)
	}

	// Extract nodes from cluster resources and check their hastate
	var onlineNodes []proxmox.Node
	var maintenanceNodes []proxmox.Node
	var nodeResources []proxmox.ClusterResource

	// Filter out node resources
	for _, resource := range resources {
		if resource.Type == "node" {
			nodeResources = append(nodeResources, resource)
		}
	}

	logging.Debugf("Found %d nodes in cluster", len(nodeResources))

	// Check each node's hastate for maintenance mode
	for _, nodeResource := range nodeResources {
		// Convert ClusterResource to Node for compatibility
		node := proxmox.Node{
			Node:   nodeResource.Node,
			Status: nodeResource.Status,
		}

		// Check both status and hastate for maintenance
		switch {
		case nodeResource.Status == "online" && nodeResource.HAState != "maintenance":
			onlineNodes = append(onlineNodes, node)
		case nodeResource.HAState == "maintenance":
			maintenanceNodes = append(maintenanceNodes, node)
			logging.Debugf("Node %s is in maintenance mode (hastate: %s, status: %s)", nodeResource.Node, nodeResource.HAState, nodeResource.Status)
		default:
			// Nodes with other statuses (offline, unknown, etc.) are not considered for maintenance migration
			logging.Debugf("Node %s has status %s and hastate %s, not considered for maintenance operations", nodeResource.Node, nodeResource.Status, nodeResource.HAState)
		}
	}

	if len(maintenanceNodes) == 0 {
		logging.Debug("No nodes in maintenance mode found")
		return nil
	}

	if len(onlineNodes) == 0 {
		logging.Warn("No online nodes available for migration from maintenance nodes")
		return nil
	}

	logging.Debugf("Found %d nodes in maintenance mode and %d online nodes available for migration", len(maintenanceNodes), len(onlineNodes))

	var totalMigrated int

	for _, maintenanceNode := range maintenanceNodes {
		migrated, err := s.migrateVMsFromMaintenanceNode(maintenanceNode.Node, onlineNodes)
		if err != nil {
			logging.Errorf("Failed to migrate VMs from maintenance node %s: %v", maintenanceNode.Node, err)
			// Continue with other nodes
			continue
		}
		totalMigrated += migrated
	}

	if totalMigrated > 0 {
		logging.Infof("Successfully migrated %d VMs/templates from maintenance nodes", totalMigrated)
	} else {
		logging.Debug("No VMs/templates needed migration from maintenance nodes")
	}

	return nil
}

// migrateVMsFromMaintenanceNode migrates eligible VMs and templates from a maintenance node
func (s *Server) migrateVMsFromMaintenanceNode(maintenanceNode string, onlineNodes []proxmox.Node) (int, error) {
	logging.Debugf("Checking VMs on maintenance node %s for migration", maintenanceNode)

	// Get VMs on the maintenance node
	vms, err := s.proxmox.GetNodeVMs(maintenanceNode)
	if err != nil {
		return 0, fmt.Errorf("failed to get VMs for maintenance node %s: %w", maintenanceNode, err)
	}

	// Get HA resources to check group assignments
	haResources, err := s.proxmox.GetClusterHAResources()
	if err != nil {
		return 0, fmt.Errorf("failed to get HA resources: %w", err)
	}

	// Create a map of VM ID to HA resource for quick lookup
	haResourceMap := make(map[int]*proxmox.ClusterHAResource)
	for _, resource := range haResources {
		if resource.Type == haResourceType {
			// Extract VM ID from resource SID (format: "vm:123")
			parts := strings.Split(resource.SID, ":")
			if len(parts) == 2 {
				if vmid := parseVMID(parts[1]); vmid > 0 {
					haResourceMap[vmid] = &resource
				}
			}
		}
	}

	var migratedCount int

	for _, vm := range vms {
		// Skip VMs with crs-skip tag
		if s.hasVMSkipTag(vm.Tags) {
			logging.Debugf("Skipping VM %d (%s) with crs-skip tag on maintenance node %s", vm.VMID, vm.Name, maintenanceNode)
			continue
		}

		if s.shouldMigrateVMFromMaintenance(vm, haResourceMap[vm.VMID]) {
			targetNode, err := s.selectMigrationTarget(vm, onlineNodes)
			if err != nil {
				logging.Errorf("Failed to select migration target for VM %d (%s): %v", vm.VMID, vm.Name, err)
				continue
			}

			if err := s.migrateVM(maintenanceNode, vm, targetNode); err != nil {
				logging.Errorf("Failed to migrate VM %d (%s) from %s to %s: %v", vm.VMID, vm.Name, maintenanceNode, targetNode, err)
				continue
			}

			migratedCount++
			logging.Infof("Successfully migrated VM %d (%s) from maintenance node %s to %s", vm.VMID, vm.Name, maintenanceNode, targetNode)

			// Sleep to avoid overwhelming the Proxmox API
			s.rateLimitSleep()
		}
	}

	return migratedCount, nil
}

// shouldMigrateVMFromMaintenance determines if a VM should be migrated from maintenance node
func (s *Server) shouldMigrateVMFromMaintenance(vm proxmox.VM, haResource *proxmox.ClusterHAResource) bool {
	// Templates should be migrated if they're on shared storage
	if vm.Template == vmTemplateFlag {
		logging.Debugf("VM %d (%s) is a template, checking if it's on shared storage for migration", vm.VMID, vm.Name)
		return s.isVMOnSharedStorage(vm)
	}

	// Running VMs are handled by HA automatically, don't migrate them manually
	if vm.Status == vmStatusRunning {
		logging.Debugf("VM %d (%s) is running, letting HA handle migration", vm.VMID, vm.Name)
		return false
	}

	// For stopped VMs, check if they're in crs-vm-prefer group
	if haResource != nil && strings.Contains(haResource.Group, "crs-vm-prefer") {
		logging.Debugf("VM %d (%s) is stopped and in prefer group (%s), should migrate", vm.VMID, vm.Name, haResource.Group)
		return true
	}

	// VMs in pin groups should stay on their assigned node
	if haResource != nil && strings.Contains(haResource.Group, "crs-vm-pin") {
		logging.Debugf("VM %d (%s) is in pin group (%s), should not migrate", vm.VMID, vm.Name, haResource.Group)
		return false
	}

	// VMs without HA resources shouldn't be migrated automatically
	logging.Debugf("VM %d (%s) has no HA resource or is not in CRS group, not migrating", vm.VMID, vm.Name)
	return false
}

// isVMOnSharedStorage checks if a VM (template) is stored on shared storage
func (s *Server) isVMOnSharedStorage(vm proxmox.VM) bool {
	// Get VM configuration to check storage
	config, err := s.proxmox.GetVMConfig(vm.Node, vm.VMID)
	if err != nil {
		logging.Errorf("Failed to get VM %d config to check storage: %v", vm.VMID, err)
		return false
	}

	// Check if all VM disks are on shared storage
	allShared, err := s.areAllVMDisksShared(config.Disks)
	if err != nil {
		logging.Errorf("Failed to analyze VM %d storage: %v", vm.VMID, err)
		return false
	}

	return allShared
}

// selectMigrationTarget selects the best target node for migration
func (s *Server) selectMigrationTarget(vm proxmox.VM, onlineNodes []proxmox.Node) (string, error) {
	if len(onlineNodes) == 0 {
		return "", fmt.Errorf("no online nodes available")
	}

	// For now, use round-robin selection based on VM ID
	// In the future, this could be enhanced with load balancing
	targetIndex := vm.VMID % len(onlineNodes)
	targetNode := onlineNodes[targetIndex].Node

	logging.Debugf("Selected target node %s for VM %d (%s) migration", targetNode, vm.VMID, vm.Name)
	return targetNode, nil
}

// migrateVM performs the actual VM migration
func (s *Server) migrateVM(sourceNode string, vm proxmox.VM, targetNode string) error {
	logging.Infof("Migrating VM %d (%s) from %s to %s", vm.VMID, vm.Name, sourceNode, targetNode)

	migrationOptions := proxmox.MigrationOptions{
		Target:    targetNode,
		Online:    false, // Offline migration for stopped VMs and templates
		WithDisks: true,  // Move disks if they're on shared storage
	}

	taskID, err := s.proxmox.MigrateVM(sourceNode, vm.VMID, migrationOptions)
	if err != nil {
		return fmt.Errorf("failed to start migration: %w", err)
	}

	logging.Debugf("Started migration task %s for VM %d", taskID, vm.VMID)
	return nil
}

// parseVMID safely parses a VM ID string to integer
func parseVMID(vmidStr string) int {
	vmid, err := strconv.Atoi(vmidStr)
	if err != nil {
		return 0
	}
	return vmid
}
