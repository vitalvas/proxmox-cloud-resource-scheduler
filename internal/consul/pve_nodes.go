package consul

import (
	"fmt"

	"github.com/hashicorp/consul/api"
)

const pveServiceName = "proxmox-pve"

func (c *Consul) GetPVENodesURL() ([]string, error) {
	entries, _, err := c.client.Health().Service(pveServiceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get service '%s' error: %w", pveServiceName, err)
	}

	passingEntries := make([]*api.ServiceEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Checks.AggregatedStatus() == api.HealthPassing {
			passingEntries = append(passingEntries, entry)
		}
	}

	resp := make([]string, 0, len(passingEntries))

	if passingEntries == nil {
		return nil, fmt.Errorf("no healthy nodes found for service '%s'", pveServiceName)
	}

	for _, entry := range passingEntries {
		resp = append(resp, fmt.Sprintf("https://%s:%d", entry.Node.Address, entry.Service.Port))
	}

	return resp, nil
}
