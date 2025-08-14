package server

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/tools"
)

const (
	crsMaxNodePriority = 1000
	crsMinNodePriority = 1
	crsSkipTag         = "crs-skip"
)

func (s *Server) SetupCRS() error {
	// Try to register CRS tag, but don't fail if it doesn't work
	if err := s.ensureCRSTagRegistered(); err != nil {
		logging.Warnf("Failed to register CRS tag (this may be expected): %v", err)
	}

	if err := s.SetupVMPin(); err != nil {
		return fmt.Errorf("setup VM pin: %w", err)
	}

	if err := s.SetupVMPrefer(); err != nil {
		return fmt.Errorf("setup VM prefer: %w", err)
	}

	if err := s.CleanupOrphanedHAGroups(); err != nil {
		return fmt.Errorf("cleanup orphaned HA groups: %w", err)
	}

	return nil
}

func (s *Server) SetupVMPin() error {
	haGroups, err := s.proxmox.GetClusterHAGroups()
	if err != nil {
		return err
	}

	nodes, err := s.proxmox.GetNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		haGroupPin := tools.GetHAVMPinGroupName(node.Node)
		groupExists := false

		for _, group := range haGroups {
			if haGroupPin == group.Group {
				groupExists = true
				break
			}
		}

		if !groupExists {
			logging.Infof("creating ha group %s", haGroupPin)

			_, err := s.proxmox.CreateClusterHAGroup(proxmox.ClusterHAGroup{
				Group:      haGroupPin,
				Nodes:      fmt.Sprintf("%s:%d", node.Node, crsMaxNodePriority),
				NoFailback: 1,
				Restricted: 1,
			})
			if err != nil {
				return fmt.Errorf("failed to create ha group %s: %s", haGroupPin, err)
			}
		}
	}

	return nil
}

func (s *Server) SetupVMPrefer() error {
	hasSharedStorage, err := s.proxmox.HasSharedStorage()
	if err != nil {
		return err
	}

	if !hasSharedStorage {
		return nil
	}

	haGroups, err := s.proxmox.GetClusterHAGroups()
	if err != nil {
		return err
	}

	nodes, err := s.proxmox.GetNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		haGroupPrefer := tools.GetHAVMPreferGroupName(node.Node)
		groupExists := false

		for _, group := range haGroups {
			if haGroupPrefer == group.Group {
				groupExists = true
				break
			}
		}

		if !groupExists {
			logging.Infof("creating ha group %s", haGroupPrefer)

			var groupNodes []string

			for _, n := range nodes {
				if n.Node == node.Node {
					groupNodes = append(groupNodes,
						fmt.Sprintf("%s:%d", n.Node, crsMaxNodePriority),
					)
				} else {
					groupNodes = append(groupNodes,
						fmt.Sprintf("%s:%d", n.Node, crsMinNodePriority),
					)
				}
			}

			sort.Strings(groupNodes)

			if _, err := s.proxmox.CreateClusterHAGroup(proxmox.ClusterHAGroup{
				Group:      haGroupPrefer,
				Nodes:      strings.Join(groupNodes, ","),
				NoFailback: 1,
				Restricted: 1,
			}); err != nil {
				return fmt.Errorf("failed to create ha group %s: %s", haGroupPrefer, err)
			}
		}
	}

	return nil
}

func (s *Server) generateActualHAGroupNames() (map[string]bool, error) {
	actualGroups := make(map[string]bool)

	nodes, err := s.proxmox.GetNodes()
	if err != nil {
		return nil, err
	}

	// Always generate pin groups for all nodes
	for _, node := range nodes {
		pinGroup := tools.GetHAVMPinGroupName(node.Node)
		actualGroups[pinGroup] = true
	}

	// Generate prefer groups only if shared storage exists
	hasSharedStorage, err := s.proxmox.HasSharedStorage()
	if err != nil {
		return nil, err
	}

	if hasSharedStorage {
		for _, node := range nodes {
			preferGroup := tools.GetHAVMPreferGroupName(node.Node)
			actualGroups[preferGroup] = true
		}
	}

	return actualGroups, nil
}

func (s *Server) removeVMsFromHAGroup(groupName string) error {
	resources, err := s.proxmox.GetClusterHAResources()
	if err != nil {
		return err
	}

	for _, resource := range resources {
		if resource.Group == groupName {
			logging.Infof("removing HA resource %s from group %s", resource.SID, groupName)
			if err := s.proxmox.DeleteClusterHAResource(resource.SID); err != nil {
				return fmt.Errorf("failed to remove HA resource %s from group %s: %w", resource.SID, groupName, err)
			}

			// Sleep 500ms to avoid overwhelming the Proxmox API
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

func (s *Server) CleanupOrphanedHAGroups() error {
	actualGroups, err := s.generateActualHAGroupNames()
	if err != nil {
		return err
	}

	haGroups, err := s.proxmox.GetClusterHAGroups()
	if err != nil {
		return err
	}

	for _, group := range haGroups {
		// Check if group has 'crs-' prefix and is not in actual groups
		if strings.HasPrefix(group.Group, "crs-") && !actualGroups[group.Group] {
			logging.Infof("found orphaned HA group: %s", group.Group)

			// First, remove all VMs from the group
			if err := s.removeVMsFromHAGroup(group.Group); err != nil {
				return fmt.Errorf("failed to remove VMs from group %s: %w", group.Group, err)
			}

			// Then delete the group
			logging.Infof("deleting orphaned HA group: %s", group.Group)
			if err := s.proxmox.DeleteClusterHAGroup(group.Group); err != nil {
				return fmt.Errorf("failed to delete orphaned HA group %s: %w", group.Group, err)
			}
		}
	}

	return nil
}

func (s *Server) ensureCRSTagRegistered() error {
	logging.Debug("Getting cluster options to check registered tags")
	options, err := s.proxmox.GetClusterOptions()
	if err != nil {
		logging.Debugf("Failed to get cluster options (this may be expected on older Proxmox versions): %v", err)
		return fmt.Errorf("failed to get cluster options: %w", err)
	}

	logging.Debugf("Current registered tags: %v", options.RegisteredTags)

	// Check if crs-skip tag is already in registered tags array
	for _, tag := range options.RegisteredTags {
		if strings.TrimSpace(tag) == crsSkipTag {
			return nil
		}
	}

	// Add crs-skip to registered tags array
	newRegisteredTags := make([]string, len(options.RegisteredTags))
	copy(newRegisteredTags, options.RegisteredTags)
	newRegisteredTags = append(newRegisteredTags, crsSkipTag)

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

	logging.Infof("Registered CRS tag '%s' in cluster options", crsSkipTag)
	return nil
}
