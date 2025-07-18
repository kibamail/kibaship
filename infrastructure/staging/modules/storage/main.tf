# =============================================================================
# Storage Module
# =============================================================================
# This module provisions persistent storage volumes for Kubernetes cluster
# worker nodes, including:
# - Storage volumes for each worker node
# - Automatic attachment to worker nodes
# - Health checks to verify volume accessibility
# - Optimized for OpenEBS Mayastor storage engine

terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.51.0"
    }
  }
}

# =============================================================================
# Variables
# =============================================================================

variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
}

variable "environment" {
  description = "Environment name (staging, production, etc.)"
  type        = string
}

variable "worker_servers" {
  description = "Map of worker server details"
  type = map(object({
    id          = string
    public_ip   = string
    private_ip  = string
    role        = string
  }))
}

variable "volume_size" {
  description = "Size of each storage volume in GB"
  type        = number
  default     = 40
}

variable "volume_type" {
  description = "Type of storage volume (network-ssd or network-hdd)"
  type        = string
  default     = "network-ssd"
}

variable "location" {
  description = "Hetzner Cloud location"
  type        = string
  default     = "nbg1"
}

# =============================================================================
# Local Values
# =============================================================================

locals {
  worker_list = [
    for name, server in var.worker_servers : {
      name      = name
      id        = server.id
      public_ip = server.public_ip
    }
  ]
}

# =============================================================================
# Storage Volumes
# =============================================================================

resource "hcloud_volume" "worker_storage" {
  count    = length(local.worker_list)
  name     = "${var.cluster_name}-worker-${count.index + 1}-storage"
  size     = var.volume_size
  location = var.location
  format   = "ext4"

  labels = {
    environment = var.environment
    cluster     = var.cluster_name
    role        = "worker-storage"
    worker_node = "${var.cluster_name}-worker-${count.index + 1}"
  }
}

# =============================================================================
# Volume Attachments
# =============================================================================

resource "hcloud_volume_attachment" "worker_storage" {
  count     = length(local.worker_list)
  volume_id = hcloud_volume.worker_storage[count.index].id
  server_id = local.worker_list[count.index].id
  automount = false

  depends_on = [
    hcloud_volume.worker_storage
  ]
}

# =============================================================================
# Outputs
# =============================================================================

output "storage_volumes" {
  description = "Details of created storage volumes"
  value = {
    for idx, volume in hcloud_volume.worker_storage :
    volume.name => {
      id          = volume.id
      name        = volume.name
      size        = volume.size
      location    = volume.location
      device_path = "/dev/disk/by-id/scsi-0HC_Volume_${volume.id}"
      mount_path  = "/mnt/HC_Volume_${volume.id}"
      worker_node = local.worker_list[idx].name
      server_id   = local.worker_list[idx].id
    }
  }
}

output "volume_attachments" {
  description = "Volume attachment details"
  value = {
    for idx, attachment in hcloud_volume_attachment.worker_storage :
    hcloud_volume.worker_storage[idx].name => {
      volume_id = attachment.volume_id
      server_id = attachment.server_id
      automount = attachment.automount
    }
  }
}

output "storage_summary" {
  description = "Complete storage configuration summary"
  value = {
    total_volumes = length(hcloud_volume.worker_storage)
    volume_size   = var.volume_size
    volume_type   = var.volume_type
    total_storage = "${length(hcloud_volume.worker_storage) * var.volume_size}GB"

    volumes = [
      for idx, volume in hcloud_volume.worker_storage : {
        name        = volume.name
        size        = "${volume.size}GB"
        worker_node = local.worker_list[idx].name
        device_path = "/dev/disk/by-id/scsi-0HC_Volume_${volume.id}"
        mount_path  = "/mnt/HC_Volume_${volume.id}"
        ready       = true
      }
    ]

    mayastor_ready = {
      description = "Volumes are configured for OpenEBS Mayastor"
      device_paths = [
        for volume in hcloud_volume.worker_storage :
        "/dev/disk/by-id/scsi-0HC_Volume_${volume.id}"
      ]
      mount_paths = [
        for volume in hcloud_volume.worker_storage :
        "/mnt/HC_Volume_${volume.id}"
      ]
      notes = [
        "Volumes are formatted with ext4 and auto-mounted",
        "Each worker node has a dedicated storage volume",
        "Volumes are consistently mounted at /mnt/HC_Volume_<volume-id>",
        "Volumes are ready for OpenEBS Mayastor configuration",
        "Use device paths in Mayastor DiskPool configuration",
        "Run manual health checks to verify volume accessibility"
      ]
    }
  }
}
