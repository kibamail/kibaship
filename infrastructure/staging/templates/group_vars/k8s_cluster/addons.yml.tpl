# =============================================================================
# KibaShip Staging Cluster Addons Configuration
# =============================================================================
# This file disables all Kubespray addons except those explicitly needed.
# We manage most infrastructure components (load balancers, storage, monitoring)
# outside of Kubespray to maintain control and flexibility.

# =============================================================================
# Ingress Controllers - DISABLED
# =============================================================================
# We manage ingress through external load balancers and custom ingress setup
ingress_nginx_enabled: false
ingress_ambassador_enabled: false

# =============================================================================
# Service Mesh - DISABLED
# =============================================================================
# Service mesh will be configured separately if needed
istio_enabled: false

# =============================================================================
# Storage - DISABLED
# =============================================================================
# Storage is managed via Terraform (OpenEBS Mayastor) and custom configuration
local_path_provisioner_enabled: false
local_volume_provisioner_enabled: false
cephfs_provisioner_enabled: false
rbd_provisioner_enabled: false

# =============================================================================
# Monitoring and Logging - DISABLED
# =============================================================================
# Monitoring stack will be deployed separately for better control
prometheus_enabled: false
grafana_enabled: false
elasticsearch_enabled: false
fluentd_enabled: false

# =============================================================================
# Certificate Management - DISABLED
# =============================================================================
# Certificate management will be handled separately
cert_manager_enabled: false

# =============================================================================
# Registry - DISABLED
# =============================================================================
# Container registry will be configured separately if needed
registry_enabled: false
harbor_enabled: false

# =============================================================================
# Network Policy - DISABLED
# =============================================================================
# Network policies will be managed through Cilium directly
kube_router_enabled: false

# =============================================================================
# Dashboard - DISABLED
# =============================================================================
# Kubernetes dashboard will be deployed separately with proper security
dashboard_enabled: false

# =============================================================================
# DNS Autoscaler - ENABLED
# =============================================================================
# Keep DNS autoscaler for proper CoreDNS scaling
dns_autoscaler: true

# =============================================================================
# Gateway API - ENABLED
# =============================================================================
# Enable Gateway API CRDs for advanced traffic management
# This works with Cilium's Gateway API support
gateway_api_enabled: true

# =============================================================================
# Metrics Server - ENABLED
# =============================================================================
# Enable metrics server for basic cluster metrics (required for HPA)
metrics_server_enabled: true

# =============================================================================
# ArgoCD - ENABLED
# =============================================================================
# Enable ArgoCD for GitOps-based application deployment and management
# ArgoCD provides declarative, GitOps continuous delivery for Kubernetes
argocd_enabled: false
