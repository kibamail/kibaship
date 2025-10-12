# Variables for Kind Cluster Provisioning

# Core cluster configuration
variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
}

variable "cluster_email" {
  description = "Email address for cluster administrator"
  type        = string
}

variable "paas_features" {
  description = "Comma-separated list of PaaS features to enable"
  type        = string
  default     = "mysql,valkey,postgres"
}

# Kind-specific configuration
variable "kind_node_count" {
  description = "Number of nodes in the Kind cluster (including control plane)"
  type        = number
  default     = 1
}

# Note: Kind clusters use local Terraform state - no S3 configuration needed

# Derived variables for internal use
locals {
  cluster_tags = {
    Name        = var.cluster_name
    Environment = "development"
    ManagedBy   = "kibaship"
    Email       = var.cluster_email
    Provider    = "kind"
  }
}
