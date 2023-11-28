package app

import (
	"fmt"
	"log"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/tools"
)

func (app *App) SetupCRSQemu() error {
	resources, err := app.proxmox.ClusterHAResourcesList()
	if err != nil {
		return err
	}

	nodeList, err := app.proxmox.NodeList()
	if err != nil {
		return err
	}

	for _, node := range nodeList {
		haGroupPin := tools.GetHAPinGroupName(node.Node)

		qemuList, err := app.proxmox.NodeQEMUList(node)
		if err != nil {
			return err
		}

		for _, vm := range qemuList {
			if vm.Template == 1 {
				continue
			}

			if strings.Contains(vm.Tags, "crs-skip") {
				continue
			}

			sid := fmt.Sprintf("vm:%d", vm.VMID)
			haveResource := false

			for _, resource := range resources {
				if resource.SID == sid && !haveResource {
					haveResource = true
				}
			}

			if haveResource {
				continue
			}

			data := proxmox.ClusterHAResources{
				SID:         sid,
				Type:        "vm",
				Comment:     "crs-managed",
				MaxRelocate: proxmox.HAMaxRelocate,
				MaxRestart:  proxmox.HAMaxRestart,
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

			if err := app.proxmox.ClusterHAResourcesCreate(data); err != nil {
				return fmt.Errorf("failed to create ha resource for %s: %s", sid, err)
			}

			log.Println("add ha resource for", sid, vm.Name)
		}
	}

	return nil
}
