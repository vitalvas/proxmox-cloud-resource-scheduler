package server

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/tools"
)

// SetupVMPin creates HA groups for VM pinning to specific nodes
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
		expectedNodes := fmt.Sprintf("%s:%d", node.Node, crsMaxNodePriority)

		var existingGroup *proxmox.ClusterHAGroup
		for _, group := range haGroups {
			if haGroupPin == group.Group {
				existingGroup = &group
				break
			}
		}

		if existingGroup == nil {
			logging.Infof("creating ha group %s", haGroupPin)

			_, err := s.proxmox.CreateClusterHAGroup(proxmox.ClusterHAGroup{
				Group:      haGroupPin,
				Nodes:      expectedNodes,
				NoFailback: 1,
				Restricted: 1,
			})
			if err != nil {
				return fmt.Errorf("failed to create ha group %s: %s", haGroupPin, err)
			}
		} else if !s.compareNodeConfiguration(existingGroup.Nodes, expectedNodes) {
			// Check if existing group has correct configuration
			logging.Infof("updating ha group %s: before=%q, after=%q", haGroupPin, existingGroup.Nodes, expectedNodes)

			err := s.proxmox.UpdateClusterHAGroup(proxmox.ClusterHAGroup{
				Group:      haGroupPin,
				Nodes:      expectedNodes,
				NoFailback: 1,
				Restricted: 1,
			})
			if err != nil {
				return fmt.Errorf("failed to update ha group %s: %s", haGroupPin, err)
			}
		}
	}

	return nil
}

// SetupVMPrefer creates HA groups for VM preference with shared storage
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

		// Generate expected node configuration
		expectedNodes := s.generateExpectedPreferNodes(nodes, node.Node)

		var existingGroup *proxmox.ClusterHAGroup
		for _, group := range haGroups {
			if haGroupPrefer == group.Group {
				existingGroup = &group
				break
			}
		}

		if existingGroup == nil {
			logging.Infof("creating ha group %s", haGroupPrefer)

			if _, err := s.proxmox.CreateClusterHAGroup(proxmox.ClusterHAGroup{
				Group:      haGroupPrefer,
				Nodes:      expectedNodes,
				NoFailback: 1,
				Restricted: 1,
			}); err != nil {
				return fmt.Errorf("failed to create ha group %s: %s", haGroupPrefer, err)
			}
		} else if !s.compareNodeConfiguration(existingGroup.Nodes, expectedNodes) {
			// Check if existing group has correct configuration
			logging.Infof("updating ha group %s: before=%q, after=%q", haGroupPrefer, existingGroup.Nodes, expectedNodes)

			err := s.proxmox.UpdateClusterHAGroup(proxmox.ClusterHAGroup{
				Group:      haGroupPrefer,
				Nodes:      expectedNodes,
				NoFailback: 1,
				Restricted: 1,
			})
			if err != nil {
				return fmt.Errorf("failed to update ha group %s: %s", haGroupPrefer, err)
			}
		}
	}

	return nil
}

// generateExpectedPreferNodes generates the expected node configuration string for a prefer group
func (s *Server) generateExpectedPreferNodes(nodes []proxmox.Node, preferredNodeName string) string {
	groupNodes := make([]string, 0, len(nodes))

	// Create a copy of nodes for sorting to ensure consistent ordering
	sortedNodes := make([]proxmox.Node, len(nodes))
	copy(sortedNodes, nodes)
	sort.Slice(sortedNodes, func(i, j int) bool {
		return sortedNodes[i].Node < sortedNodes[j].Node
	})

	// Find the index of the preferred node
	preferredIndex := -1
	for i, n := range sortedNodes {
		if n.Node == preferredNodeName {
			preferredIndex = i
			break
		}
	}

	// Assign priorities using round-robin starting from preferred node
	for i, n := range sortedNodes {
		var priority int
		if n.Node == preferredNodeName {
			// Preferred node gets maximum priority
			priority = crsMaxNodePriority
		} else {
			// Calculate round-robin position relative to preferred node
			relativePosition := (i - preferredIndex + len(sortedNodes)) % len(sortedNodes)
			if relativePosition == 0 {
				relativePosition = len(sortedNodes) // Move preferred node to end for calculation
			}
			// Start from max priority and decrement by 5 for each position
			priority = crsMaxNodePriority - (relativePosition * 5)
			// Ensure priority doesn't go below minimum
			if priority < crsMinNodePriority {
				priority = crsMinNodePriority
			}
		}

		groupNodes = append(groupNodes,
			fmt.Sprintf("%s:%d", n.Node, priority),
		)
	}

	sort.Strings(groupNodes)
	return strings.Join(groupNodes, ",")
}

// compareNodeConfiguration compares two node configuration strings by normalizing them
func (s *Server) compareNodeConfiguration(existing, expected string) bool {
	// If they are exactly equal, no need to parse
	if existing == expected {
		return true
	}

	// Parse and normalize both configurations
	existingNormalized := s.normalizeNodeConfiguration(existing)
	expectedNormalized := s.normalizeNodeConfiguration(expected)

	return existingNormalized == expectedNormalized
}

// normalizeNodeConfiguration takes a node configuration string and returns a normalized version
func (s *Server) normalizeNodeConfiguration(nodeConfig string) string {
	if nodeConfig == "" {
		return ""
	}

	// Split nodes by comma
	nodes := strings.Split(nodeConfig, ",")

	// Trim spaces and sort
	for i, node := range nodes {
		nodes[i] = strings.TrimSpace(node)
	}
	sort.Strings(nodes)

	return strings.Join(nodes, ",")
}

// generateActualHAGroupNames returns the expected HA group names based on current cluster state
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

// removeVMsFromHAGroup removes all VMs from a specific HA group
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

			// Sleep to avoid overwhelming the Proxmox API
			s.rateLimitSleep()
		}
	}

	return nil
}

// CleanupOrphanedHAGroups removes HA groups that are no longer needed
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
		// Check if group has CRS prefix and is not in actual groups
		if strings.HasPrefix(group.Group, crsGroupPrefix) && !actualGroups[group.Group] {
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
