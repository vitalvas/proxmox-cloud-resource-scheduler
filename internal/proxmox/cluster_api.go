package proxmox

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
)

func (c *Client) GetClusterStatus() (*ClusterStatus, error) {
	var status ClusterStatus
	if err := c.Get("cluster/status", &status); err != nil {
		return nil, fmt.Errorf("failed to get cluster status: %w", err)
	}

	logging.Debug("Retrieved cluster status")
	return &status, nil
}

func (c *Client) GetClusterResources() ([]ClusterResource, error) {
	var resources []ClusterResource
	if err := c.Get("cluster/resources", &resources); err != nil {
		return nil, fmt.Errorf("failed to get cluster resources: %w", err)
	}

	logging.Debugf("Retrieved %d cluster resources", len(resources))
	return resources, nil
}

func (c *Client) GetClusterNodes() ([]ClusterNode, error) {
	var nodes []ClusterNode
	if err := c.Get("cluster/status", &nodes); err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	var clusterNodes []ClusterNode
	for _, node := range nodes {
		if node.Type == "node" {
			clusterNodes = append(clusterNodes, node)
		}
	}

	logging.Debugf("Retrieved %d cluster nodes", len(clusterNodes))
	return clusterNodes, nil
}

func (c *Client) GetClusterTasks() ([]Task, error) {
	var tasks []Task
	if err := c.Get("cluster/tasks", &tasks); err != nil {
		return nil, fmt.Errorf("failed to get cluster tasks: %w", err)
	}

	logging.Debugf("Retrieved %d cluster tasks", len(tasks))
	return tasks, nil
}

func (c *Client) GetClusterHA() (*ClusterHA, error) {
	var ha ClusterHA
	if err := c.Get("cluster/ha", &ha); err != nil {
		return nil, fmt.Errorf("failed to get cluster HA status: %w", err)
	}

	logging.Debug("Retrieved cluster HA status")
	return &ha, nil
}

func (c *Client) GetClusterHAGroups() ([]ClusterHAGroup, error) {
	var groups []ClusterHAGroup
	if err := c.Get("cluster/ha/groups", &groups); err != nil {
		return nil, fmt.Errorf("failed to get cluster HA groups: %w", err)
	}

	logging.Debugf("Retrieved %d cluster HA groups", len(groups))
	return groups, nil
}

func (c *Client) CreateClusterHAGroup(group ClusterHAGroup) (string, error) {
	data := url.Values{}
	data.Set("group", group.Group)
	data.Set("nodes", group.Nodes)

	if group.Restricted > 0 {
		data.Set("restricted", strconv.Itoa(group.Restricted))
	}
	if group.NoFailback > 0 {
		data.Set("nofailback", strconv.Itoa(group.NoFailback))
	}

	if err := c.Post("cluster/ha/groups", data, nil); err != nil {
		return "", fmt.Errorf("failed to create cluster HA group %s: %w", group.Group, err)
	}

	logging.Infof("Created cluster HA group %s", group.Group)
	return group.Group, nil
}

func (c *Client) UpdateClusterHAGroup(group ClusterHAGroup) error {
	endpoint := fmt.Sprintf("cluster/ha/groups/%s", group.Group)
	data := url.Values{}
	data.Set("nodes", group.Nodes)

	if group.Restricted > 0 {
		data.Set("restricted", strconv.Itoa(group.Restricted))
	}
	if group.NoFailback > 0 {
		data.Set("nofailback", strconv.Itoa(group.NoFailback))
	}

	if err := c.Put(endpoint, data, nil); err != nil {
		return fmt.Errorf("failed to update cluster HA group %s: %w", group.Group, err)
	}

	logging.Infof("Updated cluster HA group %s", group.Group)
	return nil
}

func (c *Client) DeleteClusterHAGroup(groupName string) error {
	endpoint := fmt.Sprintf("cluster/ha/groups/%s", groupName)

	if err := c.Delete(endpoint); err != nil {
		return fmt.Errorf("failed to delete cluster HA group %s: %w", groupName, err)
	}

	logging.Infof("Deleted cluster HA group %s", groupName)
	return nil
}

