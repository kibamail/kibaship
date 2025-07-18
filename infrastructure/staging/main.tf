# =============================================================================
# KibaShip Staging Infrastructure Configuration
# =============================================================================
# This configuration provisions a complete Kubernetes cluster infrastructure
# for the KibaShip staging environment on Hetzner Cloud, including:
# - Private networking infrastructure
# - Load balancers for Kubernetes API and application traffic
# - Talos OS Kubernetes cluster with Cilium CNI
# - Persistent storage volumes for worker nodes
# - OpenEBS Mayastor preparation

terraform {
  # Remote state backend - temporarily disabled due to S3 credential issues
  # Uncomment and configure once Storj S3 credentials are working

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.51.0"
    }
    talos = {
      source  = "siderolabs/talos"
      version = "~> 0.8.1"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.0.2"
    }
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = "~> 1.19.0"
    }
    http = {
      source  = "hashicorp/http"
      version = "~> 3.4.0"
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

variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
  default     = "kibaship-staging"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "staging"
}

variable "location" {
  description = "Hetzner Cloud location"
  type        = string
  default     = "nbg1"
}

variable "server_type" {
  description = "Hetzner Cloud server type"
  type        = string
  default     = "cx22"
}

variable "talos_version" {
  description = "Talos OS version to use"
  type        = string
  default     = "1.10.15"
}

variable "kubernetes_version" {
  description = "Kubernetes version to install"
  type        = string
  default     = "1.32.0"
}

variable "volume_size" {
  description = "Size of each storage volume in GB"
  type        = number
  default     = 40
}

# =============================================================================
# Provider Configuration
# =============================================================================

provider "hcloud" {
  token = var.hcloud_token
}



provider "talos" {}



provider "kubectl" {
  host                   = module.servers.kubeconfig != null ? yamldecode(module.servers.kubeconfig).clusters[0].cluster.server : null
  client_certificate     = module.servers.kubeconfig != null ? base64decode(yamldecode(module.servers.kubeconfig).users[0].user.client-certificate-data) : null
  client_key             = module.servers.kubeconfig != null ? base64decode(yamldecode(module.servers.kubeconfig).users[0].user.client-key-data) : null
  cluster_ca_certificate = module.servers.kubeconfig != null ? base64decode(yamldecode(module.servers.kubeconfig).clusters[0].cluster.certificate-authority-data) : null
}

# =============================================================================
# Networking Module
# =============================================================================

module "networking" {
  source = "./modules/networking"

  cluster_name      = var.cluster_name
  environment       = var.environment
  network_ip_range  = "10.0.0.0/16"
  subnet_ip_range   = "10.0.1.0/24"
  network_zone      = "eu-central"
}

# =============================================================================
# Load Balancers Module
# =============================================================================

module "load_balancers" {
  source = "./modules/load-balancers"

  cluster_name         = var.cluster_name
  environment          = var.environment
  network_id           = module.networking.network_id
  location             = var.location
  load_balancer_type   = "lb11"
  k8s_api_private_ip   = "10.0.1.100"
  app_private_ip       = "10.0.1.101"

  depends_on = [module.networking]
}

# =============================================================================
# Servers Module
# =============================================================================

module "servers" {
  source = "./modules/servers"

  cluster_name         = var.cluster_name
  environment          = var.environment
  network_id           = module.networking.network_id
  cluster_endpoint     = module.load_balancers.k8s_api_endpoint
  k8s_api_public_ip    = module.load_balancers.k8s_api_public_ip
  k8s_api_private_ip   = "10.0.1.100"
  talos_version        = var.talos_version
  kubernetes_version   = var.kubernetes_version
  server_type          = var.server_type
  location             = var.location
  control_plane_count  = 3
  worker_count         = 3

  depends_on = [module.load_balancers]
}

# =============================================================================
# Storage Module
# =============================================================================

module "storage" {
  source = "./modules/storage"

  cluster_name    = var.cluster_name
  environment     = var.environment
  worker_servers  = module.servers.worker_servers
  volume_size     = var.volume_size
  volume_type     = "network-ssd"
  location        = var.location

  depends_on = [module.servers]
}

# =============================================================================
# Load Balancer Targets (After Servers)
# =============================================================================

resource "hcloud_load_balancer_target" "k8s_api_targets" {
  type             = "label_selector"
  load_balancer_id = module.load_balancers.k8s_api_load_balancer.id
  label_selector   = "role=control-plane"
  use_private_ip   = true

  depends_on = [module.servers]
}

resource "hcloud_load_balancer_target" "app_targets" {
  type             = "label_selector"
  load_balancer_id = module.load_balancers.app_load_balancer.id
  label_selector   = "role=worker"
  use_private_ip   = true

  depends_on = [module.servers]
}



# =============================================================================
# Outputs
# =============================================================================

output "cluster_info" {
  description = "Complete cluster information"
  value = {
    name               = var.cluster_name
    environment        = var.environment
    endpoint           = module.load_balancers.k8s_api_endpoint
    kubernetes_version = var.kubernetes_version
    talos_version      = var.talos_version
  }
}

output "network" {
  description = "Network infrastructure details"
  value       = module.networking.network
}

output "load_balancers" {
  description = "Load balancer details"
  value = {
    k8s_api = module.load_balancers.k8s_api_load_balancer
    app     = module.load_balancers.app_load_balancer
  }
}

output "servers" {
  description = "Server details"
  value = {
    control_planes = module.servers.control_plane_servers
    workers        = module.servers.worker_servers
  }
}

output "storage" {
  description = "Storage configuration details"
  value = module.storage.storage_summary
}

output "kubeconfig" {
  description = "Kubernetes configuration for cluster access"
  value       = module.servers.kubeconfig
  sensitive   = true
}

output "talosconfig" {
  description = "Talos configuration for cluster management"
  value       = module.servers.talosconfig
  sensitive   = true
}

output "cluster_summary" {
  description = "Complete cluster deployment summary"
  value = {
    cluster = {
      name        = var.cluster_name
      environment = var.environment
      endpoint    = module.load_balancers.k8s_api_endpoint
      network     = module.networking.network
    }
    load_balancers = {
      k8s_api_ip = module.load_balancers.k8s_api_public_ip
      app_ip     = module.load_balancers.app_public_ip
    }
    servers = {
      control_planes = length(module.servers.control_plane_servers)
      workers        = length(module.servers.worker_servers)
    }
    storage = {
      total_volumes = module.storage.storage_summary.total_volumes
      total_storage = module.storage.storage_summary.total_storage
    }
    access = {
      kubernetes_api = "Use load balancer IP: ${module.load_balancers.k8s_api_public_ip}:6443"
      applications   = "Use load balancer IP: ${module.load_balancers.app_public_ip} (ports 80/443)"
      note          = "No DNS configuration required - direct IP access"
    }
  }
}