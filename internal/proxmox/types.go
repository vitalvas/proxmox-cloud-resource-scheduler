package proxmox

import "time"

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
	SID         string `json:"sid"`
	State       string `json:"state"`
	Group       string `json:"group"`
	MaxRelocate int    `json:"max_relocate"`
	MaxRestart  int    `json:"max_restart"`
	Comment     string `json:"comment"`
	Type        string `json:"type"`
}

type ClusterConfig struct {
	TotemInterface interface{} `json:"totem"`
	NodeList       interface{} `json:"nodelist"`
	QuorumProvider string      `json:"quorum_provider"`
}
