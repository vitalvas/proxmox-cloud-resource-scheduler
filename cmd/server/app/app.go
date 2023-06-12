package app

import (
	"log"
	"time"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/config"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

type App struct {
	config  *config.Config
	proxmox *proxmox.Proxmox
}

func New() *App {
	app := &App{}

	var err error
	app.config, err = config.LoadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	app.proxmox = proxmox.New()
	if app.config != nil {
		app.proxmox.SetAuth(app.config.Proxmox.User, app.config.Proxmox.Token)

		for _, row := range app.config.Proxmox.Nodes {
			app.proxmox.AddNode(row.URL)
		}
	}

	return app
}

func (app *App) Run() {
	for {
		app.SetupDRS()
		app.SetupDRSQemu()

		time.Sleep(time.Minute)
	}
}
