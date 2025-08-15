package proxmox

import (
	"encoding/json"
	"regexp"
	"time"
)

type Node struct {
	ID      string  `json:"id"`
	Node    string  `json:"node"`
	Type    string  `json:"type"`
	Status  string  `json:"status"`
	CPU     float64 `json:"cpu"`
	MaxCPU  int     `json:"maxcpu"`
	Mem     int64   `json:"mem"`
	MaxMem  int64   `json:"maxmem"`
	Disk    int64   `json:"disk"`
	MaxDisk int64   `json:"maxdisk"`
	Uptime  int     `json:"uptime"`
	Level   string  `json:"level"`
	SSLCert string  `json:"ssl_fingerprint"`
}

type VM struct {
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Node      string  `json:"node"`
	Status    string  `json:"status"`
	Template  int     `json:"template"`
	CPU       float64 `json:"cpu"`
	MaxCPU    int     `json:"maxcpu"`
	Mem       int64   `json:"mem"`
	MaxMem    int64   `json:"maxmem"`
	Disk      int64   `json:"disk"`
	MaxDisk   int64   `json:"maxdisk"`
	NetIn     int64   `json:"netin"`
	NetOut    int64   `json:"netout"`
	DiskRead  int64   `json:"diskread"`
	DiskWrite int64   `json:"diskwrite"`
	Uptime    int     `json:"uptime"`
	PID       int     `json:"pid"`
	Tags      string  `json:"tags"`
}

type Container struct {
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Node      string  `json:"node"`
	Status    string  `json:"status"`
	Template  int     `json:"template"`
	CPU       float64 `json:"cpu"`
	MaxCPU    int     `json:"maxcpu"`
	Mem       int64   `json:"mem"`
	MaxMem    int64   `json:"maxmem"`
	Swap      int64   `json:"swap"`
	MaxSwap   int64   `json:"maxswap"`
	Disk      int64   `json:"disk"`
	MaxDisk   int64   `json:"maxdisk"`
	NetIn     int64   `json:"netin"`
	NetOut    int64   `json:"netout"`
	DiskRead  int64   `json:"diskread"`
	DiskWrite int64   `json:"diskwrite"`
	Uptime    int     `json:"uptime"`
	Tags      string  `json:"tags"`
}

type Storage struct {
	Storage  string  `json:"storage"`
	Type     string  `json:"type"`
	Content  string  `json:"content"`
	Nodes    string  `json:"nodes"`
	Shared   int     `json:"shared"`
	Used     int64   `json:"used"`
	Avail    int64   `json:"avail"`
	Total    int64   `json:"total"`
	UsedFrac float64 `json:"used_fraction"`
	Enabled  int     `json:"enabled"`
	Active   int     `json:"active"`
}

type ClusterResource struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Node      string  `json:"node"`
	Storage   string  `json:"storage"`
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	HAState   string  `json:"hastate"`
	Template  int     `json:"template"`
	CPU       float64 `json:"cpu"`
	MaxCPU    int     `json:"maxcpu"`
	Mem       int64   `json:"mem"`
	MaxMem    int64   `json:"maxmem"`
	Disk      int64   `json:"disk"`
	MaxDisk   int64   `json:"maxdisk"`
	NetIn     int64   `json:"netin"`
	NetOut    int64   `json:"netout"`
	DiskRead  int64   `json:"diskread"`
	DiskWrite int64   `json:"diskwrite"`
	Uptime    int     `json:"uptime"`
	Level     string  `json:"level"`
	Tags      string  `json:"tags"`
	Pool      string  `json:"pool"`
}

type Task struct {
	UPID      string    `json:"upid"`
	Node      string    `json:"node"`
	PID       int       `json:"pid"`
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	User      string    `json:"user"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"starttime"`
	EndTime   time.Time `json:"endtime"`
	ExitCode  string    `json:"exitstatus"`
}

type MigrationOptions struct {
	Target    string `json:"target"`
	Online    bool   `json:"online"`
	WithDisks bool   `json:"with-local-disks"`
}

type BackupOptions struct {
	Storage   string `json:"storage"`
	Mode      string `json:"mode"`
	Compress  string `json:"compress"`
	MailTo    string `json:"mailto"`
	Notes     string `json:"notes"`
	Protected bool   `json:"protected"`
}

type VMConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	OS          string            `json:"ostype"`
	Memory      int               `json:"memory"`
	Cores       int               `json:"cores"`
	Sockets     int               `json:"sockets"`
	Boot        string            `json:"boot"`
	Disks       map[string]string `json:"disks"`
	Networks    map[string]string `json:"networks"`
	Tags        string            `json:"tags"`
	Startup     string            `json:"startup"`
}

