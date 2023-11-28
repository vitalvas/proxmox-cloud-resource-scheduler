package consul

import (
	"github.com/hashicorp/consul/api"
)

type Consul struct {
	client *api.Client
}

func New() (*Consul, error) {
	consulConfig := api.DefaultConfig()

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	return &Consul{
		client: client,
	}, nil
}
