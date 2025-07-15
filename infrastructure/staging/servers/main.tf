# =============================================================================
# KibaShip Staging Kubernetes Cluster Configuration
# =============================================================================
# This configuration provisions a 6-node Kubernetes cluster using Talos OS
# on Hetzner Cloud, including:
# - 3 control plane nodes (role=control-plane)
# - 3 worker nodes (role=worker)
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
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.0.2"
    }
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = "~> 1.19.0"
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

variable "cilium_version" {
  description = "Cilium CNI version to install"
  type        = string
  default     = "1.17.5"
}

# =============================================================================
# Provider Configuration
# =============================================================================

provider "hcloud" {
  token = var.hcloud_token
}

provider "talos" {}

provider "helm" {}

provider "kubectl" {
  host                   = yamldecode(talos_cluster_kubeconfig.this.kubeconfig_raw).clusters[0].cluster.server
  client_certificate     = base64decode(yamldecode(talos_cluster_kubeconfig.this.kubeconfig_raw).users[0].user.client-certificate-data)
  client_key             = base64decode(yamldecode(talos_cluster_kubeconfig.this.kubeconfig_raw).users[0].user.client-key-data)
  cluster_ca_certificate = base64decode(yamldecode(talos_cluster_kubeconfig.this.kubeconfig_raw).clusters[0].cluster.certificate-authority-data)
}

# =============================================================================
# Data Sources
# =============================================================================

data "hcloud_network" "kibaship_staging" {
  name = "kibaship-staging-network"
}

data "hcloud_image" "talos" {
  with_selector = "os=talos"
  most_recent   = true
}

# =============================================================================
# Network Configuration
# =============================================================================

locals {
  cluster_name          = "kibaship-staging"
  cluster_domain        = "cluster.local"
  cluster_api_host      = "staging.k8s.kibaship.com"
  cluster_api_port      = "6443"
  cluster_endpoint      = "https://${local.cluster_api_host}:${local.cluster_api_port}"
  network_ipv4_cidr     = "10.0.0.0/16"
  node_ipv4_cidr        = "10.0.1.0/24"
  pod_ipv4_cidr         = "10.0.16.0/20"
  service_ipv4_cidr     = "10.0.8.0/21"

  control_plane_ips = [
    "10.0.1.10",
    "10.0.1.11",
    "10.0.1.12"
  ]

  worker_ips = [
    "10.0.1.20",
    "10.0.1.21",
    "10.0.1.22"
  ]

  control_plane_public_ipv4_list = [
    for i in range(3) : hcloud_server.control_planes[i].ipv4_address
  ]
  worker_public_ipv4_list = [
    for i in range(3) : hcloud_server.workers[i].ipv4_address
  ]

  cert_SANs = distinct(
    concat(
      local.control_plane_ips,               # All control plane private IPs
      [
        local.cluster_api_host,               # Load balancer DNS name
        "127.0.0.1",                         # Localhost
        "kubernetes",                        # Service name
        "kubernetes.default",                # Service FQDN
        "kubernetes.default.svc",            # Service FQDN
        "kubernetes.default.svc.cluster.local" # Full service FQDN
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
  cluster_name         = local.cluster_name
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
  count       = 3
  name        = "${local.cluster_name}-control-plane-${count.index + 1}"
  image       = data.hcloud_image.talos.id
  server_type = "cx22"
  location    = "nbg1"
  user_data   = data.talos_machine_configuration.control_plane[count.index].machine_configuration

  labels = {
    environment = "staging"
    cluster     = local.cluster_name
    role        = "control-plane"
  }

  public_net {
    ipv4_enabled = true
    ipv6_enabled = false
  }

  network {
    network_id = data.hcloud_network.kibaship_staging.id
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
  count       = 3
  name        = "${local.cluster_name}-worker-${count.index + 1}"
  image       = data.hcloud_image.talos.id
  server_type = "cx22"
  location    = "nbg1"
  user_data   = data.talos_machine_configuration.worker[count.index].machine_configuration

  labels = {
    environment = "staging"
    cluster     = local.cluster_name
    role        = "worker"
  }

  public_net {
    ipv4_enabled = true
    ipv6_enabled = false
  }

  network {
    network_id = data.hcloud_network.kibaship_staging.id
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
  count              = 3
  cluster_name       = local.cluster_name
  cluster_endpoint   = local.cluster_endpoint
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
            forwardKubeDNSToHost = true
            resolveMemberNames   = true
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
          disabled = true
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
        }
      }
    })
  ]
}

# =============================================================================
# Talos Machine Configuration - Worker
# =============================================================================

data "talos_machine_configuration" "worker" {
  count              = 3
  cluster_name       = local.cluster_name
  cluster_endpoint   = local.cluster_endpoint
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
            forwardKubeDNSToHost = true
            resolveMemberNames   = true
          }
        }
        nodeLabels = {
          "openebs.io/engine" = "mayastor"
        }
      }
    })
  ]
}

# =============================================================================
# Cluster Bootstrap
# =============================================================================

