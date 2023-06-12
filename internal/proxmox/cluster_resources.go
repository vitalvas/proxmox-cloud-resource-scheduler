package proxmox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ClusterResource struct {
	CPU        float64 `json:"cpu"`
	CGroupMode int     `json:"cgroup-mode"`
	Content    string  `json:"content"`
	Disk       int     `json:"disk"`
	DiskRead   int     `json:"diskread"`
	DiskWrite  int     `json:"diskwrite"`
	HAState    string  `json:"hastate"`
	ID         string  `json:"id"`
	Level      string  `json:"level"`
	MaxCPU     int     `json:"maxcpu"`
	MaxDisk    int     `json:"maxdisk"`
	MaxMem     int     `json:"maxmem"`
	Mem        int     `json:"mem"`
	Name       string  `json:"name"`
	NetIn      int     `json:"netin"`
	NetOut     int     `json:"netout"`
	Node       string  `json:"node"`
	PluginType string  `json:"plugintype"`
	Pool       string  `json:"pool"`
	SDN        string  `json:"sdn"`
	Shared     int     `json:"shared"`
	Status     string  `json:"status"`
	Storage    string  `json:"storage"`
	Tags       string  `json:"tags"`
	Template   int     `json:"template"`
	Type       string  `json:"type"`
	Uptime     int     `json:"uptime"`
	VMID       int     `json:"vmid"`
}

func (cr *ClusterResource) GetTags() []string {
	return strings.Split(cr.Tags, ";")
}

func (p *Proxmox) ClutserResourceList() ([]ClusterResource, error) {
	resp, err := p.makeHTTPRequest(http.MethodGet, "cluster/resources", nil)
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
		Data []ClusterResource `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &tmp); err != nil {
		return nil, err
	}

	return tmp.Data, nil
}
