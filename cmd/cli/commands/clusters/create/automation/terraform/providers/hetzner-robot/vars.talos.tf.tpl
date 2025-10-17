# Cluster Configuration Variables
variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
}

variable "cluster_endpoint" {
  description = "Kubernetes API endpoint (typically the load balancer IP)"
  type        = string
}

variable "cluster_dns_name" {
  description = "DNS name for Kubernetes API endpoint (e.g., kube.example.com)"
  type        = string
}

variable "cluster_network_ip_range" {
  description = "Main cluster network IP range (e.g., 172.25.0.0/16)"
  type        = string
}

variable "vswitch_subnet_ip_range" {
  description = "VSwitch subnet IP range for private networking (e.g., 172.25.64.0/20)"
  type        = string
}

variable "vlan_id" {
  description = "VLAN ID for the VSwitch"
  type        = number
}

# Server-specific Network Configuration Variables
# These are dynamically discovered from Talos after provisioning
{{range .HetznerRobot.SelectedServers}}
variable "server_{{.ID}}_public_network_interface" {
  description = "Public network interface name for server {{.Name}} ({{.ID}})"
  type        = string
}

variable "server_{{.ID}}_public_address_subnet" {
  description = "Public IP address with CIDR notation for server {{.Name}} ({{.ID}})"
  type        = string
}

variable "server_{{.ID}}_public_ipv4_gateway" {
  description = "Public IPv4 gateway for server {{.Name}} ({{.ID}})"
  type        = string
}

variable "server_{{.ID}}_private_address_subnet" {
  description = "Private IP address with CIDR notation for server {{.Name}} ({{.ID}})"
  type        = string
}

variable "server_{{.ID}}_private_ipv4_gateway" {
  description = "Private IPv4 gateway (VSwitch gateway) for server {{.Name}} ({{.ID}})"
  type        = string
}

variable "server_{{.ID}}_installation_disk" {
  description = "Installation disk path for server {{.Name}} ({{.ID}})"
  type        = string
}

variable "server_{{.ID}}_storage_disks" {
  description = "Storage disks (non-installation disks) for server {{.Name}} ({{.ID}})"
  type = list(object({
    name = string
    path = string
  }))
  default = []
}

{{end}}
