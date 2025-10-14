# Kind (Kubernetes in Docker) Cluster Provisioning
# This template provisions a local Kubernetes cluster using Kind

terraform {
  required_version = ">= 1.0"

  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "~> 0.4.0"
    }
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = ">= 1.19.0"
    }
  }

  # Use local backend for Kind clusters - no S3 required
  backend "local" {
    path = "terraform.tfstate"
  }
}

# Configure the Kind Provider
provider "kind" {}

# Configure kubectl provider to use the Kind cluster
provider "kubectl" {
  host                   = kind_cluster.cluster.endpoint
  cluster_ca_certificate = kind_cluster.cluster.cluster_ca_certificate
  client_certificate     = kind_cluster.cluster.client_certificate
  client_key             = kind_cluster.cluster.client_key
  load_config_file       = false
}

# Local variables for PaaS feature detection
locals {
  paas_features_list = split(",", var.paas_features)
  has_mysql         = contains(local.paas_features_list, "mysql")
  has_postgres      = contains(local.paas_features_list, "postgres")
  has_valkey        = contains(local.paas_features_list, "valkey")
}

# Note: Longhorn storage uses container's internal ext4 filesystem
# No host bind mounts needed - this avoids macOS virtiofs compatibility issues

# Create the Kind cluster
resource "kind_cluster" "cluster" {
  name = var.cluster_name

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"

    # Disable default CNI and kube-proxy for Cilium installation
    networking {
      disable_default_cni   = true
      kube_proxy_mode      = "none"
    }

    # Control plane node
    node {
      role  = "control-plane"
      image = "kindest/node:${var.kind_k8s_version}"

      # Port mappings for services (using 140xx prefix to avoid conflicts)
      extra_port_mappings {
        container_port = 30080
        host_port      = 14080
        protocol       = "TCP"
      }

      extra_port_mappings {
        container_port = 30443
        host_port      = 14443
        protocol       = "TCP"
      }

      extra_port_mappings {
        container_port = 30053
        host_port      = 14053
        protocol       = "TCP"
      }

      extra_port_mappings {
        container_port = 30053
        host_port      = 14053
        protocol       = "UDP"
      }
      
      # Conditional port mappings based on PaaS features
      
      # MySQL port mapping (only if mysql feature is enabled)
      dynamic "extra_port_mappings" {
        for_each = local.has_mysql ? [1] : []
        content {
          container_port = 30306
          host_port      = 14306
          protocol       = "TCP"
        }
      }

      # PostgreSQL port mapping (only if postgres feature is enabled)
      dynamic "extra_port_mappings" {
        for_each = local.has_postgres ? [1] : []
        content {
          container_port = 30432
          host_port      = 14432
          protocol       = "TCP"
        }
      }

      # Valkey/Redis port mapping (only if valkey feature is enabled)
      dynamic "extra_port_mappings" {
        for_each = local.has_valkey ? [1] : []
        content {
          container_port = 30379
          host_port      = 14379
          protocol       = "TCP"
        }
      }
    }
    
    # Worker nodes (if more than 1 node requested)
    dynamic "node" {
      for_each = range(max(0, var.kind_node_count - 1))
      content {
        role  = "worker"
        image = "kindest/node:${var.kind_k8s_version}"
      }
    }
  }
  
  # Wait for cluster to be ready
  wait_for_ready = true
}

# Install Gateway API CRDs before Cilium
# These CRDs are required for Cilium Gateway API support

data "kubectl_file_documents" "gateway_crds_gatewayclasses" {
  content = file("https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_gatewayclasses.yaml")
}

data "kubectl_file_documents" "gateway_crds_gateways" {
  content = file("https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_gateways.yaml")
}

data "kubectl_file_documents" "gateway_crds_httproutes" {
  content = file("https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_httproutes.yaml")
}

data "kubectl_file_documents" "gateway_crds_referencegrants" {
  content = file("https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_referencegrants.yaml")
}

data "kubectl_file_documents" "gateway_crds_grpcroutes" {
  content = file("https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_grpcroutes.yaml")
}

