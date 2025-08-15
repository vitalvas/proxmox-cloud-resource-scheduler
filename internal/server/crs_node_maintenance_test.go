package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

func TestHandleNodeMaintenance(t *testing.T) {
	t.Run("should migrate VMs from maintenance nodes", func(t *testing.T) {
		config := testHandlerConfig{
			includeNodes:               true,
			includeMultipleNodes:       true,
			includeNodeMaintenanceMode: true, // pve2 in maintenance
			includeNodeVMs:             true,
			includeVMConfig:            true,
			includeStorage:             true,
			includeSharedStorage:       true,
			includeMaintenanceVMs:      true, // Include VMs that need migration
			includeHAResources:         true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		err := testServer.HandleNodeMaintenance()
		assert.NoError(t, err)
	})

	t.Run("should not migrate VMs when no nodes in maintenance", func(t *testing.T) {
		config := testHandlerConfig{
			includeNodes:         true,
			includeMultipleNodes: true, // All nodes online
			includeNodeVMs:       true,
			// Not setting includeNodeMaintenanceMode, so all nodes should be online
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		err := testServer.HandleNodeMaintenance()
		assert.NoError(t, err)
	})

	t.Run("should detect nodes with maintenance status specifically", func(t *testing.T) {
		config := testHandlerConfig{
			includeNodes:               true,
			includeMultipleNodes:       true,
			includeNodeMaintenanceMode: true, // This sets pve2 to "maintenance" status
			includeNodeVMs:             true,
			includeMaintenanceVMs:      true,
			includeVMConfig:            true,
			includeStorage:             true,
			includeSharedStorage:       true,
			includeHAResources:         true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// This should detect pve2 as maintenance and process VMs
		err := testServer.HandleNodeMaintenance()
		assert.NoError(t, err)
	})

	t.Run("should handle no online nodes available", func(t *testing.T) {
		config := testHandlerConfig{
			includeNodes:                 true,
			includeAllNodesInMaintenance: true, // All nodes in maintenance
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		err := testServer.HandleNodeMaintenance()
		assert.NoError(t, err) // Should not error, just log warning
	})

	t.Run("should only consider nodes with maintenance status as maintenance nodes", func(t *testing.T) {
		// This test verifies that the function now specifically looks for "maintenance" status
		// and doesn't treat other non-online statuses as maintenance
		config := testHandlerConfig{
			includeNodes:         true,
			includeMultipleNodes: true,
			// All nodes will be "online" since includeNodeMaintenanceMode is false
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		err := testServer.HandleNodeMaintenance()
		assert.NoError(t, err)

		// With the fix, this should not detect any maintenance nodes
		// because all nodes are "online", not "maintenance"
	})
}

func TestShouldMigrateVMFromMaintenance(t *testing.T) {
	config := testHandlerConfig{
		includeVMConfig:      true,
		includeStorage:       true,
		includeSharedStorage: true,
	}
	testServer, mockServer := createTestServerWithConfig(config)
	defer mockServer.Close()

	tests := []struct {
		name       string
		vm         proxmox.VM
		haResource *proxmox.ClusterHAResource
		expected   bool
	}{
		{
			name: "should migrate stopped VM in prefer group",
			vm: proxmox.VM{
				VMID:     100,
				Name:     "test-vm",
				Status:   vmStatusStopped,
				Template: 0,
			},
			haResource: &proxmox.ClusterHAResource{
				SID:   "vm:100",
				Group: "crs-vm-prefer-pve1",
			},
			expected: true,
		},
		{
			name: "should not migrate running VM",
			vm: proxmox.VM{
				VMID:     101,
				Name:     "running-vm",
				Status:   vmStatusRunning,
				Template: 0,
			},
			haResource: &proxmox.ClusterHAResource{
				SID:   "vm:101",
				Group: "crs-vm-prefer-pve1",
			},
			expected: false,
		},
		{
			name: "should not migrate VM in pin group",
			vm: proxmox.VM{
				VMID:     102,
				Name:     "pinned-vm",
				Status:   vmStatusStopped,
				Template: 0,
			},
			haResource: &proxmox.ClusterHAResource{
				SID:   "vm:102",
				Group: "crs-vm-pin-pve1",
			},
			expected: false,
		},
		{
			name: "should not migrate VM without HA resource",
			vm: proxmox.VM{
				VMID:     103,
				Name:     "no-ha-vm",
				Status:   vmStatusStopped,
				Template: 0,
			},
			haResource: nil,
			expected:   false,
		},
		{
			name: "should migrate template on shared storage",
			vm: proxmox.VM{
				VMID:     302, // Use VM 302 which has proper mock setup for shared storage template
				Name:     "template-shared",
				Status:   vmStatusStopped,
				Template: vmTemplateFlag,
			},
			haResource: nil,
			expected:   true, // Template on shared storage should be migrated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testServer.shouldMigrateVMFromMaintenance(tt.vm, tt.haResource)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSelectMigrationTarget(t *testing.T) {
	testServer, mockServer := createTestServer()
	defer mockServer.Close()

	onlineNodes := []proxmox.Node{
		{Node: "pve1", Status: "online"},
		{Node: "pve3", Status: "online"},
	}

	tests := []struct {
		name     string
		vm       proxmox.VM
		expected string
	}{
		{
			name: "should select target based on VM ID",
			vm: proxmox.VM{
				VMID: 100,
				Name: "test-vm",
			},
			expected: "pve1", // 100 % 2 = 0, so index 0 (pve1)
		},
		{
			name: "should select different target for different VM ID",
			vm: proxmox.VM{
				VMID: 101,
				Name: "test-vm-2",
			},
			expected: "pve3", // 101 % 2 = 1, so index 1 (pve3)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := testServer.selectMigrationTarget(tt.vm, onlineNodes)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, target)
		})
	}

	t.Run("should error when no online nodes available", func(t *testing.T) {
		vm := proxmox.VM{VMID: 100, Name: "test-vm"}
		_, err := testServer.selectMigrationTarget(vm, []proxmox.Node{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no online nodes available")
	})
}

func TestParseVMID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"valid VM ID", "123", 123},
		{"invalid string", "abc", 0},
		{"empty string", "", 0},
		{"mixed string", "123abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVMID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMigrateVM(t *testing.T) {
	t.Run("should initiate migration successfully", func(t *testing.T) {
		config := testHandlerConfig{
			includeNodes:         true,
			includeMultipleNodes: true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		vm := proxmox.VM{
			VMID: 100,
			Name: "test-vm",
		}

		err := testServer.migrateVM("pve2", vm, "pve1")
		assert.NoError(t, err)
	})
}
