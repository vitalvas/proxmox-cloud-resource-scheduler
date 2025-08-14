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
