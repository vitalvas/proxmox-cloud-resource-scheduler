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
	proxmox *proxmox.Proxmox
	consul  *consul.Consul
}

func New() (*App, error) {
	consul, err := consul.New()
	if err != nil {
		return nil, err
	}

	pve := proxmox.New()
	pve.GetNodesURL = consul.GetPVENodesURL
	pve.GetToken = consul.GetPVEAuthToken

	return &App{
		consul:  consul,
		proxmox: pve,
	}, err
}

func Execute() {
	app, err := New()
	if err != nil {
		log.Fatal(err)
	}

	group, ctx := errgroup.WithContext(context.Background())

	if err := app.runPeriodic(ctx); err != nil {
		log.Fatal(err)
	}

	group.Go(func() error {
		err := xcmd.PeriodicRun(ctx, app.runPeriodic, time.Minute)
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
