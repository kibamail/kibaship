# Core cluster configuration
variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
}

variable "cluster_email" {
  description = "Email address for cluster administration"
  type        = string
}

variable "paas_features" {
  description = "PaaS features to enable (none, basic, full)"
  type        = string
  default     = "none"
}

# Terraform state configuration
variable "terraform_state_bucket" {
  description = "S3 bucket for Terraform state storage"
  type        = string
}

variable "terraform_state_region" {
  description = "AWS region for Terraform state S3 bucket"
  type        = string
}

variable "terraform_state_access_key" {
  description = "AWS access key for Terraform state S3 bucket"
  type        = string
  sensitive   = true
}

variable "terraform_state_secret_key" {
  description = "AWS secret key for Terraform state S3 bucket"
  type        = string
  sensitive   = true
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

# Talos configuration
variable "cluster_endpoint" {
  description = "Kubernetes API endpoint URL"
  type        = string
}

variable "cluster_dns_name" {
  description = "DNS name for Kubernetes API endpoint (e.g., kube.example.com)"
  type        = string
}

variable "vlan_id" {
  description = "VLAN ID for private network"
  type        = number
}

variable "vswitch_subnet_ip_range" {
  description = "vSwitch subnet IP range (generated automatically)"
  type        = string
}

# Selected servers for Talos configuration
variable "selected_servers" {
  description = "List of selected servers with their configuration"
  type = list(object({
    id      = string
    name    = string
    ip      = string
    role    = string
    product = string
    dc      = string
  }))
  default = []
}

# Hetzner Robot server passwords (defined in provision.tf.tpl)
{{range .HetznerRobot.SelectedServers}}
variable "server_{{.ID}}_password" {
  description = "Root password for server {{.Name}} (ID: {{.ID}})"
  type        = string
  sensitive   = true
}

{{end}}

# Hetzner Robot server private IPs (for load balancer targets)
{{range .HetznerRobot.SelectedServers}}
variable "server_{{.ID}}_private_ip" {
  description = "Private IP address for server {{.Name}} (ID: {{.ID}})"
  type        = string
}

{{end}}

# Talos server network configuration
{{range .HetznerRobot.SelectedServers}}
variable "server_{{.ID}}_public_network_interface" {
  description = "Public network interface for server {{.Name}} (ID: {{.ID}})"
  type        = string
  default     = "enp1s0"
}

variable "server_{{.ID}}_public_address_subnet" {
  description = "Public address subnet for server {{.Name}} (ID: {{.ID}})"
  type        = string
}

variable "server_{{.ID}}_public_ipv4_gateway" {
  description = "Public IPv4 gateway for server {{.Name}} (ID: {{.ID}})"
  type        = string
}

variable "server_{{.ID}}_private_address_subnet" {
  description = "Private address subnet for server {{.Name}} (ID: {{.ID}})"
  type        = string
}

variable "server_{{.ID}}_private_ipv4_gateway" {
  description = "Private IPv4 gateway for server {{.Name}} (ID: {{.ID}})"
  type        = string
}

variable "server_{{.ID}}_installation_disk" {
  description = "Installation disk path for server {{.Name}} (ID: {{.ID}})"
  type        = string
}

{{end}}

# Derived variables for internal use
locals {
  cluster_tags = {
    Name        = var.cluster_name
    Environment = "production"
    ManagedBy   = "kibaship"
    Email       = var.cluster_email
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
