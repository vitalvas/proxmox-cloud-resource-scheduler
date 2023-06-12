package main

import (
	"log"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/cmd/imds-server/app"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	app := app.New()

	listenAddress := "169.254.169.254:80"
	if runtime.GOOS == "darwin" {
		listenAddress = "127.0.0.1:9999"
	}

	if err := app.Router.Run(listenAddress); err != nil {
		log.Fatal(err)
	}
}
