package proxmox

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNodes(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		responseStatus int
		wantErr        bool
		expectedCount  int
	}{
		{
			name: "successful response",
			responseBody: `{
				"data": [
					{
						"id": "node/pve1",
						"node": "pve1",
						"type": "node",
						"status": "online",
						"cpu": 0.05,
						"maxcpu": 8,
						"mem": 2147483648,
						"maxmem": 8589934592,
						"disk": 1073741824,
						"maxdisk": 53687091200,
						"uptime": 123456,
						"level": ""
					},
					{
						"id": "node/pve2", 
						"node": "pve2",
						"type": "node",
						"status": "online",
						"cpu": 0.03,
						"maxcpu": 4,
						"mem": 1073741824,
						"maxmem": 4294967296,
						"disk": 536870912,
						"maxdisk": 26843545600,
						"uptime": 98765,
						"level": ""
					}
				]
			}`,
			responseStatus: http.StatusOK,
			wantErr:        false,
			expectedCount:  2,
		},
		{
			name:           "server error",
			responseBody:   `{"errors": {"status": 500, "error": "Internal server error"}}`,
			responseStatus: http.StatusInternalServerError,
			wantErr:        true,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api2/json/nodes", r.URL.Path)
				assert.Equal(t, "GET", r.Method)

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
			nodes, err := client.GetNodes()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, nodes, tt.expectedCount)

			if len(nodes) > 0 {
				assert.Equal(t, "pve1", nodes[0].Node)
				assert.Equal(t, "online", nodes[0].Status)
				assert.Equal(t, 8, nodes[0].MaxCPU)
			}
		})
	}
}

func TestGetNode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/status", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"id": "node/pve1",
				"node": "pve1",
				"type": "node",
				"status": "online",
				"cpu": 0.05,
				"maxcpu": 8,
				"mem": 2147483648,
				"maxmem": 8589934592,
				"uptime": 123456
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
	node, err := client.GetNode("pve1")

	require.NoError(t, err)
	assert.Equal(t, "pve1", node.Node)
	assert.Equal(t, "online", node.Status)
	assert.Equal(t, 8, node.MaxCPU)
}

func TestGetNodeVMs(t *testing.T) {
	responseBody := `{
		"data": [
			{
				"vmid": 100,
				"name": "test-vm",
				"node": "pve1",
				"status": "running",
				"template": 0,
				"cpu": 0.1,
				"maxcpu": 2,
				"mem": 1073741824,
				"maxmem": 2147483648,
				"disk": 0,
				"maxdisk": 10737418240,
				"uptime": 3600,
				"tags": "test,production"
			}
		]
	}`

	server, client := setupSimpleGETTest(t, "/api2/json/nodes/pve1/qemu", responseBody)
	defer server.Close()

	vms, err := client.GetNodeVMs("pve1")

	require.NoError(t, err)
	assert.Len(t, vms, 1)
	assert.Equal(t, 100, vms[0].VMID)
	assert.Equal(t, "test-vm", vms[0].Name)
	assert.Equal(t, "running", vms[0].Status)
	assert.Equal(t, "test,production", vms[0].Tags)
}

func TestGetNodeContainers(t *testing.T) {
	responseBody := `{
		"data": [
			{
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
				"disk": 0,
				"maxdisk": 5368709120,
				"uptime": 1800,
				"tags": "container,test"
			}
		]
	}`

	server, client := setupSimpleGETTest(t, "/api2/json/nodes/pve1/lxc", responseBody)
	defer server.Close()

	containers, err := client.GetNodeContainers("pve1")

	require.NoError(t, err)
	assert.Len(t, containers, 1)
	assert.Equal(t, 200, containers[0].VMID)
	assert.Equal(t, "test-container", containers[0].Name)
	assert.Equal(t, "running", containers[0].Status)
	assert.Equal(t, "container,test", containers[0].Tags)
}

func TestGetNodeTasks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api2/json/nodes/pve1/tasks", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{
					"upid": "UPID:pve1:00001234:00005678:5F8A1234:vzstart:200:root@pam:",
					"node": "pve1",
					"pid": 4660,
					"type": "vzstart",
					"id": "200",
					"user": "root@pam",
					"status": "stopped"
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
	tasks, err := client.GetNodeTasks("pve1")

	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "UPID:pve1:00001234:00005678:5F8A1234:vzstart:200:root@pam:", tasks[0].UPID)
	assert.Equal(t, "pve1", tasks[0].Node)
	assert.Equal(t, "vzstart", tasks[0].Type)
	assert.Equal(t, "stopped", tasks[0].Status)
}

func TestShutdownNode(t *testing.T) {
	server, client := setupNodeCommandTest(t, "shutdown", "UPID:pve1:00001234:00005678:5F8A1234:srvshutdown::root@pam:")
	defer server.Close()

	taskID, err := client.ShutdownNode("pve1")

	require.NoError(t, err)
	assert.Contains(t, taskID, "srvshutdown")
}

func TestRebootNode(t *testing.T) {
	server, client := setupNodeCommandTest(t, "reboot", "UPID:pve1:00001234:00005678:5F8A1234:srvreboot::root@pam:")
	defer server.Close()

	taskID, err := client.RebootNode("pve1")

	require.NoError(t, err)
	assert.Contains(t, taskID, "srvreboot")
}
