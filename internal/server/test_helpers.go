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
	includeStorage                  bool
	includeSharedStorage            bool
	includeHAGroups                 bool
	includeHAResources              bool
	includeErrorHAResources         bool
	includeDisabledHAResources      bool
	includeCriticalVMResources      bool
	includeNonCRSErrorHAResource    bool
	includeNodes                    bool
	includeNodeVMs                  bool
	includeClusterOptions           bool
	includeClusterResources         bool
	includeVMConfig                 bool
	crsTagAlreadyExists             bool
	includeMultipleNodes            bool
	includeOutdatedHAGroups         bool
	includeCorrectHAGroups          bool
	includeSharedStorageVM          bool // Include VM 200 with shared storage config
	includeRunningDisabledVM        bool // Include VM 106 with running status but disabled HA
	includeLongRunningVMs           bool // Include VMs with uptime > 24h
	includeLongRunningVMHARes       bool // Include HA resource for VM 111 (long-running)
	includeLongRunningVMInPin       bool // Include VM 111 in pin group (vs prefer)
	includeNodeMaintenanceMode      bool // Include pve2 in maintenance mode
	includeMaintenanceVMs           bool // Include VMs that need migration from maintenance node
	includeAllNodesInMaintenance    bool // All nodes in maintenance mode
	includeVMWithHostPCI            bool // Include VM with hostpci devices
	includeVMWithEmptyCDROM         bool // Include VM with CD-ROM that has no media (none)
	includeVMWithSCSIHW             bool // Include VM with scsihw controller type
	includeCriticalVMInMigrateState bool // Include critical VM in migrate state
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
				switch {
				case config.includeOutdatedHAGroups:
					// Return existing HA groups with outdated configuration
					if config.includeMultipleNodes {
						w.Write([]byte(`{
							"data": [
								{
									"group": "crs-vm-pin-pve1",
									"nodes": "pve1:1000",
									"restricted": 1,
									"nofailback": 1
								},
								{
									"group": "crs-vm-prefer-pve1",
									"nodes": "pve1:1000,pve2:1",
									"restricted": 1,
									"nofailback": 1
								}
							]
						}`))
					} else {
						w.Write([]byte(`{
							"data": [
								{
									"group": "crs-vm-pin-pve1",
									"nodes": "pve1:500",
									"restricted": 1,
									"nofailback": 1
								}
							]
						}`))
					}
				case config.includeCorrectHAGroups && config.includeMultipleNodes:
					// Return existing HA groups with correct configuration but different order
					w.Write([]byte(`{
						"data": [
							{
								"group": "crs-vm-pin-pve1",
								"nodes": "pve1:1000",
								"restricted": 1,
								"nofailback": 1
							},
							{
								"group": "crs-vm-prefer-pve1",
								"nodes": "pve3:990,pve1:1000,pve2:995",
								"restricted": 1,
								"nofailback": 1
							},
							{
								"group": "crs-vm-prefer-pve2",
								"nodes": "pve1:990,pve3:995,pve2:1000",
								"restricted": 1,
								"nofailback": 1
							},
							{
								"group": "crs-vm-prefer-pve3",
								"nodes": "pve2:990,pve3:1000,pve1:995",
								"restricted": 1,
								"nofailback": 1
							}
						]
					}`))
				default:
					w.Write([]byte(`{"data": []}`))
				}
			case http.MethodPost:
				w.Write([]byte(`{"data": null}`))
			}

		case "/api2/json/cluster/ha/resources":
			switch r.Method {
			case http.MethodGet:
				if config.includeHAResources || config.includeErrorHAResources || config.includeLongRunningVMHARes || config.includeCriticalVMInMigrateState {
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
					if config.includeRunningDisabledVM {
						resources = append(resources, `{
							"sid": "vm:110",
							"state": "disabled",
							"status": "disabled",
							"crm-state": "disabled",
							"request": "disabled",
							"group": "crs-vm-pin-pve1",
							"type": "vm",
							"node": "pve1"
						}`)
					}
					if config.includeLongRunningVMHARes {
						haGroup := "crs-vm-prefer-pve1"
						if config.includeLongRunningVMInPin {
							haGroup = "crs-vm-pin-pve1"
						}
						resources = append(resources, fmt.Sprintf(`{
							"sid": "vm:111",
							"state": "started",
							"status": "started",
							"crm-state": "started",
							"request": "started",
							"group": "%s",
							"type": "vm",
							"node": "pve1"
						}`, haGroup))
					}
					if config.includeCriticalVMInMigrateState {
						resources = append(resources, `{
							"sid": "vm:115",
							"state": "started",
							"status": "started",
							"crm-state": "started",
							"request": "started",
							"group": "crs-vm-prefer-pve01",
							"type": "vm",
							"node": "pve1"
						}`)
					}
					if config.includeMaintenanceVMs {
						// Add HA resources for VMs on maintenance node
						resources = append(resources, `{
							"sid": "vm:300",
							"state": "stopped",
							"status": "stopped",
							"crm-state": "stopped",
							"request": "stopped",
							"group": "crs-vm-prefer-pve2",
							"type": "vm",
							"node": "pve2"
						}`, `{
							"sid": "vm:301",
							"state": "started",
							"status": "started",
							"crm-state": "started",
							"request": "started",
							"group": "crs-vm-prefer-pve2",
							"type": "vm",
							"node": "pve2"
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
				switch {
				case config.includeAllNodesInMaintenance:
					// All nodes in maintenance
					w.Write([]byte(`{
						"data": [
							{
								"node": "pve1",
								"status": "maintenance",
								"type": "node"
							},
							{
								"node": "pve2",
								"status": "maintenance",
								"type": "node"
							},
							{
								"node": "pve3",
								"status": "maintenance",
								"type": "node"
							}
						]
					}`))
				case config.includeMultipleNodes:
					pve2Status := "online"
					if config.includeNodeMaintenanceMode {
						pve2Status = "maintenance"
					}
					fmt.Fprintf(w, `{
						"data": [
							{
								"node": "pve1",
								"status": "online",
								"type": "node"
							},
							{
								"node": "pve2",
								"status": "%s",
								"type": "node"
							},
							{
								"node": "pve3",
								"status": "online",
								"type": "node"
							}
						]
					}`, pve2Status)
				default:
					w.Write([]byte(`{
						"data": [
							{
								"node": "pve1",
								"status": "online",
								"type": "node"
							}
						]
					}`))
				}
			} else {
				w.Write([]byte(`{"data": []}`))
			}

		case "/api2/json/nodes/pve1/qemu":
			if config.includeNodeVMs {
				vmList := `[
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
						}`
				if config.includeSharedStorageVM {
					vmList += `,
						{
							"vmid": 200,
							"name": "shared-storage-vm",
							"status": "running",
							"template": 0,
							"tags": ""
						}`
				}
				if config.includeDisabledHAResources {
					vmList += `,
						{
							"vmid": 105,
							"name": "disabled-vm",
							"status": "stopped",
							"template": 0,
							"tags": ""
						}`
				}
				if config.includeRunningDisabledVM {
					vmList += `,
						{
							"vmid": 110,
							"name": "running-disabled-vm",
							"status": "running",
							"template": 0,
							"tags": ""
						}`
				}
				if config.includeLongRunningVMs {
					vmList += `,
						{
							"vmid": 111,
							"name": "long-running-vm",
							"status": "running",
							"template": 0,
							"tags": ""
						}`
				}
				if config.includeVMWithEmptyCDROM {
					vmList += `,
						{
							"vmid": 401,
							"name": "empty-cdrom-vm",
							"status": "running",
							"template": 0,
							"tags": ""
						}`
				}
				if config.includeVMWithSCSIHW {
					vmList += `,
						{
							"vmid": 402,
							"name": "scsihw-vm",
							"status": "running",
							"template": 0,
							"tags": ""
						}`
				}
				vmList += `
					]`
				fmt.Fprintf(w, `{"data": %s}`, vmList)
			} else {
				w.Write([]byte(`{"data": []}`))
			}

		case "/api2/json/nodes/pve2/qemu":
			if config.includeNodeVMs && config.includeMaintenanceVMs {
				// VMs on maintenance node pve2 that need migration
				w.Write([]byte(`{
					"data": [
						{
							"vmid": 300,
							"name": "stopped-prefer-vm",
							"status": "stopped",
							"template": 0,
							"tags": ""
						},
						{
							"vmid": 301,
							"name": "running-prefer-vm",
							"status": "running",
							"template": 0,
							"tags": ""
						},
						{
							"vmid": 302,
							"name": "template-shared",
							"status": "stopped",
							"template": 1,
							"tags": ""
						},
						{
							"vmid": 303,
							"name": "skip-vm",
							"status": "stopped",
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
				sharedStorageValue := 0
				if config.includeSharedStorage {
					sharedStorageValue = 1
				}
				// Return both local and shared storage for testing mixed scenarios
				fmt.Fprintf(w, `{
					"data": [
						{
							"storage": "local",
							"type": "dir",
							"shared": 0,
							"content": "images,vztmpl,backup,iso"
						},
						{
							"storage": "shared-storage",
							"type": "cephfs",
							"shared": %d,
							"content": "images,vztmpl,backup,iso"
						},
						{
							"storage": "local-lvm",
							"type": "lvm",
							"shared": 0,
							"content": "images"
						}
					]
				}`, sharedStorageValue)
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
			if config.includeClusterResources || config.includeErrorHAResources || config.includeDisabledHAResources || config.includeCriticalVMResources || config.includeRunningDisabledVM || config.includeLongRunningVMs || config.includeLongRunningVMHARes || config.includeCriticalVMInMigrateState {
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
				if config.includeCriticalVMInMigrateState {
					resources = append(resources, `{
						"id": "vm/115",
						"type": "qemu",
						"vmid": 115,
						"name": "critical-vm-migrating",
						"node": "pve1",
						"status": "running",
						"hastate": "migrate",
						"tags": "crs-critical"
					}`)
				}
				if config.includeRunningDisabledVM {
					resources = append(resources, `{
						"id": "vm/110",
						"type": "qemu",
						"vmid": 110,
						"name": "running-disabled-vm",
						"node": "pve1",
						"status": "running",
						"hastate": "disabled",
						"tags": ""
					}`)
				}
				if config.includeLongRunningVMs {
					resources = append(resources, `{
						"id": "vm/111",
						"type": "qemu",
						"vmid": 111,
						"name": "long-running-vm",
						"node": "pve1",
						"status": "running",
						"hastate": "started",
						"uptime": 90000,
						"tags": ""
					}`)
				}
				if config.includeMaintenanceVMs {
					resources = append(resources, `{
						"id": "vm/300",
						"type": "qemu",
						"vmid": 300,
						"name": "stopped-prefer-vm",
						"node": "pve2",
						"status": "stopped",
						"hastate": "stopped",
						"tags": ""
					}`, `{
						"id": "vm/301",
						"type": "qemu",
						"vmid": 301,
						"name": "running-prefer-vm",
						"node": "pve2",
						"status": "running",
						"hastate": "started",
						"tags": ""
					}`, `{
						"id": "vm/302",
						"type": "qemu",
						"vmid": 302,
						"name": "template-shared",
						"node": "pve2",
						"status": "stopped",
						"hastate": "started",
						"tags": ""
					}`, `{
						"id": "vm/303",
						"type": "qemu",
						"vmid": 303,
						"name": "skip-vm",
						"node": "pve2",
						"status": "stopped",
						"hastate": "started",
						"tags": "crs-skip"
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

				// Add node resources to cluster resources
				if config.includeNodes || config.includeMultipleNodes || config.includeNodeMaintenanceMode || config.includeAllNodesInMaintenance {
					switch {
					case config.includeAllNodesInMaintenance:
						resources = append(resources, `{
							"id": "node/pve1",
							"type": "node",
							"node": "pve1",
							"status": "online",
							"hastate": "maintenance"
						}`, `{
							"id": "node/pve2",
							"type": "node",
							"node": "pve2",
							"status": "online",
							"hastate": "maintenance"
						}`, `{
							"id": "node/pve3",
							"type": "node",
							"node": "pve3",
							"status": "online",
							"hastate": "maintenance"
						}`)
					case config.includeMultipleNodes:
						pve2HAState := "online"
						if config.includeNodeMaintenanceMode {
							pve2HAState = "maintenance"
						}
						resources = append(resources, `{
							"id": "node/pve1",
							"type": "node",
							"node": "pve1",
							"status": "online",
							"hastate": "online"
						}`, fmt.Sprintf(`{
							"id": "node/pve2",
							"type": "node",
							"node": "pve2",
							"status": "online",
							"hastate": "%s"
						}`, pve2HAState), `{
							"id": "node/pve3",
							"type": "node",
							"node": "pve3",
							"status": "online",
							"hastate": "online"
						}`)
					default:
						resources = append(resources, `{
							"id": "node/pve1",
							"type": "node",
							"node": "pve1",
							"status": "online",
							"hastate": "online"
						}`)
					}
				}

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
								"sockets": "1",
								"disks": {}
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
								"sockets": "1",
								"disks": {}
							}
						}`))
						return
					}
					// Handle storage-based VM configuration tests
					if strings.Contains(r.URL.Path, "/qemu/200/config") {
						// VM with all storage devices on shared storage
						w.Write([]byte(`{
							"data": {
								"name": "shared-storage-vm",
								"memory": "2048",
								"disks": {
									"virtio0": "shared-storage:vm-200-disk-0,size=32G",
									"virtio1": "shared-storage:vm-200-disk-1,size=100G",
									"ide2": "shared-storage:iso/ubuntu-20.04.iso,media=cdrom"
								}
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/201/config") {
						// VM with all disks on local storage
						w.Write([]byte(`{
							"data": {
								"name": "local-storage-vm",
								"memory": "1024",
								"disks": {
									"virtio0": "local:vm-201-disk-0.qcow2",
									"scsi0": "local-lvm:vm-201-disk-1,size=50G"
								}
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/202/config") {
						// VM with mixed storage (shared + local) including CD-ROM on local storage
						w.Write([]byte(`{
							"data": {
								"name": "mixed-storage-vm",
								"memory": "2048",
								"disks": {
									"virtio0": "shared-storage:vm-202-disk-0,size=32G",
									"virtio1": "local:vm-202-disk-1.qcow2",
									"ide2": "local:iso/ubuntu-20.04.iso,media=cdrom"
								}
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/203/config") {
						// VM with no disks
						w.Write([]byte(`{
							"data": {
								"name": "no-disk-vm",
								"memory": "512",
								"disks": {}
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/105/config") {
						// VM with disabled HA resource
						w.Write([]byte(`{
							"data": {
								"name": "disabled-vm",
								"memory": "1024",
								"disks": {
									"virtio0": "local:vm-105-disk-0.qcow2"
								}
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/110/config") {
						// Running VM with disabled HA resource
						w.Write([]byte(`{
							"data": {
								"name": "running-disabled-vm",
								"memory": "2048",
								"disks": {
									"virtio0": "local:vm-110-disk-0.qcow2"
								}
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/111/config") {
						// Long-running VM - storage config depends on test scenario
						if config.includeLongRunningVMHARes && config.includeLongRunningVMInPin && config.includeSharedStorage {
							// Simulate post-detachment state: only shared storage left
							w.Write([]byte(`{
								"data": {
									"name": "long-running-vm",
									"memory": "2048",
									"disks": {
										"virtio0": "shared-storage:vm-111-disk-0,size=32G"
									}
								}
							}`))
						} else {
							// Pre-detachment state: has CD-ROM on local storage
							w.Write([]byte(`{
								"data": {
									"name": "long-running-vm",
									"memory": "2048",
									"disks": {
										"virtio0": "shared-storage:vm-111-disk-0,size=32G",
										"ide2": "local:iso/installer.iso,media=cdrom"
									}
								}
							}`))
						}
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/300/config") {
						// Stopped VM on maintenance node in prefer group (local storage)
						w.Write([]byte(`{
							"data": {
								"name": "stopped-prefer-vm",
								"memory": "1024",
								"disks": {
									"virtio0": "local:vm-300-disk-0.qcow2"
								}
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/301/config") {
						// Running VM on maintenance node (should not be migrated)
						w.Write([]byte(`{
							"data": {
								"name": "running-prefer-vm",
								"memory": "1024",
								"disks": {
									"virtio0": "local:vm-301-disk-0.qcow2"
								}
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/302/config") {
						// Template on shared storage (should be migrated)
						w.Write([]byte(`{
							"data": {
								"name": "template-shared",
								"memory": "512",
								"disks": {
									"virtio0": "shared-storage:vm-302-disk-0,size=10G"
								}
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/303/config") {
						// VM with crs-skip tag (should not be migrated)
						w.Write([]byte(`{
							"data": {
								"name": "skip-vm",
								"memory": "512",
								"disks": {
									"virtio0": "local:vm-303-disk-0.qcow2"
								}
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/400/config") {
						// VM with hostpci device (should force pin group)
						w.Write([]byte(`{
							"data": {
								"name": "hostpci-vm",
								"memory": "4096",
								"disks": {
									"virtio0": "shared-storage:vm-400-disk-0,size=50G"
								},
								"hostpci0": "01:00.0,pcie=1",
								"hostpci1": "02:00.0,rombar=0"
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/401/config") {
						// VM with empty CD-ROM (none,media=cdrom)
						w.Write([]byte(`{
							"data": {
								"name": "empty-cdrom-vm",
								"memory": "2048",
								"disks": {
									"virtio0": "local:vm-401-disk-0.qcow2",
									"ide2": "none,media=cdrom"
								}
							}
						}`))
						return
					}
					if strings.Contains(r.URL.Path, "/qemu/402/config") {
						// VM with scsihw controller type that should not be treated as disk
						w.Write([]byte(`{
							"data": {
								"name": "scsihw-vm",
								"memory": "2048",
								"cores": 2,
								"sockets": 1,
								"scsi0": "local:vm-402-disk-0,size=32G",
								"scsihw": "virtio-scsi-single"
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
							"sockets": "1",
							"disks": {
								"virtio0": "local:vm-100-disk-0.qcow2"
							}
						}
					}`))
					return
				case http.MethodPut:
					// VM config update successful
					w.Write([]byte(`{"data": null}`))
					return
				case http.MethodPost:
					// Handle VM operations like migration
					if strings.Contains(r.URL.Path, "/migrate") {
						// VM migration endpoint
						w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:qmigrate:100:root@pam:"}`))
						return
					}
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
			// Handle PUT operations for updating HA groups and resources
			if r.Method == http.MethodPut {
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
		proxmox:          pveClient,
		consul:           consul,
		disableRateLimit: true, // Disable rate limiting for faster tests
	}

	return testServer, server
}

func createTestServer() (*Server, *httptest.Server) {
	return createTestServerWithConfig(testHandlerConfig{})
}
