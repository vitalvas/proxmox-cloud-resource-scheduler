package server

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/vitalvas/gokit/xcmd"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/consul"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	proxmox          *proxmox.Client
	consul           *consul.Consul
	disableRateLimit bool // For testing purposes
}

func New() (*Server, error) {
	consul, err := consul.New()
	if err != nil {
		return nil, err
	}

	// Get endpoints and token from consul
	endpoints, err := consul.GetPVENodesURL()
	if err != nil {
		return nil, err
	}

	token, err := consul.GetPVEAuthToken()
	if err != nil {
		return nil, err
	}

	// Create proxmox client configuration
	config := &proxmox.Config{
		Endpoints: endpoints,
		Auth: proxmox.AuthConfig{
			Method:   "token",
			APIToken: token,
		},
		TLS: proxmox.TLSConfig{
			InsecureSkipVerify: true,
		},
	}

	pveClient := proxmox.NewClient(config)

	return &Server{
		consul:  consul,
		proxmox: pveClient,
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	group, groupCtx := errgroup.WithContext(ctx)

	if err := s.runPeriodic(); err != nil {
		return err
	}

	group.Go(func() error {
		run := func(_ context.Context) error {
			return s.runPeriodic()
		}

		err := xcmd.PeriodicRun(groupCtx, run, time.Duration(periodicTime)*time.Second)
		if err != nil {
			log.Println(err)
		}

		return err
	})

	group.Go(func() error {
		xcmd.WaitInterrupted(groupCtx)
		log.Println("shutting down...")
		os.Exit(0)

		return nil
	})

	return group.Wait()
}
