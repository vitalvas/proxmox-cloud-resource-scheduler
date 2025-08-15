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
	includeStorage               bool
	includeSharedStorage         bool
	includeHAGroups              bool
	includeHAResources           bool
	includeErrorHAResources      bool
	includeDisabledHAResources   bool
	includeCriticalVMResources   bool
	includeNonCRSErrorHAResource bool
	includeNodes                 bool
	includeNodeVMs               bool
	includeClusterOptions        bool
	includeClusterResources      bool
	includeVMConfig              bool
	crsTagAlreadyExists          bool
}

//nolint:gocyclo // Test helper function with many mock scenarios is acceptable
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
				if config.includeHAResources || config.includeErrorHAResources {
					var resources []string
					if config.includeHAResources {
						resources = append(resources, `{
							"sid": "vm:100",
							"state": "started",
							"status": "started",
							"crm-state": "started",
							"request": "started",
							"group": "crs-vm-pin-pve1",
							"type": "vm",
							"node": "pve1"
						}`, `{
							"sid": "vm:102",
							"state": "started",
							"status": "started",
							"crm-state": "started", 
							"request": "started",
							"group": "crs-vm-prefer-pve1",
							"type": "vm",
							"node": "pve1"
						}`)
					}
					if config.includeErrorHAResources {
						resources = append(resources, `{
							"sid": "vm:103",
							"state": "started",
							"status": "error",
							"crm-state": "error",
							"request": "started",
							"group": "crs-vm-pin-pve1",
							"type": "vm",
							"node": "pve1"
						}`)
					}
					if config.includeNonCRSErrorHAResource {
						resources = append(resources, `{
							"sid": "vm:104",
							"state": "started",
							"status": "error",
							"crm-state": "error",
							"request": "started",
							"group": "legacy-ha-group",
							"type": "vm",
							"node": "pve1"
						}`)
					}
					if config.includeDisabledHAResources {
						resources = append(resources, `{
							"sid": "vm:105",
							"state": "disabled",
							"status": "disabled",
							"crm-state": "disabled",
							"request": "disabled",
							"group": "crs-vm-prefer-pve1",
							"type": "vm",
							"node": "pve1"
						}`)
					}
					if config.includeCriticalVMResources {
						resources = append(resources, `{
							"sid": "vm:106",
							"state": "disabled",
							"status": "disabled",
							"crm-state": "disabled",
							"request": "disabled",
							"group": "crs-vm-pin-pve1",
							"type": "vm",
							"node": "pve1"
						}`)
					}
					fmt.Fprintf(w, `{"data": [%s]}`, strings.Join(resources, ","))
				} else {
					w.Write([]byte(`{"data": []}`))
				}
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

		case "/api2/json/cluster/options":
			switch r.Method {
			case http.MethodGet:
				if config.includeClusterOptions {
					registeredTags := `["production", "development"]`
					if config.crsTagAlreadyExists {
						registeredTags = `["production", "development", "crs-skip"]`
					}
					fmt.Fprintf(w, `{
						"data": {
							"registered-tags": %s
						}
					}`, registeredTags)
				} else {
					w.Write([]byte(`{"data": {}}`))
				}
			case http.MethodPut:
				w.Write([]byte(`{"data": null}`))
			}

		case "/api2/json/cluster/resources":
			if config.includeClusterResources || config.includeErrorHAResources || config.includeDisabledHAResources || config.includeCriticalVMResources {
				var resources []string
				if config.includeClusterResources {
					resources = append(resources, `{
						"id": "vm/100",
						"type": "qemu",
						"vmid": 100,
						"name": "test-vm",
						"node": "pve1",
						"status": "running",
						"hastate": "started",
						"tags": ""
					}`, `{
						"id": "vm/101", 
						"type": "qemu",
						"vmid": 101,
						"name": "template-vm",
						"node": "pve1", 
						"status": "stopped",
						"hastate": "started",
						"tags": ""
					}`, `{
						"id": "vm/102",
						"type": "qemu", 
						"vmid": 102,
						"name": "skip-vm",
						"node": "pve1",
						"status": "running",
						"hastate": "started",
						"tags": "crs-skip"
					}`)
				}
				if config.includeErrorHAResources {
					resources = append(resources, `{
						"id": "vm/103",
						"type": "qemu",
						"vmid": 103,
						"name": "error-vm",
						"node": "pve1",
						"status": "running",
						"hastate": "error",
						"tags": ""
					}`)
				}
				if config.includeNonCRSErrorHAResource {
					resources = append(resources, `{
						"id": "vm/104",
						"type": "qemu",
						"vmid": 104,
						"name": "legacy-error-vm",
						"node": "pve1",
						"status": "running",
						"hastate": "error",
						"tags": ""
					}`)
				}
				if config.includeDisabledHAResources {
					resources = append(resources, `{
						"id": "vm/105",
						"type": "qemu",
						"vmid": 105,
						"name": "disabled-vm",
						"node": "pve1",
						"status": "stopped",
						"hastate": "disabled",
						"tags": ""
					}`)
				}
				if config.includeCriticalVMResources {
					resources = append(resources, `{
						"id": "vm/106",
						"type": "qemu",
						"vmid": 106,
						"name": "critical-vm",
						"node": "pve1",
						"status": "stopped",
						"hastate": "disabled",
						"tags": "crs-critical"
					}`)
				}
				// Add VM with crs-skip tag that would normally be processed
				resources = append(resources, `{
					"id": "vm/107",
					"type": "qemu",
					"vmid": 107,
					"name": "skip-error-vm",
					"node": "pve1",
					"status": "running",
					"hastate": "error",
					"tags": "crs-skip;production"
				}`, `{
					"id": "vm/108",
					"type": "qemu",
					"vmid": 108,
					"name": "skip-disabled-vm",
					"node": "pve1",
					"status": "stopped",
					"hastate": "disabled",
					"tags": "crs-skip;backup"
				}`, `{
					"id": "vm/109",
					"type": "qemu",
					"vmid": 109,
					"name": "skip-critical-vm",
					"node": "pve1",
					"status": "stopped",
					"hastate": "disabled",
					"tags": "crs-skip;crs-critical"
				}`)
				fmt.Fprintf(w, `{"data": [%s]}`, strings.Join(resources, ","))
			} else {
				w.Write([]byte(`{"data": []}`))
			}

		default:
			// Handle VM config endpoints
			if strings.Contains(r.URL.Path, "/qemu/") && strings.HasSuffix(r.URL.Path, "/config") {
				if !config.includeVMConfig {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				switch r.Method {
				case http.MethodGet:
					// Extract VM ID from path like /api2/json/nodes/pve1/qemu/106/config
					if strings.Contains(r.URL.Path, "/qemu/106/config") {
						// Critical VM with no startup order set, include memory as string to simulate real API
						w.Write([]byte(`{
							"data": {
								"name": "critical-vm",
								"tags": "crs-critical",
								"startup": "",
								"memory": "2048",
								"cores": "2",
								"sockets": "1"
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/110/config") {
						// Critical VM that already has correct startup order
						w.Write([]byte(`{
							"data": {
								"name": "critical-vm-already-set",
								"tags": "crs-critical",
								"startup": "order=1",
								"memory": "2048",
								"cores": "2",
								"sockets": "1"
							}
						}`))
						return
					}
					// Default VM config response
					w.Write([]byte(`{
						"data": {
							"name": "test-vm",
							"tags": "",
							"startup": "order=2",
							"memory": "1024",
							"cores": "1",
							"sockets": "1"
						}
					}`))
					return
				case http.MethodPut:
					// VM config update successful
					w.Write([]byte(`{"data": null}`))
					return
				}
			}

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
			// Handle PUT operations for updating HA resources
			if r.Method == http.MethodPut {
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
		proxmox:          pveClient,
		consul:           consul,
		disableRateLimit: true, // Disable rate limiting for faster tests
	}

	return testServer, server
}
