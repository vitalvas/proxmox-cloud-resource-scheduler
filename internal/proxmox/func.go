package proxmox

import "strings"

func (p *Proxmox) HasSharedStorage() (bool, error) {
	list, err := p.StorageList()
	if err != nil {
		return false, err
	}

	for _, row := range list {
		if strings.Contains(row.Content, "images") || strings.Contains(row.Content, "rootdir") {
			if row.Shared == 1 {
				return true, nil
			}
		}
	}

	return false, nil
}
