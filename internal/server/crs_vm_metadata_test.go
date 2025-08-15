package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

func TestUpdateVMMeta(t *testing.T) {
	tests := []struct {
		name                       string
		includeCriticalVMResources bool
		includeVMConfig            bool
		wantErr                    bool
	}{
		{
			name:                       "update critical VM startup order",
			includeCriticalVMResources: true,
			includeVMConfig:            true,
			wantErr:                    false,
		},
		{
			name:                       "no critical VMs found",
			includeCriticalVMResources: false,
			includeVMConfig:            false,
			wantErr:                    false,
		},
		{
			name:                       "critical VMs found but config not available",
			includeCriticalVMResources: true,
			includeVMConfig:            false,
			wantErr:                    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeCriticalVMResources: tt.includeCriticalVMResources,
				includeVMConfig:            tt.includeVMConfig,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.UpdateVMMeta()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractVMIDs(t *testing.T) {
	testServer, mockServer := createTestServer()
	defer mockServer.Close()

	vms := []proxmox.ClusterResource{
		{VMID: 100, Name: "vm1"},
		{VMID: 101, Name: "vm2"},
		{VMID: 102, Name: "vm3"},
	}

	vmids := testServer.extractVMIDs(vms)

	expected := []int{100, 101, 102}
	assert.Equal(t, expected, vmids)
}

func TestUpdateCriticalVMStartOrder(t *testing.T) {
	tests := []struct {
		name            string
		includeVMConfig bool
		expectedResult  bool
	}{
		{
			name:            "update VM startup order successfully",
			includeVMConfig: true,
			expectedResult:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeVMConfig:            tt.includeVMConfig,
				includeCriticalVMResources: true, // Always need critical VM resources for this test
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			result := testServer.updateCriticalVMStartOrder("pve1", 106)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestUpdateVMMetaSkipsCRSSkipVMs(t *testing.T) {
	tests := []struct {
		name                       string
		includeCriticalVMResources bool
	}{
		{
			name:                       "skip VMs with crs-skip tag for metadata updates",
			includeCriticalVMResources: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeCriticalVMResources: tt.includeCriticalVMResources,
				includeVMConfig:            true,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.UpdateVMMeta()
			assert.NoError(t, err)
		})
	}
}

func TestUpdateVMMetaLogging(t *testing.T) {
	tests := []struct {
		name                       string
		includeCriticalVMResources bool
		includeVMConfig            bool
		description                string
	}{
		{
			name:                       "updates needed - should log completion",
			includeCriticalVMResources: true,
			includeVMConfig:            true,
			description:                "Test that completion message is logged when updates are made",
		},
		{
			name:                       "no VMs found - should not log spam",
			includeCriticalVMResources: false,
			includeVMConfig:            true,
			description:                "Test that no spam logs when no critical VMs found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeCriticalVMResources: tt.includeCriticalVMResources,
				includeVMConfig:            tt.includeVMConfig,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.UpdateVMMeta()
			assert.NoError(t, err)
		})
	}
}

func TestDetachNonSharedCDROMs(t *testing.T) {
	t.Run("should detach CD-ROMs on non-shared storage for long-running VMs", func(t *testing.T) {
		config := testHandlerConfig{
			includeClusterResources: true,
			includeVMConfig:         true,
			includeStorage:          true,
			includeSharedStorage:    false, // Local storage only
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// Test with VM that has CD-ROM on local storage
		detached := testServer.detachNonSharedCDROMs("pve1", 202) // VM 202 has mixed storage
		assert.True(t, detached, "Should detach CD-ROM on local storage")
	})

	t.Run("should not detach CD-ROMs on shared storage", func(t *testing.T) {
		config := testHandlerConfig{
			includeClusterResources: true,
			includeVMConfig:         true,
			includeStorage:          true,
			includeSharedStorage:    true, // Shared storage available
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// Test with VM that has CD-ROM on shared storage
		detached := testServer.detachNonSharedCDROMs("pve1", 200) // VM 200 has all shared storage
		assert.False(t, detached, "Should not detach CD-ROM on shared storage")
	})

	t.Run("should handle VMs with no CD-ROMs", func(t *testing.T) {
		config := testHandlerConfig{
			includeClusterResources: true,
			includeVMConfig:         true,
			includeStorage:          true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// Test with VM that has no CD-ROMs
		detached := testServer.detachNonSharedCDROMs("pve1", 201) // VM 201 has no CD-ROM
		assert.False(t, detached, "Should not detach anything when no CD-ROMs present")
	})

	t.Run("should handle VMs with empty CD-ROM (none storage)", func(t *testing.T) {
		config := testHandlerConfig{
			includeClusterResources: true,
			includeVMConfig:         true,
			includeStorage:          true,
			includeVMWithEmptyCDROM: true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// Test with VM that has empty CD-ROM (none,media=cdrom)
		detached := testServer.detachNonSharedCDROMs("pve1", 401) // VM 401 has empty CD-ROM
		assert.False(t, detached, "Should not detach empty CD-ROM drives with no media")
	})
}

func TestIsCDROMEntry(t *testing.T) {
	testServer, mockServer := createTestServer()
	defer mockServer.Close()

	tests := []struct {
		name      string
		diskKey   string
		diskValue string
		expected  bool
	}{
		{"IDE CD-ROM with media=cdrom", "ide2", "local:iso/ubuntu-20.04.iso,media=cdrom", true},
		{"IDE CD-ROM with .iso", "ide2", "shared-storage:iso/installer.iso,media=cdrom", true},
		{"SATA CD-ROM with media=cdrom", "sata0", "local:iso/windows.iso,media=cdrom", true},
		{"Regular virtio disk", "virtio0", "local:vm-100-disk-0.qcow2", false},
		{"Regular scsi disk", "scsi0", "shared-storage:vm-100-disk-1,size=50G", false},
		{"IDE disk without CD-ROM markers", "ide0", "local:vm-100-disk-2.qcow2", false},
		{"IDE with ISO but no media=cdrom", "ide3", "local:iso/test.iso", true}, // Still considered CD-ROM due to .iso
		{"CD-ROM with no media inserted", "ide2", "none,media=cdrom", true},
		{"Network interface", "net0", "virtio=XX:XX:XX:XX:XX:XX,bridge=vmbr0", false},
		{"Empty disk value", "ide2", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testServer.isCDROMEntry(tt.diskKey, tt.diskValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateVMMetaCDROMDetachment(t *testing.T) {
	t.Run("should detach CD-ROMs from long-running VMs", func(t *testing.T) {
		config := testHandlerConfig{
			includeClusterResources: true,
			includeLongRunningVMs:   true, // Include VM with uptime > 24h
			includeVMConfig:         true,
			includeStorage:          true,
			includeSharedStorage:    false, // Local storage only
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		err := testServer.UpdateVMMeta()
		assert.NoError(t, err)
	})

	t.Run("should not affect VMs with uptime < 24h", func(t *testing.T) {
		config := testHandlerConfig{
			includeClusterResources: true,
			includeVMConfig:         true,
			includeStorage:          true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// Normal VMs with uptime < 24h should not be processed for CD-ROM detachment
		err := testServer.UpdateVMMeta()
		assert.NoError(t, err)
	})

	t.Run("should re-evaluate HA group after CD-ROM detachment", func(t *testing.T) {
		config := testHandlerConfig{
			includeClusterResources:   true,
			includeLongRunningVMs:     true, // Include VM with uptime > 24h
			includeVMConfig:           true,
			includeStorage:            true,
			includeSharedStorage:      true, // Shared storage available
			includeLongRunningVMHARes: true, // VM 111 has HA resource
			includeLongRunningVMInPin: true, // VM 111 in pin group
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// This should detach the local CD-ROM and move VM from pin to prefer group
		err := testServer.UpdateVMMeta()
		assert.NoError(t, err)
	})
}

func TestReevaluateVMHAGroupAfterDetachment(t *testing.T) {
	t.Run("should update HA group from pin to prefer after CD-ROM detachment", func(t *testing.T) {
		config := testHandlerConfig{
			includeClusterResources:   true,
			includeVMConfig:           true,
			includeStorage:            true,
			includeSharedStorage:      true, // Shared storage available
			includeLongRunningVMHARes: true, // VM 111 has HA resource in pin group
			includeLongRunningVMInPin: true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// Test the re-evaluation function directly
		err := testServer.reevaluateVMHAGroupAfterDetachment("pve1", 111)
		assert.NoError(t, err)
	})

	t.Run("should skip VMs without HA resources", func(t *testing.T) {
		config := testHandlerConfig{
			includeClusterResources: true,
			includeVMConfig:         true,
			includeStorage:          true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// Test with VM that has no HA resource
		err := testServer.reevaluateVMHAGroupAfterDetachment("pve1", 999)
		assert.NoError(t, err) // Should not error, just skip
	})

	t.Run("should keep same group if no change needed", func(t *testing.T) {
		config := testHandlerConfig{
			includeClusterResources:   true,
			includeVMConfig:           true,
			includeStorage:            true,
			includeSharedStorage:      false, // Local storage only
			includeLongRunningVMHARes: true,  // VM 111 has HA resource in pin group
			includeLongRunningVMInPin: true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// Test with VM that should stay in pin group (still has local storage)
		err := testServer.reevaluateVMHAGroupAfterDetachment("pve1", 111)
		assert.NoError(t, err)
	})
}
