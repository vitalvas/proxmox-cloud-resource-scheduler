package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

func TestUpdateHAStatusSkipsMigratingVMs(t *testing.T) {
	t.Run("should skip critical VMs in migrate state", func(t *testing.T) {
		config := testHandlerConfig{
			includeHAResources:              true,
			includeClusterResources:         true,
			includeCriticalVMInMigrateState: true, // Include VM with critical tag in migrate state
		}

		testServer, mockServer := createTestServerWithConfig(config)
		defer mockServer.Close()

		// The VM with critical tag in migrate state should be skipped
		err := testServer.UpdateHAStatusWithOptions(1, 1*time.Millisecond)
		assert.NoError(t, err)
	})
}
