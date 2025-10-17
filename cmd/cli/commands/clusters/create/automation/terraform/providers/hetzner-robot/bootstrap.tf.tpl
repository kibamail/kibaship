terraform {
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
      version = "~> 3.0"
    }
  }

  backend "local" {
    path = "terraform.tfstate"
  }
}

# Read kubeconfig from Talos stage
data "terraform_remote_state" "talos" {
  backend = "local"
  config = {
    path = "../bare-metal-talos-bootstrap/terraform.tfstate"
  }
}

# Configure Kubernetes provider with kubeconfig from Talos
provider "kubernetes" {
  host                   = yamldecode(data.terraform_remote_state.talos.outputs.kubeconfig).clusters[0].cluster.server
  client_certificate     = base64decode(yamldecode(data.terraform_remote_state.talos.outputs.kubeconfig).users[0].user["client-certificate-data"])
  client_key             = base64decode(yamldecode(data.terraform_remote_state.talos.outputs.kubeconfig).users[0].user["client-key-data"])
  cluster_ca_certificate = base64decode(yamldecode(data.terraform_remote_state.talos.outputs.kubeconfig).clusters[0].cluster["certificate-authority-data"])
}

# Configure Helm provider with kubeconfig from Talos
provider "helm" {
  kubernetes = {
    # Parse kubeconfig from Talos state
    host                   = yamldecode(data.terraform_remote_state.talos.outputs.kubeconfig).clusters[0].cluster.server
    client_certificate     = base64decode(yamldecode(data.terraform_remote_state.talos.outputs.kubeconfig).users[0].user["client-certificate-data"])
    client_key             = base64decode(yamldecode(data.terraform_remote_state.talos.outputs.kubeconfig).users[0].user["client-key-data"])
    cluster_ca_certificate = base64decode(yamldecode(data.terraform_remote_state.talos.outputs.kubeconfig).clusters[0].cluster["certificate-authority-data"])
  }
}

# Install Cilium using Helm
resource "helm_release" "cilium" {
  name       = "cilium"
  repository = "https://helm.cilium.io/"
  chart      = "cilium"
  version    = "1.18.0"
  namespace  = "kube-system"

  # Wait for resources to be ready
  wait    = true
  atomic  = true

  set = [
    {
      name  = "k8sServiceHost"
      value = "localhost"
    },
    {
      name  = "k8sServicePort"
      value = "7445"
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
    }
  ]

  depends_on = [
    data.terraform_remote_state.talos
  ]
}

# Install Longhorn using Helm
resource "helm_release" "longhorn" {
  name             = "longhorn"
  repository       = "https://charts.longhorn.io"
  chart            = "longhorn"
  version          = "1.10.0"
  namespace        = "longhorn"
  create_namespace = true
  cleanup_on_fail  = true

  # Wait for resources to be ready
  wait    = true
  atomic  = true
  timeout = 600

  set = [
    {
      name  = "persistence.defaultClass"
      value = "false"
    },
    {
      name = "defaultSettings.replicaAutoBalance"
      value = "strict-local"
    }
  ]

  depends_on = [
    helm_release.cilium
  ]
}

# Create Longhorn storage classes
resource "kubernetes_storage_class" "storage_replica_1" {
  metadata {
    name = "storage-replica-1"
  }

  storage_provisioner = "driver.longhorn.io"
  reclaim_policy      = "Delete"
  volume_binding_mode = "WaitForFirstCustomer"
  allow_volume_expansion = true

  parameters = {
    numberOfReplicas    = "1"
    staleReplicaTimeout = "30"
    fromBackup          = ""
    fsType              = "ext4"
  }

  depends_on = [
    helm_release.longhorn
  ]
}

resource "kubernetes_storage_class" "storage_replica_2" {
  metadata {
    name = "storage-replica-2"
  }

  storage_provisioner = "driver.longhorn.io"
  reclaim_policy      = "Delete"
  volume_binding_mode = "WaitForFirstCustomer"
  allow_volume_expansion = true

  parameters = {
    numberOfReplicas    = "2"
    staleReplicaTimeout = "30"
    fromBackup          = ""
    fsType              = "ext4"
  }

  depends_on = [
    helm_release.longhorn
  ]
}

