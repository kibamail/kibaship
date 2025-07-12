# =============================================================================
# KibaShip Staging Kubernetes Cluster Infrastructure
# =============================================================================

terraform {
  required_version = ">= 1.0"
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.45"
    }
  }
}

# =============================================================================
# Provider Configuration
# =============================================================================

provider "hcloud" {}

# =============================================================================
# Variables
# =============================================================================

variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
  default     = "kibaship-staging"
}

variable "location" {
  description = "Hetzner Cloud location"
  type        = string
  default     = "nbg1"
}

variable "network_zone" {
  description = "Hetzner Cloud network zone"
  type        = string
  default     = "eu-central"
}

variable "domain_name" {
  description = "Domain name for the Kubernetes API"
  type        = string
  default     = "staging.k8s.kibaship.com"
}

# =============================================================================
# SSH Key
# =============================================================================

resource "hcloud_ssh_key" "cluster_key" {
  name       = "${var.cluster_name}-key"
  public_key = file("~/.ssh/id_rsa.pub")
}

# =============================================================================
# Private Network
# =============================================================================

resource "hcloud_network" "cluster_network" {
  name     = "${var.cluster_name}-network"
  ip_range = "10.0.0.0/16"
  labels = {
    cluster = var.cluster_name
    purpose = "kubernetes"
  }
}

resource "hcloud_network_subnet" "cluster_subnet" {
  type         = "cloud"
  network_id   = hcloud_network.cluster_network.id
  network_zone = var.network_zone
  ip_range     = "10.0.1.0/24"
}

# =============================================================================
# Control Plane Servers
# =============================================================================

resource "hcloud_server" "control_plane" {
  count       = 3
  name        = "${var.cluster_name}-control-${count.index + 1}"
  image       = "ubuntu-24.04"
  server_type = "cx22"
  location    = var.location
  ssh_keys    = [hcloud_ssh_key.cluster_key.id]

  network {
    network_id = hcloud_network.cluster_network.id
    ip         = "10.0.1.${10 + count.index}"
  }

  labels = {
    cluster = var.cluster_name
    role    = "control-plane"
    type    = "kubernetes"
  }

  depends_on = [hcloud_network_subnet.cluster_subnet]
}

# =============================================================================
# Worker Nodes
# =============================================================================

resource "hcloud_server" "worker" {
  count       = 3
  name        = "${var.cluster_name}-worker-${count.index + 1}"
  image       = "ubuntu-24.04"
  server_type = "cx22"
  location    = var.location
  ssh_keys    = [hcloud_ssh_key.cluster_key.id]

  network {
    network_id = hcloud_network.cluster_network.id
    ip         = "10.0.1.${20 + count.index}"
  }

  labels = {
    cluster = var.cluster_name
    role    = "worker"
    type    = "kubernetes"
  }

  depends_on = [hcloud_network_subnet.cluster_subnet]
}

# =============================================================================
# Load Balancer for Kubernetes API
# =============================================================================

resource "hcloud_load_balancer" "k8s_api_lb" {
  name               = "${var.cluster_name}-k8s-api-lb"
  load_balancer_type = "lb11"
  location           = var.location

  labels = {
    cluster = var.cluster_name
    purpose = "kubernetes-api"
  }
}

resource "hcloud_load_balancer_network" "k8s_api_lb_network" {
  load_balancer_id = hcloud_load_balancer.k8s_api_lb.id
  network_id       = hcloud_network.cluster_network.id
  ip               = "10.0.1.100"
}

resource "hcloud_load_balancer_target" "k8s_api_lb_target" {
  load_balancer_id = hcloud_load_balancer.k8s_api_lb.id
  type             = "label_selector"
  label_selector   = "role=control-plane"
  use_private_ip   = true

  depends_on = [hcloud_load_balancer_network.k8s_api_lb_network]
}

resource "hcloud_load_balancer_service" "k8s_api_lb_service" {
  load_balancer_id = hcloud_load_balancer.k8s_api_lb.id
  protocol         = "https"
  listen_port      = 80
  destination_port = 6443

  http {
    certificates = [hcloud_managed_certificate.k8s_api_cert.id]
  }

  health_check {
    protocol = "tcp"
    port     = 6443
    interval = 10
    timeout  = 5
    retries  = 3
  }
}

# =============================================================================
# Load Balancer for Application Traffic
# =============================================================================

resource "hcloud_load_balancer" "app_lb" {
  name               = "${var.cluster_name}-app-lb"
  load_balancer_type = "lb11"
  location           = var.location

  labels = {
    cluster = var.cluster_name
    purpose = "application-traffic"
  }
}

resource "hcloud_load_balancer_network" "app_lb_network" {
  load_balancer_id = hcloud_load_balancer.app_lb.id
  network_id       = hcloud_network.cluster_network.id
  ip               = "10.0.1.101"
}

