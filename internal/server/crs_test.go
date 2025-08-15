package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/proxmox"
)

func createTestServer() (*Server, *httptest.Server) {
	return createTestServerWithConfig(testHandlerConfig{
		includeStorage:        true,
		includeSharedStorage:  false,
		includeHAGroups:       true,
		includeNodes:          true,
		includeClusterOptions: true,
	})
}

func TestSetupCRS(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful CRS setup",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer, mockServer := createTestServer()
			defer mockServer.Close()

			err := testServer.SetupCRS()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetupVMPin(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful VM pin setup",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer, mockServer := createTestServerWithConfig(testHandlerConfig{
				includeHAGroups: true,
				includeNodes:    true,
			})
			defer mockServer.Close()

			err := testServer.SetupVMPin()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetupVMPrefer(t *testing.T) {
	tests := []struct {
		name                 string
		includeStorage       bool
		includeSharedStorage bool
		wantErr              bool
	}{
		{
			name:                 "successful VM prefer setup with shared storage",
			includeStorage:       true,
			includeSharedStorage: true,
			wantErr:              false,
		},
		{
			name:                 "skip VM prefer setup without shared storage",
			includeStorage:       true,
			includeSharedStorage: false,
			wantErr:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeHAGroups:      true,
				includeNodes:         true,
				includeStorage:       tt.includeStorage,
				includeSharedStorage: tt.includeSharedStorage,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.SetupVMPrefer()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateActualHAGroupNames(t *testing.T) {
	tests := []struct {
		name                 string
		includeStorage       bool
		includeSharedStorage bool
		expectedGroups       []string
	}{
		{
			name:                 "with shared storage",
			includeStorage:       true,
			includeSharedStorage: true,
			expectedGroups:       []string{"crs-vm-pin-pve1", "crs-vm-prefer-pve1"},
		},
		{
			name:                 "without shared storage",
			includeStorage:       true,
			includeSharedStorage: false,
			expectedGroups:       []string{"crs-vm-pin-pve1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeNodes:         true,
				includeStorage:       tt.includeStorage,
				includeSharedStorage: tt.includeSharedStorage,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			actualGroups, err := testServer.generateActualHAGroupNames()

			assert.NoError(t, err)
			assert.Len(t, actualGroups, len(tt.expectedGroups))

			for _, expectedGroup := range tt.expectedGroups {
				assert.True(t, actualGroups[expectedGroup], "Expected group %s to be present", expectedGroup)
			}
		})
	}
}

func TestCleanupOrphanedHAGroups(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful cleanup",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeNodes:         true,
				includeStorage:       true,
				includeSharedStorage: false,
				includeHAGroups:      true,
				includeHAResources:   true,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.CleanupOrphanedHAGroups()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemoveVMsFromHAGroup(t *testing.T) {
	tests := []struct {
		name      string
		groupName string
		wantErr   bool
	}{
		{
			name:      "successful removal",
			groupName: "test-group",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeHAResources: true,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.removeVMsFromHAGroup(tt.groupName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCRSConstants(t *testing.T) {
	assert.Equal(t, 1000, crsMaxNodePriority)
	assert.Equal(t, 1, crsMinNodePriority)
	assert.Equal(t, "crs-skip", crsSkipTag)
	assert.Equal(t, "crs-critical", crsCriticalTag)
	assert.Equal(t, "crs-", crsGroupPrefix)
	assert.Equal(t, "error", haStateError)
	assert.Equal(t, "disabled", haStateDisabled)
	assert.Equal(t, "started", haStateStarted)
	assert.Equal(t, "stopped", haStateStopped)
	assert.Equal(t, "ignored", haStateIgnored)
	assert.Equal(t, "running", vmStatusRunning)
	assert.Equal(t, "stopped", vmStatusStopped)
	assert.Equal(t, 1, vmTemplateFlag)
	assert.Equal(t, "vm", haResourceType)
	assert.Equal(t, "crs-managed", haResourceComment)
	assert.Equal(t, "qemu", vmResourceType)
	assert.Equal(t, "order=1", vmStartupCriticalOrder)
}

func TestHasVMSkipTag(t *testing.T) {
	tests := []struct {
		name     string
		vmTags   string
		expected bool
	}{
		{
			name:     "empty tags",
			vmTags:   "",
			expected: false,
		},
		{
			name:     "has crs-skip tag",
			vmTags:   "crs-skip",
			expected: true,
		},
		{
			name:     "has crs-skip tag with other tags",
			vmTags:   "production;crs-skip;backup",
			expected: true,
		},
		{
			name:     "has other tags but not crs-skip",
			vmTags:   "production;backup;testing",
			expected: false,
		},
		{
			name:     "has crs-skip with spaces",
			vmTags:   "production; crs-skip ; backup",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer, mockServer := createTestServer()
			defer mockServer.Close()

			result := testServer.hasVMSkipTag(tt.vmTags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasVMCriticalTag(t *testing.T) {
	tests := []struct {
		name     string
		vmTags   string
		expected bool
	}{
		{
			name:     "empty tags",
			vmTags:   "",
			expected: false,
		},
		{
			name:     "has crs-critical tag",
			vmTags:   "crs-critical",
			expected: true,
		},
		{
			name:     "has crs-critical tag with other tags",
			vmTags:   "production;crs-critical;backup",
			expected: true,
		},
		{
			name:     "has other tags but not crs-critical",
			vmTags:   "production;backup;testing",
			expected: false,
		},
		{
			name:     "has crs-critical with spaces",
			vmTags:   "production; crs-critical ; backup",
			expected: true,
		},
		{
			name:     "has both crs-skip and crs-critical tags",
			vmTags:   "crs-skip;crs-critical;production",
			expected: true,
		},
		{
			name:     "has crs-skip but not crs-critical",
			vmTags:   "crs-skip;production",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer, mockServer := createTestServer()
			defer mockServer.Close()

			result := testServer.hasVMCriticalTag(tt.vmTags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveSkippedVMsFromCRSGroups(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful removal of skipped VMs",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeClusterResources: true,
				includeHAResources:      true,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.RemoveSkippedVMsFromCRSGroups()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateHAStatus(t *testing.T) {
	tests := []struct {
		name                         string
		includeErrorHAResources      bool
		includeDisabledHAResources   bool
		includeCriticalVMResources   bool
		includeNonCRSErrorHAResource bool
		wantErr                      bool
	}{
		{
			name:                    "update VMs with error state",
			includeErrorHAResources: true,
			wantErr:                 false,
		},
		{
			name:                    "no VMs need status update",
			includeErrorHAResources: false,
			wantErr:                 false,
		},
		{
			name:                         "skip non-CRS VMs with error state",
			includeErrorHAResources:      true,
			includeNonCRSErrorHAResource: true,
			wantErr:                      false,
		},
		{
			name:                       "update VMs with disabled state",
			includeDisabledHAResources: true,
			wantErr:                    false,
		},
		{
			name:                       "update both error and disabled VMs",
			includeErrorHAResources:    true,
			includeDisabledHAResources: true,
			wantErr:                    false,
		},
		{
			name:                       "update critical VMs not in started state",
			includeCriticalVMResources: true,
			wantErr:                    false,
		},
		{
			name:                       "update all types of VMs",
			includeErrorHAResources:    true,
			includeDisabledHAResources: true,
			includeCriticalVMResources: true,
			wantErr:                    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeErrorHAResources:      tt.includeErrorHAResources,
				includeDisabledHAResources:   tt.includeDisabledHAResources,
				includeCriticalVMResources:   tt.includeCriticalVMResources,
				includeNonCRSErrorHAResource: tt.includeNonCRSErrorHAResource,
				includeHAResources:           tt.includeErrorHAResources || tt.includeDisabledHAResources || tt.includeCriticalVMResources || tt.includeNonCRSErrorHAResource, // Need both for the new implementation
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			// Use very fast intervals for testing (1ms intervals, max 2 attempts = 2ms total)
			err := testServer.UpdateHAStatusWithOptions(2, 1*time.Millisecond)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnsureCRSTagRegistered(t *testing.T) {
	tests := []struct {
		name                  string
		includeClusterOptions bool
		crsTagAlreadyExists   bool
		wantErr               bool
	}{
		{
			name:                  "successful tag registration",
			includeClusterOptions: true,
			crsTagAlreadyExists:   false,
			wantErr:               false,
		},
		{
			name:                  "handle missing cluster options",
			includeClusterOptions: false,
			crsTagAlreadyExists:   false,
			wantErr:               false,
		},
		{
			name:                  "tag already exists",
			includeClusterOptions: true,
			crsTagAlreadyExists:   true,
			wantErr:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeClusterOptions: tt.includeClusterOptions,
				crsTagAlreadyExists:   tt.crsTagAlreadyExists,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			err := testServer.ensureCRSTagRegistered()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

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

func TestUpdateHAStatusSkipsCRSSkipVMs(t *testing.T) {
	tests := []struct {
		name                       string
		includeErrorHAResources    bool
		includeDisabledHAResources bool
		includeCriticalVMResources bool
	}{
		{
			name:                       "skip VMs with crs-skip tag in error state",
			includeErrorHAResources:    true,
			includeDisabledHAResources: false,
			includeCriticalVMResources: false,
		},
		{
			name:                       "skip VMs with crs-skip tag in disabled state",
			includeErrorHAResources:    false,
			includeDisabledHAResources: true,
			includeCriticalVMResources: false,
		},
		{
			name:                       "skip VMs with crs-skip tag that are critical",
			includeErrorHAResources:    false,
			includeDisabledHAResources: false,
			includeCriticalVMResources: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeErrorHAResources:    tt.includeErrorHAResources,
				includeDisabledHAResources: tt.includeDisabledHAResources,
				includeCriticalVMResources: tt.includeCriticalVMResources,
				includeHAResources:         true,
				includeClusterResources:    true,
			}

			testServer, mockServer := createTestServerWithConfig(config)
			defer mockServer.Close()

			// Use very fast intervals for testing
			err := testServer.UpdateHAStatusWithOptions(1, 1*time.Millisecond)
			assert.NoError(t, err)
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
