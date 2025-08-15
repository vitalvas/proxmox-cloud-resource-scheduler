package server

import (
	"fmt"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

// UpdateVMMeta updates VM metadata for critical VMs
func (s *Server) UpdateVMMeta() error {
	logging.Debug("Checking VMs for metadata updates")

	// Get cluster resources to find VMs that need metadata updates
	resources, err := s.proxmox.GetClusterResources()
	if err != nil {
		return fmt.Errorf("failed to get cluster resources: %w", err)
	}

	var criticalVMs []proxmox.ClusterResource

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
		}
	}

	if len(criticalVMs) == 0 {
		logging.Debug("No critical VMs found that need metadata updates")
		return nil
	}

	logging.Debugf("Found %d critical VMs that need metadata updates: %v", len(criticalVMs), s.extractVMIDs(criticalVMs))

	var updatedCount int

	for _, vm := range criticalVMs {
		if s.updateCriticalVMStartOrder(vm.Node, vm.VMID) {
			updatedCount++
		}
	}

	if updatedCount > 0 {
		logging.Infof("Updated startup order for %d critical VMs", updatedCount)
	} else {
		logging.Debug("No critical VMs required startup order updates")
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