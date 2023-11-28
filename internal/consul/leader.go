package consul

import (
	"fmt"

	"github.com/hashicorp/consul/api"
)

func (c *Consul) GetLeader(key string, interval int) (bool, string, error) {
	session := &api.SessionEntry{
		Name:     fmt.Sprintf("proxmox-crs-leader-%s", key),
		Behavior: api.SessionBehaviorDelete,
		TTL:      fmt.Sprintf("%ds", interval),
	}

	sessionID, _, err := c.client.Session().Create(session, nil)
	if err != nil {
		return false, "", err
	}

	isLeader, _, err := c.client.KV().Acquire(&api.KVPair{
		Key:     fmt.Sprintf("crs/_internal/leader/%s", key),
		Value:   []byte(sessionID),
		Session: sessionID,
	}, nil)
	if err != nil {
		return false, "", err
	}

	return isLeader, sessionID, nil
}

func (c *Consul) RenewLeader(sessionID string, interval int) error {
	return c.client.Session().RenewPeriodic(
		fmt.Sprintf("%ds", interval),
		sessionID,
		nil,
		nil,
	)
}
