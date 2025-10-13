# DigitalOcean Kubernetes Cluster Provisioning
# This template provisions a Kubernetes cluster on DigitalOcean

terraform {
  required_version = ">= 1.0"
  
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.67.0"
    }
  }
  
  backend "s3" {
    # Backend configuration will be provided via terraform init -backend-config
    # or via environment variables
  }
}

# Configure the DigitalOcean Provider
provider "digitalocean" {
  token = var.do_token
}

# Data source for available Kubernetes versions
data "digitalocean_kubernetes_versions" "kubernetes_version" {}

# Create the Kubernetes cluster
resource "digitalocean_kubernetes_cluster" "cluster" {
  name    = var.cluster_name
  region  = var.do_region
  version = data.digitalocean_kubernetes_versions.kubernetes_version.latest_version

  node_pool {
    name       = "${var.cluster_name}-default-pool"
    size       = var.do_node_size
    node_count = var.do_node_count
  }
}

# Local variables for PaaS feature detection
locals {
  paas_features_list = split(",", var.paas_features)
  has_mysql         = contains(local.paas_features_list, "mysql")
  has_postgres      = contains(local.paas_features_list, "postgres")
  has_valkey        = contains(local.paas_features_list, "valkey")
}

# Create Load Balancer for cluster services
resource "digitalocean_loadbalancer" "cluster_lb" {
  name   = "${var.cluster_name}-lb"
  region = var.do_region

  # Forward HTTP traffic (always enabled)
  forwarding_rule {
    entry_protocol  = "http"
    entry_port      = 80
    target_protocol = "tcp"
    target_port     = 30080
  }

  # Forward HTTPS traffic with TLS passthrough (always enabled)
  forwarding_rule {
    entry_protocol  = "https"
    entry_port      = 443
    target_protocol = "tcp"
    target_port     = 30443
    tls_passthrough = true
  }

  # Forward DNS over TLS (always enabled)
  forwarding_rule {
    entry_protocol  = "tcp"
    entry_port      = 53
    target_protocol = "tcp"
    target_port     = 30053
    tls_passthrough = false
  }

  # Forward DNS over UDP (always enabled)
  forwarding_rule {
    entry_protocol  = "udp"
    entry_port      = 53
    target_protocol = "udp"
    target_port     = 30053
    tls_passthrough = false
  }

  # Conditional forwarding rules based on PaaS features

  # PostgreSQL forwarding rule (only if postgres feature is enabled)
  dynamic "forwarding_rule" {
    for_each = local.has_postgres ? [1] : []
    content {
      entry_protocol  = "tcp"
      entry_port      = 5432
      target_protocol = "tcp"
      target_port     = 30432
      tls_passthrough = true
    }
  }

  # MySQL forwarding rule (only if mysql feature is enabled)
  dynamic "forwarding_rule" {
    for_each = local.has_mysql ? [1] : []
    content {
      entry_protocol  = "tcp"
      entry_port      = 3306
      target_protocol = "tcp"
      target_port     = 30306
      tls_passthrough = true
    }
  }

  # Valkey/Redis forwarding rule (only if valkey feature is enabled)
  dynamic "forwarding_rule" {
    for_each = local.has_valkey ? [1] : []
    content {
      entry_protocol  = "tcp"
      entry_port      = 6379
      target_protocol = "tcp"
      target_port     = 30379
      tls_passthrough = true
    }
  }

  # Health check configuration
  healthcheck {
    protocol               = "tcp"
    port                   = 30080
    check_interval_seconds = 10
    response_timeout_seconds = 5
    unhealthy_threshold    = 3
    healthy_threshold      = 2
  }

  droplet_tag = "${var.cluster_name}-cluster"
}

# Output important cluster information
output "cluster_id" {
  description = "ID of the Kubernetes cluster"
  value       = digitalocean_kubernetes_cluster.cluster.id
}

output "cluster_endpoint" {
  description = "Endpoint for the Kubernetes cluster"
  value       = digitalocean_kubernetes_cluster.cluster.endpoint
  sensitive   = true
}

output "cluster_status" {
  description = "Status of the Kubernetes cluster"
  value       = digitalocean_kubernetes_cluster.cluster.status
}

output "cluster_version" {
  description = "Version of the Kubernetes cluster"
  value       = digitalocean_kubernetes_cluster.cluster.version
}

output "kubeconfig" {
  description = "Kubernetes configuration for cluster access"
  value       = digitalocean_kubernetes_cluster.cluster.kube_config[0].raw_config
  sensitive   = true
}

# Load Balancer outputs
output "load_balancer_id" {
  description = "ID of the load balancer"
  value       = digitalocean_loadbalancer.cluster_lb.id
}

output "load_balancer_ip" {
  description = "Public IP address of the load balancer"
  value       = digitalocean_loadbalancer.cluster_lb.ip
}

output "load_balancer_status" {
  description = "Status of the load balancer"
  value       = digitalocean_loadbalancer.cluster_lb.status
}

output "load_balancer_forwarding_rules" {
  description = "Active forwarding rules based on enabled PaaS features"
  value = {
    http_80    = "80 -> 30080 (HTTP)"
    https_443  = "443 -> 30443 (HTTPS TLS Passthrough)"
    dns_tcp_53 = "53 -> 30053 (DNS over TCP)"
    dns_udp_53 = "53 -> 30053 (DNS over UDP)"
    mysql_3306 = local.has_mysql ? "3306 -> 30306 (MySQL TLS Passthrough)" : "disabled"
    postgres_5432 = local.has_postgres ? "5432 -> 30432 (PostgreSQL TLS Passthrough)" : "disabled"
    valkey_6379 = local.has_valkey ? "6379 -> 30379 (Valkey/Redis TLS Passthrough)" : "disabled"
  }
}
