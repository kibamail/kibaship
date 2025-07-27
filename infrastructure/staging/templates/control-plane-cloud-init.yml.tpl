#cloud-config

# =============================================================================
# Control Plane Node Cloud-Init Configuration
# =============================================================================
# This cloud-init configuration sets up control plane nodes with:
# 1. Ubuntu user setup
# 2. SSH key configuration for jump server access
# 3. Networking modules for Kubernetes

# =============================================================================
# User Configuration
# =============================================================================
users:
  - name: ubuntu
    groups: [adm, sudo]
    shell: /bin/bash
    sudo: ['ALL=(ALL) NOPASSWD:ALL']
    ssh_authorized_keys:
      - ${ssh_public_key}

# =============================================================================
# Kernel Modules
# =============================================================================
write_files:
  # Networking modules for Kubernetes
  - path: /etc/modules-load.d/networking.conf
    content: |
      overlay
      br_netfilter
    permissions: '0644'

  # Network sysctl configuration for Kubernetes
  - path: /etc/sysctl.d/99-networking.conf
    content: |
      net.bridge.bridge-nf-call-iptables = 1
      net.bridge.bridge-nf-call-ip6tables = 1
      net.ipv4.ip_forward = 1
    permissions: '0644'

# =============================================================================
# Boot Commands
# =============================================================================
bootcmd:
  # Load networking modules immediately
  - modprobe overlay
  - modprobe br_netfilter

# =============================================================================
# Run Commands
# =============================================================================
runcmd:
  # Apply sysctl settings
  - sysctl --system
  # Enable SSH service
  - systemctl enable ssh
  - systemctl start ssh
  # Log completion
  - echo 'Control plane node setup completed successfully' > /var/log/cloud-init-setup.log

final_message: "Control plane node setup completed successfully"
