# Install

This project used consul for configuration storage, node discovery, distributed lock and another.

## Requirements

* OS with `systemd` (tested on Ubuntu 22.04)
* Consul (tested on 1.17.0)

### Resources

VM with next resources:

* 1 CPU
* 1 GB RAM
* 32 GB HDD

## Configuration

### Consul

#### Consul Server

Consul server need install close to the these applications.

File: `/etc/consul.d/consul.hcl`

```hcl
# make same as proxmox cluster name: `grep cluster_name /etc/pve/corosync.conf`
datacenter = "lab-cloud01"

client_addr = "0.0.0.0"

server = true

# For production need to use minimum 3 nodes (read consul requirements). I have only one node.
bootstrap_expect = 1


# Must have: `openssl rand -base64 32`
encrypt = "..."
```

#### Consul Agent

File: `/etc/consul.d/consul.hcl`

```hcl
# Must be same as consul server
datacenter = "lab-c141"

data_dir = "/opt/consul"
client_addr = "127.0.0.1"
server = false

# Must be same as consul server
encrypt = "..."

# List of consul servers
retry_join = ["100.64.0.5"]
```
