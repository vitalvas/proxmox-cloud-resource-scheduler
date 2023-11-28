# Configure

## Proxmox

```shell
pveum user add cloud-resource-scheduler@pve
pveum user token add cloud-resource-scheduler@pve scheduler

pveum acl modify / --roles PVEAdmin --user 'cloud-resource-scheduler@pve'
pveum acl modify / --roles PVEAdmin --tokens 'cloud-resource-scheduler@pve!scheduler'
```

## Config

All configuration is stored in consul.

### Add authentication information to consul

```shell
echo '{"user":"cloud-resource-scheduler@pve", "token":"scheduler=2e7ccf22-32f8-427b-ba44-29b327f32460"}' | consul kv put crs/config/proxmox/auth -
```
