package main

import (
	"context"
	"log"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/server"
)

func main() {
	srv, err := server.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := srv.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
