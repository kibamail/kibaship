# =============================================================================
# Load Balancers Module
# =============================================================================
# This module provisions load balancers for the KibaShip staging environment,
# including:
# - Kubernetes API Load Balancer (public IP access on port 6443)
# - Application Load Balancer (*.staging.kibaship.app:80/443)
# - Network attachments and health checks

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

variable "network_id" {
  description = "ID of the private network"
  type        = string
}

variable "location" {
  description = "Hetzner Cloud location"
  type        = string
  default     = "nbg1"
}

variable "load_balancer_type" {
  description = "Type of load balancer"
  type        = string
  default     = "lb11"
}

variable "k8s_api_private_ip" {
  description = "Private IP for Kubernetes API load balancer"
  type        = string
  default     = "10.0.1.100"
}

variable "app_private_ip" {
  description = "Private IP for application load balancer"
  type        = string
  default     = "10.0.1.101"
}

# =============================================================================
# Kubernetes API Load Balancer
# =============================================================================

resource "hcloud_load_balancer" "k8s_api" {
  name               = "${var.cluster_name}-k8s-api"
  load_balancer_type = var.load_balancer_type
  location           = var.location

  labels = {
    environment = var.environment
    purpose     = "kubernetes-api"
    cluster     = var.cluster_name
  }
}

resource "hcloud_load_balancer_network" "k8s_api_network" {
  load_balancer_id = hcloud_load_balancer.k8s_api.id
  network_id       = var.network_id
  ip               = var.k8s_api_private_ip
}

resource "hcloud_load_balancer_service" "k8s_api_service" {
  load_balancer_id = hcloud_load_balancer.k8s_api.id
  protocol         = "tcp"
  listen_port      = 6443
  destination_port = 6443

  health_check {
    protocol = "tcp"
    port     = 6443
    interval = 10
    timeout  = 5
    retries  = 3
  }
}

resource "hcloud_load_balancer_target" "k8s_api_targets" {
  type             = "label_selector"
  load_balancer_id = hcloud_load_balancer.k8s_api.id
  label_selector   = "role=control-plane"
  use_private_ip   = true

  depends_on = [hcloud_load_balancer_network.k8s_api_network]
}



# =============================================================================
# Application Load Balancer
# =============================================================================

resource "hcloud_load_balancer" "app" {
  name               = "app.${var.cluster_name}"
  load_balancer_type = var.load_balancer_type
  location           = var.location

  labels = {
    environment = var.environment
    purpose     = "application"
    cluster     = var.cluster_name
  }
}

resource "hcloud_load_balancer_network" "app_network" {
  load_balancer_id = hcloud_load_balancer.app.id
  network_id       = var.network_id
  ip               = var.app_private_ip
}

resource "hcloud_load_balancer_service" "app_http_service" {
  load_balancer_id = hcloud_load_balancer.app.id
  protocol         = "tcp"
  listen_port      = 80
  destination_port = 30080

  health_check {
    protocol = "tcp"
    port     = 30080
    interval = 10
    timeout  = 5
    retries  = 3
  }
}

resource "hcloud_load_balancer_service" "app_https_service" {
  load_balancer_id = hcloud_load_balancer.app.id
  protocol         = "tcp"
  listen_port      = 443
  destination_port = 30443

  health_check {
    protocol = "tcp"
    port     = 30443
    interval = 10
    timeout  = 5
    retries  = 3
  }
}

resource "hcloud_load_balancer_target" "app_targets" {
  type             = "label_selector"
  load_balancer_id = hcloud_load_balancer.app.id
  label_selector   = "role=worker"
  use_private_ip   = true

  depends_on = [hcloud_load_balancer_network.app_network]
}



# =============================================================================
# Outputs
# =============================================================================

output "k8s_api_load_balancer" {
  description = "Kubernetes API Load Balancer details"
  value = {
    id          = hcloud_load_balancer.k8s_api.id
    name        = hcloud_load_balancer.k8s_api.name
    ipv4        = hcloud_load_balancer.k8s_api.ipv4
    ipv6        = hcloud_load_balancer.k8s_api.ipv6
    private_ip  = hcloud_load_balancer_network.k8s_api_network.ip
    port        = 6443
    targets     = "servers with label role=control-plane"
  }
}

output "app_load_balancer" {
  description = "Application Load Balancer details"
  value = {
    id          = hcloud_load_balancer.app.id
    name        = hcloud_load_balancer.app.name
    ipv4        = hcloud_load_balancer.app.ipv4
    ipv6        = hcloud_load_balancer.app.ipv6
    private_ip  = hcloud_load_balancer_network.app_network.ip
    domain      = "*.staging.kibaship.app"
    ports       = "80 -> 30080, 443 -> 30443"
    targets     = "servers with label role=worker"
  }
}

output "k8s_api_public_ip" {
  description = "Public IP of Kubernetes API load balancer"
  value       = hcloud_load_balancer.k8s_api.ipv4
}

output "k8s_api_endpoint" {
  description = "Kubernetes API endpoint URL"
  value       = "https://${hcloud_load_balancer.k8s_api.ipv4}:6443"
}

output "app_public_ip" {
  description = "Public IP of application load balancer"
  value       = hcloud_load_balancer.app.ipv4
}
