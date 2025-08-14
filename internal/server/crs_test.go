package server

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestServer() (*Server, *httptest.Server) {
	return createTestServerWithConfig(testHandlerConfig{
		includeStorage:  true,
		includeHAGroups: true,
		includeNodes:    true,
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
		name           string
		includeStorage bool
		wantErr        bool
	}{
		{
			name:           "successful VM prefer setup with shared storage",
			includeStorage: true,
			wantErr:        false,
		},
		{
			name:           "skip VM prefer setup without shared storage",
			includeStorage: false,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testHandlerConfig{
				includeHAGroups: true,
				includeNodes:    true,
				includeStorage:  tt.includeStorage,
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

func TestCRSConstants(t *testing.T) {
	assert.Equal(t, 1000, crsMaxNodePriority)
	assert.Equal(t, 1, crsMinNodePriority)
}