data "kubectl_file_documents" "gateway_crds_tlsroutes" {
  content = file("https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/experimental/gateway.networking.k8s.io_tlsroutes.yaml")
}

resource "kubectl_manifest" "gateway_crd_gatewayclasses" {
  depends_on = [kind_cluster.cluster]
  for_each   = data.kubectl_file_documents.gateway_crds_gatewayclasses.manifests
  yaml_body  = each.value
}

resource "kubectl_manifest" "gateway_crd_gateways" {
  depends_on = [kind_cluster.cluster]
  for_each   = data.kubectl_file_documents.gateway_crds_gateways.manifests
  yaml_body  = each.value
}

resource "kubectl_manifest" "gateway_crd_httproutes" {
  depends_on = [kind_cluster.cluster]
  for_each   = data.kubectl_file_documents.gateway_crds_httproutes.manifests
  yaml_body  = each.value
}

resource "kubectl_manifest" "gateway_crd_referencegrants" {
  depends_on = [kind_cluster.cluster]
  for_each   = data.kubectl_file_documents.gateway_crds_referencegrants.manifests
  yaml_body  = each.value
}

resource "kubectl_manifest" "gateway_crd_grpcroutes" {
  depends_on = [kind_cluster.cluster]
  for_each   = data.kubectl_file_documents.gateway_crds_grpcroutes.manifests
  yaml_body  = each.value
}

resource "kubectl_manifest" "gateway_crd_tlsroutes" {
  depends_on = [kind_cluster.cluster]
  for_each   = data.kubectl_file_documents.gateway_crds_tlsroutes.manifests
  yaml_body  = each.value
}

# Output important cluster information
output "cluster_name" {
  description = "Name of the Kind cluster"
  value       = kind_cluster.cluster.name
}

output "cluster_endpoint" {
  description = "Endpoint for the Kubernetes cluster"
  value       = kind_cluster.cluster.endpoint
  sensitive   = true
}

output "cluster_client_certificate" {
  description = "Client certificate for cluster access"
  value       = kind_cluster.cluster.client_certificate
  sensitive   = true
}

output "cluster_client_key" {
  description = "Client key for cluster access"
  value       = kind_cluster.cluster.client_key
  sensitive   = true
}

output "cluster_ca_certificate" {
  description = "CA certificate for cluster access"
  value       = kind_cluster.cluster.cluster_ca_certificate
  sensitive   = true
}

output "kubeconfig" {
  description = "Kubernetes configuration for cluster access"
  value       = kind_cluster.cluster.kubeconfig
  sensitive   = true
}

# Kind cluster information
output "kind_cluster_info" {
  description = "Kind cluster information and port mappings"
  value = {
    cluster_name = kind_cluster.cluster.name
    node_count   = var.kind_node_count
    endpoint     = "https://localhost:${kind_cluster.cluster.endpoint}"
    docker_network = "kind"
    cni_disabled = true
    kube_proxy_disabled = true
    cilium_ready = "Cluster is ready for Cilium installation"
    storage_enabled = true
    storage_type = "container-internal"
    storage_info = "Longhorn uses container's internal ext4 filesystem (no host bind mounts)"
    storage_note = "Storage persists across pod restarts but not cluster recreation"
  }
}

output "port_mappings" {
  description = "Active port mappings based on enabled PaaS features"
  value = {
    http_14080    = "localhost:14080 -> cluster:30080 (HTTP)"
    https_14443   = "localhost:14443 -> cluster:30443 (HTTPS)"
    dns_tcp_14053 = "localhost:14053 -> cluster:30053 (DNS over TCP)"
    dns_udp_14053 = "localhost:14053 -> cluster:30053 (DNS over UDP)"
    mysql_14306   = local.has_mysql ? "localhost:14306 -> cluster:30306 (MySQL)" : "disabled"
    postgres_14432 = local.has_postgres ? "localhost:14432 -> cluster:30432 (PostgreSQL)" : "disabled"
    valkey_14379  = local.has_valkey ? "localhost:14379 -> cluster:30379 (Valkey/Redis)" : "disabled"
  }
}
