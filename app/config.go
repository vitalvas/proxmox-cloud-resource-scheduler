package app

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type Config struct {
	Proxmox ConfigProxmox `json:"proxmox"`
	DRS     ConfigDRS     `jsob:"drs"`
}

type ConfigProxmox struct {
	User  string              `json:"user"`
	Token string              `json:"token"`
	Nodes []ConfigProxmoxNode `json:"nodes"`
}

type ConfigProxmoxNode struct {
	URL string `json:"url"`
}

type ConfigDRS struct {
	Maintenance map[string]ConfigDRSMaintenance `json:"maintenance"`
}

type ConfigDRSMaintenance struct {
	RestoreTime time.Duration `json:"restore-time"`
}

func LoadConfig(file string) *Config {
	jsonFile, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	var conf *Config
	json.Unmarshal(byteValue, &conf)

	return conf
}
