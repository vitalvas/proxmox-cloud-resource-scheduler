package app

import (
	"fmt"
	"log"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/app/proxmox"
)

func (this *App) SetupDRSQemu() {
	resources := this.proxmox.ClusterHAResourcesList()

	for _, node := range this.proxmox.NodeList() {
		haGroupPin := fmt.Sprintf("drs-pin-node-%s", strings.ToLower(node.Node))

		for _, vm := range this.proxmox.NodeQEMUList(node) {
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
				this.proxmox.ClusterHAResourcesCreate(data)

				log.Println("add ha resource for", sid, vm.Name)
			}
		}
	}
}
