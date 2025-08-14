package app

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

type App struct {
	proxmox *proxmox.Client
	consul  *consul.Consul
}

func New() (*App, error) {
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

	return &App{
		consul:  consul,
		proxmox: pveClient,
	}, nil
}

func Execute() {
	app, err := New()
	if err != nil {
		log.Fatal(err)
	}

	group, ctx := errgroup.WithContext(context.Background())

	if err := app.runPeriodic(); err != nil {
		log.Fatal(err)
	}

	group.Go(func() error {
		run := func(_ context.Context) error {
			return app.runPeriodic()
		}

		err := xcmd.PeriodicRun(ctx, run, time.Duration(periodicTime)*time.Second)
		if err != nil {
			log.Println(err)
		}

		return err
	})

	group.Go(func() error {
		xcmd.WaitInterrupted(ctx)
		log.Println("shutting down...")
		os.Exit(0)

		return nil
	})

	if err := group.Wait(); err != nil {
		log.Fatal(err)
	}
}
