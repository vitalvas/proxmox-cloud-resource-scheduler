package proxmox

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
)

func (c *Client) GetContainer(node string, vmid int) (*Container, error) {
	var container Container
	endpoint := fmt.Sprintf("nodes/%s/lxc/%d/status/current", node, vmid)
	if err := c.Get(endpoint, &container); err != nil {
		return nil, fmt.Errorf("failed to get container %d on node %s: %w", vmid, node, err)
	}

	logging.Debugf("Retrieved container %d on node %s", vmid, node)
	return &container, nil
}

func (c *Client) CreateContainer(node string, vmid int, config ContainerConfig) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/lxc", node)
	data := url.Values{}
	data.Set("vmid", strconv.Itoa(vmid))
	data.Set("ostemplate", config.OSTemplate)

	if config.Hostname != "" {
		data.Set("hostname", config.Hostname)
	}
	if config.Description != "" {
		data.Set("description", config.Description)
	}
	if config.Memory > 0 {
		data.Set("memory", strconv.Itoa(config.Memory))
	}
	if config.Swap > 0 {
		data.Set("swap", strconv.Itoa(config.Swap))
	}
	if config.Cores > 0 {
		data.Set("cores", strconv.Itoa(config.Cores))
	}
	if config.RootFS != "" {
		data.Set("rootfs", config.RootFS)
	}
	if config.Tags != "" {
		data.Set("tags", config.Tags)
	}
	if config.Unprivileged {
		data.Set("unprivileged", "1")
	}

	for key, value := range config.Networks {
		data.Set(key, value)
	}

	var taskID string
	if err := c.Post(endpoint, data, &taskID); err != nil {
		return "", fmt.Errorf("failed to create container %d on node %s: %w", vmid, node, err)
	}

	return taskID, nil
}

func (c *Client) StartContainer(node string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/lxc/%d/status/start", node, vmid)

	var taskID string
	if err := c.Post(endpoint, nil, &taskID); err != nil {
		return "", fmt.Errorf("failed to start container %d on node %s: %w", vmid, node, err)
	}

	return taskID, nil
}

func (c *Client) StopContainer(node string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/lxc/%d/status/stop", node, vmid)

	var taskID string
	if err := c.Post(endpoint, nil, &taskID); err != nil {
		return "", fmt.Errorf("failed to stop container %d on node %s: %w", vmid, node, err)
	}

	return taskID, nil
}

func (c *Client) ShutdownContainer(node string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/lxc/%d/status/shutdown", node, vmid)

	var taskID string
	if err := c.Post(endpoint, nil, &taskID); err != nil {
		return "", fmt.Errorf("failed to shutdown container %d on node %s: %w", vmid, node, err)
	}

	return taskID, nil
}

func (c *Client) RebootContainer(node string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/lxc/%d/status/reboot", node, vmid)

	var taskID string
	if err := c.Post(endpoint, nil, &taskID); err != nil {
		return "", fmt.Errorf("failed to reboot container %d on node %s: %w", vmid, node, err)
	}

	return taskID, nil
}

func (c *Client) DeleteContainer(node string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/lxc/%d", node, vmid)

	var taskID string
	if err := c.Delete(endpoint); err != nil {
		return "", fmt.Errorf("failed to delete container %d on node %s: %w", vmid, node, err)
	}

	return taskID, nil
}

func (c *Client) MigrateContainer(node string, vmid int, options MigrationOptions) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/lxc/%d/migrate", node, vmid)
	data := url.Values{}
	data.Set("target", options.Target)

	if options.Online {
		data.Set("online", "1")
	}
	if options.WithDisks {
		data.Set("with-local-disks", "1")
	}

	var taskID string
	if err := c.Post(endpoint, data, &taskID); err != nil {
		return "", fmt.Errorf("failed to migrate container %d from node %s to %s: %w", vmid, node, options.Target, err)
	}

	return taskID, nil
}

func (c *Client) CloneContainer(node string, vmid int, newid int, full bool) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/lxc/%d/clone", node, vmid)
	data := url.Values{}
	data.Set("newid", strconv.Itoa(newid))

	if full {
		data.Set("full", "1")
	}

	var taskID string
	if err := c.Post(endpoint, data, &taskID); err != nil {
		return "", fmt.Errorf("failed to clone container %d to %d on node %s: %w", vmid, newid, node, err)
	}

	return taskID, nil
}

func (c *Client) CreateContainerSnapshot(node string, vmid int, snapname string, description string) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/lxc/%d/snapshot", node, vmid)
	data := url.Values{}
	data.Set("snapname", snapname)

	if description != "" {
		data.Set("description", description)
	}

	var taskID string
	if err := c.Post(endpoint, data, &taskID); err != nil {
		return "", fmt.Errorf("failed to create snapshot %s for container %d on node %s: %w", snapname, vmid, node, err)
	}

	return taskID, nil
}

func (c *Client) DeleteContainerSnapshot(node string, vmid int, snapname string) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/lxc/%d/snapshot/%s", node, vmid, snapname)

	var taskID string
	if err := c.DeleteWithResponse(endpoint, &taskID); err != nil {
		return "", fmt.Errorf("failed to delete snapshot %s for container %d on node %s: %w", snapname, vmid, node, err)
	}

	return taskID, nil
}

func (c *Client) CreateContainerBackup(node string, vmid int, options BackupOptions) (string, error) {
	return c.createBackup("lxc", node, vmid, options)
}
