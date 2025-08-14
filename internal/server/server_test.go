package server

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		expectError bool
	}{
		{
			name: "missing consul environment",
			setupEnv: func() {
				os.Unsetenv("CONSUL_URL")
				os.Unsetenv("CONSUL_TOKEN")
			},
			cleanupEnv:  func() {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			server, err := New()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, server)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, server)
				assert.NotNil(t, server.consul)
				assert.NotNil(t, server.proxmox)
			}
		})
	}
}

func TestServer_RunContextCancellation(t *testing.T) {
	t.Skip("Skipping Run test as it requires proper initialization and would run indefinitely")
}

func TestPeriodicTime(t *testing.T) {
	assert.Equal(t, 30, periodicTime)
}
