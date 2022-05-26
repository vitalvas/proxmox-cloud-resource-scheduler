package proxmox

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type NodeQEMU struct {
	VMID      uint    `json:"vmid"`
	Name      string  `json:"name"`
	NetIN     uint64  `json:"netin"`
	NetOUT    uint64  `json:"netout"`
	CPU       float32 `json:"cpu"`
	Uptime    uint64  `json:"uptime"`
	Diskread  uint64  `json:"diskread"`
	Diskwrite uint64  `json:"diskwrite"`
	Pid       uint64  `json:"pid"`
	Disk      uint64  `json:"disk"`
	Status    string  `json:"status"`
	Maxmem    uint64  `json:"maxmem"`
	Maxdisk   uint64  `json:"maxdisk"`
	Mem       uint64  `json:"mem"`
	CPUs      uint    `json:"cpus"`
	Lock      string  `json:"lock"`
	Template  uint    `json:"template"`
}

func (this *Proxmox) NodeQEMUList(node Node) []NodeQEMU {
	path := fmt.Sprintf("nodes/%s/qemu", node.Node)

	resp, err := this.makeHTTPRequest(http.MethodGet, path, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatal("wrong status code:", resp.StatusCode)
		return nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var tmp struct {
		Data []NodeQEMU `json:"data"`
	}

	json.Unmarshal(bodyBytes, &tmp)
	return tmp.Data
}
