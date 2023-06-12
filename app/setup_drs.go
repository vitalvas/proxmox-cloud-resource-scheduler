package app

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/app/proxmox"
)

const (
	drsMaxNodePriority = 1000
	drsMinNodePriority = 1
)

func (app *App) SetupDRS() {
	hasSharedStorage := app.proxmox.HasSharedStorage()

	haGroups := app.proxmox.ClusterHAGroupList()
	nodes := app.proxmox.NodeList()

	actualHaGroups := make(map[string]bool)

	for _, row := range nodes {
		createdPin := false
		createdPrefer := false

		haGroupPin := fmt.Sprintf("drs-pin-node-%s", strings.ToLower(row.Node))
		haGroupPrefer := fmt.Sprintf("drs-prefer-node-%s", strings.ToLower(row.Node))

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
				Nodes:      fmt.Sprintf("%s:%d", row.Node, drsMaxNodePriority),
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
						fmt.Sprintf("%s:%d", node.Node, drsMaxNodePriority),
					)
				} else {
					groupNodes = append(groupNodes,
						fmt.Sprintf("%s:%d", node.Node, drsMinNodePriority),
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

	for _, row := range app.proxmox.ClusterHAGroupList() {
		if strings.HasPrefix(row.Group, "drs-pin-node-") ||
			strings.HasPrefix(row.Group, "drs-prefer-node-") {

			if _, exists := actualHaGroups[row.Group]; !exists {
				log.Println("deleting ha group", row.Group)

				app.proxmox.ClusterHAGroupDelete(proxmox.ClusterHAGroup{Group: row.Group})
			}

		}
	}

}
