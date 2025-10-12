# Variables for DigitalOcean Kubernetes Cluster Provisioning
# These variables will be populated via TF_VAR environment variables

# Core cluster configuration
variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
}

variable "cluster_email" {
  description = "Email address for cluster administration and Let's Encrypt certificates"
  type        = string
}

variable "paas_features" {
  description = "PaaS features to install (comma-separated: mysql,valkey,postgres,none)"
  type        = string
}

# DigitalOcean provider configuration
variable "do_token" {
  description = "DigitalOcean API token"
  type        = string
  sensitive   = true
}

variable "do_region" {
  description = "DigitalOcean region for the cluster"
  type        = string
}

variable "do_node_count" {
  description = "Number of nodes in the node pool"
  type        = number
}

variable "do_node_size" {
  description = "DigitalOcean droplet size for worker nodes"
  type        = string
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

# Derived variables for internal use
locals {
  cluster_tags = {
    Name        = var.cluster_name
    Environment = "production"
    ManagedBy   = "kibaship"
    Email       = var.cluster_email
  }
}
