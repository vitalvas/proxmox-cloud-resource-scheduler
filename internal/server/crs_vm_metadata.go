package server

import (
	"fmt"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

// UpdateVMMeta updates VM metadata for critical VMs and handles CD-ROM detachment
func (s *Server) UpdateVMMeta() error {
	logging.Debug("Checking VMs for metadata updates")

	// Get cluster resources to find VMs that need metadata updates
	resources, err := s.proxmox.GetClusterResources()
	if err != nil {
		return fmt.Errorf("failed to get cluster resources: %w", err)
	}

	var criticalVMs []proxmox.ClusterResource
	var longRunningVMs []proxmox.ClusterResource

	for _, resource := range resources {
		// Only check VMs (type=qemu)
		if resource.Type == vmResourceType {
			// Skip VMs with crs-skip tag
			if s.hasVMSkipTag(resource.Tags) {
				logging.Debugf("Skipping VM %d (%s) with crs-skip tag for metadata operations", resource.VMID, resource.Name)
				continue
			}

			// Check if VM has critical tag
			if s.hasVMCriticalTag(resource.Tags) {
				criticalVMs = append(criticalVMs, resource)
				logging.Debugf("VM %d with critical tag detected: name=%s, node=%s, tags=%s",
					resource.VMID, resource.Name, resource.Node, resource.Tags)
			}

			// Check if VM is running and has uptime > 24 hours (86400 seconds)
			if resource.Status == vmStatusRunning && resource.Uptime > 86400 {
				longRunningVMs = append(longRunningVMs, resource)
				logging.Debugf("VM %d is long-running: name=%s, node=%s, uptime=%ds",
					resource.VMID, resource.Name, resource.Node, resource.Uptime)
			}
		}
	}

	var totalUpdatedCount int

	// Handle critical VM startup order updates
	if len(criticalVMs) > 0 {
		logging.Debugf("Found %d critical VMs that need metadata updates: %v", len(criticalVMs), s.extractVMIDs(criticalVMs))

		var criticalUpdatedCount int
		for _, vm := range criticalVMs {
			if s.updateCriticalVMStartOrder(vm.Node, vm.VMID) {
				criticalUpdatedCount++
			}
		}

		if criticalUpdatedCount > 0 {
			logging.Infof("Updated startup order for %d critical VMs", criticalUpdatedCount)
			totalUpdatedCount += criticalUpdatedCount
		} else {
			logging.Debug("No critical VMs required startup order updates")
		}
	} else {
		logging.Debug("No critical VMs found that need metadata updates")
	}

	// Handle CD-ROM detachment for long-running VMs
	if len(longRunningVMs) > 0 {
		logging.Debugf("Found %d long-running VMs that may need CD-ROM detachment: %v", len(longRunningVMs), s.extractVMIDs(longRunningVMs))

		var cdromDetachedCount int
		for _, vm := range longRunningVMs {
			if s.detachNonSharedCDROMs(vm.Node, vm.VMID) {
				cdromDetachedCount++
			}
		}

		if cdromDetachedCount > 0 {
			logging.Infof("Detached non-shared CD-ROMs from %d long-running VMs", cdromDetachedCount)
			totalUpdatedCount += cdromDetachedCount
		} else {
			logging.Debug("No long-running VMs required CD-ROM detachment")
		}
	} else {
		logging.Debug("No long-running VMs found that need CD-ROM detachment")
	}

	if totalUpdatedCount == 0 {
		logging.Debug("No VM metadata updates were needed")
	}

	return nil
}

// extractVMIDs extracts VM IDs from a list of cluster resources
func (s *Server) extractVMIDs(vms []proxmox.ClusterResource) []int {
	vmids := make([]int, len(vms))
	for i, vm := range vms {
		vmids[i] = vm.VMID
	}
	return vmids
}

// updateCriticalVMStartOrder updates the startup order for critical VMs to order=1
func (s *Server) updateCriticalVMStartOrder(node string, vmid int) bool {
	logging.Debugf("Checking startup order for critical VM %d on node %s", vmid, node)

	// Get current VM configuration
	config, err := s.proxmox.GetVMConfig(node, vmid)
	if err != nil {
		logging.Errorf("Failed to get VM %d config on node %s: %v", vmid, node, err)
		return false
	}

	// Check if startup order is already set to order=1
	if config.Startup == vmStartupCriticalOrder {
		logging.Debugf("VM %d already has correct startup order: %s", vmid, config.Startup)
		return false
	}

	logging.Infof("Updating critical VM %d startup order from '%s' to '%s'", vmid, config.Startup, vmStartupCriticalOrder)

	// Update only the startup configuration
	updateConfig := proxmox.VMConfig{
		Startup: vmStartupCriticalOrder,
	}

	if err := s.proxmox.UpdateVMConfig(node, vmid, updateConfig); err != nil {
		logging.Errorf("Failed to update VM %d startup order on node %s: %v", vmid, node, err)
		return false
	}

	logging.Infof("Successfully updated startup order for critical VM %d", vmid)
	return true
}

// detachNonSharedCDROMs detaches CD-ROM drives that are on non-shared storage
func (s *Server) detachNonSharedCDROMs(node string, vmid int) bool {
	logging.Debugf("Checking CD-ROM drives for VM %d on node %s", vmid, node)

	// Get current VM configuration
	config, err := s.proxmox.GetVMConfig(node, vmid)
	if err != nil {
		logging.Errorf("Failed to get VM %d config on node %s: %v", vmid, node, err)
		return false
	}

	// Get storage information
	storages, err := s.proxmox.GetStorage()
	if err != nil {
		logging.Errorf("Failed to get storage info for VM %d CD-ROM check: %v", vmid, err)
		return false
	}

	// Create a map of storage name to shared status for quick lookup
	storageSharedMap := make(map[string]bool)
	for _, storage := range storages {
		storageSharedMap[storage.Storage] = storage.Shared == 1
	}

	var cdromsToDetach []string
	var detachedAny bool

	// Check each disk entry for CD-ROM drives on non-shared storage
	for diskKey, diskValue := range config.Disks {
		// Check if this is a CD-ROM drive (usually ide2, but can be other IDE devices)
		if s.isCDROMEntry(diskKey, diskValue) {
			// Extract storage name from disk configuration
			storageName := s.extractStorageFromDiskConfig(diskValue)
			if storageName == "" {
				logging.Warnf("Could not extract storage name from CD-ROM config: %s=%s", diskKey, diskValue)
				continue
			}

			// Check if this storage is shared
			isShared, exists := storageSharedMap[storageName]
			if !exists {
				logging.Warnf("Storage %s not found in cluster storage list for VM %d CD-ROM %s", storageName, vmid, diskKey)
				continue
			}

			if !isShared {
				logging.Infof("Found CD-ROM %s on non-shared storage %s for VM %d, scheduling for detachment", diskKey, storageName, vmid)
				cdromsToDetach = append(cdromsToDetach, diskKey)
			} else {
				logging.Debugf("CD-ROM %s on shared storage %s for VM %d, keeping attached", diskKey, storageName, vmid)
			}
		}
	}

	if len(cdromsToDetach) == 0 {
		logging.Debugf("No non-shared CD-ROMs found for VM %d", vmid)
		return false
	}

	logging.Infof("Detaching %d non-shared CD-ROM drives from VM %d: %v", len(cdromsToDetach), vmid, cdromsToDetach)

	// Detach each CD-ROM drive by removing it from config
	updateConfig := proxmox.VMConfig{
		Disks: make(map[string]string),
	}

	for _, cdromKey := range cdromsToDetach {
		// Set to empty string to remove the CD-ROM drive
		updateConfig.Disks[cdromKey] = ""
		logging.Debugf("Removing CD-ROM drive %s from VM %d", cdromKey, vmid)
	}

	if err := s.proxmox.UpdateVMConfig(node, vmid, updateConfig); err != nil {
		logging.Errorf("Failed to detach CD-ROM drives from VM %d on node %s: %v", vmid, node, err)
		return false
	}

	logging.Infof("Successfully detached %d non-shared CD-ROM drives from VM %d", len(cdromsToDetach), vmid)
	detachedAny = true

	// Re-evaluate HA group assignment after CD-ROM detachment
	// Storage configuration may have changed from mixed to all-shared
	if err := s.reevaluateVMHAGroupAfterDetachment(node, vmid); err != nil {
		logging.Errorf("Failed to re-evaluate HA group for VM %d after CD-ROM detachment: %v", vmid, err)
		// Don't return error as CD-ROM detachment was successful
	}

	// Sleep to avoid overwhelming the Proxmox API
	s.rateLimitSleep()

	return detachedAny
}

// isCDROMEntry checks if a disk entry represents a CD-ROM drive
func (s *Server) isCDROMEntry(diskKey, diskValue string) bool {
	// CD-ROM drives are typically IDE devices with media=cdrom parameter
	// or ISO files referenced directly
	if !s.isDiskEntry(diskKey) {
		return false
	}

	// Check if the disk value contains media=cdrom (explicit CD-ROM)
	if diskValue != "" && (strings.Contains(diskValue, "media=cdrom") ||
		strings.Contains(diskValue, ".iso")) {
		return true
	}

	// Additional check: IDE devices are commonly used for CD-ROM
	// but only consider them CD-ROM if they have specific indicators
	if strings.HasPrefix(diskKey, "ide") && strings.Contains(diskValue, ".iso") {
		return true
	}

	return false
}

// reevaluateVMHAGroupAfterDetachment re-evaluates and updates VM HA group assignment after CD-ROM detachment
func (s *Server) reevaluateVMHAGroupAfterDetachment(node string, vmid int) error {
	logging.Debugf("Re-evaluating HA group assignment for VM %d after CD-ROM detachment", vmid)

	// Get current HA resource to see what group it's currently in
	resources, err := s.proxmox.GetClusterHAResources()
	if err != nil {
		return fmt.Errorf("failed to get cluster HA resources: %w", err)
	}

	sid := fmt.Sprintf("%s:%d", haResourceType, vmid)
	var currentResource *proxmox.ClusterHAResource
	for _, resource := range resources {
		if resource.SID == sid {
			currentResource = &resource
			break
		}
	}

	if currentResource == nil {
		logging.Debugf("VM %d has no HA resource, skipping group re-evaluation", vmid)
		return nil
	}

	// Determine new appropriate HA group based on current storage configuration
	newHAGroup, err := s.determineVMHAGroup(node, vmid)
	if err != nil {
		return fmt.Errorf("failed to determine new HA group: %w", err)
	}

	// Check if HA group needs to be changed
	if currentResource.Group == newHAGroup {
		logging.Debugf("VM %d HA group '%s' is still appropriate after CD-ROM detachment", vmid, currentResource.Group)
		return nil
	}

	logging.Infof("VM %d HA group needs update after CD-ROM detachment: '%s' -> '%s'", vmid, currentResource.Group, newHAGroup)

	// Update the HA resource with new group
	updatedResource := *currentResource
	updatedResource.Group = newHAGroup

	if err := s.proxmox.UpdateClusterHAResource(updatedResource); err != nil {
		return fmt.Errorf("failed to update HA resource group: %w", err)
	}

	logging.Infof("Successfully updated VM %d HA group from '%s' to '%s' after CD-ROM detachment", vmid, currentResource.Group, newHAGroup)
	return nil
}
