terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.52"
    }
  }

  backend "s3" {
    bucket = "{{.TerraformState.S3Bucket}}"
    key    = "clusters/{{.Name}}/bare-metal-cloud-load-balancer/terraform.tfstate"
    region = "{{.TerraformState.S3Region}}"
    encrypt = true
  }
}

provider "hcloud" {
  token = var.hcloud_token
}

# Create Hetzner Cloud network
resource "hcloud_network" "cluster_network" {
  name     = "${var.cluster_name}-network"
  ip_range = var.cluster_network_ip_range
  expose_routes_to_vswitch = true
  labels = {
    cluster    = var.cluster_name
    managed_by = "kibaship"
  }
}

# Create cloud subnet and attach to vSwitch
resource "hcloud_network_subnet" "vswitch_subnet" {
  network_id   = hcloud_network.cluster_network.id
  type         = "vswitch"
  network_zone = var.network_zone
  ip_range     = var.cluster_vswitch_subnet_ip_range
  vswitch_id   = var.vswitch_id
}

# Create load balancer subnet
resource "hcloud_network_subnet" "load_balancer_subnet" {
  network_id   = hcloud_network.cluster_network.id
  type         = "cloud"
  network_zone = var.network_zone
  ip_range     = var.cluster_subnet_ip_range
}

# Create ingress load balancer
resource "hcloud_load_balancer" "ingress" {
  name               = "ingress.${var.cluster_name}"
  load_balancer_type = "lb11"
  location           = var.location

  labels = {
    cluster    = var.cluster_name
    managed_by = "kibaship"
    type       = "ingress"
  }
}

# Attach load balancer to network
resource "hcloud_load_balancer_network" "ingress_network" {
  load_balancer_id = hcloud_load_balancer.ingress.id
  network_id       = hcloud_network.cluster_network.id

  depends_on = [
    hcloud_network_subnet.vswitch_subnet,
    hcloud_network_subnet.load_balancer_subnet
  ]
}

# HTTP service (port 80 -> 30080)
resource "hcloud_load_balancer_service" "ingress_http" {
  load_balancer_id = hcloud_load_balancer.ingress.id
  protocol         = "tcp"
  listen_port      = 80
  destination_port = 30080

  health_check {
    protocol = "tcp"
    port     = 30080
    interval = 15
    timeout  = 10
    retries  = 3
  }
}

# HTTPS service (port 443 -> 30443) - TCP passthrough
resource "hcloud_load_balancer_service" "ingress_https" {
  load_balancer_id = hcloud_load_balancer.ingress.id
  protocol         = "tcp"
  listen_port      = 443
  destination_port = 30443

  health_check {
    protocol = "tcp"
    port     = 30443
    interval = 15
    timeout  = 10
    retries  = 3
  }
}

# Valkey service (port 6379 -> 30379)
resource "hcloud_load_balancer_service" "ingress_valkey" {
  load_balancer_id = hcloud_load_balancer.ingress.id
  protocol         = "tcp"
  listen_port      = 6379
  destination_port = 30379

  health_check {
    protocol = "tcp"
    port     = 30379
    interval = 15
    timeout  = 10
    retries  = 3
  }
}

# MySQL service (port 3306 -> 30306)
resource "hcloud_load_balancer_service" "ingress_mysql" {
  load_balancer_id = hcloud_load_balancer.ingress.id
  protocol         = "tcp"
  listen_port      = 3306
  destination_port = 30306

  health_check {
    protocol = "tcp"
    port     = 30306
    interval = 15
    timeout  = 10
    retries  = 3
  }
}

# PostgreSQL service (port 5432 -> 30432)
resource "hcloud_load_balancer_service" "ingress_postgres" {
  load_balancer_id = hcloud_load_balancer.ingress.id
  protocol         = "tcp"
  listen_port      = 5432
  destination_port = 30432

  health_check {
    protocol = "tcp"
    port     = 30432
    interval = 15
    timeout  = 10
    retries  = 3
  }
}

