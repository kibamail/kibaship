# =============================================================================
# Jump Server (Bastion Host) Module
# =============================================================================
# This module provisions a jump server (bastion host) that serves as the only
# public-facing server in the infrastructure. All other servers will be private
# and accessible only through this jump server. The jump server includes:
# - Public IP for external SSH access
# - Private network connectivity to access cluster nodes
# - SSH key management for accessing other servers
# - Basic security hardening

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

variable "server_type" {
  description = "Hetzner Cloud server type for the jump server"
  type        = string
  default     = "cx22"
}

variable "location" {
  description = "Hetzner Cloud location"
  type        = string
  default     = "nbg1"
}

variable "ubuntu_version" {
  description = "Ubuntu version to use"
  type        = string
  default     = "24.04"
}

variable "ssh_key_id" {
  description = "Hetzner Cloud SSH key ID"
  type        = string
}

variable "ssh_private_key" {
  description = "SSH private key content to copy to jump server"
  type        = string
  sensitive   = true
}

variable "ssh_public_key" {
  description = "SSH public key content"
  type        = string
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
  jump_server_private_ip = "10.0.1.5"

  cloud_init_config = templatefile("${path.module}/../../templates/jump-server-cloud-init.yml.tpl", {
    ssh_private_key = var.ssh_private_key
    ssh_public_key  = var.ssh_public_key
  })
}

# =============================================================================
# Jump Server
# =============================================================================

resource "hcloud_server" "jump_server" {
  name        = "${var.cluster_name}-jump-server"
  image       = data.hcloud_image.ubuntu.id
  server_type = var.server_type
  location    = var.location
  ssh_keys    = [var.ssh_key_id]
  user_data   = local.cloud_init_config

  labels = {
    environment = var.environment
    cluster     = var.cluster_name
    role        = "jump-server"
  }

  public_net {
    ipv4_enabled = true
    ipv6_enabled = false
  }

  network {
    network_id = var.network_id
    ip         = local.jump_server_private_ip
  }
}

# =============================================================================
# Jump Server Readiness Check
# =============================================================================

resource "null_resource" "jump_server_ready" {
  triggers = {
    server_id = hcloud_server.jump_server.id
  }

  provisioner "local-exec" {
    command = "echo 'Jump server ${hcloud_server.jump_server.name} is ready at ${hcloud_server.jump_server.ipv4_address}'"
  }

  depends_on = [hcloud_server.jump_server]
}

# =============================================================================
# Outputs
# =============================================================================

output "jump_server" {
  description = "Jump server details"
  value = {
    id         = hcloud_server.jump_server.id
    name       = hcloud_server.jump_server.name
    public_ip  = hcloud_server.jump_server.ipv4_address
    private_ip = local.jump_server_private_ip
    role       = "jump-server"
  }
}

output "jump_server_public_ip" {
  description = "Public IP address of the jump server"
  value       = hcloud_server.jump_server.ipv4_address
}

output "jump_server_private_ip" {
  description = "Private IP address of the jump server"
  value       = local.jump_server_private_ip
}

output "jump_server_ready" {
  description = "Indicates when jump server is ready"
  value       = "Jump server provisioned and configured successfully."
  depends_on  = [null_resource.jump_server_ready]
}

output "ssh_access_info" {
  description = "Information about SSH access through jump server"
  value = {
    jump_server_ip = hcloud_server.jump_server.ipv4_address
    ssh_command    = "ssh -i .secrets/staging/ssh_key ubuntu@${hcloud_server.jump_server.ipv4_address}"
    note          = "Use this jump server to access all other cluster nodes via their private IPs"
  }
}
