# =============================================================================
# KibaShip Staging Kubernetes Cluster Configuration
# =============================================================================
# This file contains core Kubernetes cluster settings that override Kubespray
# defaults. Only essential configurations that differ from defaults are included.

# =============================================================================
# Kubernetes User and Ownership Configuration
# =============================================================================
# Set Kubernetes file ownership to root to avoid permission issues with
# CNI plugins and system containers like Cilium
kube_owner: root

# =============================================================================
# Kubernetes Version Configuration
# =============================================================================
# Use a stable Kubernetes version that's well-tested with our CNI setup
kube_version: 1.31.2

# =============================================================================
# Cluster Hardening
# =============================================================================
# Enable basic security hardening
kube_encrypt_secret_data: true   # Encrypt secrets at rest in etcd
