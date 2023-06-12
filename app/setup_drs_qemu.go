package app

import (
	"fmt"
	"log"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

func (app *App) SetupDRSQemu() error {
	resources, err := app.proxmox.ClusterHAResourcesList()
	if err != nil {
		return err
	}

	nodeList, err := app.proxmox.NodeList()
	if err != nil {
		return err
	}

	for _, node := range nodeList {
		haGroupPin := fmt.Sprintf("drs-pin-node-%s", strings.ToLower(node.Node))

		qemuList, err := app.proxmox.NodeQEMUList(node)
		if err != nil {
			return err
		}

		for _, vm := range qemuList {
			if vm.Template == 1 {
				continue
			}

			sid := fmt.Sprintf("vm:%d", vm.VMID)
			haveResource := false

			for _, resource := range resources {
				if resource.SID == sid {
					haveResource = true
					break
				}
			}

			data := proxmox.ClusterHAResources{
				SID:         sid,
				Type:        "vm",
				Comment:     "drs-managed",
				MaxRelocate: 10,
				MaxRestart:  10,
				Group:       haGroupPin,
			}

			switch vm.Status {
			case "running":
				data.State = "started"

			case "stopped":
				data.State = "stopped"

			default:
				data.State = "ignored"
			}

			if !haveResource {
				app.proxmox.ClusterHAResourcesCreate(data)

				log.Println("add ha resource for", sid, vm.Name)
			}
		}
	}

	return nil
}
