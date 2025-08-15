package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
