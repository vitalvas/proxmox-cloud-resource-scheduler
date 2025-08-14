package server

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/tools"
)

const (
	crsMaxNodePriority = 1000
	crsMinNodePriority = 1
)

func (s *Server) SetupCRS() error {
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
			log.Println("creating ha group", haGroupPin)

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
			log.Println("creating ha group", haGroupPrefer)

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
			log.Printf("removing HA resource %s from group %s", resource.SID, groupName)
			if err := s.proxmox.DeleteClusterHAResource(resource.SID); err != nil {
				return fmt.Errorf("failed to remove HA resource %s from group %s: %w", resource.SID, groupName, err)
			}
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
			log.Printf("found orphaned HA group: %s", group.Group)

			// First, remove all VMs from the group
			if err := s.removeVMsFromHAGroup(group.Group); err != nil {
				return fmt.Errorf("failed to remove VMs from group %s: %w", group.Group, err)
			}

			// Then delete the group
			log.Printf("deleting orphaned HA group: %s", group.Group)
			if err := s.proxmox.DeleteClusterHAGroup(group.Group); err != nil {
				return fmt.Errorf("failed to delete orphaned HA group %s: %w", group.Group, err)
			}
		}
	}

	return nil
}
