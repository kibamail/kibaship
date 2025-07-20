# =============================================================================
# KibaShip Staging Infrastructure Configuration
# =============================================================================
# This configuration provisions a complete Kubernetes cluster infrastructure
# for the KibaShip staging environment on Hetzner Cloud, including:
# - Private networking infrastructure
# - Load balancers for Kubernetes API and application traffic
# - Ubuntu 24.04 Kubernetes cluster with kubeadm and Cilium CNI
# - Persistent storage volumes for worker nodes
# - OpenEBS Mayastor preparation
# - SSH key management for secure server access

terraform {
  # Remote state backend - temporarily disabled due to S3 credential issues
  # Uncomment and configure once Storj S3 credentials are working

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.51.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2.0"
    }
    time = {
      source  = "hashicorp/time"
      version = "~> 0.12.0"
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

variable "api_load_balancer_private_ip" {
  description = "Private IP address for the API load balancer"
  type        = string
  default     = "10.0.1.100"
}

variable "ubuntu_version" {
  description = "Ubuntu version to use"
  type        = string
  default     = "24.04"
}



variable "volume_size" {
  description = "Size of each storage volume in GB"
  type        = number
  default     = 40
}

variable "ssh_public_key_path" {
  description = "Path to store the generated SSH public key"
  type        = string
  default     = ".secrets/staging/ssh_key.pub"
}

variable "ssh_private_key_path" {
  description = "Path to store the generated SSH private key"
  type        = string
  default     = ".secrets/staging/ssh_key"
}

# =============================================================================
# Provider Configuration
# =============================================================================

provider "hcloud" {
  token = var.hcloud_token
}

provider "tls" {}

provider "local" {}

provider "null" {}

# kubectl provider configuration is handled in individual modules
# that need it, since the kubeconfig file doesn't exist at plan time

# =============================================================================
# SSH Key Management Module
# =============================================================================

module "ssh_keys" {
  source = "./modules/ssh-keys"

  cluster_name         = var.cluster_name
  environment          = var.environment
  ssh_public_key_path  = var.ssh_public_key_path
  ssh_private_key_path = var.ssh_private_key_path
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
  k8s_api_private_ip   = var.api_load_balancer_private_ip
  ubuntu_version       = var.ubuntu_version
  server_type          = var.server_type
  location             = var.location
  control_plane_count  = 3
  worker_count         = 3
  ssh_key_id           = module.ssh_keys.ssh_key_id
  ssh_private_key      = module.ssh_keys.ssh_private_key
  ssh_public_key       = module.ssh_keys.ssh_public_key

  depends_on = [module.load_balancers, module.ssh_keys]
}

# =============================================================================
# Storage Module
# =============================================================================

module "storage" {
  source = "./modules/storage"

  cluster_name     = var.cluster_name
  environment      = var.environment
  worker_servers   = module.servers.worker_servers
  volume_size      = var.volume_size
  volume_type      = "network-ssd"
  location         = var.location
  ssh_private_key  = module.ssh_keys.ssh_private_key

  depends_on = [module.servers]
}

# =============================================================================
# Infrastructure Complete
# =============================================================================
# This configuration provisions the complete infrastructure including:
# - Servers, networking, load balancers, and storage
# - Servers are prepared with basic system configuration
# - Ready for application deployment

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
  description = "Complete infrastructure information"
  value = {
    name           = var.cluster_name
    environment    = var.environment
    endpoint       = module.load_balancers.k8s_api_endpoint
    ubuntu_version = var.ubuntu_version
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

output "servers_ready" {
  description = "Server deployment status"
  value       = module.servers.servers_ready
}

output "ssh_private_key" {
  description = "SSH private key for server access"
  value       = module.ssh_keys.ssh_private_key
  sensitive   = true
}

output "ssh_public_key" {
  description = "SSH public key for server access"
  value       = module.ssh_keys.ssh_public_key
}

output "control_plane_ips" {
  description = "Public IP addresses of control plane nodes"
  value       = module.servers.control_plane_ips
}

output "worker_ips" {
  description = "Public IP addresses of worker nodes"
  value       = module.servers.worker_ips
}

output "deployment_info" {
  description = "Infrastructure deployment information"
  value = <<-EOT
    Infrastructure provisioned successfully!

    Server Details:
    - Control Plane IPs: ${join(", ", module.servers.control_plane_ips)}
    - Worker IPs: ${join(", ", module.servers.worker_ips)}
    - SSH Key: .secrets/staging/ssh_key

    Infrastructure is ready for application deployment.
  EOT
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