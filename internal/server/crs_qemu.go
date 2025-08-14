package server

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/tools"
)

func (s *Server) SetupCRSQemu() error {
	resources, err := s.proxmox.GetClusterHAResources()
	if err != nil {
		return err
	}

	nodeList, err := s.proxmox.GetNodes()
	if err != nil {
		return err
	}

	for _, node := range nodeList {
		haGroupPin := tools.GetHAVMPinGroupName(node.Node)

		vmList, err := s.proxmox.GetNodeVMs(node.Node)
		if err != nil {
			return err
		}

		for _, vm := range vmList {
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

			data := proxmox.ClusterHAResource{
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

			if _, err := s.proxmox.CreateClusterHAResource(data); err != nil {
				return fmt.Errorf("failed to create ha resource for %s: %s", sid, err)
			}

			log.Println("add ha resource for", sid, vm.Name)

			// Sleep 500ms to avoid overwhelming the Proxmox API
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}
