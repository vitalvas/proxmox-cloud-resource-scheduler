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
		name          string
		wantErr       bool
		expectedCalls int
	}{
		{
			name:          "successful CRS setup",
			wantErr:       false,
			expectedCalls: 1,
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

func TestCRSConstants(t *testing.T) {
	assert.Equal(t, 1000, crsMaxNodePriority)
	assert.Equal(t, 1, crsMinNodePriority)
}
