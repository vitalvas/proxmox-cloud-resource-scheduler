package proxmox

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetClusterStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/cluster/status", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"name": "test-cluster",
				"version": "7.4-3",
				"local": 1,
				"nodeid": 1,
				"nodes": 3,
				"expected_votes": 3,
				"quorate": 1
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
	status, err := client.GetClusterStatus()

	require.NoError(t, err)
	assert.Equal(t, "test-cluster", status.Name)
	assert.Equal(t, "7.4-3", status.Version)
	assert.Equal(t, 1, status.Local)
	assert.Equal(t, 1, status.NodeID)
	assert.Equal(t, 3, status.Nodes)
	assert.Equal(t, 1, status.Quorate)
}

func TestGetClusterResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/cluster/resources", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{
					"id": "node/pve1",
					"type": "node",
					"node": "pve1",
					"status": "online",
					"cpu": 0.05,
					"maxcpu": 8,
					"mem": 2147483648,
					"maxmem": 8589934592,
					"uptime": 123456
				},
				{
					"id": "qemu/100",
					"type": "qemu",
					"node": "pve1",
					"vmid": 100,
					"name": "test-vm",
					"status": "running",
					"cpu": 0.1,
					"maxcpu": 2,
					"mem": 1073741824,
					"maxmem": 2147483648,
					"tags": "production,test"
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
	resources, err := client.GetClusterResources()

	require.NoError(t, err)
	assert.Len(t, resources, 2)
	assert.Equal(t, "node/pve1", resources[0].ID)
	assert.Equal(t, "node", resources[0].Type)
	assert.Equal(t, "qemu/100", resources[1].ID)
	assert.Equal(t, "qemu", resources[1].Type)
	assert.Equal(t, 100, resources[1].VMID)
	assert.Equal(t, "production,test", resources[1].Tags)
}

func TestGetClusterHAGroups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/cluster/ha/groups", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{
					"group": "test-group",
					"nodes": "pve1:1000,pve2:500",
					"restricted": 1,
					"nofailback": 1,
					"type": "group"
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
	groups, err := client.GetClusterHAGroups()

	require.NoError(t, err)
	assert.Len(t, groups, 1)
	assert.Equal(t, "test-group", groups[0].Group)
	assert.Equal(t, "pve1:1000,pve2:500", groups[0].Nodes)
	assert.Equal(t, 1, groups[0].Restricted)
	assert.Equal(t, 1, groups[0].NoFailback)
}

func TestCreateClusterHAGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/cluster/ha/groups", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "test-group", r.Form.Get("group"))
		assert.Equal(t, "pve1:1000,pve2:500", r.Form.Get("nodes"))
		assert.Equal(t, "1", r.Form.Get("restricted"))
		assert.Equal(t, "1", r.Form.Get("nofailback"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": null}`))
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
	group := ClusterHAGroup{
		Group:      "test-group",
		Nodes:      "pve1:1000,pve2:500",
		Restricted: 1,
		NoFailback: 1,
	}

	result, err := client.CreateClusterHAGroup(group)

	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestDeleteClusterHAGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/cluster/ha/groups/test-group", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": null}`))
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
	err := client.DeleteClusterHAGroup("test-group")

	require.NoError(t, err)
}

func TestGetClusterHAResources(t *testing.T) {
	responseBody := `{
		"data": [
			{
				"sid": "vm:100",
				"state": "started",
				"group": "test-group",
				"max_relocate": 10,
				"max_restart": 10,
				"comment": "test-vm",
				"type": "vm"
			}
		]
	}`

	server, client := setupSimpleGETTest(t, "/api2/json/cluster/ha/resources", responseBody)
	defer server.Close()

	resources, err := client.GetClusterHAResources()

	require.NoError(t, err)
	assert.Len(t, resources, 1)
	assert.Equal(t, "vm:100", resources[0].SID)
	assert.Equal(t, "started", resources[0].State)
	assert.Equal(t, "test-group", resources[0].Group)
	assert.Equal(t, 10, resources[0].MaxRelocate)
	assert.Equal(t, 10, resources[0].MaxRestart)
}

func TestCreateClusterHAResource(t *testing.T) {
	formValidation := func(t *testing.T, r *http.Request) {
		assert.Equal(t, "vm:100", r.Form.Get("sid"))
		assert.Equal(t, "test-group", r.Form.Get("group"))
		assert.Equal(t, "10", r.Form.Get("max_relocate"))
		assert.Equal(t, "10", r.Form.Get("max_restart"))
		assert.Equal(t, "started", r.Form.Get("state"))
	}

	server, client := setupFormPOSTTest(t, "/api2/json/cluster/ha/resources", formValidation, `{"data": null}`)
	defer server.Close()

	resource := ClusterHAResource{
		SID:         "vm:100",
		Group:       "test-group",
		MaxRelocate: 10,
		MaxRestart:  10,
		State:       "started",
	}

	result, err := client.CreateClusterHAResource(resource)

	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestDeleteClusterHAResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/cluster/ha/resources/vm:100", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": null}`))
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
	err := client.DeleteClusterHAResource("vm:100")

	require.NoError(t, err)
}

func TestGetTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/tasks/UPID:pve1:00001234:00005678:5F8A1234:qmstart:100:root@pam:/status", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"upid": "UPID:pve1:00001234:00005678:5F8A1234:qmstart:100:root@pam:",
				"node": "pve1",
				"pid": 4660,
				"type": "qmstart",
				"id": "100",
				"user": "root@pam",
				"status": "stopped"
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
	task, err := client.GetTask("pve1", "UPID:pve1:00001234:00005678:5F8A1234:qmstart:100:root@pam:")

	require.NoError(t, err)
	assert.Equal(t, "UPID:pve1:00001234:00005678:5F8A1234:qmstart:100:root@pam:", task.UPID)
	assert.Equal(t, "pve1", task.Node)
	assert.Equal(t, "qmstart", task.Type)
	assert.Equal(t, "100", task.ID)
	assert.Equal(t, "stopped", task.Status)
}

func TestStopTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/tasks/UPID:pve1:00001234:00005678:5F8A1234:qmstart:100:root@pam:", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": null}`))
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
	err := client.StopTask("pve1", "UPID:pve1:00001234:00005678:5F8A1234:qmstart:100:root@pam:")

	require.NoError(t, err)
}
