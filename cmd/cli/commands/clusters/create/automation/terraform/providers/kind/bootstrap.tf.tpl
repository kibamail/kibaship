# Kind Cluster Bootstrap Configuration
# This template installs essential components for the Kind cluster

terraform {
  required_version = ">= 1.0"

  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.0.2"
    }

    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.38.0"
    }

    null = {
      source  = "hashicorp/null"
      version = "~> 3.2.0"
    }

    local = {
      source  = "hashicorp/local"
      version = "~> 2.5.0"
    }

    time = {
      source  = "hashicorp/time"
      version = "~> 0.12.0"
    }
  }

  # Use local backend for Kind clusters - no S3 required
  backend "local" {
    path = "terraform.tfstate"
  }
}

# Get cluster credentials from provision state
data "terraform_remote_state" "provision" {
  backend = "local"
  config = {
    path = "../provision/terraform.tfstate"
  }
}

# Configure Kubernetes provider using remote state credentials
provider "kubernetes" {
  host                   = data.terraform_remote_state.provision.outputs.cluster_endpoint
  cluster_ca_certificate = data.terraform_remote_state.provision.outputs.cluster_ca_certificate
  client_certificate     = data.terraform_remote_state.provision.outputs.cluster_client_certificate
  client_key             = data.terraform_remote_state.provision.outputs.cluster_client_key
}

# Configure Helm provider using remote state credentials
provider "helm" {
  kubernetes = {
    host                   = data.terraform_remote_state.provision.outputs.cluster_endpoint
    cluster_ca_certificate = data.terraform_remote_state.provision.outputs.cluster_ca_certificate
    client_certificate     = data.terraform_remote_state.provision.outputs.cluster_client_certificate
    client_key             = data.terraform_remote_state.provision.outputs.cluster_client_key
  }
}

# Local variables for component configuration
locals {
  # For Kind clusters, use control plane hostname which is in the API server certificate
  # Format: <cluster-name>-control-plane resolves via Docker's internal DNS
  cluster_name = data.terraform_remote_state.provision.outputs.cluster_name
  api_host = "${local.cluster_name}-control-plane"
  api_port = "6443"
}

# Install Cilium CNI using Helm
resource "helm_release" "cilium" {
  name             = "cilium"
  repository       = "https://helm.cilium.io"
  chart            = "cilium"
  version          = "1.18.2"
  namespace        = "kube-system"
  cleanup_on_fail  = true
  replace          = true
  timeout          = 600
  atomic           = true

  set = [
    {
      name  = "k8sServiceHost"
      value = local.api_host
    },
    {
      name  = "k8sServicePort"
      value = local.api_port
    },
    {
      name  = "kubeProxyReplacement"
      value = "true"
    },
    {
      name  = "tunnelProtocol"
      value = "vxlan"
    },
    {
      name  = "gatewayAPI.enabled"
      value = "true"
    },
    {
      name  = "gatewayAPI.hostNetwork.enabled"
      value = "true"
    },
    {
      name  = "gatewayAPI.enableAlpn"
      value = "true"
    },
    {
      name  = "gatewayAPI.hostNetwork.nodeLabelSelector"
      value = "ingress.kibaship.com/ready=true"
    },
    {
      name  = "gatewayAPI.enableProxyProtocol"
      value = "true"
    },
    {
      name  = "gatewayAPI.enableAppProtocol"
      value = "true"
    },
    {
      name  = "ipam.mode"
      value = "kubernetes"
    },
    {
      name  = "loadBalancer.mode"
      value = "snat"
    },
    {
      name  = "operator.replicas"
      value = "2"
    },
    {
      name  = "bpf.masquerade"
      value = "true"
    },
    {
      name  = "securityContext.capabilities.ciliumAgent"
      value = "{CHOWN,KILL,NET_ADMIN,NET_RAW,IPC_LOCK,SYS_ADMIN,SYS_RESOURCE,DAC_OVERRIDE,FOWNER,SETGID,SETUID}"
    },
    {
      name  = "securityContext.capabilities.cleanCiliumState"
      value = "{NET_ADMIN,SYS_ADMIN,SYS_RESOURCE}"
    },
    {
      name  = "cgroup.autoMount.enabled"
      value = "false"
    },
    {
      name  = "cgroup.hostRoot"
      value = "/sys/fs/cgroup"
    },
    {
      name  = "hostServices.enabled"
      value = "false"
    },
    {
      name  = "externalIPs.enabled"
      value = "true"
    },
    {
      name  = "nodePort.enabled"
      value = "true"
    },
    {
      name  = "hostPort.enabled"
      value = "true"
    },
    {
      name  = "image.pullPolicy"
      value = "IfNotPresent"
    },
    {
      name  = "ipam.operator.clusterPoolIPv4PodCIDRList"
      value = "10.244.0.0/16"
    }
  ]
}

