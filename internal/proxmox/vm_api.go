package proxmox

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
)

func (c *Client) GetVM(node string, vmid int) (*VM, error) {
	var vm VM
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/current", node, vmid)
	if err := c.Get(endpoint, &vm); err != nil {
		return nil, fmt.Errorf("failed to get VM %d on node %s: %w", vmid, node, err)
	}

	logging.Debugf("Retrieved VM %d on node %s", vmid, node)
	return &vm, nil
}

func (c *Client) CreateVM(node string, vmid int, config VMConfig) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu", node)
	data := url.Values{}
	data.Set("vmid", strconv.Itoa(vmid))

	if config.Name != "" {
		data.Set("name", config.Name)
	}
	if config.Description != "" {
		data.Set("description", config.Description)
	}
	if config.OS != "" {
		data.Set("ostype", config.OS)
	}
	if config.Memory > 0 {
		data.Set("memory", strconv.Itoa(config.Memory))
	}
	if config.Cores > 0 {
		data.Set("cores", strconv.Itoa(config.Cores))
	}
	if config.Sockets > 0 {
		data.Set("sockets", strconv.Itoa(config.Sockets))
	}
	if config.Boot != "" {
		data.Set("boot", config.Boot)
	}
	if config.Tags != "" {
		data.Set("tags", config.Tags)
	}

	for key, value := range config.Disks {
		data.Set(key, value)
	}

	for key, value := range config.Networks {
		data.Set(key, value)
	}

	var taskID string
	if err := c.Post(endpoint, data, &taskID); err != nil {
		return "", fmt.Errorf("failed to create VM %d on node %s: %w", vmid, node, err)
	}

	logging.Infof("Created VM %d on node %s, task: %s", vmid, node, taskID)
	return taskID, nil
}

func (c *Client) StartVM(node string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/start", node, vmid)

	var taskID string
	if err := c.Post(endpoint, nil, &taskID); err != nil {
		return "", fmt.Errorf("failed to start VM %d on node %s: %w", vmid, node, err)
	}

	logging.Infof("Started VM %d on node %s, task: %s", vmid, node, taskID)
	return taskID, nil
}

func (c *Client) StopVM(node string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/stop", node, vmid)

	var taskID string
	if err := c.Post(endpoint, nil, &taskID); err != nil {
		return "", fmt.Errorf("failed to stop VM %d on node %s: %w", vmid, node, err)
	}

	logging.Infof("Stopped VM %d on node %s, task: %s", vmid, node, taskID)
	return taskID, nil
}

func (c *Client) ShutdownVM(node string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/shutdown", node, vmid)

	var taskID string
	if err := c.Post(endpoint, nil, &taskID); err != nil {
		return "", fmt.Errorf("failed to shutdown VM %d on node %s: %w", vmid, node, err)
	}

	logging.Infof("Shutdown VM %d on node %s, task: %s", vmid, node, taskID)
	return taskID, nil
}

func (c *Client) RebootVM(node string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/reboot", node, vmid)

	var taskID string
	if err := c.Post(endpoint, nil, &taskID); err != nil {
		return "", fmt.Errorf("failed to reboot VM %d on node %s: %w", vmid, node, err)
	}

	logging.Infof("Rebooted VM %d on node %s, task: %s", vmid, node, taskID)
	return taskID, nil
}

func (c *Client) ResetVM(node string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/status/reset", node, vmid)

	var taskID string
	if err := c.Post(endpoint, nil, &taskID); err != nil {
		return "", fmt.Errorf("failed to reset VM %d on node %s: %w", vmid, node, err)
	}

	logging.Infof("Reset VM %d on node %s, task: %s", vmid, node, taskID)
	return taskID, nil
}

func (c *Client) DeleteVM(node string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d", node, vmid)

	var taskID string
	if err := c.Delete(endpoint); err != nil {
		return "", fmt.Errorf("failed to delete VM %d on node %s: %w", vmid, node, err)
	}

	logging.Infof("Deleted VM %d on node %s, task: %s", vmid, node, taskID)
	return taskID, nil
}

func (c *Client) MigrateVM(node string, vmid int, options MigrationOptions) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/migrate", node, vmid)
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
		return "", fmt.Errorf("failed to migrate VM %d from node %s to %s: %w", vmid, node, options.Target, err)
	}

	logging.Infof("Migrating VM %d from node %s to %s, task: %s", vmid, node, options.Target, taskID)
	return taskID, nil
}

func (c *Client) CloneVM(node string, vmid int, newid int, full bool) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/clone", node, vmid)
	data := url.Values{}
	data.Set("newid", strconv.Itoa(newid))

	if full {
		data.Set("full", "1")
	}

	var taskID string
	if err := c.Post(endpoint, data, &taskID); err != nil {
		return "", fmt.Errorf("failed to clone VM %d to %d on node %s: %w", vmid, newid, node, err)
	}

	logging.Infof("Cloned VM %d to %d on node %s, task: %s", vmid, newid, node, taskID)
	return taskID, nil
}

func (c *Client) CreateVMSnapshot(node string, vmid int, snapname string, description string) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/snapshot", node, vmid)
	data := url.Values{}
	data.Set("snapname", snapname)

	if description != "" {
		data.Set("description", description)
	}

	var taskID string
	if err := c.Post(endpoint, data, &taskID); err != nil {
		return "", fmt.Errorf("failed to create snapshot %s for VM %d on node %s: %w", snapname, vmid, node, err)
	}

	logging.Infof("Created snapshot %s for VM %d on node %s, task: %s", snapname, vmid, node, taskID)
	return taskID, nil
}

func (c *Client) DeleteVMSnapshot(node string, vmid int, snapname string) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/qemu/%d/snapshot/%s", node, vmid, snapname)

	var taskID string
	if err := c.DeleteWithResponse(endpoint, &taskID); err != nil {
		return "", fmt.Errorf("failed to delete snapshot %s for VM %d on node %s: %w", snapname, vmid, node, err)
	}

	logging.Infof("Deleted snapshot %s for VM %d on node %s, task: %s", snapname, vmid, node, taskID)
	return taskID, nil
}

func (c *Client) CreateVMBackup(node string, vmid int, options BackupOptions) (string, error) {
	return c.createBackup("qemu", node, vmid, options)
}
