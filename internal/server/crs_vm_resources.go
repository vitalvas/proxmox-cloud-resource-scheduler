package server

import (
	"fmt"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/tools"
)

// SetupVMHAResources creates HA resources for VMs that don't have them
func (s *Server) SetupVMHAResources() error {
	logging.Debug("Setting up HA resources for VMs")

	// Get existing HA resources to avoid duplicates
	resources, err := s.proxmox.GetClusterHAResources()
	if err != nil {
		return fmt.Errorf("failed to get cluster HA resources: %w", err)
	}

	// Get all nodes to process their VMs
	nodeList, err := s.proxmox.GetNodes()
	if err != nil {
		return fmt.Errorf("failed to get nodes: %w", err)
	}

	var createdCount int

	for _, node := range nodeList {
		haGroupPin := tools.GetHAVMPinGroupName(node.Node)

		vmList, err := s.proxmox.GetNodeVMs(node.Node)
		if err != nil {
			return fmt.Errorf("failed to get VMs for node %s: %w", node.Node, err)
		}

		for _, vm := range vmList {
			// Skip templates
			if vm.Template == vmTemplateFlag {
				logging.Debugf("Skipping template VM %d (%s) on node %s", vm.VMID, vm.Name, node.Node)
				continue
			}

			// Skip VMs with crs-skip tag
			if s.hasVMSkipTag(vm.Tags) {
				logging.Debugf("Skipping VM %d (%s) with crs-skip tag on node %s", vm.VMID, vm.Name, node.Node)
				continue
			}

			sid := fmt.Sprintf("%s:%d", haResourceType, vm.VMID)

			// Check if VM already has HA resource
			haveResource := false
			for _, resource := range resources {
				if resource.SID == sid {
					haveResource = true
					break
				}
			}

			if haveResource {
				logging.Debugf("VM %d (%s) already has HA resource, skipping", vm.VMID, vm.Name)
				continue
			}

			// Determine initial HA state based on VM status
			var haState string
			switch vm.Status {
			case vmStatusRunning:
				haState = haStateStarted
			case vmStatusStopped:
				haState = haStateStopped
			default:
				haState = haStateIgnored
			}

			// Create HA resource
			data := proxmox.ClusterHAResource{
				SID:         sid,
				Type:        haResourceType,
				Comment:     haResourceComment,
				MaxRelocate: proxmox.HAMaxRelocate,
				MaxRestart:  proxmox.HAMaxRestart,
				Group:       haGroupPin,
				State:       haState,
			}

			if _, err := s.proxmox.CreateClusterHAResource(data); err != nil {
				return fmt.Errorf("failed to create HA resource for %s (%s): %w", sid, vm.Name, err)
			}

			logging.Infof("Created HA resource for VM %d (%s) on node %s with state %s", vm.VMID, vm.Name, node.Node, haState)
			createdCount++

			// Sleep to avoid overwhelming the Proxmox API
			s.rateLimitSleep()
		}
	}

	if createdCount > 0 {
		logging.Infof("Created %d new HA resources for VMs", createdCount)
	} else {
		logging.Debug("No new HA resources needed for VMs")
	}

	return nil
}
