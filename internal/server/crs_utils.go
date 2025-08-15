package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

// rateLimitSleep applies API rate limiting unless disabled for testing
func (s *Server) rateLimitSleep() {
	if !s.disableRateLimit {
		time.Sleep(apiRateLimit)
	}
}

// hasVMSkipTag checks if a VM has the crs-skip tag
func (s *Server) hasVMSkipTag(vmTags string) bool {
	if vmTags == "" {
		return false
	}

	tags := strings.Split(vmTags, ";")
	for _, tag := range tags {
		if strings.TrimSpace(tag) == crsSkipTag {
			return true
		}
	}
	return false
}

// hasVMCriticalTag checks if a VM has the crs-critical tag
func (s *Server) hasVMCriticalTag(vmTags string) bool {
	if vmTags == "" {
		return false
	}

	tags := strings.Split(vmTags, ";")
	for _, tag := range tags {
		if strings.TrimSpace(tag) == crsCriticalTag {
			return true
		}
	}
	return false
}

// RemoveSkippedVMsFromCRSGroups removes VMs with crs-skip tag from CRS HA groups
func (s *Server) RemoveSkippedVMsFromCRSGroups() error {
	logging.Debug("Checking for VMs with crs-skip tag in CRS HA groups")

	resources, err := s.proxmox.GetClusterResources()
	if err != nil {
		return fmt.Errorf("failed to get cluster resources: %w", err)
	}

	haResources, err := s.proxmox.GetClusterHAResources()
	if err != nil {
		return fmt.Errorf("failed to get HA resources: %w", err)
	}

	for _, haResource := range haResources {
		if !strings.HasPrefix(haResource.Group, crsGroupPrefix) {
			continue
		}

		var vmTags string
		for _, resource := range resources {
			resourceSID := fmt.Sprintf("vm:%d", resource.VMID)
			if haResource.SID == resourceSID {
				vmTags = resource.Tags
				break
			}
		}

		if s.hasVMSkipTag(vmTags) {
			logging.Infof("Removing VM %s with crs-skip tag from CRS HA group %s", haResource.SID, haResource.Group)
			if err := s.proxmox.DeleteClusterHAResource(haResource.SID); err != nil {
				return fmt.Errorf("failed to remove VM %s from HA group %s: %w", haResource.SID, haResource.Group, err)
			}

			s.rateLimitSleep()
		}
	}

	return nil
}

// ensureCRSTagRegistered ensures CRS tags are registered in cluster options
func (s *Server) ensureCRSTagRegistered() error {
	logging.Debug("Getting cluster options to check registered tags")
	options, err := s.proxmox.GetClusterOptions()
	if err != nil {
		logging.Debugf("Failed to get cluster options (this may be expected on older Proxmox versions): %v", err)
		return fmt.Errorf("failed to get cluster options: %w", err)
	}

	logging.Debugf("Current registered tags: %v", options.RegisteredTags)

	// Define CRS tags that need to be registered
	crsTagsToRegister := []string{crsSkipTag, crsCriticalTag}

	// Check which CRS tags are missing
	var missingTags []string
	for _, crsTag := range crsTagsToRegister {
		found := false
		for _, existingTag := range options.RegisteredTags {
			if strings.TrimSpace(existingTag) == crsTag {
				found = true
				break
			}
		}
		if !found {
			missingTags = append(missingTags, crsTag)
		}
	}

	// If all tags are already registered, nothing to do
	if len(missingTags) == 0 {
		logging.Debug("All CRS tags are already registered")
		return nil
	}

	// Add missing CRS tags to registered tags array
	newRegisteredTags := make([]string, len(options.RegisteredTags))
	copy(newRegisteredTags, options.RegisteredTags)
	newRegisteredTags = append(newRegisteredTags, missingTags...)

	logging.Debugf("New registered tags array: %v", newRegisteredTags)

	// Update cluster options with new registered tags
	updateOptions := proxmox.ClusterOptions{
		RegisteredTags: newRegisteredTags,
	}

	logging.Debug("Updating cluster options with new registered tags")
	if err := s.proxmox.UpdateClusterOptions(updateOptions); err != nil {
		logging.Debugf("Failed to update cluster options (this may be expected on older Proxmox versions or limited permissions): %v", err)
		return fmt.Errorf("failed to update cluster options with CRS tag: %w", err)
	}

	logging.Infof("Registered CRS tags %v in cluster options", missingTags)
	return nil
}
