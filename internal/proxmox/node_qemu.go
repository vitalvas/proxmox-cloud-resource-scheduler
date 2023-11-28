package proxmox

import (
	"encoding/json"
	"fmt"
	"io"
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
	Tags      string  `json:"tags"`
}

func (p *Proxmox) NodeQEMUList(node Node) ([]NodeQEMU, error) {
	path := fmt.Sprintf("nodes/%s/qemu", node.Node)

	resp, err := p.makeHTTPRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wrong status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tmp struct {
		Data []NodeQEMU `json:"data"`
	}

	json.Unmarshal(bodyBytes, &tmp)

	return tmp.Data, nil
}
