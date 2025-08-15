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
