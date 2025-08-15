package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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