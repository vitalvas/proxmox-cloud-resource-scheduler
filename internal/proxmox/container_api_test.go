package proxmox

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetContainer(t *testing.T) {
	responseBody := `{
		"data": {
			"vmid": 200,
			"name": "test-container",
			"node": "pve1",
			"status": "running",
			"template": 0,
			"cpu": 0.05,
			"maxcpu": 1,
			"mem": 536870912,
			"maxmem": 1073741824,
			"swap": 0,
			"maxswap": 536870912,
			"uptime": 1800,
			"tags": "container,test"
		}
	}`

	server, client := setupSimpleGETTest(t, "/api2/json/nodes/pve1/lxc/200/status/current", responseBody)
	defer server.Close()

	container, err := client.GetContainer("pve1", 200)

	require.NoError(t, err)
	assert.Equal(t, 200, container.VMID)
	assert.Equal(t, "test-container", container.Name)
	assert.Equal(t, "running", container.Status)
	assert.Equal(t, "container,test", container.Tags)
}

func TestCreateContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/lxc", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "200", r.Form.Get("vmid"))
		assert.Equal(t, "local:vztmpl/ubuntu-20.04-standard_20.04-1_amd64.tar.gz", r.Form.Get("ostemplate"))
		assert.Equal(t, "test-container", r.Form.Get("hostname"))
		assert.Equal(t, "1024", r.Form.Get("memory"))
		assert.Equal(t, "1", r.Form.Get("cores"))
		assert.Equal(t, "1", r.Form.Get("unprivileged"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:vzcreate:200:root@pam:"}`))
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
	containerConfig := ContainerConfig{
		OSTemplate:   "local:vztmpl/ubuntu-20.04-standard_20.04-1_amd64.tar.gz",
		Hostname:     "test-container",
		Memory:       1024,
		Cores:        1,
		Unprivileged: true,
	}

	taskID, err := client.CreateContainer("pve1", 200, containerConfig)

	require.NoError(t, err)
	assert.Contains(t, taskID, "vzcreate")
}

func TestStartContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/lxc/200/status/start", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:vzstart:200:root@pam:"}`))
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
	taskID, err := client.StartContainer("pve1", 200)

	require.NoError(t, err)
	assert.Contains(t, taskID, "vzstart")
}

func TestStopContainer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/lxc/200/status/stop", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:vzstop:200:root@pam:"}`))
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
	taskID, err := client.StopContainer("pve1", 200)

	require.NoError(t, err)
	assert.Contains(t, taskID, "vzstop")
}

func TestMigrateContainer(t *testing.T) {
	server, client := setupMigrationTest(t, "/api2/json/nodes/pve1/lxc/200/migrate", "UPID:pve1:00001234:00005678:5F8A1234:vzmigrate:200:root@pam:")
	defer server.Close()

	options := createMigrationOptions()
	taskID, err := client.MigrateContainer("pve1", 200, options)

	require.NoError(t, err)
	assert.Contains(t, taskID, "vzmigrate")
}

func TestCloneContainer(t *testing.T) {
	formValidation := func(t *testing.T, r *http.Request) {
		assert.Equal(t, "201", r.Form.Get("newid"))
		assert.Equal(t, "1", r.Form.Get("full"))
	}

	server, client := setupFormPOSTTest(t, "/api2/json/nodes/pve1/lxc/200/clone", formValidation, `{"data": "UPID:pve1:00001234:00005678:5F8A1234:vzclone:200:root@pam:"}`)
	defer server.Close()

	taskID, err := client.CloneContainer("pve1", 200, 201, true)

	require.NoError(t, err)
	assert.Contains(t, taskID, "vzclone")
}

func TestCreateContainerSnapshot(t *testing.T) {
	server, client := setupSnapshotTest(t, "/api2/json/nodes/pve1/lxc/200/snapshot", "UPID:pve1:00001234:00005678:5F8A1234:vzsnapshot:200:root@pam:")
	defer server.Close()

	taskID, err := client.CreateContainerSnapshot("pve1", 200, "test-snapshot", "Test snapshot description")

	require.NoError(t, err)
	assert.Contains(t, taskID, "vzsnapshot")
}

func TestDeleteContainerSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/lxc/200/snapshot/test-snapshot", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:vzdelsnapshot:200:root@pam:"}`))
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
	taskID, err := client.DeleteContainerSnapshot("pve1", 200, "test-snapshot")

	require.NoError(t, err)
	assert.Contains(t, taskID, "vzdelsnapshot")
}

func TestCreateContainerBackup(t *testing.T) {
	server, client := setupBackupTest(t, "/api2/json/nodes/pve1/lxc/200/backup", "UPID:pve1:00001234:00005678:5F8A1234:vzbackup:200:root@pam:")
	defer server.Close()

	options := createBackupOptions()
	taskID, err := client.CreateContainerBackup("pve1", 200, options)

	require.NoError(t, err)
	assert.Contains(t, taskID, "vzbackup")
}
