# =============================================================================
# Storage Module
# =============================================================================
# This module provisions persistent storage volumes for worker nodes, including:
# - Storage volumes for each worker node
# - Automatic attachment to worker nodes
# - Raw block devices ready for storage configuration

terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.51.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2.0"
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

variable "ssh_private_key" {
  description = "SSH private key for server access"
  type        = string
  sensitive   = true
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

  labels = {
    environment = var.environment
    cluster     = var.cluster_name
    role        = "worker-storage"
    worker_node = "${var.cluster_name}-worker-${count.index + 1}"
    purpose     = "mayastor-storage"
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
# Volume Formatting and Mounting
# =============================================================================

resource "null_resource" "verify_raw_devices" {
  count = length(local.worker_list)

  connection {
    type        = "ssh"
    user        = "ubuntu"
    private_key = var.ssh_private_key
    host        = local.worker_list[count.index].public_ip
  }

  provisioner "remote-exec" {
    inline = [
      # Verify the device is available
      "sudo lsblk /dev/disk/by-id/scsi-0HC_Volume_${hcloud_volume.worker_storage[count.index].id}",

      # Wait a moment for device to be fully ready
      "sleep 5",

      # Format the volume with ext4
      "sudo mkfs.ext4 -F /dev/disk/by-id/scsi-0HC_Volume_${hcloud_volume.worker_storage[count.index].id}",

      # Create mount point for OpenEBS Local PV
      "sudo mkdir -p /mnt/openebs-local",

      # Mount the volume
      "sudo mount /dev/disk/by-id/scsi-0HC_Volume_${hcloud_volume.worker_storage[count.index].id} /mnt/openebs-local",

      # Add to fstab for persistence across reboots
      "echo '/dev/disk/by-id/scsi-0HC_Volume_${hcloud_volume.worker_storage[count.index].id} /mnt/openebs-local ext4 defaults 0 2' | sudo tee -a /etc/fstab",

      # Create OpenEBS Local PV directory structure
      "sudo mkdir -p /mnt/openebs-local/local",
      "sudo chown root:root /mnt/openebs-local/local",
      "sudo chmod 755 /mnt/openebs-local/local",

      # Verify mount and directory
      "df -h /mnt/openebs-local",
      "ls -la /mnt/openebs-local/",

      # Mark as ready for OpenEBS Local PV
      "echo 'OpenEBS Local PV storage ready at /mnt/openebs-local/local'"
    ]
  }

  depends_on = [
    hcloud_volume_attachment.worker_storage
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
      mount_path  = "/mnt/openebs-local"
      worker_node = local.worker_list[idx].name
      server_id   = local.worker_list[idx].id
      formatted   = true
      mounted     = true
      filesystem  = "ext4"
      openebs_path = "/mnt/openebs-local/local"
    }
  }
  depends_on = [null_resource.verify_raw_devices]
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

    openebs_local_pv_ready = {
      description = "Volumes are formatted, mounted, and ready for OpenEBS Local PV"
      device_paths = [
        for volume in hcloud_volume.worker_storage :
        "/dev/disk/by-id/scsi-0HC_Volume_${volume.id}"
      ]
      mount_paths = [
        for idx in range(length(hcloud_volume.worker_storage)) :
        "/mnt/openebs-local"
      ]
      openebs_base_paths = [
        for idx in range(length(hcloud_volume.worker_storage)) :
        "/mnt/openebs-local/local"
      ]
      notes = [
        "Volumes are formatted with ext4 and mounted at /mnt/openebs-local",
        "Each worker node has a dedicated 40GB storage volume",
        "OpenEBS Local PV base path: /mnt/openebs-local/local",
        "Volumes are persistent across reboots (added to /etc/fstab)",
        "Ready for OpenEBS Local PV Hostpath provisioning"
      ]
    }
  }
}
