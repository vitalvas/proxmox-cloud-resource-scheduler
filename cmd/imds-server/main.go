package main

import (
	"log"
	"runtime"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	app := gin.Default()

	listenAddress := "169.254.169.254:80"
	if runtime.GOOS == "darwin" {
		listenAddress = "127.0.0.1:9999"
	}

	if err := app.Run(listenAddress); err != nil {
		log.Fatal(err)
	}
}
