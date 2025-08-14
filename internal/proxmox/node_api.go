package proxmox

import (
	"fmt"
	"net/url"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
)

func (c *Client) GetNodes() ([]Node, error) {
	var nodes []Node
	if err := c.Get("nodes", &nodes); err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	logging.Debugf("Retrieved %d nodes", len(nodes))
	return nodes, nil
}

func (c *Client) GetNode(name string) (*Node, error) {
	var node Node
	endpoint := fmt.Sprintf("nodes/%s/status", name)
	if err := c.Get(endpoint, &node); err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", name, err)
	}

	logging.Debugf("Retrieved node: %s", name)
	return &node, nil
}

func (c *Client) GetNodeVMs(node string) ([]VM, error) {
	var vms []VM
	endpoint := fmt.Sprintf("nodes/%s/qemu", node)
	if err := c.Get(endpoint, &vms); err != nil {
		return nil, fmt.Errorf("failed to get VMs for node %s: %w", node, err)
	}

	logging.Debugf("Retrieved %d VMs for node %s", len(vms), node)
	return vms, nil
}

func (c *Client) GetNodeContainers(node string) ([]Container, error) {
	var containers []Container
	endpoint := fmt.Sprintf("nodes/%s/lxc", node)
	if err := c.Get(endpoint, &containers); err != nil {
		return nil, fmt.Errorf("failed to get containers for node %s: %w", node, err)
	}

	logging.Debugf("Retrieved %d containers for node %s", len(containers), node)
	return containers, nil
}

func (c *Client) GetNodeTasks(node string) ([]Task, error) {
	var tasks []Task
	endpoint := fmt.Sprintf("nodes/%s/tasks", node)
	if err := c.Get(endpoint, &tasks); err != nil {
		return nil, fmt.Errorf("failed to get tasks for node %s: %w", node, err)
	}

	logging.Debugf("Retrieved %d tasks for node %s", len(tasks), node)
	return tasks, nil
}

func (c *Client) GetNodeStorage(node string) ([]Storage, error) {
	var storage []Storage
	endpoint := fmt.Sprintf("nodes/%s/storage", node)
	if err := c.Get(endpoint, &storage); err != nil {
		return nil, fmt.Errorf("failed to get storage for node %s: %w", node, err)
	}

	logging.Debugf("Retrieved %d storage entries for node %s", len(storage), node)
	return storage, nil
}

func (c *Client) ShutdownNode(node string) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/status", node)
	data := url.Values{}
	data.Set("command", "shutdown")

	var result string
	if err := c.Post(endpoint, data, &result); err != nil {
		return "", fmt.Errorf("failed to shutdown node %s: %w", node, err)
	}

	logging.Infof("Shutdown initiated for node %s", node)
	return result, nil
}

func (c *Client) RebootNode(node string) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/status", node)
	data := url.Values{}
	data.Set("command", "reboot")

	var result string
	if err := c.Post(endpoint, data, &result); err != nil {
		return "", fmt.Errorf("failed to reboot node %s: %w", node, err)
	}

	logging.Infof("Reboot initiated for node %s", node)
	return result, nil
}

func (c *Client) WakeNode(node string) error {
	endpoint := fmt.Sprintf("nodes/%s/wakeonlan", node)

	if err := c.Post(endpoint, nil, nil); err != nil {
		return fmt.Errorf("failed to wake node %s: %w", node, err)
	}

	logging.Infof("Wake-on-LAN sent to node %s", node)
	return nil
}
