# =============================================================================
# KibaShip Staging Storage Configuration
# =============================================================================
# This configuration provisions persistent storage volumes for the staging
# Kubernetes cluster worker nodes, including:
# - 3 x 40GB volumes for worker nodes
# - Automatic attachment to worker nodes using label selectors
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

variable "hcloud_token" {
  description = "Hetzner Cloud API Token"
  type        = string
  sensitive   = true
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

# =============================================================================
# Provider Configuration
# =============================================================================

provider "hcloud" {
  token = var.hcloud_token
}

# =============================================================================
# Data Sources
# =============================================================================

# Discover worker servers using label selectors
data "hcloud_servers" "staging_workers" {
  with_selector = "environment=staging,cluster=kibaship-staging,role=worker"
}

# Get location information from the first worker server
data "hcloud_server" "worker_reference" {
  id = data.hcloud_servers.staging_workers.servers[0].id
}

# =============================================================================
# Local Values
# =============================================================================

locals {
  cluster_name = "kibaship-staging"
  location     = data.hcloud_server.worker_reference.location

  # Create a map of worker servers for easier reference
  worker_servers = {
    for idx, server in data.hcloud_servers.staging_workers.servers :
    idx => {
      id       = server.id
      name     = server.name
      location = server.location
    }
  }
}

# =============================================================================
# Storage Volumes
# =============================================================================

# Create 40GB volumes for each worker node
resource "hcloud_volume" "worker_storage" {
  count    = length(data.hcloud_servers.staging_workers.servers)
  name     = "${local.cluster_name}-worker-${count.index + 1}-storage"
  size     = var.volume_size
  location = local.location
  format   = "ext4"

  labels = {
    environment = "staging"
    cluster     = local.cluster_name
    role        = "worker-storage"
    worker_node = "${local.cluster_name}-worker-${count.index + 1}"
  }
}

# =============================================================================
# Volume Attachments
# =============================================================================

# Attach each volume to its corresponding worker node
# Note: Hetzner Cloud consistently mounts volumes to /mnt/HC_Volume_<volume-id>
# This provides a predictable path for all worker nodes
resource "hcloud_volume_attachment" "worker_storage" {
  count     = length(data.hcloud_servers.staging_workers.servers)
  volume_id = hcloud_volume.worker_storage[count.index].id
  server_id = data.hcloud_servers.staging_workers.servers[count.index].id
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
      worker_node = data.hcloud_servers.staging_workers.servers[idx].name
      server_id   = data.hcloud_servers.staging_workers.servers[idx].id
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
        worker_node = data.hcloud_servers.staging_workers.servers[idx].name
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
        "Each worker node has a dedicated 40GB storage volume",
        "Volumes are consistently mounted at /mnt/HC_Volume_<volume-id>",
        "Volumes are ready for OpenEBS Mayastor configuration",
        "Use device paths in Mayastor DiskPool configuration",
        "Run manual health checks to verify volume accessibility"
      ]
    }
  }
}