package proxmox

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type Node struct {
	ID             string  `json:"id"`
	Node           string  `json:"node"`
	MaxDisk        uint64  `json:"maxdisk"`
	Disk           uint64  `json:"disk"`
	SSLFingerprint string  `json:"ssl_fingerprint"`
	CPU            float32 `json:"cpu"`
	MaxCPU         uint64  `json:"maxcpu"`
	Uptime         uint64  `json:"uptime"`
	Status         string  `json:"status"`
	Type           string  `json:"type"`
	Mem            uint64  `json:"mem"`
	MaxMem         uint64  `json:"maxmem"`
}

func (p *Proxmox) NodeList() []Node {
	resp, err := p.makeHTTPRequest(http.MethodGet, "nodes", nil)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var tmp struct {
			Data []Node `json:"data"`
		}

		json.Unmarshal(bodyBytes, &tmp)
		return tmp.Data
	} else {
		log.Fatal("wrong status code:", resp.StatusCode)
	}

	return nil
}
