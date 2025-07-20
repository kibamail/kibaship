# =============================================================================
# KibaShip Staging Cluster Configuration
# =============================================================================
# This file contains the core configuration overrides for the KibaShip staging
# Kubernetes cluster. Only essential settings that differ from Kubespray
# defaults are included here.

# =============================================================================
# Load Balancer Configuration
# =============================================================================
# Configure the external load balancer for the Kubernetes API server.
# This points to the Hetzner Cloud load balancer provisioned by Terraform.
loadbalancer_apiserver:
  address: ${k8s_api_public_ip}  # Public IP of the Kubernetes API load balancer
  port: 6443                     # Standard Kubernetes API server port

# =============================================================================
# Time Synchronization
# =============================================================================
# Enable NTP to ensure all cluster nodes have synchronized time.
# This is critical for certificate validation and etcd operation.
ntp_enabled: true

# =============================================================================
# Logging Configuration
# =============================================================================
# Enable detailed logging for troubleshooting during cluster setup.
# Shows sensitive information in logs - disable in production.
unsafe_show_logs: true

# =============================================================================
# Cloud Provider Configuration
# =============================================================================
# Disable all cloud provider integrations to prevent automatic cloud resource
# management. We manage load balancers and storage manually via Terraform.
cloud_provider: ""

# =============================================================================
# Container Runtime Configuration
# =============================================================================
# Use containerd as the container runtime (Kubespray default, but explicit)
container_manager: containerd

# =============================================================================
# Cluster Networking
# =============================================================================
# Configure cluster networking to work with our private network setup
kube_network_plugin: cilium                    # Use Cilium as the CNI plugin
kube_pods_subnet: ${kube_pods_subnet}          # Pod CIDR range
kube_service_addresses: ${kube_service_addresses}  # Service CIDR range

# =============================================================================
# DNS Configuration
# =============================================================================
# Use CoreDNS for cluster DNS resolution
dns_mode: coredns
cluster_name: ${cluster_name}