# Add each server as a target to the ingress load balancer
{{range .HetznerRobot.SelectedServers}}
resource "hcloud_load_balancer_target" "server_{{.ID}}" {
  type             = "ip"
  load_balancer_id = hcloud_load_balancer.ingress.id
  ip               = var.server_{{.ID}}_private_ip

  depends_on = [hcloud_load_balancer_network.ingress_network]
}
{{end}}

# Create Kubernetes API load balancer
resource "hcloud_load_balancer" "kube" {
  name               = "kube.${var.cluster_name}"
  load_balancer_type = "lb11"
  location           = var.location

  labels = {
    cluster    = var.cluster_name
    managed_by = "kibaship"
    type       = "kubernetes-api"
  }
}

# Attach Kubernetes API load balancer to network
resource "hcloud_load_balancer_network" "kube_network" {
  load_balancer_id = hcloud_load_balancer.kube.id
  network_id       = hcloud_network.cluster_network.id

  depends_on = [
    hcloud_network_subnet.vswitch_subnet,
    hcloud_network_subnet.load_balancer_subnet
  ]
}

# Kubernetes API service (port 6443)
resource "hcloud_load_balancer_service" "kube_api" {
  load_balancer_id = hcloud_load_balancer.kube.id
  protocol         = "tcp"
  listen_port      = 6443
  destination_port = 6443

  health_check {
    protocol = "tcp"
    port     = 50000
    interval = 15
    timeout  = 10
    retries  = 3
  }
}

# Add control plane servers as targets to the Kubernetes API load balancer
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "control-plane"}}
resource "hcloud_load_balancer_target" "kube_cp_{{.ID}}" {
  type             = "ip"
  load_balancer_id = hcloud_load_balancer.kube.id
  ip               = var.server_{{.ID}}_private_ip

  depends_on = [hcloud_load_balancer_network.kube_network]
}
{{end}}
{{end}}

# Outputs
output "network_id" {
  description = "ID of the created network"
  value       = hcloud_network.cluster_network.id
}

output "network_name" {
  description = "Name of the created network"
  value       = hcloud_network.cluster_network.name
}

output "network_ip_range" {
  description = "IP range of the network"
  value       = hcloud_network.cluster_network.ip_range
}

output "subnet_id" {
  description = "ID of the network subnet"
  value       = hcloud_network_subnet.load_balancer_subnet.id
}

output "vswitch_id" {
  description = "ID of the attached vSwitch"
  value       = var.vswitch_id
}

output "ingress_load_balancer_id" {
  description = "ID of the ingress load balancer"
  value       = hcloud_load_balancer.ingress.id
}

output "ingress_load_balancer_name" {
  description = "Name of the ingress load balancer"
  value       = hcloud_load_balancer.ingress.name
}

output "ingress_load_balancer_public_ip" {
  description = "Public IPv4 of the ingress load balancer"
  value       = hcloud_load_balancer.ingress.ipv4
}

output "ingress_load_balancer_public_ipv6" {
  description = "Public IPv6 of the ingress load balancer"
  value       = hcloud_load_balancer.ingress.ipv6
}

output "ingress_load_balancer_private_ip" {
  description = "Private IPv4 of the ingress load balancer"
  value       = hcloud_load_balancer_network.ingress_network.ip
}

output "kube_load_balancer_id" {
  description = "ID of the Kubernetes API load balancer"
  value       = hcloud_load_balancer.kube.id
}

output "kube_load_balancer_name" {
  description = "Name of the Kubernetes API load balancer"
  value       = hcloud_load_balancer.kube.name
}

output "kube_load_balancer_public_ip" {
  description = "Public IPv4 of the Kubernetes API load balancer"
  value       = hcloud_load_balancer.kube.ipv4
}

output "kube_load_balancer_private_ip" {
  description = "Private IPv4 of the Kubernetes API load balancer"
  value       = hcloud_load_balancer_network.kube_network.ip
}
