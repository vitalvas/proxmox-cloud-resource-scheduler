package app

import "github.com/gorilla/mux"

type App struct {
	Router *mux.Router
}

func New() *App {
	app := &App{}

	app.Router = app.newRouter()

	return app
}

func (app *App) newRouter() *mux.Router {
	router := mux.NewRouter()

	// router.HandleFunc("/2009-04-04/meta-data/instance-id", app.httpMetadataInstanceIDGet).Methods("GET")

	return router
}