resource "hcloud_load_balancer_target" "app_lb_target" {
  load_balancer_id = hcloud_load_balancer.app_lb.id
  type             = "label_selector"
  label_selector   = "role=worker"
  use_private_ip   = true

  depends_on = [hcloud_load_balancer_network.app_lb_network]
}

resource "hcloud_load_balancer_service" "app_lb_http" {
  load_balancer_id = hcloud_load_balancer.app_lb.id
  protocol         = "http"
  listen_port      = 80
  destination_port = 30080

  health_check {
    protocol = "http"
    port     = 30080
    interval = 10
    timeout  = 5
    retries  = 3

    http {
      path         = "/"
      status_codes = ["200", "404"]
    }
  }
}

resource "hcloud_load_balancer_service" "app_lb_https" {
  load_balancer_id = hcloud_load_balancer.app_lb.id
  protocol         = "https"
  listen_port      = 443
  destination_port = 30443

  http {
    certificates = [hcloud_managed_certificate.app_cert.id]
  }

  health_check {
    protocol = "http"
    port     = 30080
    interval = 10
    timeout  = 5
    retries  = 3

    http {
      path         = "/"
      status_codes = ["200", "404"]
    }
  }
}



# =============================================================================
# SSL Certificates
# =============================================================================

resource "hcloud_managed_certificate" "k8s_api_cert" {
  name         = "${var.cluster_name}-k8s-api-cert"
  domain_names = [var.domain_name]

  labels = {
    cluster = var.cluster_name
    purpose = "kubernetes-api-ssl"
  }
}

resource "hcloud_managed_certificate" "app_cert" {
  name         = "${var.cluster_name}-app-cert"
  domain_names = ["*.staging.kibaship.app"]

  labels = {
    cluster = var.cluster_name
    purpose = "application-ssl"
  }
}



# =============================================================================
# Outputs
# =============================================================================

output "control_plane_servers" {
  description = "Control plane server details"
  value = {
    for i, server in hcloud_server.control_plane :
    "control-${i + 1}" => {
      name       = server.name
      public_ip  = server.ipv4_address
      private_ip = tolist(server.network)[0].ip
      id         = server.id
    }
  }
}

output "worker_servers" {
  description = "Worker server details"
  value = {
    for i, server in hcloud_server.worker :
    "worker-${i + 1}" => {
      name       = server.name
      public_ip  = server.ipv4_address
      private_ip = tolist(server.network)[0].ip
      id         = server.id
    }
  }
}

output "k8s_api_load_balancer" {
  description = "Kubernetes API load balancer details"
  value = {
    name       = hcloud_load_balancer.k8s_api_lb.name
    public_ip  = hcloud_load_balancer.k8s_api_lb.ipv4
    private_ip = hcloud_load_balancer_network.k8s_api_lb_network.ip
    id         = hcloud_load_balancer.k8s_api_lb.id
    endpoint   = var.domain_name
  }
}

output "app_load_balancer" {
  description = "Application load balancer details"
  value = {
    name       = hcloud_load_balancer.app_lb.name
    public_ip  = hcloud_load_balancer.app_lb.ipv4
    private_ip = hcloud_load_balancer_network.app_lb_network.ip
    id         = hcloud_load_balancer.app_lb.id
  }
}

output "network_details" {
  description = "Network configuration details"
  value = {
    network_id   = hcloud_network.cluster_network.id
    network_name = hcloud_network.cluster_network.name
    ip_range     = hcloud_network.cluster_network.ip_range
    subnet_range = hcloud_network_subnet.cluster_subnet.ip_range
  }
}

output "ssl_certificates" {
  description = "SSL certificate details"
  value = {
    k8s_api = {
      id           = hcloud_managed_certificate.k8s_api_cert.id
      name         = hcloud_managed_certificate.k8s_api_cert.name
      domain_names = hcloud_managed_certificate.k8s_api_cert.domain_names
      fingerprint  = hcloud_managed_certificate.k8s_api_cert.fingerprint
    }
    app = {
      id           = hcloud_managed_certificate.app_cert.id
      name         = hcloud_managed_certificate.app_cert.name
      domain_names = hcloud_managed_certificate.app_cert.domain_names
      fingerprint  = hcloud_managed_certificate.app_cert.fingerprint
    }
  }
}

# Summary output for easy reference
output "cluster_summary" {
  description = "Complete cluster summary"
  value = {
    cluster_name = var.cluster_name
    location     = var.location

    control_plane_ips = [
      for server in hcloud_server.control_plane : server.ipv4_address
    ]

    worker_ips = [
      for server in hcloud_server.worker : server.ipv4_address
    ]

    k8s_api_endpoint = var.domain_name
    k8s_lb_ip       = hcloud_load_balancer.k8s_api_lb.ipv4
    app_lb_ip       = hcloud_load_balancer.app_lb.ipv4

    ssh_command_example = "ssh root@${hcloud_server.control_plane[0].ipv4_address}"
  }
}