resource "kubernetes_storage_class" "storage_replica_rwm_1" {
  metadata {
    name = "storage-replica-rwm-1"
  }

  storage_provisioner = "driver.longhorn.io"
  reclaim_policy      = "Delete"
  volume_binding_mode = "WaitForFirstCustomer"
  allow_volume_expansion = true

  parameters = {
    numberOfReplicas    = "1"
    staleReplicaTimeout = "30"
    fromBackup          = ""
    fsType              = "ext4"
    migratable          = "true"
  }

  depends_on = [
    helm_release.longhorn
  ]
}

# Install cert-manager (required by other operators)
resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
  version          = "v1.18.2"
  namespace        = "cert-manager"
  create_namespace = true
  cleanup_on_fail  = true

  # Wait for resources to be ready
  wait   = true
  atomic = true

  # Install CRDs
  set = [
    {
      name  = "installCRDs"
      value = "true"
    },
    {
      name  = "global.leaderElection.namespace"
      value = "cert-manager"
    }
  ]

  depends_on = [
    helm_release.cilium
  ]
}

# Install MySQL Operator using Helm
resource "helm_release" "mysql_operator" {
  name             = "mysql-operator"
  repository       = "https://mysql.github.io/mysql-operator/"
  chart            = "mysql-operator"
  version          = "2.2.5"
  namespace        = "mysql"
  create_namespace = true
  cleanup_on_fail  = true
  replace          = true

  # Wait for resources to be ready
  wait   = true
  atomic = true

  set = [
    {
      name  = "replicas"
      value = "3"
    },
    {
      name  = "envs.k8sClusterDomain"
      value = "cluster.local"
    }
  ]

  depends_on = [
    helm_release.cilium,
  ]
}

