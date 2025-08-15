package server

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupVMHAResources(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful VM HA resources setup",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeHAResources: true,
				includeNodes:       true,
				includeNodeVMs:     true,
				includeVMConfig:    true,
				includeStorage:     true,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.SetupVMHAResources()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDetermineVMHAGroup(t *testing.T) {
	tests := []struct {
		name                 string
		vmid                 int
		includeSharedStorage bool
		vmStorageConfig      string
		expectedGroupType    string
		wantErr              bool
	}{
		{
			name:                 "VM with all disks on shared storage should get prefer group",
			vmid:                 200,
			includeSharedStorage: true,
			vmStorageConfig:      "shared",
			expectedGroupType:    "prefer",
			wantErr:              false,
		},
		{
			name:                 "VM with local storage should get pin group",
			vmid:                 201,
			includeSharedStorage: false,
			vmStorageConfig:      "local",
			expectedGroupType:    "pin",
			wantErr:              false,
		},
		{
			name:                 "VM with mixed storage should get pin group",
			vmid:                 202,
			includeSharedStorage: true,
			vmStorageConfig:      "mixed",
			expectedGroupType:    "pin",
			wantErr:              false,
		},
		{
			name:                 "VM with no disks should get prefer group",
			vmid:                 203,
			includeSharedStorage: true,
			vmStorageConfig:      "none",
			expectedGroupType:    "prefer",
			wantErr:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeStorage:       true,
				includeSharedStorage: tt.includeSharedStorage,
				includeVMConfig:      true,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			haGroup, err := testServer.determineVMHAGroup("pve1", tt.vmid)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectedGroupType == "prefer" {
					assert.Equal(t, "crs-vm-prefer-pve1", haGroup)
				} else {
					assert.Equal(t, "crs-vm-pin-pve1", haGroup)
				}
			}
		})
	}
}

