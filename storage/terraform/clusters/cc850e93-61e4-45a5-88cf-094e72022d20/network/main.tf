terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.52"
    }
  }

  backend "s3" {
    bucket = "terraform-state-staging"
    key    = "clusters/cc850e93-61e4-45a5-88cf-094e72022d20/network/terraform.tfstate"
    region = "auto"

    endpoints = {
      s3 = "https://e61cebd91eb54d4ebeec9c5d525ae041.r2.cloudflarestorage.com"
    }

    skip_credentials_validation = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    use_path_style           = true
  }
}

variable "hcloud_token" {
  description = "Hetzner Cloud API Token"
  type        = string
  sensitive   = true
}

variable "cluster_name" {
  description = "Name of the cluster"
  type        = string
}

variable "network_zone" {
  description = "Network zone for the private network"
  type        = string
}

provider "hcloud" {
  token = var.hcloud_token
}

resource "hcloud_network" "cluster_network" {
  name              = "${var.cluster_name}-network"
  ip_range          = "10.0.0.0/16"

  labels = {
    cluster    = var.cluster_name
    managed_by = "kibaship"
  }
}

resource "hcloud_network_subnet" "cluster_subnet" {
  type         = "cloud"
  network_id   = hcloud_network.cluster_network.id
  network_zone = var.network_zone
  ip_range     = "10.0.1.0/24"
}

output "network_id" {
  description = "ID of the created private network"
  value       = hcloud_network.cluster_network.id
}

output "network_name" {
  description = "Name of the created private network"
  value       = hcloud_network.cluster_network.name
}

output "network_ip_range" {
  description = "IP range of the private network"
  value       = hcloud_network.cluster_network.ip_range
}

output "subnet_id" {
  description = "ID of the created subnet"
  value       = hcloud_network_subnet.cluster_subnet.id
}

output "subnet_ip_range" {
  description = "IP range of the subnet"
  value       = hcloud_network_subnet.cluster_subnet.ip_range
}

output "subnet_network_zone" {
  description = "Network zone of the subnet"
  value       = hcloud_network_subnet.cluster_subnet.network_zone
}

output "network_labels" {
  description = "Labels applied to the network"
  value       = hcloud_network.cluster_network.labels
}