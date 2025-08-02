# =============================================================================
# Servers Module
# =============================================================================
# This module provisions Talos OS servers on Hetzner Cloud, including:
# - Control plane nodes (role=control-plane)
# - Worker nodes (role=worker)
# - Talos OS configuration and bootstrapping
# - Network configuration for Kubernetes cluster

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
  description = "Hetzner Cloud SSH key ID for server access"
  type        = string
}

variable "control_plane_machine_configurations" {
  description = "Talos machine configurations for control plane nodes"
  type        = map(string)
  default     = {}
}

variable "worker_machine_configurations" {
  description = "Talos machine configurations for worker nodes"
  type        = map(string)
  default     = {}
}

# =============================================================================
# Data Sources
# =============================================================================

data "hcloud_image" "talos" {
  with_selector = "os=talos"
  with_architecture = "x86"
  most_recent   = true
}

# =============================================================================
# Placement Groups
# =============================================================================

resource "hcloud_placement_group" "control_plane" {
  name = "${var.cluster_name}-control-plane-pg"
  type = "spread"

  labels = {
    environment = var.environment
    cluster     = var.cluster_name
    role        = "control-plane"
  }
}

resource "hcloud_placement_group" "workers" {
  name = "${var.cluster_name}-workers-pg"
  type = "spread"

  labels = {
    environment = var.environment
    cluster     = var.cluster_name
    role        = "worker"
  }
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

  control_plane_nodes = {
    for i in range(var.control_plane_count) :
    "${var.cluster_name}-control-plane-${i + 1}" => {
      name       = "${var.cluster_name}-control-plane-${i + 1}"
      private_ip = local.control_plane_ips[i]
    }
  }

  worker_nodes = {
    for i in range(var.worker_count) :
    "${var.cluster_name}-worker-${i + 1}" => {
      name       = "${var.cluster_name}-worker-${i + 1}"
      private_ip = local.worker_ips[i]
    }
  }
}

# =============================================================================
# Control Plane Servers
# =============================================================================

resource "hcloud_server" "control_planes" {
  for_each         = local.control_plane_nodes
  name             = each.value.name
  image            = data.hcloud_image.talos.id
  server_type      = var.server_type
  location         = var.location
  ssh_keys         = [var.ssh_key_id]
  placement_group_id = hcloud_placement_group.control_plane.id
  user_data        = lookup(var.control_plane_machine_configurations, each.key, "")

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
    ip         = each.value.private_ip
  }
}

# =============================================================================
# Worker Servers
# =============================================================================

resource "hcloud_server" "workers" {
  for_each         = local.worker_nodes
  name             = each.value.name
  image            = data.hcloud_image.talos.id
  server_type      = var.server_type
  location         = var.location
  ssh_keys         = [var.ssh_key_id]
  placement_group_id = hcloud_placement_group.workers.id
  user_data        = lookup(var.worker_machine_configurations, each.key, "")

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
    ip         = each.value.private_ip
  }
}

# =============================================================================
# Infrastructure Ready
# =============================================================================

resource "null_resource" "infrastructure_ready" {
  triggers = {
    control_planes_ready = join(",", [for server in hcloud_server.control_planes : server.id])
    workers_ready        = join(",", [for server in hcloud_server.workers : server.id])
  }

  provisioner "local-exec" {
    command = "echo 'All servers are ready'"
  }

  depends_on = [hcloud_server.control_planes, hcloud_server.workers]
}


# =============================================================================
# Outputs
# =============================================================================

output "cluster_info" {
  description = "Infrastructure cluster information"
  value = {
    name     = var.cluster_name
    endpoint = var.cluster_endpoint
    os       = "talos"
  }
}

output "control_plane_servers" {
  description = "Control plane server details"
  value = {
    for name, server in hcloud_server.control_planes :
    name => {
      id          = server.id
      private_ip  = server.network[0].ip
      public_ip   = server.ipv4_address
      role        = "control-plane"
    }
  }
}

output "worker_servers" {
  description = "Worker server details"
  value = {
    for name, server in hcloud_server.workers :
    name => {
      id          = server.id
      private_ip  = server.network[0].ip
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
  value       = [for server in hcloud_server.control_planes : server.network[0].ip]
}

output "worker_ips" {
  description = "Private IP addresses of worker nodes"
  value       = [for server in hcloud_server.workers : server.network[0].ip]
}

output "control_plane_public_ips" {
  description = "Public IP addresses of control plane nodes"
  value       = [for server in hcloud_server.control_planes : server.ipv4_address]
}

output "worker_public_ips" {
  description = "Public IP addresses of worker nodes"
  value       = [for server in hcloud_server.workers : server.ipv4_address]
}

output "control_plane_nodes" {
  description = "Control plane node details for Talos configuration"
  value       = local.control_plane_nodes
}

output "worker_nodes" {
  description = "Worker node details for Talos configuration"
  value       = local.worker_nodes
}


