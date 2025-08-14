package server

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
