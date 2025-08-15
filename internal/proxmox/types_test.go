package proxmox

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVMConfigRead_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name            string
		jsonData        string
		expectedDisks   map[string]string
		expectedHostPCI map[string]string
		excludedFields  []string // Fields that should NOT be captured as disks
	}{
		{
			name: "should exclude scsihw and other non-disk scsi fields",
			jsonData: `{
				"name": "test-vm",
				"scsi0": "pve:base-10001-disk-0,discard=on,iothread=1,size=16G",
				"scsi1": "local:vm-disk-1,size=50G",
				"scsihw": "virtio-scsi-single",
				"scsicontroller": "some-value",
				"scsibus": "another-value",
				"virtio0": "local:vm-100-disk-0.qcow2",
				"ide2": "none,media=cdrom"
			}`,
			expectedDisks: map[string]string{
				"scsi0":   "pve:base-10001-disk-0,discard=on,iothread=1,size=16G",
				"scsi1":   "local:vm-disk-1,size=50G",
				"virtio0": "local:vm-100-disk-0.qcow2",
				"ide2":    "none,media=cdrom",
			},
			expectedHostPCI: map[string]string{},
			excludedFields:  []string{"scsihw", "scsicontroller", "scsibus"},
		},
		{
			name: "should capture hostpci devices and exclude non-device hostpci fields",
			jsonData: `{
				"name": "hostpci-vm",
				"virtio0": "shared-storage:vm-400-disk-0,size=50G",
				"hostpci0": "01:00.0,pcie=1",
				"hostpci1": "02:00.0,rombar=0",
				"hostpciconfig": "some-config",
				"scsihw": "virtio-scsi-pci"
			}`,
			expectedDisks: map[string]string{
				"virtio0": "shared-storage:vm-400-disk-0,size=50G",
			},
			expectedHostPCI: map[string]string{
				"hostpci0": "01:00.0,pcie=1",
				"hostpci1": "02:00.0,rombar=0",
			},
			excludedFields: []string{"scsihw", "hostpciconfig"},
		},
		{
			name: "should handle all disk device types with regex precision",
			jsonData: `{
				"name": "multi-disk-vm",
				"virtio0": "local:vm-disk-0.qcow2",
				"virtio10": "local:vm-disk-10.qcow2",
				"scsi1": "shared:vm-disk-1,size=32G",
				"ide2": "local:iso/test.iso,media=cdrom",
				"sata0": "backup:vm-disk-2,size=100G",
				"scsihw": "virtio-scsi-single",
				"virtiocontroller": "some-config",
				"idebus": "another-config",
				"satamode": "more-config",
				"net0": "virtio=12:34:56:78:9A:BC,bridge=vmbr0",
				"networkconfig": "some-net-config"
			}`,
			expectedDisks: map[string]string{
				"virtio0":  "local:vm-disk-0.qcow2",
				"virtio10": "local:vm-disk-10.qcow2",
				"scsi1":    "shared:vm-disk-1,size=32G",
				"ide2":     "local:iso/test.iso,media=cdrom",
				"sata0":    "backup:vm-disk-2,size=100G",
			},
			expectedHostPCI: map[string]string{},
			excludedFields:  []string{"scsihw", "virtiocontroller", "idebus", "satamode", "networkconfig"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config VMConfigRead
			err := json.Unmarshal([]byte(tt.jsonData), &config)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedDisks, config.Disks, "Disks map should match expected")
			assert.Equal(t, tt.expectedHostPCI, config.HostPCI, "HostPCI map should match expected")

			// Verify excluded fields are not captured in any device map
			for _, excludedField := range tt.excludedFields {
				_, inDisks := config.Disks[excludedField]
				_, inNetworks := config.Networks[excludedField]
				_, inHostPCI := config.HostPCI[excludedField]

				assert.False(t, inDisks, "Field '%s' should not be captured as a disk device", excludedField)
				assert.False(t, inNetworks, "Field '%s' should not be captured as a network device", excludedField)
				assert.False(t, inHostPCI, "Field '%s' should not be captured as a hostpci device", excludedField)
			}
		})
	}
}
