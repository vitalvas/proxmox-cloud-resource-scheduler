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