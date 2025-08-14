package proxmox

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		authHeader := r.Header.Get("Authorization")
		assert.Equal(t, "PVEAPIToken=test@pam!test=12345678-1234-1234-1234-123456789012", authHeader)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": []}`))
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

	require.NoError(t, err)
}

func TestPasswordAuthentication(t *testing.T) {
	authCalled := false
	nodesCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/access/ticket":
			authCalled = true
			assert.Equal(t, "POST", r.Method)

			err := r.ParseForm()
			require.NoError(t, err)
			assert.Equal(t, "root@pam", r.Form.Get("username"))
			assert.Equal(t, "secret", r.Form.Get("password"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"ticket": "PVE:root@pam:12345678::abcdef123456789",
					"CSRFPreventionToken": "12345678:abcdef123456789"
				}
			}`))

		case "/api2/json/nodes":
			nodesCalled = true
			assert.Equal(t, "GET", r.Method)

			cookieHeader := r.Header.Get("Cookie")
			assert.Equal(t, "PVEAuthCookie=PVE:root@pam:12345678::abcdef123456789", cookieHeader)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": []}`))

		default:
			t.Errorf("Unexpected request to %s", r.URL.Path)
		}
	}))
	defer server.Close()

	config := &Config{
		Endpoints: []string{server.URL},
		Auth: AuthConfig{
			Method:   "password",
			Username: "root",
			Password: "secret",
			Realm:    "pam",
		},
	}

	client := NewClient(config)
	_, err := client.GetNodes()

	require.NoError(t, err)
	assert.True(t, authCalled, "Authentication endpoint should be called")
	assert.True(t, nodesCalled, "Nodes endpoint should be called")
}

func TestPasswordAuthenticationWithCSRFToken(t *testing.T) {
	authCalled := false
	createCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/access/ticket":
			authCalled = true
			assert.Equal(t, "POST", r.Method)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"ticket": "PVE:root@pam:12345678::abcdef123456789",
					"CSRFPreventionToken": "12345678:abcdef123456789"
				}
			}`))

		case "/api2/json/nodes/pve1/qemu":
			createCalled = true
			assert.Equal(t, "POST", r.Method)

			cookieHeader := r.Header.Get("Cookie")
			assert.Equal(t, "PVEAuthCookie=PVE:root@pam:12345678::abcdef123456789", cookieHeader)

			csrfHeader := r.Header.Get("CSRFPreventionToken")
			assert.Equal(t, "12345678:abcdef123456789", csrfHeader)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:qmcreate:100:root@pam:"}`))

		default:
			t.Errorf("Unexpected request to %s", r.URL.Path)
		}
	}))
	defer server.Close()

	config := &Config{
		Endpoints: []string{server.URL},
		Auth: AuthConfig{
			Method:   "password",
			Username: "root",
			Password: "secret",
			Realm:    "pam",
		},
	}

	client := NewClient(config)
	vmConfig := VMConfig{Name: "test-vm"}
	_, err := client.CreateVM("pve1", 100, vmConfig)

	require.NoError(t, err)
	assert.True(t, authCalled, "Authentication endpoint should be called")
	assert.True(t, createCalled, "Create VM endpoint should be called")
}

func TestAuthenticationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/access/ticket":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors": {"status": 401, "error": "authentication failed"}}`))

		default:
			t.Errorf("Unexpected request to %s", r.URL.Path)
		}
	}))
	defer server.Close()

	config := &Config{
		Endpoints: []string{server.URL},
		Auth: AuthConfig{
			Method:   "password",
			Username: "root",
			Password: "wrongpassword",
			Realm:    "pam",
		},
	}

	client := NewClient(config)
	_, err := client.GetNodes()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestInvalidAuthMethod(t *testing.T) {
	config := &Config{
		Endpoints: []string{"https://pve.example.com:8006"},
		Auth: AuthConfig{
			Method: "invalid",
		},
	}

	client := NewClient(config)
	_, err := client.GetNodes()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported auth method: invalid")
}

func TestMissingAuthToken(t *testing.T) {
	config := &Config{
		Endpoints: []string{"https://pve.example.com:8006"},
		Auth: AuthConfig{
			Method: "token",
		},
	}

	err := config.validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_token is required")
}

func TestMissingPasswordCredentials(t *testing.T) {
	tests := []struct {
		name     string
		config   AuthConfig
		errorMsg string
	}{
		{
			name: "missing username",
			config: AuthConfig{
				Method:   "password",
				Password: "secret",
				Realm:    "pam",
			},
			errorMsg: "username and password are required",
		},
		{
			name: "missing password",
			config: AuthConfig{
				Method:   "password",
				Username: "root",
				Realm:    "pam",
			},
			errorMsg: "username and password are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Endpoints: []string{"https://pve.example.com:8006"},
				Auth:      tt.config,
			}

			err := config.validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}
