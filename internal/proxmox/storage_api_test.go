package proxmox

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStorage(t *testing.T) {
	responseBody := `{
		"data": [
			{
				"storage": "local",
				"type": "dir",
				"content": "images,vztmpl,iso,backup",
				"nodes": "",
				"shared": 0,
				"used": 1073741824,
				"avail": 53687091200,
				"total": 54760833024,
				"used_fraction": 0.0196,
				"enabled": 1,
				"active": 1
			},
			{
				"storage": "shared-storage",
				"type": "nfs",
				"content": "images,rootdir",
				"nodes": "",
				"shared": 1,
				"used": 2147483648,
				"avail": 107374182400,
				"total": 109521666048,
				"used_fraction": 0.0196,
				"enabled": 1,
				"active": 1
			}
		]
	}`

	server, client := setupSimpleGETTest(t, "/api2/json/storage", responseBody)
	defer server.Close()

	storage, err := client.GetStorage()

	require.NoError(t, err)
	assert.Len(t, storage, 2)
	assert.Equal(t, "local", storage[0].Storage)
	assert.Equal(t, "dir", storage[0].Type)
	assert.Equal(t, 0, storage[0].Shared)
	assert.Equal(t, "shared-storage", storage[1].Storage)
	assert.Equal(t, 1, storage[1].Shared)
}

func TestGetStorageContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/storage/local/content", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{
					"volid": "local:100/vm-100-disk-0.qcow2",
					"format": "qcow2",
					"size": 10737418240,
					"used": 1073741824,
					"content": "images",
					"vmid": 100
				},
				{
					"volid": "local:backup/vzdump-qemu-100-2023_01_01-00_00_00.vma.gz",
					"format": "vma.gz",
					"size": 536870912,
					"used": 536870912,
					"content": "backup",
					"vmid": 100
				}
			]
		}`))
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
	content, err := client.GetStorageContent("pve1", "local")

	require.NoError(t, err)
	assert.Len(t, content, 2)
	assert.Equal(t, "local:100/vm-100-disk-0.qcow2", content[0].VolID)
	assert.Equal(t, "qcow2", content[0].Format)
	assert.Equal(t, int64(10737418240), content[0].Size)
	assert.Equal(t, 100, content[0].VMID)
}

func TestGetStorageStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/storage/local/status", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"storage": "local",
				"type": "dir",
				"total": 54760833024,
				"used": 1073741824,
				"avail": 53687091200,
				"enabled": 1,
				"active": 1,
				"used_fraction": 0.0196
			}
		}`))
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
	status, err := client.GetStorageStatus("pve1", "local")

	require.NoError(t, err)
	assert.Equal(t, "local", status.Storage)
	assert.Equal(t, "dir", status.Type)
	assert.Equal(t, int64(54760833024), status.Total)
	assert.Equal(t, int64(1073741824), status.Used)
	assert.Equal(t, int64(53687091200), status.Avail)
	assert.Equal(t, 1, status.Enabled)
	assert.Equal(t, 1, status.Active)
}

func TestDeleteFromStorage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/storage/local/content/local:100/vm-100-disk-0.qcow2", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:imgdel::root@pam:"}`))
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
	taskID, err := client.DeleteFromStorage("pve1", "local", "local:100/vm-100-disk-0.qcow2")

	require.NoError(t, err)
	assert.Contains(t, taskID, "imgdel")
}

func TestCreateStorageBackup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/storage/backup/backup", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "backup", r.Form.Get("storage"))
		assert.Equal(t, "100,200", r.Form.Get("vmid"))
		assert.Equal(t, "snapshot", r.Form.Get("mode"))
		assert.Equal(t, "gzip", r.Form.Get("compress"))
		assert.Equal(t, "1", r.Form.Get("protected"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": "UPID:pve1:00001234:00005678:5F8A1234:vzbackup::root@pam:"}`))
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
	vmids := []int{100, 200}
	options := BackupOptions{
		Storage:   "backup",
		Mode:      "snapshot",
		Compress:  "gzip",
		Protected: true,
	}

	taskID, err := client.CreateStorageBackup("pve1", "backup", vmids, options)

	require.NoError(t, err)
	assert.Contains(t, taskID, "vzbackup")
}

func TestHasSharedStorage(t *testing.T) {
	tests := []struct {
		name          string
		responseBody  string
		expectedValue bool
	}{
		{
			name: "has shared storage",
			responseBody: `{
				"data": [
					{
						"storage": "local",
						"content": "backup,iso",
						"shared": 0
					},
					{
						"storage": "shared-nfs",
						"content": "images,rootdir",
						"shared": 1
					}
				]
			}`,
			expectedValue: true,
		},
		{
			name: "no shared storage",
			responseBody: `{
				"data": [
					{
						"storage": "local",
						"content": "images,backup,iso",
						"shared": 0
					},
					{
						"storage": "local-lvm",
						"content": "rootdir",
						"shared": 0
					}
				]
			}`,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api2/json/storage", r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
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
			hasShared, err := client.HasSharedStorage()

			require.NoError(t, err)
			assert.Equal(t, tt.expectedValue, hasShared)
		})
	}
}
