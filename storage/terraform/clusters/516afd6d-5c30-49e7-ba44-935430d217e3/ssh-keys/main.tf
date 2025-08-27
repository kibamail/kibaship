terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.52"
    }
  }

  backend "s3" {
    bucket = "terraform-state-staging"
    key    = "clusters/516afd6d-5c30-49e7-ba44-935430d217e3/ssh-keys/terraform.tfstate"
    region = "auto"

    endpoints = {
      s3 = "https://e61cebd91eb54d4ebeec9c5d525ae041.r2.cloudflarestorage.com"
    }

    skip_credentials_validation = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    use_path_style           = true
  }
}

variable "hcloud_token" {
  description = "Hetzner Cloud API Token"
  type        = string
  sensitive   = true
}



variable "cluster_name" {
  description = "Name of the cluster"
  type        = string
}

variable "public_key" {
  description = "SSH public key content"
  type        = string
}

provider "hcloud" {
  token = var.hcloud_token
}

resource "hcloud_ssh_key" "cluster_ssh_key" {
  name       = "${var.cluster_name}-ssh-key"
  public_key = var.public_key

  labels = {
    cluster    = var.cluster_name
    managed_by = "kibaship"
    created_at = timestamp()
  }
}

output "ssh_key_id" {
  description = "ID of the created SSH key"
  value       = hcloud_ssh_key.cluster_ssh_key.id
}

output "ssh_key_name" {
  description = "Name of the created SSH key"
  value       = hcloud_ssh_key.cluster_ssh_key.name
}

output "ssh_key_fingerprint" {
  description = "Fingerprint of the SSH key"
  value       = hcloud_ssh_key.cluster_ssh_key.fingerprint
}

output "ssh_key_labels" {
  description = "Labels applied to the SSH key"
  value       = hcloud_ssh_key.cluster_ssh_key.labels
}