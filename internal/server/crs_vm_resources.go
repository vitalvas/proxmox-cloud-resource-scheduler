package server

import (
	"fmt"
	"strings"

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
			var existingResource *proxmox.ClusterHAResource
			for _, resource := range resources {
				if resource.SID == sid {
					existingResource = &resource
					break
				}
			}

			if existingResource != nil {
				// Check if existing resource is disabled and needs to be started
				if existingResource.State == haStateDisabled {
					// Always switch disabled HA resources to 'started' state
					// Proxmox stops VMs when HA resource is disabled, so we want to re-enable them
					newState := haStateStarted

					logging.Infof("Updating disabled HA resource for VM %d (%s) from 'disabled' to '%s'", vm.VMID, vm.Name, newState)

					// Update the HA resource state
					updatedResource := *existingResource
					updatedResource.State = newState

					if err := s.proxmox.UpdateClusterHAResource(updatedResource); err != nil {
						return fmt.Errorf("failed to update HA resource state for %s (%s): %w", sid, vm.Name, err)
					}

					logging.Infof("Successfully updated HA resource for VM %d (%s) from 'disabled' to '%s'", vm.VMID, vm.Name, newState)
					createdCount++ // Count as a change
				} else {
					logging.Debugf("VM %d (%s) already has HA resource in state '%s', skipping", vm.VMID, vm.Name, existingResource.State)
				}
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

			// Determine appropriate HA group based on VM storage configuration
			haGroup, err := s.determineVMHAGroup(node.Node, vm.VMID)
			if err != nil {
				logging.Warnf("Failed to determine HA group for VM %d (%s), using pin group: %v", vm.VMID, vm.Name, err)
				haGroup = haGroupPin
			}

			// Create HA resource
			data := proxmox.ClusterHAResource{
				SID:         sid,
				Type:        haResourceType,
				Comment:     haResourceComment,
				MaxRelocate: proxmox.HAMaxRelocate,
				MaxRestart:  proxmox.HAMaxRestart,
				Group:       haGroup,
				State:       haState,
			}

			if _, err := s.proxmox.CreateClusterHAResource(data); err != nil {
				return fmt.Errorf("failed to create HA resource for %s (%s): %w", sid, vm.Name, err)
			}

			logging.Infof("Created HA resource for VM %d (%s) on node %s with state %s and assigned to HA group %s", vm.VMID, vm.Name, node.Node, haState, haGroup)
			createdCount++

			// Sleep to avoid overwhelming the Proxmox API
			s.rateLimitSleep()
		}
	}

	if createdCount > 0 {
		logging.Infof("Created or updated %d HA resources for VMs", createdCount)
	} else {
		logging.Debug("No HA resource changes needed for VMs")
	}

	return nil
}

// determineVMHAGroup determines the appropriate HA group for a VM based on its storage configuration and hardware devices
func (s *Server) determineVMHAGroup(nodeName string, vmid int) (string, error) {
	// Get VM configuration to analyze disk storage and hardware devices
	vmConfig, err := s.proxmox.GetVMConfig(nodeName, vmid)
	if err != nil {
		return "", fmt.Errorf("failed to get VM config: %w", err)
	}

	// Check if VM has hostpci devices (PCIe passthrough) - these require pinning to specific node
	if len(vmConfig.HostPCI) > 0 {
		pinGroup := tools.GetHAVMPinGroupName(nodeName)
		logging.Debugf("VM %d has hostpci devices (%v), must use pin group: %s", vmid, getHostPCIDeviceList(vmConfig.HostPCI), pinGroup)
		return pinGroup, nil
	}

	// Check if all VM disks are on shared storage
	allDisksShared, err := s.areAllVMDisksShared(vmConfig.Disks)
	if err != nil {
		return "", fmt.Errorf("failed to analyze VM storage: %w", err)
	}

	if allDisksShared {
		// All disks are on shared storage - use prefer group for better load distribution
		preferGroup := tools.GetHAVMPreferGroupName(nodeName)
		logging.Debugf("VM %d has all disks on shared storage, assigning to prefer group: %s", vmid, preferGroup)
		return preferGroup, nil
	}
	// At least one disk is on local/non-shared storage - use pin group to keep VM on this node
	pinGroup := tools.GetHAVMPinGroupName(nodeName)
	logging.Debugf("VM %d has local storage, assigning to pin group: %s", vmid, pinGroup)
	return pinGroup, nil
}

// getHostPCIDeviceList returns a list of hostpci device keys for logging
func getHostPCIDeviceList(hostpci map[string]string) []string {
	devices := make([]string, 0, len(hostpci))
	for device := range hostpci {
		devices = append(devices, device)
	}
	return devices
}

// areAllVMDisksShared checks if all VM storage devices (including CD-ROM) are on shared storage
func (s *Server) areAllVMDisksShared(disks map[string]string) (bool, error) {
	if len(disks) == 0 {
		// No storage devices found, consider as shared (edge case)
		return true, nil
	}

	// Get all storage information
	storages, err := s.proxmox.GetStorage()
	if err != nil {
		return false, fmt.Errorf("failed to get storage info: %w", err)
	}

	// Create a map of storage name to shared status for quick lookup
	storageSharedMap := make(map[string]bool)
	for _, storage := range storages {
		storageSharedMap[storage.Storage] = storage.Shared == 1
	}

	// Check each storage device (including CD-ROM drives)
	for diskKey, diskValue := range disks {
		// Skip non-storage entries
		if !s.isDiskEntry(diskKey) {
			continue
		}

		// Extract storage name from disk configuration
		storageName := s.extractStorageFromDiskConfig(diskValue)
		if storageName == "" {
			// Skip CD-ROM devices with no media (none,media=cdrom) - these don't affect storage analysis
			if strings.HasPrefix(diskValue, "none,") {
				logging.Debugf("Skipping CD-ROM device %s with no media: %s", diskKey, diskValue)
				continue
			}
			logging.Warnf("Could not extract storage name from storage device config: %s=%s", diskKey, diskValue)
			continue
		}

		// Check if this storage is shared
		isShared, exists := storageSharedMap[storageName]
		if !exists {
			logging.Warnf("Storage %s not found in cluster storage list", storageName)
			// Assume non-shared if we can't find the storage
			return false, nil
		}

		if !isShared {
			// Found at least one storage device on non-shared storage
			logging.Debugf("Storage device %s on storage %s is not shared", diskKey, storageName)
			return false, nil
		}
	}

	// All storage devices are on shared storage
	return true, nil
}

// isDiskEntry checks if a configuration key represents any storage device
func (s *Server) isDiskEntry(key string) bool {
	// Proxmox storage device keys: virtio0, virtio1, sata0, sata1, scsi0, scsi1, ide0, ide1, ide2, etc.
	// Include ALL storage devices including CD-ROM drives
	storageDevicePrefixes := []string{"virtio", "sata", "scsi", "ide"}

	for _, prefix := range storageDevicePrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

// extractStorageFromDiskConfig extracts the storage name from a disk configuration string
func (s *Server) extractStorageFromDiskConfig(diskConfig string) string {
	// Disk config format examples:
	// "local-lvm:vm-100-disk-0,size=32G"
	// "ceph-storage:vm-100-disk-1,size=100G,format=raw"
	// "local:100/vm-100-disk-0.qcow2"
	// "none,media=cdrom" (CD-ROM with no media inserted)

	// Handle CD-ROM with no media first - "none" is not a real storage
	if strings.HasPrefix(diskConfig, "none,") {
		return ""
	}

	parts := strings.Split(diskConfig, ":")
	if len(parts) >= 1 {
		return parts[0]
	}

	return ""
}
