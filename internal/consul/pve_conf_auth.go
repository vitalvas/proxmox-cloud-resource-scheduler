package consul

import (
	"encoding/json"
	"fmt"
)

type proxmoxAuth struct {
	User  string `json:"user"`
	Token string `json:"token"`
}

func (c *Consul) GetPVEAuthToken() (string, error) {
	kv := c.client.KV()

	pair, _, err := kv.Get("crs/config/proxmox/auth", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get auth token from consul: %w", err)
	}

	if pair == nil {
		return "", fmt.Errorf("auth token not found in consul")
	}

	var auth proxmoxAuth

	if err = json.Unmarshal(pair.Value, &auth); err != nil {
		return "", fmt.Errorf("failed to unmarshal auth token: %w", err)
	}

	return fmt.Sprintf("%s!%s", auth.User, auth.Token), nil
}
