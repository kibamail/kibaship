# =============================================================================
# Servers Module
# =============================================================================
# This module provisions a Kubernetes cluster using Talos OS on Hetzner Cloud,
# including:
# - Control plane nodes (role=control-plane)
# - Worker nodes (role=worker)
# - Automatic cluster bootstrapping
# - Cilium CNI with native routing

terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.51.0"
    }
    talos = {
      source  = "siderolabs/talos"
      version = "~> 0.8.1"
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

variable "api_port_kube_prism" {
  description = "Port for KubePrism local API proxy"
  type        = number
  default     = 7445
}

# =============================================================================
# Data Sources
# =============================================================================

data "hcloud_image" "talos" {
  with_selector = "os=talos"
  most_recent   = true
}

# =============================================================================
# Local Values
# =============================================================================

locals {
  cluster_domain        = "kibaship.internal"
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

control_plane_public_ipv4_list = [
    for i in range(var.control_plane_count) : hcloud_server.control_planes[i].ipv4_address
  ]
  
  worker_public_ipv4_list = [
    for i in range(var.worker_count) : hcloud_server.workers[i].ipv4_address
  ]

  cert_SANs = distinct(
    concat(
      local.control_plane_ips,
      [
        var.k8s_api_public_ip,                                    # Load balancer public IP
        var.k8s_api_private_ip,                                   # Load balancer private IP
        "127.0.0.1",                                             # Localhost & KubePrism
        "kubernetes",                                            # Service name
        "kubernetes.default",                                    # Service FQDN
        "kubernetes.default.svc",                                # Service FQDN
        "kubernetes.default.svc.${local.cluster_domain}"         # Full service FQDN
      ]
    )
  )
}

# =============================================================================
# Talos Machine Secrets
# =============================================================================

resource "talos_machine_secrets" "this" {
  talos_version = var.talos_version
}

data "talos_client_configuration" "this" {
  cluster_name         = var.cluster_name
  client_configuration = talos_machine_secrets.this.client_configuration
  endpoints            = local.control_plane_public_ipv4_list
  nodes                = concat(
    local.control_plane_public_ipv4_list,
    local.worker_public_ipv4_list
  )
}

# =============================================================================
# Control Plane Servers
# =============================================================================

resource "hcloud_server" "control_planes" {
  count       = var.control_plane_count
  name        = "${var.cluster_name}-control-plane-${count.index + 1}"
  image       = data.hcloud_image.talos.id
  server_type = var.server_type
  location    = var.location
  user_data   = data.talos_machine_configuration.control_plane[count.index].machine_configuration

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

  depends_on = [
    data.talos_machine_configuration.control_plane
  ]
}

# =============================================================================
# Worker Servers
# =============================================================================

resource "hcloud_server" "workers" {
  count       = var.worker_count
  name        = "${var.cluster_name}-worker-${count.index + 1}"
  image       = data.hcloud_image.talos.id
  server_type = var.server_type
  location    = var.location
  user_data   = data.talos_machine_configuration.worker[count.index].machine_configuration

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

  depends_on = [
    data.talos_machine_configuration.worker
  ]
}

# =============================================================================
# Talos Machine Configuration - Control Plane
# =============================================================================

data "talos_machine_configuration" "control_plane" {
  count              = var.control_plane_count
  cluster_name       = var.cluster_name
  cluster_endpoint   = var.cluster_endpoint
  machine_type       = "controlplane"
  machine_secrets    = talos_machine_secrets.this.machine_secrets
  talos_version      = var.talos_version
  kubernetes_version = var.kubernetes_version

  config_patches = [
    yamlencode({
      machine = {
        install = {
          image = "ghcr.io/siderolabs/installer:${var.talos_version}"
        }
        certSANs = local.cert_SANs
        kubelet = {
          extraArgs = {
            "rotate-server-certificates" = true
          }
          nodeIP = {
            validSubnets = [local.node_ipv4_cidr]
          }
        }
        network = {
          interfaces = [
            {
              interface = "eth0"
              dhcp      = true
            },
            {
              interface = "eth1"
              dhcp      = true
            }
          ]
        }
        sysctls = {
          "net.core.somaxconn"          = "65535"
          "net.core.netdev_max_backlog" = "4096"
        }
        kernel = {
          modules = [
            {
              name = "br_netfilter"
            },
            {
              name = "overlay"
            }
          ]
        }
        time = {
          servers = [
            "ntp1.hetzner.de",
            "ntp2.hetzner.com",
            "ntp3.hetzner.net",
            "time.cloudflare.com"
          ]
        }
        features = {
          kubernetesTalosAPIAccess = {
            enabled = true
            allowedRoles = ["os:reader"]
            allowedKubernetesNamespaces = ["kube-system"]
          }
          hostDNS = {
            enabled              = true
            forwardKubeDNSToHost = false
            resolveMemberNames   = true
          }
          kubePrism = {
            enabled = true
            port    = var.api_port_kube_prism
          }
        }
      }
      cluster = {
        apiServer = {
          certSANs = local.cert_SANs
          admissionControl = [
            {
              name = "PodSecurity"
              configuration = {
                apiVersion = "pod-security.admission.config.k8s.io/v1beta1"
                kind = "PodSecurityConfiguration"
                exemptions = {
                  namespaces = ["openebs"]
                }
              }
            }
          ]
        }
        allowSchedulingOnControlPlanes = true
        etcd = {
          advertisedSubnets = [local.node_ipv4_cidr]
          extraArgs = {
            "listen-metrics-urls" = "http://0.0.0.0:2381"
          }
        }
        scheduler = {
          extraArgs = {
            "bind-address" = "0.0.0.0"
          }
        }
        coreDNS = {
          disabled = false
        }
        proxy = {
          disabled = true
        }
        network = {
          cni = {
            name = "none"
          }
          podSubnets     = [local.pod_ipv4_cidr]
          serviceSubnets = [local.service_ipv4_cidr]
          dnsDomain = local.cluster_domain
        }
      }
    })
  ]
}

# =============================================================================
# Talos Machine Configuration - Worker
# =============================================================================

data "talos_machine_configuration" "worker" {
  count              = var.worker_count
  cluster_name       = var.cluster_name
  cluster_endpoint   = var.cluster_endpoint
  machine_type       = "worker"
  machine_secrets    = talos_machine_secrets.this.machine_secrets
  talos_version      = var.talos_version
  kubernetes_version = var.kubernetes_version

  config_patches = [
    yamlencode({
      machine = {
        install = {
          image = "ghcr.io/siderolabs/installer:${var.talos_version}"
        }
        certSANs = local.cert_SANs
        kubelet = {
          extraArgs = {
            "rotate-server-certificates" = true
          }
          nodeIP = {
            validSubnets = [local.node_ipv4_cidr]
          }
          extraMounts = [
            {
              destination = "/var/local"
              type        = "bind"
              source      = "/var/local"
              options     = ["bind", "rshared", "rw"]
            }
          ]
        }
        network = {
          interfaces = [
            {
              interface = "eth0"
              dhcp      = true
            },
            {
              interface = "eth1"
              dhcp      = true
            }
          ]
        }
        sysctls = {
          "net.core.somaxconn"          = "65535"
          "net.core.netdev_max_backlog" = "4096"
          "vm.nr_hugepages"             = "1024"
        }
        kernel = {
          modules = [
            {
              name = "br_netfilter"
            },
            {
              name = "overlay"
            }
          ]
        }
        time = {
          servers = [
            "ntp1.hetzner.de",
            "ntp2.hetzner.com",
            "ntp3.hetzner.net",
            "time.cloudflare.com"
          ]
        }
        features = {
          hostDNS = {
            enabled              = true
            forwardKubeDNSToHost = false
            resolveMemberNames   = true
          }
          kubePrism = {
            enabled = true
            port    = var.api_port_kube_prism
          }
        }
        nodeLabels = {
          "openebs.io/engine" = "mayastor"
        }
      }
      cluster = {
        network = {
           cni = {
            name = "none"
          }
          podSubnets     = [
            local.pod_ipv4_cidr
          ]
          serviceSubnets = [
            local.service_ipv4_cidr
          ]
          dnsDomain = local.cluster_domain
        }
      }
    })
  ]
}

# =============================================================================
# Machine Configuration Apply
# =============================================================================

resource "talos_machine_configuration_apply" "control_plane" {
  count                       = var.control_plane_count
  client_configuration        = talos_machine_secrets.this.client_configuration
  machine_configuration_input = data.talos_machine_configuration.control_plane[count.index].machine_configuration
  node                        = local.control_plane_public_ipv4_list[count.index]
  endpoint                    = local.control_plane_public_ipv4_list[count.index]

  depends_on = [hcloud_server.control_planes]
}

resource "talos_machine_configuration_apply" "worker" {
  count                       = var.worker_count
  client_configuration        = talos_machine_secrets.this.client_configuration
  machine_configuration_input = data.talos_machine_configuration.worker[count.index].machine_configuration
  node                        = local.worker_public_ipv4_list[count.index]
  endpoint                    = local.worker_public_ipv4_list[count.index]

  depends_on = [
    hcloud_server.workers,
    talos_machine_bootstrap.this
  ]
}

# =============================================================================
# Cluster Bootstrap
# =============================================================================

# Wait for control plane nodes to be ready before bootstrap
resource "time_sleep" "wait_for_control_plane" {
  depends_on = [talos_machine_configuration_apply.control_plane]
  create_duration = "60s"
}

resource "talos_machine_bootstrap" "this" {
  depends_on = [
    time_sleep.wait_for_control_plane
  ]
  client_configuration = talos_machine_secrets.this.client_configuration
  node                 = local.control_plane_public_ipv4_list[0]
  endpoint             = local.control_plane_public_ipv4_list[0]
}

# Wait for bootstrap to complete
resource "time_sleep" "wait_for_bootstrap" {
  depends_on = [talos_machine_bootstrap.this]
  create_duration = "120s"
}

resource "talos_cluster_kubeconfig" "this" {
  depends_on = [
    time_sleep.wait_for_bootstrap
  ]
  client_configuration = talos_machine_secrets.this.client_configuration
  node                 = local.control_plane_public_ipv4_list[0]
  endpoint             = local.control_plane_public_ipv4_list[0]
}

# =============================================================================
# Cluster Health Check
# =============================================================================

data "http" "talos_health" {
  url = "https://${local.control_plane_public_ipv4_list[0]}:6443/version"

  ca_cert_pem = base64decode(
    yamldecode(talos_cluster_kubeconfig.this.kubeconfig_raw)
    .clusters[0].cluster["certificate-authority-data"]
  )

  retry {
    attempts     = 60
    min_delay_ms = 5000
    max_delay_ms = 5000
  }
  depends_on = [talos_cluster_kubeconfig.this]
}

# =============================================================================
# Cilium CNI Configuration
# =============================================================================



# Wait for cluster to be healthy before installing Cilium
resource "time_sleep" "wait_for_cluster_health" {
  depends_on = [data.http.talos_health]
  create_duration = "60s"
}







# =============================================================================
# Outputs
# =============================================================================

output "cluster_info" {
  description = "Kubernetes cluster information"
  value = {
    name               = var.cluster_name
    endpoint           = var.cluster_endpoint
    kubernetes_version = var.kubernetes_version
    talos_version      = var.talos_version
  }
}

output "control_plane_servers" {
  description = "Control plane server details"
  value = {
    for i, server in hcloud_server.control_planes :
    server.name => {
      id          = server.id
      public_ip   = server.ipv4_address
      private_ip  = local.control_plane_ips[i]
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
      public_ip   = server.ipv4_address
      private_ip  = local.worker_ips[i]
      role        = "worker"
    }
  }
}

output "kubeconfig" {
  description = "Kubernetes configuration for cluster access"
  value       = talos_cluster_kubeconfig.this.kubeconfig_raw
  sensitive   = true
}

output "talosconfig" {
  description = "Talos configuration for cluster management"
  value       = data.talos_client_configuration.this.talos_config
  sensitive   = true
}


