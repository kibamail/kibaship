# Kind Cluster Bootstrap Configuration
# This template installs essential components for the Kind cluster

terraform {
  required_version = ">= 1.0"

  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.0.2"
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

resource "helm_release" "longhorn" {
  name             = "longhorn"
  repository       = "https://charts.longhorn.io"
  chart            = "longhorn"
  namespace        = "longhorn"
  version          = "v1.10.0"
  create_namespace = true
  cleanup_on_fail  = true
  replace          = true
  atomic           = true

  set = [
    {
      name = "defaultSettings.defaultDataPath"
      value = "/var/lib/longhorn"
    },
    {
      name = "persistence.defaultClass"
      value = false
    },
    {
      name = "defaultSettings.replicaAutoBalance"
      value = "strict-local"
    },
    {
      name = "persistence.volumeBindingMode"
      value = "WaitForFirstConsumer"
    },
    {
      name = "persistence.dataEngine",
      value = "v1"
    }
  ]

  depends_on = [helm_release.cilium]
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
    api_host           = local.api_host
    api_port           = local.api_port
    cilium_version     = helm_release.cilium.version
    longhorn_version   = helm_release.longhorn.version
    cert_manager_version = helm_release.cert_manager.version
  }
}
