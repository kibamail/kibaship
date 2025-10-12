# Kind Cluster Bootstrap Configuration
# This template will be used for deploying PaaS services to the Kind cluster

terraform {
  required_version = ">= 1.0"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11.0"
    }
  }

  # Use local backend for Kind clusters - no S3 required
  backend "local" {
    path = "terraform.tfstate"
  }
}

# Configure providers for Kind cluster
provider "kubernetes" {
  config_path = "~/.kube/config"
  config_context = "kind-${var.cluster_name}"
}

provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
    config_context = "kind-${var.cluster_name}"
  }
}

# Placeholder for future PaaS service deployments
# This will be populated with Helm charts and Kubernetes manifests
# for MySQL, PostgreSQL, Valkey/Redis, and other services

resource "kubernetes_namespace" "kibaship_system" {
  metadata {
    name = "kibaship-system"
    labels = {
      "app.kubernetes.io/name"       = "kibaship"
      "app.kubernetes.io/component"  = "system"
      "app.kubernetes.io/managed-by" = "terraform"
    }
  }
}

# Output namespace information
output "kibaship_namespace" {
  description = "Kibaship system namespace"
  value       = kubernetes_namespace.kibaship_system.metadata[0].name
}
