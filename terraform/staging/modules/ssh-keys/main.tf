# =============================================================================
# SSH Key Management Module
# =============================================================================
# This module manages SSH key generation and deployment for Ubuntu servers,
# including:
# - ED25519 SSH key pair generation
# - Hetzner Cloud SSH key registration
# - Local file storage for key management
# - Secure key handling with proper permissions

terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.51.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5.0"
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

variable "ssh_public_key_path" {
  description = "Path to store the generated SSH public key"
  type        = string
  default     = ".secrets/staging/id_ed25519.pub"
}

variable "ssh_private_key_path" {
  description = "Path to store the generated SSH private key"
  type        = string
  default     = ".secrets/staging/id_ed25519"
}

# =============================================================================
# SSH Key Generation
# =============================================================================

resource "tls_private_key" "ssh_key" {
  algorithm = "ED25519"
}

# Add email comment to the public key
locals {
  ssh_public_key_with_email = "${trimspace(tls_private_key.ssh_key.public_key_openssh)} engineering@kibaship.com"
}

# =============================================================================
# Local File Storage
# =============================================================================

resource "local_file" "ssh_private_key" {
  content         = tls_private_key.ssh_key.private_key_openssh
  filename        = var.ssh_private_key_path
  file_permission = "0600"

  provisioner "local-exec" {
    command = "mkdir -p ${dirname(var.ssh_private_key_path)}"
  }
}

resource "local_file" "ssh_public_key" {
  content         = local.ssh_public_key_with_email
  filename        = var.ssh_public_key_path
  file_permission = "0644"

  provisioner "local-exec" {
    command = "mkdir -p ${dirname(var.ssh_public_key_path)}"
  }
}

# =============================================================================
# Hetzner Cloud SSH Key Registration
# =============================================================================

resource "hcloud_ssh_key" "cluster_key" {
  name       = "${var.cluster_name}-ssh-key"
  public_key = local.ssh_public_key_with_email

  labels = {
    environment = var.environment
    cluster     = var.cluster_name
    purpose     = "cluster-access"
  }
}

# =============================================================================
# Outputs
# =============================================================================

output "ssh_key_id" {
  description = "Hetzner Cloud SSH key ID"
  value       = hcloud_ssh_key.cluster_key.id
}

output "ssh_key_name" {
  description = "Hetzner Cloud SSH key name"
  value       = hcloud_ssh_key.cluster_key.name
}

output "ssh_public_key" {
  description = "SSH public key content with email"
  value       = local.ssh_public_key_with_email
}

output "ssh_private_key" {
  description = "SSH private key content"
  value       = tls_private_key.ssh_key.private_key_openssh
  sensitive   = true
}

output "ssh_public_key_path" {
  description = "Path to SSH public key file"
  value       = local_file.ssh_public_key.filename
}

output "ssh_private_key_path" {
  description = "Path to SSH private key file"
  value       = local_file.ssh_private_key.filename
}

output "ssh_key_fingerprint" {
  description = "SSH key fingerprint"
  value       = hcloud_ssh_key.cluster_key.fingerprint
}
