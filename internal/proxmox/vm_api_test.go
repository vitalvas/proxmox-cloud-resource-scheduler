package proxmox

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVM(t *testing.T) {
	responseBody := `{
		"data": {
			"vmid": 100,
			"name": "test-vm",
			"node": "pve1",
			"status": "running",
			"template": 0,
			"cpu": 0.1,
			"maxcpu": 2,
			"mem": 1073741824,
			"maxmem": 2147483648,
			"uptime": 3600,
			"tags": "test,production"
		}
	}`

	server, client := setupSimpleGETTest(t, "/api2/json/nodes/pve1/qemu/100/status/current", responseBody)
	defer server.Close()

	vm, err := client.GetVM("pve1", 100)

	require.NoError(t, err)
	assert.Equal(t, 100, vm.VMID)
	assert.Equal(t, "test-vm", vm.Name)
	assert.Equal(t, "running", vm.Status)
	assert.Equal(t, "test,production", vm.Tags)
}

func TestCreateVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/qemu", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "100", r.Form.Get("vmid"))
		assert.Equal(t, "test-vm", r.Form.Get("name"))
		assert.Equal(t, "2", r.Form.Get("cores"))
		assert.Equal(t, "2048", r.Form.Get("memory"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:qmcreate:100:root@pam:"}`))
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
	vmConfig := VMConfig{
		Name:   "test-vm",
		Cores:  2,
		Memory: 2048,
	}

	taskID, err := client.CreateVM("pve1", 100, vmConfig)

	require.NoError(t, err)
	assert.Contains(t, taskID, "qmcreate")
}

func TestStartVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/qemu/100/status/start", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:qmstart:100:root@pam:"}`))
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
	taskID, err := client.StartVM("pve1", 100)

	require.NoError(t, err)
	assert.Contains(t, taskID, "qmstart")
}

func TestStopVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/qemu/100/status/stop", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:qmstop:100:root@pam:"}`))
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
	taskID, err := client.StopVM("pve1", 100)

	require.NoError(t, err)
	assert.Contains(t, taskID, "qmstop")
}

func TestMigrateVM(t *testing.T) {
	server, client := setupMigrationTest(t, "/api2/json/nodes/pve1/qemu/100/migrate", "UPID:pve1:00001234:00005678:5F8A1234:qmigrate:100:root@pam:")
	defer server.Close()

	options := createMigrationOptions()
	taskID, err := client.MigrateVM("pve1", 100, options)

	require.NoError(t, err)
	assert.Contains(t, taskID, "qmigrate")
}

func TestCloneVM(t *testing.T) {
	formValidation := func(t *testing.T, r *http.Request) {
		assert.Equal(t, "101", r.Form.Get("newid"))
		assert.Equal(t, "1", r.Form.Get("full"))
	}

	server, client := setupFormPOSTTest(t, "/api2/json/nodes/pve1/qemu/100/clone", formValidation, `{"data": "UPID:pve1:00001234:00005678:5F8A1234:qmclone:100:root@pam:"}`)
	defer server.Close()

	taskID, err := client.CloneVM("pve1", 100, 101, true)

	require.NoError(t, err)
	assert.Contains(t, taskID, "qmclone")
}

func TestCreateVMSnapshot(t *testing.T) {
	server, client := setupSnapshotTest(t, "/api2/json/nodes/pve1/qemu/100/snapshot", "UPID:pve1:00001234:00005678:5F8A1234:qmsnapshot:100:root@pam:")
	defer server.Close()

	taskID, err := client.CreateVMSnapshot("pve1", 100, "test-snapshot", "Test snapshot description")

	require.NoError(t, err)
	assert.Contains(t, taskID, "qmsnapshot")
}

func TestDeleteVMSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/qemu/100/snapshot/test-snapshot", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:qmdelsnapshot:100:root@pam:"}`))
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
	taskID, err := client.DeleteVMSnapshot("pve1", 100, "test-snapshot")

	require.NoError(t, err)
	assert.Contains(t, taskID, "qmdelsnapshot")
}

func TestCreateVMBackup(t *testing.T) {
	server, client := setupBackupTest(t, "/api2/json/nodes/pve1/qemu/100/backup", "UPID:pve1:00001234:00005678:5F8A1234:qmbackup:100:root@pam:")
	defer server.Close()

	options := createBackupOptions()
	taskID, err := client.CreateVMBackup("pve1", 100, options)

	require.NoError(t, err)
	assert.Contains(t, taskID, "qmbackup")
}
