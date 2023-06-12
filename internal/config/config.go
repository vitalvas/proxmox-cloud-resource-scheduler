package config

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

type Config struct {
	Proxmox Proxmox `json:"proxmox"`
	DRS     DRS     `jsob:"drs"`
}

type Proxmox struct {
	User  string        `json:"user"`
	Token string        `json:"token"`
	Nodes []ProxmoxNode `json:"nodes"`
}

type ProxmoxNode struct {
	URL string `json:"url"`
}

type DRS struct {
	Maintenance map[string]DRS `json:"maintenance"`
}

type DRSMaintenance struct {
	RestoreTime time.Duration `json:"restore-time"`
}

func LoadConfig(file string) (*Config, error) {
	jsonFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	var conf *Config
	json.Unmarshal(byteValue, &conf)

	return conf, nil
}
