# Configure

## Proxmox

```shell
pveum user add cloud-resource-scheduler@pve
pveum user token add cloud-resource-scheduler@pve scheduler

pveum acl modify / --roles PVEAdmin --user 'cloud-resource-scheduler@pve'
pveum acl modify / --roles PVEAdmin --tokens 'cloud-resource-scheduler@pve!scheduler'
```

## Config file

```json
{
    "proxmox": {
        "user": "cloud-resource-scheduler@pve",
        "token": "scheduler=2e7ccf22-32f8-427b-ba44-29b327f32460",
        "nodes": [
            {"url":"https://pve-pool01-host01.example.com:8006"},
            {"url":"https://pve-pool01-host02.example.com:8006"},
            {"url":"https://pve-pool01-host03.example.com:8006"},
            {"url":"https://pve-pool01-host04.example.com:8006"},
            {"url":"https://pve-pool01-host05.example.com:8006"}
        ]
    }
}
```
