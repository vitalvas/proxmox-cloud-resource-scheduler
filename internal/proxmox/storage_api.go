package proxmox

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
)

func (c *Client) GetStorage() ([]Storage, error) {
	var storage []Storage
	if err := c.Get("storage", &storage); err != nil {
		return nil, fmt.Errorf("failed to get storage: %w", err)
	}

	logging.Debugf("Retrieved %d storage entries", len(storage))
	return storage, nil
}

func (c *Client) GetStorageContent(node, storage string) ([]StorageContent, error) {
	var content []StorageContent
	endpoint := fmt.Sprintf("nodes/%s/storage/%s/content", node, storage)
	if err := c.Get(endpoint, &content); err != nil {
		return nil, fmt.Errorf("failed to get storage content for %s on node %s: %w", storage, node, err)
	}

	logging.Debugf("Retrieved %d content items for storage %s on node %s", len(content), storage, node)
	return content, nil
}

func (c *Client) GetStorageStatus(node, storage string) (*StorageStatus, error) {
	var status StorageStatus
	endpoint := fmt.Sprintf("nodes/%s/storage/%s/status", node, storage)
	if err := c.Get(endpoint, &status); err != nil {
		return nil, fmt.Errorf("failed to get storage status for %s on node %s: %w", storage, node, err)
	}

	logging.Debugf("Retrieved storage status for %s on node %s", storage, node)
	return &status, nil
}

func (c *Client) UploadToStorage(node, storage, filename string, content []byte) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/storage/%s/upload", node, storage)

	data := url.Values{}
	data.Set("filename", filename)
	data.Set("content", string(content))

	var taskID string
	if err := c.Post(endpoint, data, &taskID); err != nil {
		return "", fmt.Errorf("failed to upload %s to storage %s on node %s: %w", filename, storage, node, err)
	}

	logging.Infof("Uploaded %s to storage %s on node %s, task: %s", filename, storage, node, taskID)
	return taskID, nil
}

func (c *Client) DeleteFromStorage(node, storage, volid string) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/storage/%s/content/%s", node, storage, volid)

	var taskID string
	if err := c.DeleteWithResponse(endpoint, &taskID); err != nil {
		return "", fmt.Errorf("failed to delete %s from storage %s on node %s: %w", volid, storage, node, err)
	}

	logging.Infof("Deleted %s from storage %s on node %s, task: %s", volid, storage, node, taskID)
	return taskID, nil
}

func (c *Client) CreateStorageBackup(node, storage string, vmids []int, options BackupOptions) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/storage/%s/backup", node, storage)
	data := url.Values{}
	data.Set("storage", options.Storage)

	vmidStr := ""
	for i, vmid := range vmids {
		if i > 0 {
			vmidStr += ","
		}
		vmidStr += fmt.Sprintf("%d", vmid)
	}
	data.Set("vmid", vmidStr)

	if options.Mode != "" {
		data.Set("mode", options.Mode)
	}
	if options.Compress != "" {
		data.Set("compress", options.Compress)
	}
	if options.MailTo != "" {
		data.Set("mailto", options.MailTo)
	}
	if options.Notes != "" {
		data.Set("notes", options.Notes)
	}
	if options.Protected {
		data.Set("protected", "1")
	}

	var taskID string
	if err := c.Post(endpoint, data, &taskID); err != nil {
		return "", fmt.Errorf("failed to create storage backup on %s: %w", storage, err)
	}

	logging.Infof("Created storage backup on %s, task: %s", storage, taskID)
	return taskID, nil
}

func (c *Client) HasSharedStorage() (bool, error) {
	storageList, err := c.GetStorage()
	if err != nil {
		return false, fmt.Errorf("failed to get storage list: %w", err)
	}

	for _, storage := range storageList {
		if storage.Shared == 1 && strings.Contains(storage.Content, "images") {
			return true, nil
		}
	}

	return false, nil
}