# Create observability namespace
resource "kubernetes_namespace" "observability" {
  metadata {
    name = "observability"
  }

  depends_on = [
    kubernetes_storage_class.storage_replica_1,
    kubernetes_storage_class.storage_replica_2,
    kubernetes_storage_class.storage_replica_rwm_1
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
  atomic     = true

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
                storage = "100Gi"
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
        enabled = true
      }

      kubeScheduler = {
        enabled = true
      }

      kubeProxy = {
        enabled = false
      }

      kubeEtcd = {
        enabled = true
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

  depends_on = [
    kubernetes_namespace.observability,
    helm_release.longhorn,
    kubernetes_storage_class.storage_replica_1
  ]
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
  atomic     = true

  values = [
    yamlencode({
      ################################################
      # VictoriaLogs Server
      ################################################
      server = {
        retentionPeriod = "90d"
        persistentVolume = {
          enabled = true
          size    = "50Gi"
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
        role = "Agent"
        dataDir = "/vector-data-dir"

        # Expose UDP ports for Talos logs on host network
        containerPorts = [
          {
            name = "talos-kernel"
            containerPort = 6050
            protocol = "UDP"
            hostPort = 6050
          },
          {
            name = "talos-service"
            containerPort = 6051
            protocol = "UDP"
            hostPort = 6051
          },
          {
            name = "prom-exporter"
            containerPort = 9090
            protocol = "TCP"
          }
        ]

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

        # Custom Vector configuration for Talos + K8s logs
        customConfig = {
          data_dir = "/vector-data-dir"

          api = {
            enabled = false
            address = "0.0.0.0:8686"
            playground = true
          }

          # Sources: Talos logs + K8s pod logs
          sources = {
            # Talos kernel logs via UDP
            talos_kernel_logs = {
              type = "socket"
              mode = "udp"
              address = "0.0.0.0:6050"
              max_length = 102400
              decoding = {
                codec = "json"
              }
              host_key = "__host"
            }

            # Talos service logs via UDP
            talos_service_logs = {
              type = "socket"
              mode = "udp"
              address = "0.0.0.0:6051"
              max_length = 102400
              decoding = {
                codec = "json"
              }
              host_key = "__host"
            }

            # Kubernetes pod logs
            k8s_logs = {
              type = "kubernetes_logs"
            }

            # Internal metrics for monitoring Vector itself
            internal_metrics = {
              type = "internal_metrics"
            }
          }

          # Transforms: Parse and enrich logs
          transforms = {
            # Parse K8s pod logs
            k8s_parser = {
              type = "remap"
              inputs = ["k8s_logs"]
              source = <<-EOT
                .log = parse_json(.message) ?? .message
                del(.message)
              EOT
            }

            # Enrich Talos kernel logs with metadata
            talos_kernel_transform = {
              type = "remap"
              inputs = ["talos_kernel_logs"]
              source = <<-EOT
                .log_type = "talos-kernel"
                .hostname = .__host
              EOT
            }

            # Enrich Talos service logs with metadata
            talos_service_transform = {
              type = "remap"
              inputs = ["talos_service_logs"]
              source = <<-EOT
                .log_type = "talos-service"
                .hostname = .__host
              EOT
            }
          }

          # Sinks: Send all logs to VictoriaLogs
          sinks = {
            # Prometheus exporter for Vector metrics
            prom_exporter = {
              type = "prometheus_exporter"
              address = "0.0.0.0:9090"
              inputs = ["internal_metrics"]
            }

            # VictoriaLogs sink for K8s pod logs
            vlogs_k8s = {
              type = "elasticsearch"
              inputs = ["k8s_parser"]
              mode = "bulk"
              api_version = "v8"
              compression = "gzip"
              endpoint = "http://vc-logs-victoria-logs-single-server.observability.svc.cluster.local:9428"
              healthcheck = {
                enabled = false
              }
              request = {
                headers = {
                  VL-Time-Field = "timestamp"
                  VL-Stream-Fields = "stream,kubernetes.pod_name,kubernetes.container_name,kubernetes.pod_namespace"
                  VL-Msg-Field = "message,msg,_msg,log.msg,log.message,log"
                  AccountID = "0"
                  ProjectID = "0"
                }
              }
            }

            # VictoriaLogs sink for Talos kernel logs
            vlogs_talos_kernel = {
              type = "elasticsearch"
              inputs = ["talos_kernel_transform"]
              mode = "bulk"
              api_version = "v8"
              compression = "gzip"
              endpoint = "http://vc-logs-victoria-logs-single-server.observability.svc.cluster.local:9428"
              healthcheck = {
                enabled = false
              }
              request = {
                headers = {
                  VL-Time-Field = "talos-time"
                  VL-Stream-Fields = "hostname,facility,log_type"
                  VL-Msg-Field = "msg"
                  AccountID = "0"
                  ProjectID = "0"
                }
              }
            }

            # VictoriaLogs sink for Talos service logs
            vlogs_talos_service = {
              type = "elasticsearch"
              inputs = ["talos_service_transform"]
              mode = "bulk"
              api_version = "v8"
              compression = "gzip"
              endpoint = "http://vc-logs-victoria-logs-single-server.observability.svc.cluster.local:9428"
              healthcheck = {
                enabled = false
              }
              request = {
                headers = {
                  VL-Time-Field = "talos-time"
                  VL-Stream-Fields = "hostname,talos-service,log_type"
                  VL-Msg-Field = "msg"
                  AccountID = "0"
                  ProjectID = "0"
                }
              }
            }
          }
        }
      }
    })
  ]

  depends_on = [
    kubernetes_namespace.observability,
    helm_release.victoria_metrics,
    kubernetes_storage_class.storage_replica_1
  ]
}

# Output Cilium installation status
output "cilium_status" {
  description = "Cilium installation status"
  value = {
    chart_version = helm_release.cilium.version
    namespace     = helm_release.cilium.namespace
    status        = helm_release.cilium.status
  }
}

# Output Longhorn installation status
output "longhorn_status" {
  description = "Longhorn installation status"
  value = {
    chart_version = helm_release.longhorn.version
    namespace     = helm_release.longhorn.namespace
    status        = helm_release.longhorn.status
  }
}

# Output cert-manager installation status
output "cert_manager_status" {
  description = "cert-manager installation status"
  value = {
    chart_version = helm_release.cert_manager.version
    namespace     = helm_release.cert_manager.namespace
    status        = helm_release.cert_manager.status
  }
}

# Output MySQL Operator installation status
output "mysql_operator_status" {
  description = "MySQL Operator installation status"
  value = {
    chart_version = helm_release.mysql_operator.version
    namespace     = helm_release.mysql_operator.namespace
    status        = helm_release.mysql_operator.status
  }
}

# Output VictoriaMetrics installation status
output "victoria_metrics_status" {
  description = "VictoriaMetrics monitoring stack installation status"
  value = {
    chart_version = helm_release.victoria_metrics.version
    namespace     = helm_release.victoria_metrics.namespace
    status        = helm_release.victoria_metrics.status
  }
}

# Output VictoriaLogs installation status
output "victoria_logs_status" {
  description = "VictoriaLogs log aggregation installation status"
  value = {
    chart_version = helm_release.victoria_logs.version
    namespace     = helm_release.victoria_logs.namespace
    status        = helm_release.victoria_logs.status
  }
}
