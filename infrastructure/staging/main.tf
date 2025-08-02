# =============================================================================
# KibaShip Staging Infrastructure Configuration
# =============================================================================
# This configuration provisions a complete Kubernetes cluster infrastructure
# for the KibaShip staging environment on Hetzner Cloud, including:
# - Private networking infrastructure
# - Load balancers for Kubernetes API and application traffic
# - Talos linux K8s cluster with Cilium CNI
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
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0.6"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5.0"
    }
    time = {
      source  = "hashicorp/time"
      version = "~> 0.12.0"
    }
    http = {
      source  = "hashicorp/http"
      version = ">= 3.5.0"
    }
    kubectl = {
      source  = "alekc/kubectl"
      version = ">= 2.0.4"
    }
    helm = {
      source  = "hashicorp/helm"
      version = ">= 3.0.2"
    }
    talos = {
      source  = "siderolabs/talos"
      version = "~> 0.8.1"
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
  default     = "cpx21"
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

variable "kube_pods_subnet" {
  description = "Kubernetes pod subnet CIDR"
  type        = string
  default     = "10.0.16.0/20"
}

variable "kube_service_addresses" {
  description = "Kubernetes service subnet CIDR"
  type        = string
  default     = "10.0.8.0/21"
}

variable "volume_size" {
  description = "Size of each storage volume in GB"
  type        = number
  default     = 40
}

provider "hcloud" {
  token = var.hcloud_token
}

# =============================================================================
# SSH Key Management Module
# =============================================================================

module "ssh_keys" {
  source = "./modules/ssh-keys"

  cluster_name         = var.cluster_name
  environment          = var.environment
  ssh_public_key_path  = ".secrets/ssh/id_ed25519.pub"
  ssh_private_key_path = ".secrets/ssh/id_ed25519"
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

# =============================================================================
# Local Values for Node Configuration
# =============================================================================

locals {
  control_plane_count = 3
  worker_count        = 3

  control_plane_ips = [
    for i in range(local.control_plane_count) : "10.0.1.${10 + i}"
  ]

  worker_ips = [
    for i in range(local.worker_count) : "10.0.1.${20 + i}"
  ]

  control_plane_nodes = {
    for i in range(local.control_plane_count) :
    "${var.cluster_name}-control-plane-${i + 1}" => {
      name       = "${var.cluster_name}-control-plane-${i + 1}"
      private_ip = local.control_plane_ips[i]
    }
  }

  worker_nodes = {
    for i in range(local.worker_count) :
    "${var.cluster_name}-worker-${i + 1}" => {
      name       = "${var.cluster_name}-worker-${i + 1}"
      private_ip = local.worker_ips[i]
    }
  }
}

# =============================================================================
# Talos Configuration Module
# =============================================================================

module "talos" {
  source = "./modules/talos"

  cluster_name                    = var.cluster_name
  talos_version                  = "v1.8"
  kubernetes_version             = "v1.31.1"
  k8s_api_public_ip              = module.load_balancers.k8s_api_public_ip
  k8s_api_private_ip             = var.api_load_balancer_private_ip
  control_plane_nodes            = local.control_plane_nodes
  worker_nodes                   = local.worker_nodes
  enable_alias_ip                = true
  cluster_domain                 = "cluster.local"
  control_plane_private_vip_ipv4 = var.api_load_balancer_private_ip

  depends_on = [module.load_balancers]
}

# =============================================================================
# Servers Module
# =============================================================================

module "servers" {
  source = "./modules/servers"

  cluster_name                        = var.cluster_name
  environment                         = var.environment
  network_id                          = module.networking.network_id
  cluster_endpoint                    = module.load_balancers.k8s_api_endpoint
  k8s_api_public_ip                   = module.load_balancers.k8s_api_public_ip
  k8s_api_private_ip                  = var.api_load_balancer_private_ip
  server_type                         = var.server_type
  location                            = var.location
  control_plane_count                 = local.control_plane_count
  worker_count                        = local.worker_count
  ssh_key_id                          = module.ssh_keys.ssh_key_id
  control_plane_machine_configurations = module.talos.control_plane_machine_configurations
  worker_machine_configurations       = module.talos.worker_machine_configurations

  depends_on = [module.load_balancers, module.ssh_keys, module.talos]
}

# =============================================================================
# Talos Bootstrap
# =============================================================================

resource "talos_machine_bootstrap" "cluster" {
  count                = local.control_plane_count > 0 ? 1 : 0
  client_configuration = module.talos.machine_secrets.client_configuration
  endpoint             = module.servers.control_plane_public_ips[0]
  node                 = module.servers.control_plane_public_ips[0]

  depends_on = [module.servers]
}

data "talos_client_configuration" "cluster" {
  cluster_name         = var.cluster_name
  client_configuration = module.talos.machine_secrets.client_configuration
  endpoints            = module.servers.control_plane_public_ips

  depends_on = [module.servers]
}

resource "talos_cluster_kubeconfig" "cluster" {
  count                = local.control_plane_count > 0 ? 1 : 0
  client_configuration = module.talos.machine_secrets.client_configuration
  node                 = module.servers.control_plane_public_ips[0]

  depends_on = [talos_machine_bootstrap.cluster]
}

locals {
  kubeconfig = local.control_plane_count > 0 ? replace(
    talos_cluster_kubeconfig.cluster[0].kubeconfig_raw,
    "https://${var.api_load_balancer_private_ip}:6443",
    "https://${module.load_balancers.k8s_api_public_ip}:6443"
  ) : ""
}

# =============================================================================
# Outputs
# =============================================================================

output "talos_config" {
  description = "Talos client configuration"
  value       = data.talos_client_configuration.cluster.talos_config
  sensitive   = true
}

output "kubeconfig" {
  description = "Kubernetes configuration"
  value       = local.kubeconfig
  sensitive   = true
}

output "cluster_endpoint" {
  description = "Kubernetes API endpoint"
  value       = "https://${module.load_balancers.k8s_api_public_ip}:6443"
}