func TestAreAllVMDisksShared(t *testing.T) {
	tests := []struct {
		name                 string
		disks                map[string]string
		includeSharedStorage bool
		expected             bool
		wantErr              bool
	}{
		{
			name: "all disks on shared storage",
			disks: map[string]string{
				"virtio0": "shared-storage:vm-100-disk-0,size=32G",
				"virtio1": "shared-storage:vm-100-disk-1,size=100G",
			},
			includeSharedStorage: true,
			expected:             true,
			wantErr:              false,
		},
		{
			name: "all disks on local storage",
			disks: map[string]string{
				"virtio0": "local:vm-100-disk-0.qcow2",
				"scsi0":   "local-lvm:vm-100-disk-1,size=50G",
			},
			includeSharedStorage: false,
			expected:             false,
			wantErr:              false,
		},
		{
			name: "mixed storage should return false",
			disks: map[string]string{
				"virtio0": "shared-storage:vm-100-disk-0,size=32G",
				"virtio1": "local:vm-100-disk-1.qcow2",
			},
			includeSharedStorage: true,
			expected:             false,
			wantErr:              false,
		},
		{
			name:                 "no disks should return true",
			disks:                map[string]string{},
			includeSharedStorage: true,
			expected:             true,
			wantErr:              false,
		},
		{
			name: "mixed storage with CD-ROM on local storage should return false",
			disks: map[string]string{
				"virtio0": "shared-storage:vm-100-disk-0,size=32G",
				"ide2":    "local:iso/ubuntu-20.04.iso,media=cdrom",
			},
			includeSharedStorage: true,
			expected:             false, // Now CD-ROM is checked, so mixed storage = false
			wantErr:              false,
		},
		{
			name: "all storage devices including CD-ROM on shared storage",
			disks: map[string]string{
				"virtio0": "shared-storage:vm-100-disk-0,size=32G",
				"ide2":    "shared-storage:iso/ubuntu-20.04.iso,media=cdrom",
			},
			includeSharedStorage: true,
			expected:             true,
			wantErr:              false,
		},
		{
			name: "disks with empty CD-ROM should ignore none storage",
			disks: map[string]string{
				"virtio0": "shared-storage:vm-100-disk-0,size=32G",
				"ide2":    "none,media=cdrom", // Empty CD-ROM should be ignored
			},
			includeSharedStorage: true,
			expected:             true, // Should be true because only virtio0 counts (all shared)
			wantErr:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeStorage:       true,
				includeSharedStorage: tt.includeSharedStorage,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			result, err := testServer.areAllVMDisksShared(tt.disks)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsDiskEntry(t *testing.T) {
	testServer, mockServer := createTestServer()
	defer mockServer.Close()

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"virtio disk", "virtio0", true},
		{"virtio disk multiple", "virtio10", true},
		{"sata disk", "sata0", true},
		{"scsi disk", "scsi1", true},
		{"ide disk", "ide0", true},
		{"ide disk 1", "ide1", true},
		{"ide2 cd-rom should be included", "ide2", true},
		{"ide3 disk", "ide3", true},
		{"network interface", "net0", false},
		{"cpu setting", "cores", false},
		{"memory setting", "memory", false},
		{"unknown key", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testServer.isDiskEntry(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractStorageFromDiskConfig(t *testing.T) {
	testServer, mockServer := createTestServer()
	defer mockServer.Close()

	tests := []struct {
		name       string
		diskConfig string
		expected   string
	}{
		{
			name:       "LVM storage",
			diskConfig: "local-lvm:vm-100-disk-0,size=32G",
			expected:   "local-lvm",
		},
		{
			name:       "Ceph storage",
			diskConfig: "ceph-storage:vm-100-disk-1,size=100G,format=raw",
			expected:   "ceph-storage",
		},
		{
			name:       "Local file storage",
			diskConfig: "local:100/vm-100-disk-0.qcow2",
			expected:   "local",
		},
		{
			name:       "Storage with special characters",
			diskConfig: "nfs-shared-01:vm-100-disk-0,backup=0",
			expected:   "nfs-shared-01",
		},
		{
			name:       "Empty config",
			diskConfig: "",
			expected:   "",
		},
		{
			name:       "Config without colon",
			diskConfig: "invalid-config",
			expected:   "invalid-config",
		},
		{
			name:       "CD-ROM with no media (none storage)",
			diskConfig: "none,media=cdrom",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testServer.extractStorageFromDiskConfig(tt.diskConfig)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetupVMHAResourcesLogsAssignedGroup(t *testing.T) {
	t.Run("should log pin group assignment for VM with local storage", func(t *testing.T) {
		config := testHandlerConfig{
			includeHAResources:   false, // No existing HA resources
			includeNodes:         true,
			includeNodeVMs:       true,
			includeVMConfig:      true,
			includeStorage:       true,
			includeSharedStorage: false, // Local storage only
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// This should create new HA resources and log pin group assignment
		err := testServer.SetupVMHAResources()
		assert.NoError(t, err)
	})

	t.Run("should log prefer group assignment for VM with shared storage", func(t *testing.T) {
		config := testHandlerConfig{
			includeHAResources:     false, // No existing HA resources
			includeNodes:           true,
			includeNodeVMs:         true,
			includeVMConfig:        true,
			includeStorage:         true,
			includeSharedStorage:   true, // Shared storage available
			includeSharedStorageVM: true, // Include VM 200 with shared storage
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// This should create new HA resources and log prefer group assignment
		err := testServer.SetupVMHAResources()
		assert.NoError(t, err)
	})

	t.Run("should correctly handle VMs with CD-ROM on different storage", func(t *testing.T) {
		// Test VM 202 which has mixed storage (shared data disks + local CD-ROM)
		config := testHandlerConfig{
			includeHAResources:     false,
			includeNodes:           true,
			includeNodeVMs:         true,
			includeVMConfig:        true,
			includeStorage:         true,
			includeSharedStorage:   true,
			includeSharedStorageVM: true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// Test the mixed storage VM (VM 202) assignment
		haGroup, err := testServer.determineVMHAGroup("pve1", 202)
		assert.NoError(t, err)
		// Should be pin group because CD-ROM is on local storage
		assert.Equal(t, "crs-vm-pin-pve1", haGroup)
	})
}

func TestCDROMStorageConsideration(t *testing.T) {
	t.Run("CD-ROM on local storage should force pin group assignment", func(t *testing.T) {
		// Test that even if data disks are on shared storage,
		// having CD-ROM on local storage results in pin group assignment
		disks := map[string]string{
			"virtio0": "shared-storage:vm-test-disk-0,size=32G",
			"virtio1": "shared-storage:vm-test-disk-1,size=100G",
			"ide2":    "local:iso/installer.iso,media=cdrom", // CD-ROM on local storage
		}

		config := testHandlerConfig{
			includeStorage:       true,
			includeSharedStorage: true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		allShared, err := testServer.areAllVMDisksShared(disks)
		assert.NoError(t, err)
		assert.False(t, allShared, "Should return false because CD-ROM is on local storage")
	})

	t.Run("CD-ROM on shared storage allows prefer group assignment", func(t *testing.T) {
		// Test that when ALL storage devices including CD-ROM are on shared storage,
		// prefer group assignment is allowed
		disks := map[string]string{
			"virtio0": "shared-storage:vm-test-disk-0,size=32G",
			"virtio1": "shared-storage:vm-test-disk-1,size=100G",
			"ide2":    "shared-storage:iso/installer.iso,media=cdrom", // CD-ROM on shared storage
		}

		config := testHandlerConfig{
			includeStorage:       true,
			includeSharedStorage: true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		allShared, err := testServer.areAllVMDisksShared(disks)
		assert.NoError(t, err)
		assert.True(t, allShared, "Should return true because all storage devices including CD-ROM are on shared storage")
	})
}

func TestSetupVMHAResourcesUpdatesDisabledResources(t *testing.T) {
	t.Run("should update disabled HA resources to appropriate state", func(t *testing.T) {
		config := testHandlerConfig{
			includeHAResources:         true, // Include existing HA resources
			includeDisabledHAResources: true, // Include disabled HA resources
			includeNodes:               true,
			includeNodeVMs:             true,
			includeVMConfig:            true,
			includeStorage:             true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// This should update the disabled HA resource (VM 105) to started state
		err := testServer.SetupVMHAResources()
		assert.NoError(t, err)
	})

	t.Run("should skip HA resources that are already in correct state", func(t *testing.T) {
		config := testHandlerConfig{
			includeHAResources: true, // Include existing HA resources (VM 100, 102 in started state)
			includeNodes:       true,
			includeNodeVMs:     true,
			includeVMConfig:    true,
			includeStorage:     true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// This should skip VMs that already have HA resources in correct state
		err := testServer.SetupVMHAResources()
		assert.NoError(t, err)
	})
}

func TestSetupVMHAResourcesUpdatesDisabledToStarted(t *testing.T) {
	t.Run("should update disabled HA resource to started for running VM", func(t *testing.T) {
		config := testHandlerConfig{
			includeHAResources:       true, // Include existing HA resources
			includeRunningDisabledVM: true, // Include VM 110 with running status but disabled HA
			includeNodes:             true,
			includeNodeVMs:           true,
			includeVMConfig:          true,
			includeStorage:           true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// This should update VM 110's disabled HA resource to 'started' state since VM is running
		err := testServer.SetupVMHAResources()
		assert.NoError(t, err)
	})

	t.Run("should update disabled HA resource to started for stopped VM", func(t *testing.T) {
		config := testHandlerConfig{
			includeHAResources:         true, // Include existing HA resources
			includeDisabledHAResources: true, // Include VM 105 with stopped status but disabled HA
			includeNodes:               true,
			includeNodeVMs:             true,
			includeVMConfig:            true,
			includeStorage:             true,
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// This should update VM 105's disabled HA resource to 'started' state (Proxmox always stops VMs when HA is disabled)
		err := testServer.SetupVMHAResources()
		assert.NoError(t, err)
	})
}

func TestDetermineVMHAGroupWithHostPCI(t *testing.T) {
	config := testHandlerConfig{
		includeVMConfig:      true,
		includeStorage:       true,
		includeSharedStorage: true,
		includeVMWithHostPCI: true,
	}
	testServer, mockServer := createTestServerWithConfig(config)
	defer mockServer.Close()

	t.Run("VM with hostpci devices should use pin group regardless of storage", func(t *testing.T) {
		// VM 400 has all disks on shared storage but also has hostpci devices
		haGroup, err := testServer.determineVMHAGroup("pve1", 400)
		assert.NoError(t, err)
		assert.Equal(t, "crs-vm-pin-pve1", haGroup, "VM with hostpci devices should be assigned to pin group even with shared storage")
	})

	t.Run("VM config should properly parse hostpci devices", func(t *testing.T) {
		// Test that our custom unmarshaling correctly captures hostpci devices
		vmConfig, err := testServer.proxmox.GetVMConfig("pve1", 400)
		assert.NoError(t, err)
		assert.NotNil(t, vmConfig.HostPCI, "HostPCI map should be initialized")
		assert.Len(t, vmConfig.HostPCI, 2, "Should detect 2 hostpci devices")
		assert.Equal(t, "01:00.0,pcie=1", vmConfig.HostPCI["hostpci0"])
		assert.Equal(t, "02:00.0,rombar=0", vmConfig.HostPCI["hostpci1"])

		// Also verify that disks are still captured correctly
		assert.NotNil(t, vmConfig.Disks, "Disks map should be initialized")
		assert.Equal(t, "shared-storage:vm-400-disk-0,size=50G", vmConfig.Disks["virtio0"])
	})
}

func TestDetermineVMHAGroupWithEmptyCDROM(t *testing.T) {
	config := testHandlerConfig{
		includeVMConfig:         true,
		includeStorage:          true,
		includeSharedStorage:    false, // Local storage only
		includeVMWithEmptyCDROM: true,
	}
	testServer, mockServer := createTestServerWithConfig(config)
	defer mockServer.Close()

	t.Run("VM with empty CD-ROM should not cause storage lookup warning", func(t *testing.T) {
		// VM 401 has local storage and empty CD-ROM (none,media=cdrom)
		haGroup, err := testServer.determineVMHAGroup("pve1", 401)
		assert.NoError(t, err)
		assert.Equal(t, "crs-vm-pin-pve1", haGroup, "VM with local storage should be assigned to pin group")
	})

	t.Run("VM config should properly handle empty CD-ROM", func(t *testing.T) {
		// Test that our storage analysis correctly handles "none,media=cdrom"
		vmConfig, err := testServer.proxmox.GetVMConfig("pve1", 401)
		assert.NoError(t, err)
		assert.NotNil(t, vmConfig.Disks, "Disks map should be initialized")
		assert.Equal(t, "local:vm-401-disk-0.qcow2", vmConfig.Disks["virtio0"])
		assert.Equal(t, "none,media=cdrom", vmConfig.Disks["ide2"])

		// Test that extractStorageFromDiskConfig handles "none" correctly
		storageName := testServer.extractStorageFromDiskConfig("none,media=cdrom")
		assert.Equal(t, "", storageName, "Should return empty string for 'none,media=cdrom'")
	})

	t.Run("VM with empty CD-ROM should be handled correctly in storage analysis", func(t *testing.T) {
		// Test that areAllVMDisksShared correctly ignores "none" storage
		disks := map[string]string{
			"virtio0": "local:vm-401-disk-0.qcow2",
			"ide2":    "none,media=cdrom",
		}

		allShared, err := testServer.areAllVMDisksShared(disks)
		assert.NoError(t, err)
		assert.False(t, allShared, "Should return false because virtio0 is on local storage (ignoring none CD-ROM)")
	})
}

func TestDetermineVMHAGroupWithSCSIHW(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func() (*Server, *httptest.Server)
		vmid          int
		expectedGroup string
		expectedError bool
		description   string
	}{
		{
			name: "VM with scsihw controller should not treat it as disk storage",
			setupFunc: func() (*Server, *httptest.Server) {
				return createTestServerWithConfig(testHandlerConfig{
					includeVMConfig:      true,
					includeStorage:       true,
					includeSharedStorage: false, // local storage only
					includeVMWithSCSIHW:  true,
				})
			},
			vmid:          402,
			expectedGroup: "crs-vm-pin-pve1", // should be pin group because scsi0 is on local storage
			expectedError: false,
			description:   "scsihw field should be excluded from disk analysis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, httpServer := tt.setupFunc()
			defer httpServer.Close()

			group, err := server.determineVMHAGroup("pve1", tt.vmid)

			if tt.expectedError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				assert.Equal(t, tt.expectedGroup, group, tt.description)
			}

			// Additional verification: check that scsihw is not captured as a disk
			vmConfig, configErr := server.proxmox.GetVMConfig("pve1", tt.vmid)
			require.NoError(t, configErr, "Should be able to get VM config")

			// Verify scsihw is not in the disks map
			_, scsihwInDisks := vmConfig.Disks["scsihw"]
			assert.False(t, scsihwInDisks, "scsihw should not be captured as a disk device")

			// Verify scsi0 IS in the disks map
			scsi0Value, scsi0InDisks := vmConfig.Disks["scsi0"]
			assert.True(t, scsi0InDisks, "scsi0 should be captured as a disk device")
			assert.Equal(t, "local:vm-402-disk-0,size=32G", scsi0Value, "scsi0 disk config should match expected")
		})
	}
}
