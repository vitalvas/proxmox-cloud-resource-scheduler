package proxmox

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type Storage struct {
	Content      string `json:"content"`
	PruneBackups string `json:"prune-backups"`
	Server       string `json:"server"`
	Type         string `json:"type"`
	Digest       string `json:"digest"`
	Fingerprint  string `json:"fingerprint"`
	Datastore    string `json:"datastore"`
	Storage      string `json:"storage"`
	Username     string `json:"username"`
	Shared       uint   `json:"shared"`
	ThinPool     string `json:"thinpool"`
	VgName       string `json:"vgname"`
	Path         string `json:"path"`
}

func (this *Proxmox) StorageList() []Storage {
	resp, err := this.makeHTTPRequest(http.MethodGet, "storage", nil)
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
			Data []Storage `json:"data"`
		}

		json.Unmarshal(bodyBytes, &tmp)
		return tmp.Data
	} else {
		log.Fatal("wrong status code:", resp.StatusCode)
	}

	return nil
}
