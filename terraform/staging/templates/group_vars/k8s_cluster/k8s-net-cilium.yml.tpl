# =============================================================================
# KibaShip Staging Cilium CNI Configuration
# =============================================================================
# This file configures Cilium as the Container Network Interface (CNI) plugin
# with Gateway API support and Hubble observability enabled.
# Only variables that are actually supported by Kubespray are included.

# =============================================================================
# Tunnel Mode Configuration
# =============================================================================
# Disable tunneling for better performance in private networks
# Use native routing since all nodes are in the same private network
cilium_tunnel_mode: disabled

# =============================================================================
# Gateway API Support
# =============================================================================
# Enable Cilium Gateway API for advanced traffic management and ingress
cilium_gateway_api_enabled: true

# =============================================================================
# Hubble Observability
# =============================================================================
# Enable Hubble for network observability, monitoring, and security
cilium_enable_hubble: true

# Enable Hubble UI for visual network monitoring
cilium_enable_hubble_ui: true

# Configure Hubble metrics for Prometheus integration
cilium_hubble_metrics:
  - dns
  - drop
  - tcp
  - flow
  - icmp
  - http

# =============================================================================
# IPAM Configuration
# =============================================================================
# Use cluster-pool IPAM mode for better IP management
cilium_ipam_mode: cluster-pool

# =============================================================================
# Load Balancer Configuration
# =============================================================================
# Configure load balancer mode for service traffic
cilium_loadbalancer_mode: snat

# =============================================================================
# Monitoring Integration
# =============================================================================
# Enable Prometheus metrics collection
cilium_enable_prometheus: true

# =============================================================================
# Operator Configuration
# =============================================================================
# Configure Cilium operator for cluster management
cilium_operator_replicas: 2

# =============================================================================
# BPF Configuration
# =============================================================================
# Enable BPF masquerading for better performance
cilium_enable_bpf_masquerade: true

# =============================================================================
# Kube-proxy Replacement
# =============================================================================
# Enable kube-proxy replacement for better performance
# This allows Cilium to handle all service load balancing
cilium_kube_proxy_replacement: true
