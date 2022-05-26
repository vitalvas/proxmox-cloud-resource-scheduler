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

func (this *Proxmox) ClusterHAGroupList() []ClusterHAGroup {
	resp, err := this.makeHTTPRequest(http.MethodGet, "cluster/ha/groups", nil)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var tmp struct {
			Data []ClusterHAGroup `json:"data"`
		}

		json.Unmarshal(bodyBytes, &tmp)
		return tmp.Data
	} else {
		log.Fatal("wrong status code:", resp.StatusCode)
	}

	return nil
}

func (this *Proxmox) ClusterHAGroupCreate(group ClusterHAGroup) {
	data := url.Values{}

	data.Add("group", group.Group)
	data.Add("nodes", group.Nodes)
	data.Add("nofailback", strconv.Itoa(group.NoFailback))
	data.Add("restricted", strconv.Itoa(group.Restricted))

	encodedData := data.Encode()

	resp, err := this.makeHTTPRequest(http.MethodPost, "cluster/ha/groups", strings.NewReader(encodedData))
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatal("wrong status code:", resp.StatusCode)
	}
}

func (this *Proxmox) ClusterHAGroupDelete(group ClusterHAGroup) {
	path := fmt.Sprintf("cluster/ha/groups/%s", group.Group)

	resp, err := this.makeHTTPRequest(http.MethodDelete, path, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatal("wrong status code:", resp.StatusCode)
	}
}
