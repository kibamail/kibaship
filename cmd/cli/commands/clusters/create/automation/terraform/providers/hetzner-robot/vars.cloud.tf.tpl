# Core cluster configuration
variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
}

# Hetzner Cloud configuration
variable "hcloud_token" {
  description = "Hetzner Cloud API Token"
  type        = string
  sensitive   = true
}

variable "location" {
  description = "Location for the network and load balancer"
  type        = string
  default     = "nbg1"
}

variable "network_zone" {
  description = "Network zone (eu-central, us-east, etc.)"
  type        = string
  default     = "eu-central"
}

variable "vswitch_id" {
  description = "Hetzner Robot vSwitch ID"
  type        = number
}

# Network configuration (generated automatically)
variable "cluster_network_ip_range" {
  description = "IP range for the cluster network (generated automatically)"
  type        = string
}

variable "cluster_vswitch_subnet_ip_range" {
  description = "IP range for the vSwitch subnet (generated automatically)"
  type        = string
}

variable "cluster_subnet_ip_range" {
  description = "IP range for the load balancer subnet (generated automatically)"
  type        = string
}

# Hetzner Robot server private IPs (for load balancer targets)
{{range .HetznerRobot.SelectedServers}}
variable "server_{{.ID}}_private_ip" {
  description = "Private IP address for server {{.Name}} (ID: {{.ID}})"
  type        = string
}

{{end}}

# Derived variables for internal use
locals {
  cluster_tags = {
    Name        = var.cluster_name
    Environment = "production"
    ManagedBy   = "kibaship"
  }

  # Server information for reference
  selected_servers = {
{{range .HetznerRobot.SelectedServers}}
    "{{.ID}}" = {
      name       = "{{.Name}}"
      ip         = "{{.IP}}"
      product    = "{{.Product}}"
      dc         = "{{.DC}}"
      role       = "{{.Role}}"
    }
{{end}}
  }

  # Network configuration
  network_config = {
    cluster_network_ip_range         = var.cluster_network_ip_range
    cluster_vswitch_subnet_ip_range  = var.cluster_vswitch_subnet_ip_range
    cluster_subnet_ip_range          = var.cluster_subnet_ip_range
    location                         = var.location
    network_zone                     = var.network_zone
    vswitch_id                       = var.vswitch_id
  }
}
