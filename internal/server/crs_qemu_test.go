package server

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestServerForQemu() (*Server, *httptest.Server) {
	return createTestServerWithConfig(testHandlerConfig{
		includeHAResources: true,
		includeNodes:       true,
		includeNodeVMs:     true,
	})
}

func TestSetupCRSQemu(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful QEMU CRS setup",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testServer, mockServer := createTestServerForQemu()
			defer mockServer.Close()

			err := testServer.SetupCRSQemu()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