# Install cert-manager using Helm
resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
  version          = "v1.16.2"
  namespace        = "cert-manager"
  create_namespace = true
  cleanup_on_fail  = true
  replace          = true
  timeout          = 600
  atomic           = true   

  set = [
    {
      name  = "crds.enabled"
      value = "true"
    },
    {
      name  = "replicaCount"
      value = "2"
    },
    {
      name  = "webhook.replicaCount"
      value = "2"
    },
    {
      name  = "cainjector.replicaCount"
      value = "2"
    },
    {
      name  = "prometheus.enabled"
      value = "false"
    }
  ]

  depends_on = [helm_release.cilium]
}

# Write kubeconfig to a temporary file for kubectl commands
# Note: .kubeconfig is gitignored - it's only used during terraform apply
resource "local_file" "kubeconfig" {
  content  = data.terraform_remote_state.provision.outputs.kubeconfig
  filename = "${path.module}/.kubeconfig"

  lifecycle {
    ignore_changes = [content]
  }
}

# Mark Kind's built-in storage class as non-default
# Kind v0.11.0+ comes with local-path-provisioner pre-installed
resource "null_resource" "patch_kind_storage_class" {
  provisioner "local-exec" {
    command = <<-EOT
      export KUBECONFIG="${path.module}/.kubeconfig"

      # Mark Kind's default 'standard' storage class as non-default
      # so we can use our custom storage classes as default
      kubectl patch storageclass standard -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}' || true
    EOT
  }

  depends_on = [
    helm_release.cert_manager,
    local_file.kubeconfig
  ]
}

# Create storage classes using local-path-provisioner
# Note: Local-path-provisioner doesn't support actual replication or RWX,
# but we maintain the naming convention for compatibility with existing deployments

# Storage class with "1 replica" (local-path doesn't replicate, naming for compatibility)
resource "kubernetes_storage_class" "storage_replica_1" {
  metadata {
    name = "storage-replica-1"
    annotations = {
      "storageclass.kubernetes.io/is-default-class" = "true"
    }
  }

  storage_provisioner = "rancher.io/local-path"
  reclaim_policy      = "Delete"
  volume_binding_mode = "WaitForFirstConsumer"
  allow_volume_expansion = true

  depends_on = [null_resource.patch_kind_storage_class]
}

# Storage class with "2 replicas" (local-path doesn't replicate, naming for compatibility)
resource "kubernetes_storage_class" "storage_replica_2" {
  metadata {
    name = "storage-replica-2"
  }

  storage_provisioner = "rancher.io/local-path"
  reclaim_policy      = "Delete"
  volume_binding_mode = "WaitForFirstConsumer"
  allow_volume_expansion = true

  depends_on = [null_resource.patch_kind_storage_class]
}

# Storage class for "ReadWriteMany" (local-path only supports RWO, naming for compatibility)
resource "kubernetes_storage_class" "storage_replica_rwm_1" {
  metadata {
    name = "storage-replica-rwm-1"
  }

  storage_provisioner = "rancher.io/local-path"
  reclaim_policy      = "Delete"
  volume_binding_mode = "WaitForFirstConsumer"
  allow_volume_expansion = true

  depends_on = [null_resource.patch_kind_storage_class]
}

{{- template "mysql-operator" -}}

resource "kubernetes_namespace" "observability" {
  metadata {
    name = "observability"
  }

  depends_on = [
    kubernetes_storage_class.storage_replica_1,
    kubernetes_storage_class.storage_replica_2,
    kubernetes_storage_class.storage_replica_rwm_1,
    helm_release.cilium,
    null_resource.patch_kind_storage_class,
    helm_release.cert_manager
  ]
}

