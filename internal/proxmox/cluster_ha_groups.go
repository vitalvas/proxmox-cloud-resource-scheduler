package proxmox

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type ClusterHAGroup struct {
	Type       string `json:"type"`
	Group      string `json:"group"`
	NoFailback int    `json:"nofailback"`
	Restricted int    `json:"restricted"`
	Digest     string `json:"digest"`
	Nodes      string `json:"nodes"`
}

func (p *Proxmox) ClusterHAGroupList() ([]ClusterHAGroup, error) {
	resp, err := p.makeHTTPRequest(http.MethodGet, "cluster/ha/groups", nil)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var tmp struct {
			Data []ClusterHAGroup `json:"data"`
		}

		json.Unmarshal(bodyBytes, &tmp)

		return tmp.Data, nil
	}

	return nil, fmt.Errorf("wrong status code: %d", resp.StatusCode)
}

func (p *Proxmox) ClusterHAGroupCreate(group ClusterHAGroup) error {
	data := url.Values{}

	data.Add("group", group.Group)
	data.Add("nodes", group.Nodes)
	data.Add("nofailback", strconv.Itoa(group.NoFailback))
	data.Add("restricted", strconv.Itoa(group.Restricted))

	encodedData := data.Encode()

	resp, err := p.makeHTTPRequest(http.MethodPost, "cluster/ha/groups", strings.NewReader(encodedData))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong status code: %d", resp.StatusCode)
	}

	return nil
}

func (p *Proxmox) ClusterHAGroupDelete(group ClusterHAGroup) error {
	path := fmt.Sprintf("cluster/ha/groups/%s", group.Group)

	resp, err := p.makeHTTPRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong status code: %d", resp.StatusCode)
	}

	return nil
}
