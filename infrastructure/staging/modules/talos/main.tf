# =============================================================================
# Talos Configuration Module
# =============================================================================
# This module handles Talos OS configuration for the Kubernetes cluster,
# including:
# - Machine secrets generation
# - Control plane and worker node configurations
# - Cluster bootstrapping
# - Kubeconfig generation

terraform {
  required_providers {
    talos = {
      source  = "siderolabs/talos"
      version = "~> 0.8.1"
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

variable "talos_version" {
  description = "Talos version to use"
  type        = string
}

variable "kubernetes_version" {
  description = "Kubernetes version to use"
  type        = string
}

variable "k8s_api_public_ip" {
  description = "Public IP of the Kubernetes API load balancer"
  type        = string
}

variable "k8s_api_private_ip" {
  description = "Private IP of the Kubernetes API load balancer"
  type        = string
}

variable "control_plane_nodes" {
  description = "Map of control plane nodes with their details"
  type = map(object({
    name       = string
    private_ip = string
  }))
}

variable "worker_nodes" {
  description = "Map of worker nodes with their details"
  type = map(object({
    name       = string
    private_ip = string
  }))
}

variable "control_plane_public_ips" {
  description = "List of control plane public IP addresses"
  type        = list(string)
  default     = []
}

variable "cluster_domain" {
  description = "Cluster domain name"
  type        = string
  default     = "cluster.local"
}

variable "control_plane_private_vip_ipv4" {
  description = "Private VIP IPv4 address for control plane alias"
  type        = string
  default     = "10.0.1.100"
}

# =============================================================================
# Talos Configuration
# =============================================================================

resource "talos_machine_secrets" "this" {
  talos_version = var.talos_version
}

locals {
  api_port_k8s        = 6443
  api_port_kube_prism = 7445

  cluster_api_host_private = "kube.${var.cluster_domain}"
  control_plane_private_vip_ipv4 = var.control_plane_private_vip_ipv4

  cluster_endpoint_internal = local.cluster_api_host_private
  cluster_endpoint_url = "https://${local.cluster_endpoint_internal}:${local.api_port_k8s}"

  cert_SANs = distinct(concat(
    [var.k8s_api_public_ip, var.k8s_api_private_ip],
    [for node in var.control_plane_nodes : node.private_ip],
    var.control_plane_public_ips,
    [local.control_plane_private_vip_ipv4, local.cluster_api_host_private]
  ))

  extra_host_entries = [
    {
      ip = local.control_plane_private_vip_ipv4
      aliases = [local.cluster_api_host_private]
    }
  ]
}

data "talos_machine_configuration" "control_plane" {
  for_each           = var.control_plane_nodes
  talos_version      = var.talos_version
  cluster_name       = var.cluster_name
  cluster_endpoint   = local.cluster_endpoint_url
  kubernetes_version = var.kubernetes_version
  machine_type       = "controlplane"
  machine_secrets    = talos_machine_secrets.this.machine_secrets

  config_patches = [
    yamlencode({
      machine = {
        install = {
          image = "ghcr.io/siderolabs/installer:${var.talos_version}"
          extraKernelArgs = [
            "ipv6.disable=1",
          ]
        }
        network = {
          hostname = each.value.name
          interfaces = [{
            interface = "eth0"
            addresses = [each.value.private_ip]
          }]
          extraHostEntries = local.extra_host_entries
        }
        kubelet = {
          extraArgs = {
            "rotate-server-certificates" = "true"
          }
        }
        certSANs = local.cert_SANs
      }
      cluster = {
        allowSchedulingOnControlPlanes = false
        controllerManager = {
          extraArgs = {
            "bind-address" = "0.0.0.0"
          }
        }
        scheduler = {
          extraArgs = {
            "bind-address" = "0.0.0.0"
          }
        }
        discovery = {
          registries = {
            kubernetes = {
              disabled = false
            }
            service = {
              disabled = true
            }
          }
        }
        proxy = {
          disabled = true
        }
      }
    })
  ]

  docs     = false
  examples = false
}

data "talos_machine_configuration" "worker" {
  for_each           = var.worker_nodes
  talos_version      = var.talos_version
  cluster_name       = var.cluster_name
  cluster_endpoint   = local.cluster_endpoint_url
  kubernetes_version = var.kubernetes_version
  machine_type       = "worker"
  machine_secrets    = talos_machine_secrets.this.machine_secrets

  config_patches = [
    yamlencode({
      machine = {
        install = {
          image = "ghcr.io/siderolabs/installer:${var.talos_version}"
          extraKernelArgs = [
            "ipv6.disable=1",
          ]
        }
        network = {
          hostname = each.value.name
          interfaces = [{
            interface = "eth0"
            addresses = [each.value.private_ip]
          }]
          extraHostEntries = local.extra_host_entries
        }
        kubelet = {
          extraArgs = {
            "rotate-server-certificates" = "true"
          }
        }
        certSANs = local.cert_SANs
      }
      cluster = {
        discovery = {
          registries = {
            kubernetes = {
              disabled = false
            }
            service = {
              disabled = true
            }
          }
        }
        proxy = {
          disabled = true
        }
      }
    })
  ]

  docs     = false
  examples = false
}

# =============================================================================
# Talos Bootstrap and Kubeconfig
# =============================================================================

resource "talos_machine_bootstrap" "this" {
  count                = length(var.control_plane_public_ips) > 0 ? 1 : 0
  client_configuration = talos_machine_secrets.this.client_configuration
  endpoint             = var.control_plane_public_ips[0]
  node                 = var.control_plane_public_ips[0]
}

data "talos_client_configuration" "this" {
  cluster_name         = var.cluster_name
  client_configuration = talos_machine_secrets.this.client_configuration
  endpoints            = [var.k8s_api_public_ip]
}

resource "talos_cluster_kubeconfig" "this" {
  count                = length(var.control_plane_public_ips) > 0 ? 1 : 0
  client_configuration = talos_machine_secrets.this.client_configuration
  node                 = var.control_plane_public_ips[0]

  depends_on = [talos_machine_bootstrap.this]
}

locals {
  kubeconfig = length(var.control_plane_public_ips) > 0 ? replace(
    talos_cluster_kubeconfig.this[0].kubeconfig_raw,
    local.cluster_endpoint_url,
    "https://${var.k8s_api_public_ip}:${local.api_port_k8s}"
  ) : ""
}

# =============================================================================
# Outputs
# =============================================================================

output "control_plane_machine_configurations" {
  description = "Talos machine configurations for control plane nodes"
  value = {
    for key, config in data.talos_machine_configuration.control_plane :
    key => config.machine_configuration
  }
}

output "worker_machine_configurations" {
  description = "Talos machine configurations for worker nodes"
  value = {
    for key, config in data.talos_machine_configuration.worker :
    key => config.machine_configuration
  }
}

output "talos_config" {
  description = "Talos client configuration"
  value       = data.talos_client_configuration.this.talos_config
  sensitive   = true
}

output "kubeconfig" {
  description = "Kubernetes configuration"
  value       = local.kubeconfig
  sensitive   = true
}

output "machine_secrets" {
  description = "Talos machine secrets"
  value       = talos_machine_secrets.this.machine_secrets
  sensitive   = true
}

output "bootstrap_ready" {
  description = "Indicates when Talos bootstrap is complete"
  value       = length(talos_machine_bootstrap.this) > 0 ? talos_machine_bootstrap.this[0].id : "no-bootstrap"
}