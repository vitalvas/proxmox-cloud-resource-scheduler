# Configure

## Proxmox

```shell
pveum user add cloud-resource-scheduler@pve
pveum user token add cloud-resource-scheduler@pve scheduler

pveum acl modify / --roles PVEAdmin --user 'cloud-resource-scheduler@pve'
pveum acl modify / --roles PVEAdmin --tokens 'cloud-resource-scheduler@pve!scheduler'
```
