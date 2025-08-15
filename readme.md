# Proxmox Cloud Resource Scheduler

Make your cloud out of a Proxmox Cluster!

## Overview

Proxmox Cloud Resource Scheduler (CRS) automates high availability management and resource optimization for Proxmox VE clusters.

## Features

- Automatic HA Group Management: Assigns VMs to optimal HA groups based on storage type and hardware requirements
- Node Maintenance Automation: Automatically migrates VMs from nodes in maintenance mode
- Critical VM Support: Prioritizes startup order for mission-critical workloads
- Storage Optimization: Manages CD-ROM attachments and storage-based VM placement
- PCIe Passthrough Awareness: Ensures hardware-dependent VMs stay on correct nodes

## Global Tags

- `crs-critical`: Marks VMs as mission-critical with guaranteed startup order
- `crs-skip`: Excludes VMs from automated CRS management

## Roadmap

* Dynamic Resource Scheduler
* Auto Scaler
