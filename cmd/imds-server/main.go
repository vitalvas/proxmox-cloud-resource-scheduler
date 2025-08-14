package main

import (
	"log"
	"net/http"
	"runtime"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/cmd/imds-server/app"
)

func main() {
	app := app.New()

	listenAddress := "169.254.169.254:80"
	if runtime.GOOS == "darwin" {
		listenAddress = "127.0.0.1:9999"
	}

	if err := http.ListenAndServe(listenAddress, app.Router); err != nil {
		log.Fatal(err)
	}
}
