package app

import (
	"log"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func (app *App) httpServer() {
	router := gin.Default()

	if err := router.Run(); err != nil {
		log.Fatal(err)
	}
}
