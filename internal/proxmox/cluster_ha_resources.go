package proxmox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type ClusterHAResources struct {
	SID         string `json:"sid"`
	Type        string `json:"type"`
	MaxRelocate int    `json:"max_relocate"`
	MaxRestart  int    `json:"max_restart"`
	Group       string `json:"group"`
	Comment     string `json:"comment"`
	Digest      string `json:"digest"`
	State       string `json:"state"`
}

func (p *Proxmox) ClusterHAResourcesList() ([]ClusterHAResources, error) {
	resp, err := p.makeHTTPRequest(http.MethodGet, "cluster/ha/resources", nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wrong status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tmp struct {
		Data []ClusterHAResources `json:"data"`
	}

	json.Unmarshal(bodyBytes, &tmp)

	return tmp.Data, nil
}

func (p *Proxmox) ClusterHAResourcesCreate(resource ClusterHAResources) error {
	data := url.Values{}

	data.Add("sid", resource.SID)
	data.Add("max_relocate", strconv.Itoa(resource.MaxRelocate))
	data.Add("max_restart", strconv.Itoa(resource.MaxRestart))
	data.Add("group", resource.Group)
	data.Add("comment", resource.Comment)
	data.Add("state", resource.State)

	encodedData := data.Encode()

	resp, err := p.makeHTTPRequest(http.MethodPost, "cluster/ha/resources", strings.NewReader(encodedData))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong status code: %d", resp.StatusCode)
	}

	return nil
}
