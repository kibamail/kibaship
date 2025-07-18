# =============================================================================
# Networking Module
# =============================================================================
# This module provisions the core networking infrastructure for the KibaShip
# staging environment, including:
# - Private network with configurable IP range
# - Subnet configuration for the staging environment
# - Network labels for resource organization

terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.51"
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

variable "network_ip_range" {
  description = "IP range for the private network"
  type        = string
  default     = "10.0.0.0/16"
}

variable "subnet_ip_range" {
  description = "IP range for the subnet"
  type        = string
  default     = "10.0.1.0/24"
}

variable "network_zone" {
  description = "Network zone for the subnet"
  type        = string
  default     = "eu-central"
}

# =============================================================================
# Private Network Infrastructure
# =============================================================================

resource "hcloud_network" "main" {
  name     = "${var.cluster_name}-network"
  ip_range = var.network_ip_range

  labels = {
    environment = var.environment
    cluster     = var.cluster_name
  }
}

resource "hcloud_network_subnet" "main" {
  network_id   = hcloud_network.main.id
  type         = "cloud"
  network_zone = var.network_zone
  ip_range     = var.subnet_ip_range
}

# =============================================================================
# Outputs
# =============================================================================

output "network" {
  description = "Private network details"
  value = {
    id       = hcloud_network.main.id
    name     = hcloud_network.main.name
    ip_range = hcloud_network.main.ip_range
  }
}

output "subnet" {
  description = "Subnet details"
  value = {
    id       = hcloud_network_subnet.main.id
    ip_range = hcloud_network_subnet.main.ip_range
  }
}

output "network_id" {
  description = "Network ID for use in other modules"
  value       = hcloud_network.main.id
}

output "subnet_id" {
  description = "Subnet ID for use in other modules"
  value       = hcloud_network_subnet.main.id
}
