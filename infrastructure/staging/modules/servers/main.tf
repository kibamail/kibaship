# =============================================================================
# Servers Module
# =============================================================================
# This module provisions Ubuntu 24.04 servers on Hetzner Cloud, including:
# - Control plane nodes (role=control-plane)
# - Worker nodes (role=worker)
# - Basic system preparation
# - SSH key management and secure access

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
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5.0"
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

variable "network_id" {
  description = "ID of the private network"
  type        = string
}

variable "cluster_endpoint" {
  description = "Kubernetes API endpoint URL"
  type        = string
}

variable "k8s_api_public_ip" {
  description = "Public IP of the Kubernetes API load balancer"
  type        = string
}

variable "k8s_api_private_ip" {
  description = "Private IP of the Kubernetes API load balancer"
  type        = string
  default     = "10.0.1.100"
}

variable "ubuntu_version" {
  description = "Ubuntu version to use"
  type        = string
  default     = "24.04"
}

variable "server_type" {
  description = "Hetzner Cloud server type"
  type        = string
  default     = "cx22"
}

variable "location" {
  description = "Hetzner Cloud location"
  type        = string
  default     = "nbg1"
}

variable "control_plane_count" {
  description = "Number of control plane nodes"
  type        = number
  default     = 3
}

variable "worker_count" {
  description = "Number of worker nodes"
  type        = number
  default     = 3
}

variable "ssh_key_id" {
  description = "Hetzner Cloud SSH key ID"
  type        = string
}

variable "ssh_private_key" {
  description = "SSH private key for server access"
  type        = string
  sensitive   = true
}

variable "ssh_public_key" {
  description = "SSH public key content"
  type        = string
}

variable "jump_server_public_ip" {
  description = "Public IP address of the jump server for SSH access"
  type        = string
}

variable "jump_server_private_ip" {
  description = "Private IP address of the jump server for routing"
  type        = string
  default     = "10.0.1.5"
}

# =============================================================================
# Data Sources
# =============================================================================

data "hcloud_image" "ubuntu" {
  name = "ubuntu-${var.ubuntu_version}"
}

# =============================================================================
# Local Values
# =============================================================================

locals {
  network_ipv4_cidr     = "10.0.0.0/16"
  node_ipv4_cidr        = "10.0.1.0/24"
  pod_ipv4_cidr         = "10.0.16.0/20"
  service_ipv4_cidr     = "10.0.8.0/21"

  control_plane_ips = [
    for i in range(var.control_plane_count) : "10.0.1.${10 + i}"
  ]

  worker_ips = [
    for i in range(var.worker_count) : "10.0.1.${20 + i}"
  ]
}

# =============================================================================
# Cloud-Init Configuration
# =============================================================================

locals {
  # Control plane cloud-init configurations (one per server)
  control_plane_cloud_init_configs = [
    for i in range(var.control_plane_count) : templatefile("${path.module}/../../templates/control-plane-cloud-init.yml.tpl", {
      ssh_public_key           = var.ssh_public_key
      control_plane_private_ip = local.control_plane_ips[i]
      jump_server_private_ip   = var.jump_server_private_ip
    })
  ]

  # Worker cloud-init configurations (one per server)
  worker_cloud_init_configs = [
    for i in range(var.worker_count) : templatefile("${path.module}/../../templates/worker-cloud-init.yml.tpl", {
      ssh_public_key         = var.ssh_public_key
      worker_private_ip      = local.worker_ips[i]
      jump_server_private_ip = var.jump_server_private_ip
    })
  ]
}

# =============================================================================
# Control Plane Servers
# =============================================================================

resource "hcloud_server" "control_planes" {
  count       = var.control_plane_count
  name        = "${var.cluster_name}-control-plane-${count.index + 1}"
  image       = data.hcloud_image.ubuntu.id
  server_type = var.server_type
  location    = var.location
  ssh_keys    = [var.ssh_key_id]
  user_data   = local.control_plane_cloud_init_configs[count.index]

  labels = {
    environment = var.environment
    cluster     = var.cluster_name
    role        = "control-plane"
  }

  public_net {
    ipv4_enabled = true
    ipv6_enabled = false
  }

  network {
    network_id = var.network_id
    ip         = local.control_plane_ips[count.index]
  }
}



# =============================================================================
# Worker Servers
# =============================================================================

resource "hcloud_server" "workers" {
  count       = var.worker_count
  name        = "${var.cluster_name}-worker-${count.index + 1}"
  image       = data.hcloud_image.ubuntu.id
  server_type = var.server_type
  location    = var.location
  ssh_keys    = [var.ssh_key_id]
  user_data   = local.worker_cloud_init_configs[count.index]

  labels = {
    environment = var.environment
    cluster     = var.cluster_name
    role        = "worker"
  }

  public_net {
    ipv4_enabled = true
    ipv6_enabled = false
  }

  network {
    network_id = var.network_id
    ip         = local.worker_ips[count.index]
  }
}






# =============================================================================
# Infrastructure Ready
# =============================================================================

resource "null_resource" "infrastructure_ready" {
  triggers = {
    control_planes_ready = join(",", [for i in range(var.control_plane_count) : hcloud_server.control_planes[i].id])
    workers_ready        = join(",", [for i in range(var.worker_count) : hcloud_server.workers[i].id])
  }

  provisioner "local-exec" {
    command = "echo 'All servers are ready and configured via cloud-init'"
  }

  depends_on = [hcloud_server.control_planes, hcloud_server.workers]
}


# =============================================================================
# Outputs
# =============================================================================

output "cluster_info" {
  description = "Infrastructure cluster information"
  value = {
    name           = var.cluster_name
    endpoint       = var.cluster_endpoint
    ubuntu_version = var.ubuntu_version
  }
}

output "control_plane_servers" {
  description = "Control plane server details"
  value = {
    for i, server in hcloud_server.control_planes :
    server.name => {
      id          = server.id
      private_ip  = local.control_plane_ips[i]
      public_ip   = server.ipv4_address
      role        = "control-plane"
    }
  }
}

output "worker_servers" {
  description = "Worker server details"
  value = {
    for i, server in hcloud_server.workers :
    server.name => {
      id          = server.id
      private_ip  = local.worker_ips[i]
      public_ip   = server.ipv4_address
      role        = "worker"
    }
  }
}

output "servers_ready" {
  description = "Indicates when servers are ready"
  value       = "Infrastructure provisioned successfully."
}

output "infrastructure_ready" {
  description = "Indicates when the infrastructure is ready"
  value       = true
  depends_on  = [null_resource.infrastructure_ready]
}

output "control_plane_ips" {
  description = "Private IP addresses of control plane nodes"
  value       = local.control_plane_ips
}

output "worker_ips" {
  description = "Private IP addresses of worker nodes"
  value       = local.worker_ips
}

output "control_plane_public_ips" {
  description = "Public IP addresses of control plane nodes"
  value       = [for server in hcloud_server.control_planes : server.ipv4_address]
}

output "worker_public_ips" {
  description = "Public IP addresses of worker nodes"
  value       = [for server in hcloud_server.workers : server.ipv4_address]
}

output "cloud_init_debug" {
  description = "Debug information about cloud-init configuration"
  value = {
    control_plane_cloud_init_configured = length(local.control_plane_cloud_init_configs) > 0
    worker_cloud_init_configured = length(local.worker_cloud_init_configs) > 0
    setup_method = "cloud-init"
  }
}


