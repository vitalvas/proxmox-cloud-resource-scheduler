package app

import "github.com/vitalvas/proxmox-cloud-resource-scheduler/app/proxmox"

type App struct {
	config  *Config
	proxmox *proxmox.Proxmox
}

func New() *App {
	app := &App{}

	app.config = LoadConfig("config.json")

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
	app.SetupDRS()
	app.SetupDRSQemu()
}
