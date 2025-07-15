# =============================================================================
# KibaShip Staging Load Balancers Configuration
# =============================================================================
# This configuration provisions load balancers for the KibaShip staging
# environment on Hetzner Cloud, including:
# - Kubernetes API Load Balancer (staging.k8s.kibaship.com:6443)
# - Application Load Balancer (*.staging.kibaship.app:80/443)

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

variable "hcloud_token" {
  description = "Hetzner Cloud API Token"
  type        = string
  sensitive   = true
}

# =============================================================================
# Provider Configuration
# =============================================================================

provider "hcloud" {
  token = var.hcloud_token
}

# =============================================================================
# Network Infrastructure
# =============================================================================

# Private network for load balancers and servers
resource "hcloud_network" "kibaship_staging" {
  name     = "kibaship-staging-network"
  ip_range = "10.0.0.0/16"

  labels = {
    environment = "staging"
    cluster     = "kibaship-staging"
  }
}

# Subnet for the staging environment
resource "hcloud_network_subnet" "kibaship_staging_subnet" {
  network_id   = hcloud_network.kibaship_staging.id
  type         = "cloud"
  network_zone = "eu-central"
  ip_range     = "10.0.1.0/24"
}

# =============================================================================
# Kubernetes API Load Balancer
# =============================================================================
# Routes traffic to Kubernetes API servers on control plane nodes
# Domain: staging.k8s.kibaship.com
# Port: 6443 (TCP passthrough)

resource "hcloud_load_balancer" "k8s_api" {
  name               = "kibaship-staging-k8s-api"
  load_balancer_type = "lb11"
  location           = "nbg1"

  labels = {
    environment = "staging"
    purpose     = "kubernetes-api"
    cluster     = "kibaship-staging"
  }
}

resource "hcloud_load_balancer_network" "k8s_api_network" {
  load_balancer_id = hcloud_load_balancer.k8s_api.id
  network_id       = hcloud_network.kibaship_staging.id
  ip               = "10.0.1.100"

  depends_on = [hcloud_network_subnet.kibaship_staging_subnet]
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

  depends_on = [
    hcloud_load_balancer_service.k8s_api_service,
    hcloud_load_balancer_network.k8s_api_network
  ]
}

# =============================================================================
# Application Load Balancer
# =============================================================================
# Routes application traffic to worker nodes via NodePort services
# Domain: *.staging.kibaship.app
# Ports: 80->30080 (HTTP), 443->30443 (HTTPS)

resource "hcloud_load_balancer" "app" {
  name               = "kibaship-staging-app"
  load_balancer_type = "lb11"
  location           = "nbg1"

  labels = {
    environment = "staging"
    purpose     = "application"
    cluster     = "kibaship-staging"
  }
}

resource "hcloud_load_balancer_network" "app_network" {
  load_balancer_id = hcloud_load_balancer.app.id
  network_id       = hcloud_network.kibaship_staging.id
  ip               = "10.0.1.101"

  depends_on = [hcloud_network_subnet.kibaship_staging_subnet]
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

  depends_on = [
    hcloud_load_balancer_service.app_http_service,
    hcloud_load_balancer_service.app_https_service,
    hcloud_load_balancer_network.app_network
  ]
}

# =============================================================================
# Outputs
# =============================================================================

output "network" {
  description = "Private network details"
  value = {
    id       = hcloud_network.kibaship_staging.id
    name     = hcloud_network.kibaship_staging.name
    ip_range = hcloud_network.kibaship_staging.ip_range
    subnet = {
      id       = hcloud_network_subnet.kibaship_staging_subnet.id
      ip_range = hcloud_network_subnet.kibaship_staging_subnet.ip_range
    }
  }
}

output "k8s_api_load_balancer" {
  description = "Kubernetes API Load Balancer details"
  value = {
    id          = hcloud_load_balancer.k8s_api.id
    name        = hcloud_load_balancer.k8s_api.name
    ipv4        = hcloud_load_balancer.k8s_api.ipv4
    ipv6        = hcloud_load_balancer.k8s_api.ipv6
    private_ip  = hcloud_load_balancer_network.k8s_api_network.ip
    domain      = "staging.k8s.kibaship.com"
    port        = 6443
    targets     = "servers with label role=control-plane"
    network_id  = hcloud_network.kibaship_staging.id
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
    network_id  = hcloud_network.kibaship_staging.id
  }
}

output "load_balancer_summary" {
  description = "Summary of all load balancers"
  value = {
    network = {
      name     = hcloud_network.kibaship_staging.name
      ip_range = hcloud_network.kibaship_staging.ip_range
    }
    k8s_api = {
      name       = hcloud_load_balancer.k8s_api.name
      public_ip  = hcloud_load_balancer.k8s_api.ipv4
      private_ip = hcloud_load_balancer_network.k8s_api_network.ip
      domain     = "staging.k8s.kibaship.com"
      purpose    = "Kubernetes API (port 6443)"
    }
    app = {
      name       = hcloud_load_balancer.app.name
      public_ip  = hcloud_load_balancer.app.ipv4
      private_ip = hcloud_load_balancer_network.app_network.ip
      domain     = "*.staging.kibaship.app"
      purpose    = "Application traffic (ports 80/443 -> 30080/30443)"
    }
  }
}