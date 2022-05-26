package proxmox

import "strings"

func (this *Proxmox) HasSharedStorage() bool {
	for _, row := range this.StorageList() {
		if strings.Contains(row.Content, "images") || strings.Contains(row.Content, "rootdir") {
			if row.Shared == 1 {
				return true
			}
		}
	}

	return false
}
