package proxmox

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedError  string
	}{
		{
			name:           "400 bad request",
			responseStatus: http.StatusBadRequest,
			responseBody:   `{"status": 400, "error": "Bad request"}`,
			expectedError:  "API error 400: Bad request",
		},
		{
			name:           "401 unauthorized",
			responseStatus: http.StatusUnauthorized,
			responseBody:   `{"status": 401, "error": "Authentication failed"}`,
			expectedError:  "API error 401: Authentication failed",
		},
		{
			name:           "403 forbidden",
			responseStatus: http.StatusForbidden,
			responseBody:   `{"status": 403, "error": "Permission denied"}`,
			expectedError:  "API error 403: Permission denied",
		},
		{
			name:           "404 not found",
			responseStatus: http.StatusNotFound,
			responseBody:   `{"status": 404, "error": "VM not found"}`,
			expectedError:  "API error 404: VM not found",
		},
		{
			name:           "500 internal server error",
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"status": 500, "error": "Internal server error"}`,
			expectedError:  "API error 500: Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			config := &Config{
				Endpoints: []string{server.URL},
				Auth: AuthConfig{
					Method:   "token",
					APIToken: "test@pam!test=12345678-1234-1234-1234-123456789012",
				},
			}

			client := NewClient(config)
			_, err := client.GetNodes()

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestHTTPErrorWithoutAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Plain text error"))
	}))
	defer server.Close()

	config := &Config{
		Endpoints: []string{server.URL},
		Auth: AuthConfig{
			Method:   "token",
			APIToken: "test@pam!test=12345678-1234-1234-1234-123456789012",
		},
	}

	client := NewClient(config)
	_, err := client.GetNodes()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestNon2xxStatusCodes(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		response     string
		wantError    string
		expectsError bool
	}{
		{
			name:         "301 redirect with API error",
			statusCode:   301,
			response:     `{"error": "Moved permanently"}`,
			wantError:    "API error 301: Moved permanently",
			expectsError: true,
		},
		{
			name:         "302 redirect with plain text", 
			statusCode:   302,
			response:     `Plain text redirect`,
			wantError:    "HTTP 302",
			expectsError: true,
		},
		{
			name:         "304 not modified",
			statusCode:   304,
			response:     ``,
			wantError:    "HTTP 304",
			expectsError: true,
		},
		{
			name:         "204 no content (should be successful)",
			statusCode:   204,
			response:     `{"data": null}`,
			expectsError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.response != "" {
					w.Write([]byte(tt.response))
				}
			}))
			defer server.Close()

			config := &Config{
				Endpoints: []string{server.URL},
				Auth: AuthConfig{
					Method:   "token",
					APIToken: "test@pam!test=12345678-1234-1234-1234-123456789012",
				},
			}

			client := NewClient(config)
			_, err := client.GetNodes()
			
			if tt.expectsError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNetworkError(t *testing.T) {
	config := &Config{
		Endpoints: []string{"http://invalid-host:8006"},
		Auth: AuthConfig{
			Method:   "token",
			APIToken: "test@pam!test=12345678-1234-1234-1234-123456789012",
		},
	}

	client := NewClient(config)
	_, err := client.GetNodes()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}

func TestInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": invalid json}`))
	}))
	defer server.Close()

	config := &Config{
		Endpoints: []string{server.URL},
		Auth: AuthConfig{
			Method:   "token",
			APIToken: "test@pam!test=12345678-1234-1234-1234-123456789012",
		},
	}

	client := NewClient(config)
	_, err := client.GetNodes()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestConfigValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectedErr string
	}{
		{
			name: "no endpoints",
			config: &Config{
				Endpoints: []string{},
				Auth: AuthConfig{
					Method:   "token",
					APIToken: "test@pam!test=12345678-1234-1234-1234-123456789012",
				},
			},
			expectedErr: "no endpoints specified",
		},
		{
			name: "unsupported auth method",
			config: &Config{
				Endpoints: []string{"https://pve.example.com:8006"},
				Auth: AuthConfig{
					Method: "oauth",
				},
			},
			expectedErr: "unsupported auth method: oauth",
		},
		{
			name: "missing API token",
			config: &Config{
				Endpoints: []string{"https://pve.example.com:8006"},
				Auth: AuthConfig{
					Method: "token",
				},
			},
			expectedErr: "api_token is required when method is 'token'",
		},
		{
			name: "missing username for password auth",
			config: &Config{
				Endpoints: []string{"https://pve.example.com:8006"},
				Auth: AuthConfig{
					Method:   "password",
					Password: "secret",
				},
			},
			expectedErr: "username and password are required when method is 'password'",
		},
		{
			name: "missing password for password auth",
			config: &Config{
				Endpoints: []string{"https://pve.example.com:8006"},
				Auth: AuthConfig{
					Method:   "password",
					Username: "root",
				},
			},
			expectedErr: "username and password are required when method is 'password'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestEndpointLoadBalancing(t *testing.T) {
	callCount := make(map[string]int)

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount["server1"]++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount["server2"]++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
	}))
	defer server2.Close()

	config := &Config{
		Endpoints: []string{server1.URL, server2.URL},
		Auth: AuthConfig{
			Method:   "token",
			APIToken: "test@pam!test=12345678-1234-1234-1234-123456789012",
		},
	}

	client := NewClient(config)

	for i := 0; i < 10; i++ {
		_, err := client.GetNodes()
		require.NoError(t, err)
	}

	totalCalls := callCount["server1"] + callCount["server2"]
	assert.Equal(t, 10, totalCalls)
	assert.True(t, callCount["server1"] > 0 || callCount["server2"] > 0, "At least one server should receive calls")
}

func TestBuildURLError(t *testing.T) {
	client := &Client{
		config: &Config{
			Endpoints: []string{},
		},
	}

	url := client.buildURL("test")
	assert.Empty(t, url)
}
