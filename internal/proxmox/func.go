package proxmox

import "strings"

func (p *Proxmox) HasSharedStorage() bool {
	for _, row := range p.StorageList() {
		if strings.Contains(row.Content, "images") || strings.Contains(row.Content, "rootdir") {
			if row.Shared == 1 {
				return true
			}
		}
	}

	return false
}
