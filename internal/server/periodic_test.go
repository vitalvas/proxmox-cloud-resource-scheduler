package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunPeriodic(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() *Server
		expectedError bool
	}{
		{
			name: "successful periodic run",
			setupServer: func() *Server {
				srv, _ := createTestServerWithConfig(testHandlerConfig{
					includeStorage:     true,
					includeHAGroups:    true,
					includeNodes:       true,
					includeHAResources: true,
					includeNodeVMs:     true,
				})
				return srv
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			err := server.runPeriodic()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
