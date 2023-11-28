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

	if err := app.runPeriodic(); err != nil {
		log.Fatal(err)
	}

	group.Go(func() error {
		var leaderSession string

		run := func(_ context.Context) error {
			leaderInterval := periodicTime * 3

			var isLeader bool

			if len(leaderSession) == 0 {
				leader, leaderSessionID, err := app.consul.GetLeader("periodic", leaderInterval)
				if err != nil {
					return err
				}

				leaderSession = leaderSessionID
				isLeader = leader
			} else {
				if err := app.consul.RenewLeader(leaderSession, leaderInterval); err != nil {
					return err
				}
				isLeader = true
			}

			if isLeader {
				if err := app.runPeriodic(); err != nil {
					return err
				}
			}

			return nil
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
