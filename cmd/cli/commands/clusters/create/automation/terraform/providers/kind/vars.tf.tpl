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

variable "kind_storage_per_node" {
  description = "Storage size per node in GB (note: Kind uses container internal storage, not host bind mounts)"
  type        = number
  default     = 75
}

variable "kind_k8s_version" {
  description = "Kubernetes version for Kind cluster (e.g., v1.34.0, v1.31.0)"
  type        = string
  default     = "v1.34.0@sha256:7416a61b42b1662ca6ca89f02028ac133a309a2a30ba309614e8ec94d976dc5a"
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