type VMConfigRead struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	OS          string            `json:"ostype"`
	Memory      interface{}       `json:"memory"`  // Can be string or int
	Cores       interface{}       `json:"cores"`   // Can be string or int
	Sockets     interface{}       `json:"sockets"` // Can be string or int
	Boot        string            `json:"boot"`
	Disks       map[string]string `json:"disks"`
	Networks    map[string]string `json:"networks"`
	Tags        string            `json:"tags"`
	Startup     string            `json:"startup"`
	HostPCI     map[string]string `json:"-"` // PCIe passthrough devices (populated via UnmarshalJSON)
}

// UnmarshalJSON custom unmarshaling to capture hostpci devices and other dynamic fields
func (v *VMConfigRead) UnmarshalJSON(data []byte) error {
	// First unmarshal into a generic map to capture all fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Create an alias type to avoid infinite recursion
	type Alias VMConfigRead
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(v),
	}

	// Unmarshal into the alias first to populate standard fields
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Initialize maps if they don't exist
	if v.Disks == nil {
		v.Disks = make(map[string]string)
	}
	if v.Networks == nil {
		v.Networks = make(map[string]string)
	}
	if v.HostPCI == nil {
		v.HostPCI = make(map[string]string)
	}

	// Define regex patterns for device identification
	var (
		diskDevicePattern    = regexp.MustCompile(`^(virtio|ide|sata|scsi)([0-9]+)$`)
		networkDevicePattern = regexp.MustCompile(`^net([0-9]+)$`)
		hostpciDevicePattern = regexp.MustCompile(`^hostpci([0-9]+)$`)
	)

	// Process all fields to capture dynamic ones
	for key, value := range raw {
		strValue, ok := value.(string)
		if !ok {
			continue
		}

		switch {
		case diskDevicePattern.MatchString(key):
			// Disk devices (virtio0, scsi0, ide2, sata1, etc.)
			v.Disks[key] = strValue
		case networkDevicePattern.MatchString(key):
			// Network devices (net0, net1, etc.)
			v.Networks[key] = strValue
		case hostpciDevicePattern.MatchString(key):
			// PCIe passthrough devices (hostpci0, hostpci1, etc.)
			v.HostPCI[key] = strValue
		}
	}

	return nil
}

type ContainerConfig struct {
	OSTemplate   string            `json:"ostemplate"`
	Hostname     string            `json:"hostname"`
	Description  string            `json:"description"`
	Memory       int               `json:"memory"`
	Swap         int               `json:"swap"`
	Cores        int               `json:"cores"`
	RootFS       string            `json:"rootfs"`
	Networks     map[string]string `json:"networks"`
	Tags         string            `json:"tags"`
	Unprivileged bool              `json:"unprivileged"`
}

type StorageContent struct {
	VolID  string `json:"volid"`
	Format string `json:"format"`
	Size   int64  `json:"size"`
	Used   int64  `json:"used"`
	Type   string `json:"content"`
	VMID   int    `json:"vmid"`
}

type StorageStatus struct {
	Storage  string  `json:"storage"`
	Type     string  `json:"type"`
	Total    int64   `json:"total"`
	Used     int64   `json:"used"`
	Avail    int64   `json:"avail"`
	Enabled  int     `json:"enabled"`
	Active   int     `json:"active"`
	UsedFrac float64 `json:"used_fraction"`
}

type ClusterStatus struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Local    int    `json:"local"`
	NodeID   int    `json:"nodeid"`
	Nodes    int    `json:"nodes"`
	Expected int    `json:"expected_votes"`
	Quorate  int    `json:"quorate"`
}

type ClusterNode struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Online int    `json:"online"`
	Local  int    `json:"local"`
	NodeID int    `json:"nodeid"`
	IP     string `json:"ip"`
	Level  string `json:"level"`
}

type ClusterHA struct {
	Manager string `json:"manager"`
	State   string `json:"state"`
}

type ClusterHAGroup struct {
	Group      string `json:"group"`
	Nodes      string `json:"nodes"`
	Restricted int    `json:"restricted"`
	NoFailback int    `json:"nofailback"`
	Type       string `json:"type"`
}

type ClusterHAResource struct {
	SID            string `json:"sid"`
	State          string `json:"state"`
	Group          string `json:"group"`
	MaxRelocate    int    `json:"max_relocate"`
	MaxRestart     int    `json:"max_restart"`
	Comment        string `json:"comment"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	Node           string `json:"node"`
	CRMState       string `json:"crm-state"`
	RequestedState string `json:"request"`
}

type ClusterConfig struct {
	TotemInterface interface{} `json:"totem"`
	NodeList       interface{} `json:"nodelist"`
	QuorumProvider string      `json:"quorum_provider"`
}

type ClusterOptions struct {
	RegisteredTags []string `json:"registered-tags,omitempty"`
}
