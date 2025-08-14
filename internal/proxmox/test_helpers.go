package proxmox

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestClient creates a test client with token authentication
func createTestClient(serverURL string) *Client {
	config := &Config{
		Endpoints: []string{serverURL},
		Auth: AuthConfig{
			Method:   "token",
			APIToken: "test@pam!test=12345678-1234-1234-1234-123456789012",
		},
	}
	return NewClient(config)
}

// writeJSONResponse writes a JSON response with the given status and body
func writeJSONResponse(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(body))
}

// assertHTTPRequest checks that the HTTP request matches expected method and path
func assertHTTPRequest(t *testing.T, r *http.Request, expectedMethod, expectedPath string) {
	require.Equal(t, expectedMethod, r.Method)
	require.Equal(t, expectedPath, r.URL.Path)
}

// setupBackupTest creates a test server for backup operations
func setupBackupTest(t *testing.T, expectedPath, expectedTaskID string) (*httptest.Server, *Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertHTTPRequest(t, r, "POST", expectedPath)

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "local", r.Form.Get("storage"))
		assert.Equal(t, "snapshot", r.Form.Get("mode"))
		assert.Equal(t, "gzip", r.Form.Get("compress"))
		assert.Equal(t, "1", r.Form.Get("protected"))

		writeJSONResponse(w, http.StatusOK, `{"data": "`+expectedTaskID+`"}`)
	}))

	client := createTestClient(server.URL)
	return server, client
}

// createBackupOptions returns standard backup options for testing
func createBackupOptions() BackupOptions {
	return BackupOptions{
		Storage:   "local",
		Mode:      "snapshot",
		Compress:  "gzip",
		Protected: true,
	}
}

// setupMigrationTest creates a test server for migration operations
func setupMigrationTest(t *testing.T, expectedPath, expectedTaskID string) (*httptest.Server, *Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertHTTPRequest(t, r, "POST", expectedPath)

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "pve2", r.Form.Get("target"))
		assert.Equal(t, "1", r.Form.Get("online"))

		writeJSONResponse(w, http.StatusOK, `{"data": "`+expectedTaskID+`"}`)
	}))

	client := createTestClient(server.URL)
	return server, client
}

// createMigrationOptions returns standard migration options for testing
func createMigrationOptions() MigrationOptions {
	return MigrationOptions{
		Target: "pve2",
		Online: true,
	}
}

// setupSimpleGETTest creates a test server for simple GET operations
func setupSimpleGETTest(t *testing.T, expectedPath, responseBody string) (*httptest.Server, *Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertHTTPRequest(t, r, "GET", expectedPath)
		writeJSONResponse(w, http.StatusOK, responseBody)
	}))

	client := createTestClient(server.URL)
	return server, client
}

// setupFormPOSTTest creates a test server for POST operations with form data validation
func setupFormPOSTTest(t *testing.T, expectedPath string, formValidation func(*testing.T, *http.Request), responseBody string) (*httptest.Server, *Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertHTTPRequest(t, r, "POST", expectedPath)

		err := r.ParseForm()
		require.NoError(t, err)

		if formValidation != nil {
			formValidation(t, r)
		}

		writeJSONResponse(w, http.StatusOK, responseBody)
	}))

	client := createTestClient(server.URL)
	return server, client
}

// setupSnapshotTest creates a test server for snapshot operations
func setupSnapshotTest(t *testing.T, expectedPath, expectedTaskID string) (*httptest.Server, *Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertHTTPRequest(t, r, "POST", expectedPath)

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "test-snapshot", r.Form.Get("snapname"))
		assert.Equal(t, "Test snapshot description", r.Form.Get("description"))

		writeJSONResponse(w, http.StatusOK, `{"data": "`+expectedTaskID+`"}`)
	}))

	client := createTestClient(server.URL)
	return server, client
}

// setupNodeCommandTest creates a test server for node command operations (shutdown/reboot)
func setupNodeCommandTest(t *testing.T, expectedCommand, expectedTaskID string) (*httptest.Server, *Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertHTTPRequest(t, r, "POST", "/api2/json/nodes/pve1/status")

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, expectedCommand, r.Form.Get("command"))

		writeJSONResponse(w, http.StatusOK, `{"data": "`+expectedTaskID+`"}`)
	}))

	client := createTestClient(server.URL)
	return server, client
}