# Install VictoriaMetrics monitoring stack
resource "helm_release" "victoria_metrics" {
  name       = "vc-metrics"
  repository = "https://victoriametrics.github.io/helm-charts/"
  chart      = "victoria-metrics-k8s-stack"
  version    = "0.60.1"
  namespace  = kubernetes_namespace.observability.metadata[0].name
  cleanup_on_fail  = true
  timeout    = 900
  wait       = false
  replace    = true

  values = [
    yamlencode({
      ################################################
      # VictoriaMetrics Operator (Auto-installed)
      ################################################
      victoria-metrics-operator = {
        enabled = true
        operator = {
          disable_prometheus_converter = false
        }
      }

      ################################################
      # VictoriaMetrics Single (Metrics Storage)
      ################################################
      vmsingle = {
        enabled = true
        spec = {
          retentionPeriod = "30d"
          storage = {
            storageClassName = "storage-replica-1"
            resources = {
              requests = {
                storage = "1Gi"
              }
            }
          }
          resources = {
            requests = {
              cpu    = "500m"
              memory = "1Gi"
            }
            limits = {
              cpu    = "2"
              memory = "4Gi"
            }
          }
        }
      }

      ################################################
      # VMAgent (Metrics Collection)
      ################################################
      vmagent = {
        enabled = true
        spec = {
          selectAllByDefault = true
          scrapeInterval     = "30s"
          resources = {
            requests = {
              cpu    = "250m"
              memory = "512Mi"
            }
            limits = {
              cpu    = "1"
              memory = "2Gi"
            }
          }
        }
      }

      ################################################
      # Kubernetes Component Scraping
      ################################################
      kubelet = {
        enabled = true
        spec = {
          interval = "30s"
        }
      }

      kubeApiServer = {
        enabled = true
      }

      kubeControllerManager = {
        # Disabled for Kind: controller-manager binds metrics to localhost only (127.0.0.1:10257)
        # and cannot be scraped from vmagent pods
        enabled = false
      }

      kubeScheduler = {
        # Disabled for Kind: scheduler binds metrics to localhost only (127.0.0.1:10259)
        # and cannot be scraped from vmagent pods
        enabled = false
      }

      kubeProxy = {
        enabled = false
      }

      kubeEtcd = {
        # Disabled for Kind: etcd binds metrics to localhost only (127.0.0.1:2381)
        # and cannot be scraped from vmagent pods
        enabled = false
      }

      coreDns = {
        enabled = true
      }

      ################################################
      # Dependencies (Auto-installed)
      ################################################
      prometheus-node-exporter = {
        enabled = true
        service = {
          labels = {
            jobLabel = "node-exporter"
          }
        }
        vmScrape = {
          enabled = true
        }
      }

      kube-state-metrics = {
        enabled = true
        vmScrape = {
          enabled = true
        }
      }

      grafana = {
        enabled       = true
        persistence = {
          enabled = true
          size    = "1Gi"
          type    = "pvc"
          storageClassName = "storage-replica-1"
          accessModes = ["ReadWriteOnce"]
        }
        vmScrape = {
          enabled = true
        }
      }

      ################################################
      # Default Dashboards & Rules
      ################################################
      defaultDashboards = {
        enabled = true
        dashboards = {
          victoriametrics-operator = {
            enabled = true
          }
          victoriametrics-vmalert = {
            enabled = true
          }
          node-exporter-full = {
            enabled = true
          }
        }
      }

      defaultRules = {
        create = true
        rules = {
          vmagent   = true
          vmsingle  = true
          vmhealth  = true
          k8s       = true
        }
      }
    })
  ]

  depends_on = [kubernetes_namespace.observability]
}

# Wait for VictoriaMetrics operator webhooks to be fully ready
# This prevents webhook connection errors when installing VictoriaLogs
resource "time_sleep" "wait_for_victoria_metrics_operator" {
  depends_on = [helm_release.victoria_metrics]

  create_duration = "30s"
}

# Install VictoriaLogs for log aggregation
resource "helm_release" "victoria_logs" {
  name       = "vc-logs"
  repository = "https://victoriametrics.github.io/helm-charts/"
  chart      = "victoria-logs-single"
  version    = "0.11.11"
  namespace  = kubernetes_namespace.observability.metadata[0].name
  cleanup_on_fail  = true
  timeout    = 900
  wait       = false
  replace    = true

  values = [
    yamlencode({
      ################################################
      # VictoriaLogs Server
      ################################################
      server = {
        retentionPeriod = "90d"
        persistentVolume = {
          enabled = true
          size    = "1Gi"
          accessModes = ["ReadWriteOnce"]
          storageClassName = "storage-replica-1"
        }
        resources = {
          requests = {
            cpu    = "500m"
            memory = "1Gi"
          }
          limits = {
            cpu    = "2"
            memory = "4Gi"
          }
        }

        vmServiceScrape = {
          enabled = true
        }
      }

      dashboards = {
        enabled = true
      }

      ################################################
      # Vector Log Collector (Auto-deployed as DaemonSet)
      ################################################
      vector = {
        enabled = true
        resources = {
          requests = {
            cpu    = "100m"
            memory = "256Mi"
          }
          limits = {
            cpu    = "500m"
            memory = "512Mi"
          }
        }
      }
    })
  ]

  depends_on = [time_sleep.wait_for_victoria_metrics_operator]
}


# Output information

output "cilium_status" {
  description = "Cilium CNI installation status"
  value       = "Cilium ${helm_release.cilium.version} installed via Helm"
}

output "cert_manager_status" {
  description = "Cert-manager installation status"
  value       = "Cert-manager ${helm_release.cert_manager.version} installed via Helm"
}

output "cluster_info" {
  description = "Cluster configuration details"
  value = {
    api_host             = local.api_host
    api_port             = local.api_port
    cilium_version       = helm_release.cilium.version
    storage_provisioner  = "local-path-provisioner (Kind built-in)"
    cert_manager_version = helm_release.cert_manager.version
    storage_note         = "Using Kind's built-in local-path-provisioner (no iSCSI required)"
  }
}

output "storage_classes" {
  description = "Available storage classes"
  value = {
    storage_replica_1 = {
      name = kubernetes_storage_class.storage_replica_1.metadata[0].name
      provisioner = "rancher.io/local-path"
      note = "Default storage class, RWO only"
    }
    storage_replica_2 = {
      name = kubernetes_storage_class.storage_replica_2.metadata[0].name
      provisioner = "rancher.io/local-path"
      note = "Named for compatibility, no actual replication, RWO only"
    }
    storage_replica_rwm_1 = {
      name = kubernetes_storage_class.storage_replica_rwm_1.metadata[0].name
      provisioner = "rancher.io/local-path"
      note = "Named for RWX compatibility, but only supports RWO"
    }
  }
}
