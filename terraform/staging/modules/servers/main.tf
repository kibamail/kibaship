# =============================================================================
# Servers Module
# =============================================================================
# This module provisions Ubuntu 24.04 servers on Hetzner Cloud, including:
# - Control plane nodes (role=control-plane)
# - Worker nodes (role=worker)
# - Basic system preparation
# - SSH key management and secure access

terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.51.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5.0"
    }
    http = {
      source  = "hashicorp/http"
      version = "~> 3.4.0"
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

variable "cluster_endpoint" {
  description = "Kubernetes API endpoint URL"
  type        = string
}

variable "k8s_api_public_ip" {
  description = "Public IP of the Kubernetes API load balancer"
  type        = string
}

variable "k8s_api_private_ip" {
  description = "Private IP of the Kubernetes API load balancer"
  type        = string
  default     = "10.0.1.100"
}

variable "ubuntu_version" {
  description = "Ubuntu version to use"
  type        = string
  default     = "24.04"
}

variable "server_type" {
  description = "Hetzner Cloud server type"
  type        = string
  default     = "cx22"
}

variable "location" {
  description = "Hetzner Cloud location"
  type        = string
  default     = "nbg1"
}

variable "control_plane_count" {
  description = "Number of control plane nodes"
  type        = number
  default     = 3
}

variable "worker_count" {
  description = "Number of worker nodes"
  type        = number
  default     = 3
}

variable "ssh_key_id" {
  description = "Hetzner Cloud SSH key ID"
  type        = string
}

variable "ssh_private_key" {
  description = "SSH private key for server access"
  type        = string
  sensitive   = true
}

variable "ssh_public_key" {
  description = "SSH public key content"
  type        = string
}

# =============================================================================
# Data Sources
# =============================================================================

data "hcloud_image" "ubuntu" {
  name = "ubuntu-${var.ubuntu_version}"
}

# =============================================================================
# Local Values
# =============================================================================

locals {
  cluster_domain        = "kibaship.internal"
  network_ipv4_cidr     = "10.0.0.0/16"
  node_ipv4_cidr        = "10.0.1.0/24"
  pod_ipv4_cidr         = "10.0.16.0/20"
  service_ipv4_cidr     = "10.0.8.0/21"

  control_plane_ips = [
    for i in range(var.control_plane_count) : "10.0.1.${10 + i}"
  ]

  worker_ips = [
    for i in range(var.worker_count) : "10.0.1.${20 + i}"
  ]

  control_plane_public_ipv4_list = [
    for i in range(var.control_plane_count) : hcloud_server.control_planes[i].ipv4_address
  ]

  worker_public_ipv4_list = [
    for i in range(var.worker_count) : hcloud_server.workers[i].ipv4_address
  ]

  cert_SANs = distinct(
    concat(
      local.control_plane_ips,
      [
        var.k8s_api_public_ip,
        var.k8s_api_private_ip,
        "127.0.0.1",
        "kubernetes",
        "kubernetes.default",
        "kubernetes.default.svc",
        "kubernetes.default.svc.${local.cluster_domain}"
      ]
    )
  )
}

# =============================================================================
# Cloud-Init Configuration
# =============================================================================

locals {
  # Load setup scripts from YAML files
  common_setup_yaml = yamldecode(file("${path.module}/../../scripts/common-setup.yaml"))
  worker_specific_yaml = yamldecode(file("${path.module}/../../scripts/worker-specific.yaml"))
  post_reboot_yaml = yamldecode(file("${path.module}/../../scripts/post-reboot-verification.yaml"))

  # Build script arrays and filter out empty strings
  common_setup_steps = [for cmd in flatten([
    local.common_setup_yaml.common_setup.user_management,
    local.common_setup_yaml.common_setup.ssh_key_setup,
    local.common_setup_yaml.common_setup.system_update,
    local.common_setup_yaml.common_setup.package_installation,
    local.common_setup_yaml.common_setup.networking_modules,
    local.common_setup_yaml.common_setup.sysctl_networking,
    local.common_setup_yaml.common_setup.disable_swap,
    local.common_setup_yaml.common_setup.time_sync,
    local.common_setup_yaml.common_setup.helm_installation,
    local.common_setup_yaml.common_setup.ssh_security,
    local.common_setup_yaml.common_setup.completion_message
  ]) : cmd if cmd != ""]

  worker_specific_steps = [for cmd in flatten([
    local.worker_specific_yaml.worker_specific.storage_modules,
    local.worker_specific_yaml.worker_specific.hugepages_sysctl,
    local.worker_specific_yaml.worker_specific.hugepages_immediate,
    local.worker_specific_yaml.worker_specific.grub_configuration,
    local.worker_specific_yaml.worker_specific.verification,
    local.worker_specific_yaml.worker_specific.reboot_message
  ]) : cmd if cmd != ""]

  post_reboot_steps = [for cmd in flatten([
    local.post_reboot_yaml.post_reboot.connection_test,
    local.post_reboot_yaml.post_reboot.hugepages_check,
    local.post_reboot_yaml.post_reboot.modules_check,
    local.post_reboot_yaml.post_reboot.time_sync_check,
    local.post_reboot_yaml.post_reboot.final_verification
  ]) : cmd if cmd != ""]

  # Complete script arrays
  control_plane_script = concat(local.common_setup_steps, ["echo 'Rebooting server to apply all configurations...'", "reboot"])
  worker_script = concat(local.common_setup_steps, local.worker_specific_steps, ["reboot"])
}

# =============================================================================
# Control Plane Servers
# =============================================================================

resource "hcloud_server" "control_planes" {
  count       = var.control_plane_count
  name        = "${var.cluster_name}-control-plane-${count.index + 1}"
  image       = data.hcloud_image.ubuntu.id
  server_type = var.server_type
  location    = var.location
  ssh_keys    = [var.ssh_key_id]
  # No user_data - using remote-exec for setup

  labels = {
    environment = var.environment
    cluster     = var.cluster_name
    role        = "control-plane"
  }

  public_net {
    ipv4_enabled = true
    ipv6_enabled = false
  }

  network {
    network_id = var.network_id
    ip         = local.control_plane_ips[count.index]
  }


}

# =============================================================================
# Control Plane Server Setup
# =============================================================================

resource "null_resource" "control_plane_setup" {
  count = var.control_plane_count

  connection {
    type        = "ssh"
    user        = "root"
    private_key = var.ssh_private_key
    host        = hcloud_server.control_planes[count.index].ipv4_address
    timeout     = "10m"
  }

  provisioner "remote-exec" {
    inline = [for cmd in local.control_plane_script : replace(cmd, "$${ssh_public_key}", var.ssh_public_key)]
  }

  depends_on = [hcloud_server.control_planes]
}

# =============================================================================
# Worker Servers
# =============================================================================

resource "hcloud_server" "workers" {
  count       = var.worker_count
  name        = "${var.cluster_name}-worker-${count.index + 1}"
  image       = data.hcloud_image.ubuntu.id
  server_type = var.server_type
  location    = var.location
  ssh_keys    = [var.ssh_key_id]
  # No user_data - using remote-exec for setup

  labels = {
    environment = var.environment
    cluster     = var.cluster_name
    role        = "worker"
  }

  public_net {
    ipv4_enabled = true
    ipv6_enabled = false
  }

  network {
    network_id = var.network_id
    ip         = local.worker_ips[count.index]
  }


}

# =============================================================================
# Worker Server Setup
# =============================================================================

resource "null_resource" "worker_setup" {
  count = var.worker_count

  connection {
    type        = "ssh"
    user        = "root"
    private_key = var.ssh_private_key
    host        = hcloud_server.workers[count.index].ipv4_address
    timeout     = "10m"
  }

  provisioner "remote-exec" {
    inline = [for cmd in local.worker_script : replace(cmd, "$${ssh_public_key}", var.ssh_public_key)]
  }

  depends_on = [hcloud_server.workers]
}

# =============================================================================
# Server Readiness Check
# =============================================================================

resource "time_sleep" "wait_for_reboot_delay" {
  create_duration = "60s"

  depends_on = [
    null_resource.control_plane_setup,
    null_resource.worker_setup
  ]
}

resource "null_resource" "wait_for_reboot" {
  count = var.control_plane_count + var.worker_count

  connection {
    type        = "ssh"
    user        = "ubuntu"
    private_key = var.ssh_private_key
    host        = count.index < var.control_plane_count ? local.control_plane_public_ipv4_list[count.index] : local.worker_public_ipv4_list[count.index - var.control_plane_count]
    timeout     = "10m"
  }

  provisioner "remote-exec" {
    inline = local.post_reboot_steps
  }

  depends_on = [
    time_sleep.wait_for_reboot_delay
  ]
}


# =============================================================================
# Infrastructure Ready
# =============================================================================

resource "null_resource" "infrastructure_ready" {
  triggers = {
    control_planes_ready = join(",", [for i in range(var.control_plane_count) : hcloud_server.control_planes[i].id])
    workers_ready        = join(",", [for i in range(var.worker_count) : hcloud_server.workers[i].id])
    reboot_complete      = join(",", null_resource.wait_for_reboot[*].id)
  }

  provisioner "local-exec" {
    command = "echo 'Infrastructure provisioning completed.'"
  }

  depends_on = [
    null_resource.wait_for_reboot
  ]
}


# =============================================================================
# Outputs
# =============================================================================

output "cluster_info" {
  description = "Infrastructure cluster information"
  value = {
    name           = var.cluster_name
    endpoint       = var.cluster_endpoint
    ubuntu_version = var.ubuntu_version
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

output "servers_ready" {
  description = "Indicates when servers are ready"
  value       = "Infrastructure provisioned successfully."
}

output "infrastructure_ready" {
  description = "Indicates when the infrastructure is ready"
  value       = true
  depends_on  = [null_resource.infrastructure_ready]
}

output "control_plane_ips" {
  description = "Public IP addresses of control plane nodes"
  value       = local.control_plane_public_ipv4_list
}

output "worker_ips" {
  description = "Public IP addresses of worker nodes"
  value       = local.worker_public_ipv4_list
}

output "script_debug" {
  description = "Debug information about generated scripts"
  value = {
    common_setup_steps_count = length(local.common_setup_steps)
    worker_specific_steps_count = length(local.worker_specific_steps)
    post_reboot_steps_count = length(local.post_reboot_steps)
    control_plane_script_count = length(local.control_plane_script)
    worker_script_count = length(local.worker_script)
  }
}


