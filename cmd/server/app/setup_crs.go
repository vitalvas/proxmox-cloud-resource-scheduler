package app

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

const (
	crsMaxNodePriority = 1000
	crsMinNodePriority = 1
)

func (app *App) SetupCRS() error {
	hasSharedStorage, err := app.proxmox.HasSharedStorage()
	if err != nil {
		return err
	}

	haGroups, err := app.proxmox.ClusterHAGroupList()
	if err != nil {
		return err
	}

	nodes, err := app.proxmox.NodeList()
	if err != nil {
		return err
	}

	actualHaGroups := make(map[string]bool)

	for _, row := range nodes {
		createdPin := false
		createdPrefer := false

		haGroupPin := fmt.Sprintf("crs-pin-node-%s", strings.ToLower(row.Node))
		haGroupPrefer := fmt.Sprintf("crs-prefer-node-%s", strings.ToLower(row.Node))

		actualHaGroups[haGroupPin] = true

		if hasSharedStorage {
			actualHaGroups[haGroupPrefer] = true
		}

		for _, group := range haGroups {
			if !createdPin && haGroupPin == group.Group {
				createdPin = true
			}

			if hasSharedStorage && !createdPrefer && haGroupPrefer == group.Group {
				createdPrefer = true
			}
		}

		if !createdPin {
			log.Println("creating ha group", haGroupPin)

			app.proxmox.ClusterHAGroupCreate(proxmox.ClusterHAGroup{
				Group:      haGroupPin,
				Nodes:      fmt.Sprintf("%s:%d", row.Node, crsMaxNodePriority),
				NoFailback: 1,
				Restricted: 1,
			})
		}

		if hasSharedStorage && !createdPrefer {
			log.Println("creating ha group", haGroupPrefer)

			var groupNodes []string

			for _, node := range nodes {
				if node.Node == row.Node {
					groupNodes = append(groupNodes,
						fmt.Sprintf("%s:%d", node.Node, crsMaxNodePriority),
					)
				} else {
					groupNodes = append(groupNodes,
						fmt.Sprintf("%s:%d", node.Node, crsMinNodePriority),
					)
				}
			}

			sort.Strings(groupNodes)

			app.proxmox.ClusterHAGroupCreate(proxmox.ClusterHAGroup{
				Group:      haGroupPrefer,
				Nodes:      strings.Join(groupNodes, ","),
				NoFailback: 1,
				Restricted: 1,
			})
		}
	}

	clusterHAGroupList, err := app.proxmox.ClusterHAGroupList()
	if err != nil {
		return err
	}
	for _, row := range clusterHAGroupList {
		if strings.HasPrefix(row.Group, "crs-pin-node-") ||
			strings.HasPrefix(row.Group, "crs-prefer-node-") {

			if _, exists := actualHaGroups[row.Group]; !exists {
				log.Println("deleting ha group", row.Group)

				app.proxmox.ClusterHAGroupDelete(proxmox.ClusterHAGroup{Group: row.Group})
			}

		}
	}

	return nil
}
