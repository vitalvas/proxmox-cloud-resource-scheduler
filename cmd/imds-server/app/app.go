package app

import "github.com/gin-gonic/gin"

type App struct {
	Router *gin.Engine
}

func New() *App {
	app := &App{}

	app.Router = app.newRouter()

	return app
}

func (app *App) newRouter() *gin.Engine {
	router := gin.Default()

	// router.GET("/2009-04-04/meta-data/instance-id", app.httpMetadataInstanceIDGet)

	return router
}