resource "talos_machine_bootstrap" "this" {
  depends_on = [
    talos_machine_configuration_apply.control_plane
  ]
  client_configuration = talos_machine_secrets.this.client_configuration
  node                 = local.control_plane_public_ipv4_list[0]
  endpoint             = local.control_plane_public_ipv4_list[0]
}

# Wait for cluster to be fully ready before extracting kubeconfig
resource "talos_cluster_kubeconfig" "this" {
  depends_on = [
    talos_machine_bootstrap.this
  ]
  client_configuration = talos_machine_secrets.this.client_configuration
  node                 = local.control_plane_public_ipv4_list[0]
  endpoint             = local.control_plane_public_ipv4_list[0]
}

# Check cluster health using HTTP endpoint with proper certificate validation
data "http" "talos_health" {
  url = "https://${local.cluster_api_host}:${local.cluster_api_port}/version"

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

# Generate Cilium manifests only after cluster is ready
data "helm_template" "cilium" {
  depends_on = [
    data.http.talos_health
  ]

  name         = "cilium"
  namespace    = "kube-system"
  repository   = "https://helm.cilium.io"
  chart        = "cilium"
  version      = var.cilium_version
  kube_version = var.kubernetes_version

  # Skip Kubernetes version validation since we're generating templates
  skip_crds = false

  values = [
    yamlencode({
      operator = {
        replicas = 2
      }
      ipam = {
        mode = "kubernetes"
      }
      routingMode = "native"
      ipv4NativeRoutingCIDR = local.pod_ipv4_cidr
      kubeProxyReplacement = true
      bpf = {
        masquerade = false
      }
      loadBalancer = {
        acceleration = "native"
      }
      encryption = {
        enabled = false
      }
      securityContext = {
        capabilities = {
          ciliumAgent = [
            "CHOWN", "KILL", "NET_ADMIN", "NET_RAW", "IPC_LOCK",
            "SYS_ADMIN", "SYS_RESOURCE", "DAC_OVERRIDE", "FOWNER",
            "SETGID", "SETUID"
          ]
          cleanCiliumState = ["NET_ADMIN", "SYS_ADMIN", "SYS_RESOURCE"]
        }
      }
      cgroup = {
        autoMount = {
          enabled = false
        }
        hostRoot = "/sys/fs/cgroup"
      }
      k8sServiceHost = local.cluster_api_host
      k8sServicePort = local.cluster_api_port
      hubble = {
        enabled = false
      }
    })
  ]
}

# =============================================================================
# Cluster Readiness Verification
# =============================================================================
# Note: Mayastor configuration is prepared but not deployed in this bootstrap.
# The cluster is Mayastor-ready with:
# - Huge pages configured (vm.nr_hugepages = 1024)
# - Node labels (openebs.io/engine = mayastor)
# - Extra mounts (/var/local with rshared)
# - Pod security exemptions for openebs namespace
# Deploy Mayastor separately after cluster is healthy.

# =============================================================================
# Cilium Deployment
# =============================================================================

resource "talos_machine_configuration_apply" "control_plane" {
  count                       = 3
  client_configuration        = talos_machine_secrets.this.client_configuration
  machine_configuration_input = data.talos_machine_configuration.control_plane[count.index].machine_configuration
  node                        = local.control_plane_public_ipv4_list[count.index]
  endpoint                    = local.control_plane_public_ipv4_list[count.index]

  depends_on = [hcloud_server.control_planes]
}

resource "talos_machine_configuration_apply" "worker" {
  count                       = 3
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
# Cilium Installation
# =============================================================================

data "kubectl_file_documents" "cilium" {
  content = data.helm_template.cilium.manifest
}

# Deploy Cilium only after cluster is fully bootstrapped
resource "kubectl_manifest" "apply_cilium" {
  for_each   = data.kubectl_file_documents.cilium.manifests
  yaml_body  = each.value
  apply_only = true
  depends_on = [data.http.talos_health]
}

# =============================================================================
# Outputs
# =============================================================================

output "cluster_info" {
  description = "Kubernetes cluster information"
  value = {
    name             = local.cluster_name
    endpoint         = local.cluster_endpoint
    kubernetes_version = var.kubernetes_version
    talos_version    = var.talos_version
    cilium_version   = var.cilium_version
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

output "cluster_summary" {
  description = "Complete cluster deployment summary"
  value = {
    cluster = {
      name     = local.cluster_name
      endpoint = local.cluster_endpoint
      network  = {
        cidr         = local.network_ipv4_cidr
        nodes        = local.node_ipv4_cidr
        pods         = local.pod_ipv4_cidr
        services     = local.service_ipv4_cidr
      }
    }
    servers = {
      control_planes = [
        for i, server in hcloud_server.control_planes : {
          name       = server.name
          public_ip  = server.ipv4_address
          private_ip = local.control_plane_ips[i]
        }
      ]
      workers = [
        for i, server in hcloud_server.workers : {
          name       = server.name
          public_ip  = server.ipv4_address
          private_ip = local.worker_ips[i]
        }
      ]
    }
    load_balancers = {
      k8s_api = "staging.k8s.kibaship.com:6443"
      apps    = "*.staging.kibaship.app:80/443"
    }
  }
}