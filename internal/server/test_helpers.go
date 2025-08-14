package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/consul"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

type testHandlerConfig struct {
	includeStorage       bool
	includeSharedStorage bool
	includeHAGroups      bool
	includeHAResources   bool
	includeNodes         bool
	includeNodeVMs       bool
}

func createTestServerWithConfig(config testHandlerConfig) (*Server, *httptest.Server) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/api2/json/cluster/ha/groups":
			switch r.Method {
			case http.MethodGet:
				w.Write([]byte(`{"data": []}`))
			case http.MethodPost:
				w.Write([]byte(`{"data": null}`))
			}

		case "/api2/json/cluster/ha/resources":
			switch r.Method {
			case http.MethodGet:
				w.Write([]byte(`{"data": []}`))
			case http.MethodPost:
				w.Write([]byte(`{"data": null}`))
			}

		case "/api2/json/nodes":
			if config.includeNodes {
				w.Write([]byte(`{
					"data": [
						{
							"node": "pve1",
							"status": "online",
							"type": "node"
						}
					]
				}`))
			} else {
				w.Write([]byte(`{"data": []}`))
			}

		case "/api2/json/nodes/pve1/qemu":
			if config.includeNodeVMs {
				w.Write([]byte(`{
					"data": [
						{
							"vmid": 100,
							"name": "test-vm",
							"status": "running",
							"template": 0,
							"tags": ""
						},
						{
							"vmid": 101,
							"name": "template-vm", 
							"status": "stopped",
							"template": 1,
							"tags": ""
						},
						{
							"vmid": 102,
							"name": "skip-vm",
							"status": "running",
							"template": 0,
							"tags": "crs-skip"
						}
					]
				}`))
			} else {
				w.Write([]byte(`{"data": []}`))
			}

		case "/api2/json/storage":
			if config.includeStorage {
				sharedValue := 0
				contentValue := "vztmpl,backup,iso"
				if config.includeSharedStorage {
					sharedValue = 1
					contentValue = "images,vztmpl,backup,iso"
				}
				fmt.Fprintf(w, `{
					"data": [
						{
							"storage": "local",
							"type": "dir",
							"shared": %d,
							"content": "%s"
						}
					]
				}`, sharedValue, contentValue)
			} else {
				w.Write([]byte(`{"data": []}`))
			}

		default:
			// Handle DELETE operations for specific HA groups and resources
			if r.Method == http.MethodDelete {
				if strings.HasPrefix(r.URL.Path, "/api2/json/cluster/ha/groups/") {
					w.Write([]byte(`{"data": null}`))
					return
				}
				if strings.HasPrefix(r.URL.Path, "/api2/json/cluster/ha/resources/") {
					w.Write([]byte(`{"data": null}`))
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	proxmoxConfig := &proxmox.Config{
		Endpoints: []string{server.URL},
		Auth: proxmox.AuthConfig{
			Method:   "token",
			APIToken: "test@pam!test=12345678-1234-1234-1234-123456789012",
		},
		TLS: proxmox.TLSConfig{
			InsecureSkipVerify: true,
		},
	}

	pveClient := proxmox.NewClient(proxmoxConfig)
	consul := &consul.Consul{}

	testServer := &Server{
		proxmox: pveClient,
		consul:  consul,
	}

	return testServer, server
}
