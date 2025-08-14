package proxmox

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid token config",
			config: &Config{
				Endpoints: []string{"https://pve.example.com:8006"},
				Auth: AuthConfig{
					Method:   "token",
					APIToken: "user@pam!token=12345678-1234-1234-1234-123456789012",
				},
			},
			wantErr: false,
		},
		{
			name: "valid password config",
			config: &Config{
				Endpoints: []string{"https://pve.example.com:8006"},
				Auth: AuthConfig{
					Method:   "password",
					Username: "root",
					Password: "secret",
					Realm:    "pam",
				},
			},
			wantErr: false,
		},
		{
			name: "empty endpoints",
			config: &Config{
				Endpoints: []string{},
				Auth: AuthConfig{
					Method:   "token",
					APIToken: "user@pam!token=12345678-1234-1234-1234-123456789012",
				},
			},
			wantErr: true,
		},
		{
			name: "token method without api_token",
			config: &Config{
				Endpoints: []string{"https://pve.example.com:8006"},
				Auth: AuthConfig{
					Method: "token",
				},
			},
			wantErr: true,
		},
		{
			name: "password method without username",
			config: &Config{
				Endpoints: []string{"https://pve.example.com:8006"},
				Auth: AuthConfig{
					Method:   "password",
					Password: "secret",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	config := &Config{
		Endpoints: []string{"https://pve.example.com:8006"},
		Auth: AuthConfig{
			Method:   "token",
			APIToken: "user@pam!token=12345678-1234-1234-1234-123456789012",
		},
		Timeout: 30 * time.Second,
		TLS: TLSConfig{
			InsecureSkipVerify: true,
		},
	}

	client := NewClient(config)
	assert.NotNil(t, client)
	assert.Equal(t, config, client.config)
	assert.NotNil(t, client.httpClient)
}

func TestClientGetRandomEndpoint(t *testing.T) {
	tests := []struct {
		name      string
		endpoints []string
		wantEmpty bool
	}{
		{
			name:      "single endpoint",
			endpoints: []string{"https://pve1.example.com:8006"},
			wantEmpty: false,
		},
		{
			name:      "multiple endpoints",
			endpoints: []string{"https://pve1.example.com:8006", "https://pve2.example.com:8006"},
			wantEmpty: false,
		},
		{
			name:      "no endpoints",
			endpoints: []string{},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: &Config{
					Endpoints: tt.endpoints,
				},
			}

			endpoint := client.getRandomEndpoint()
			if tt.wantEmpty {
				assert.Empty(t, endpoint)
			} else {
				assert.NotEmpty(t, endpoint)
				assert.Contains(t, tt.endpoints, endpoint)
			}
		})
	}
}

func TestClientBuildURL(t *testing.T) {
	client := &Client{
		config: &Config{
			Endpoints: []string{"https://pve.example.com:8006"},
		},
	}

	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{
			name:     "simple endpoint",
			endpoint: "nodes",
			want:     "https://pve.example.com:8006/api2/json/nodes",
		},
		{
			name:     "nested endpoint",
			endpoint: "nodes/pve1/qemu",
			want:     "https://pve.example.com:8006/api2/json/nodes/pve1/qemu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := client.buildURL(tt.endpoint)
			assert.Equal(t, tt.want, url)
		})
	}
}
