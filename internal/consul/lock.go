package consul

import "github.com/hashicorp/consul/api"

func (c *Consul) GetLock(key string) (*api.Lock, error) {
	session := &api.SessionEntry{
		Name:     "proxmox-cloud-resource-scheduler",
		Behavior: api.SessionBehaviorDelete,
		TTL:      "30s",
	}

	sessionID, _, err := c.client.Session().Create(session, nil)
	if err != nil {
		return nil, err
	}

	opts := &api.LockOptions{
		Key:     key,
		Session: sessionID,
	}

	lock, err := c.client.LockOpts(opts)
	if err != nil {
		return nil, err
	}

	return lock, nil
}