func (c *Client) GetClusterHAResources() ([]ClusterHAResource, error) {
	var resources []ClusterHAResource
	if err := c.Get("cluster/ha/resources", &resources); err != nil {
		return nil, fmt.Errorf("failed to get cluster HA resources: %w", err)
	}

	logging.Debugf("Retrieved %d cluster HA resources", len(resources))
	return resources, nil
}

func (c *Client) CreateClusterHAResource(resource ClusterHAResource) (string, error) {
	data := url.Values{}
	data.Set("sid", resource.SID)

	if resource.Group != "" {
		data.Set("group", resource.Group)
	}
	if resource.MaxRelocate > 0 {
		data.Set("max_relocate", strconv.Itoa(resource.MaxRelocate))
	}
	if resource.MaxRestart > 0 {
		data.Set("max_restart", strconv.Itoa(resource.MaxRestart))
	}
	if resource.State != "" {
		data.Set("state", resource.State)
	}
	if resource.Comment != "" {
		data.Set("comment", resource.Comment)
	}

	if err := c.Post("cluster/ha/resources", data, nil); err != nil {
		return "", fmt.Errorf("failed to create cluster HA resource %s: %w", resource.SID, err)
	}

	logging.Infof("Created cluster HA resource %s", resource.SID)
	return resource.SID, nil
}

func (c *Client) UpdateClusterHAResource(resource ClusterHAResource) error {
	endpoint := fmt.Sprintf("cluster/ha/resources/%s", resource.SID)
	data := url.Values{}

	if resource.Group != "" {
		data.Set("group", resource.Group)
	}
	if resource.MaxRelocate > 0 {
		data.Set("max_relocate", strconv.Itoa(resource.MaxRelocate))
	}
	if resource.MaxRestart > 0 {
		data.Set("max_restart", strconv.Itoa(resource.MaxRestart))
	}
	if resource.State != "" {
		data.Set("state", resource.State)
	}
	if resource.Comment != "" {
		data.Set("comment", resource.Comment)
	}

	if err := c.Put(endpoint, data, nil); err != nil {
		return fmt.Errorf("failed to update cluster HA resource %s: %w", resource.SID, err)
	}

	logging.Infof("Updated cluster HA resource %s", resource.SID)
	return nil
}

func (c *Client) DeleteClusterHAResource(sid string) error {
	endpoint := fmt.Sprintf("cluster/ha/resources/%s", sid)

	if err := c.Delete(endpoint); err != nil {
		return fmt.Errorf("failed to delete cluster HA resource %s: %w", sid, err)
	}

	logging.Infof("Deleted cluster HA resource %s", sid)
	return nil
}

func (c *Client) GetClusterConfig() (*ClusterConfig, error) {
	var config ClusterConfig
	if err := c.Get("cluster/config", &config); err != nil {
		return nil, fmt.Errorf("failed to get cluster config: %w", err)
	}

	logging.Debug("Retrieved cluster config")
	return &config, nil
}

func (c *Client) GetTask(node, upid string) (*Task, error) {
	var task Task
	endpoint := fmt.Sprintf("nodes/%s/tasks/%s/status", node, upid)
	if err := c.Get(endpoint, &task); err != nil {
		return nil, fmt.Errorf("failed to get task %s on node %s: %w", upid, node, err)
	}

	logging.Debugf("Retrieved task %s on node %s", upid, node)
	return &task, nil
}

func (c *Client) StopTask(node, upid string) error {
	endpoint := fmt.Sprintf("nodes/%s/tasks/%s", node, upid)

	if err := c.Delete(endpoint); err != nil {
		return fmt.Errorf("failed to stop task %s on node %s: %w", upid, node, err)
	}

	logging.Infof("Stopped task %s on node %s", upid, node)
	return nil
}
