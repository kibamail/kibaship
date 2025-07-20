#cloud-config

# =============================================================================
# Worker Node Cloud-Init Configuration
# =============================================================================
# This cloud-init configuration sets up worker nodes with:
# 1. Ubuntu user setup
# 2. SSH key configuration for jump server access
# 3. Networking modules for Kubernetes
# 4. Storage modules for OpenEBS Mayastor
# 5. Hugepages configuration
# 6. GRUB configuration for persistent settings

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
# Kernel Modules and Configuration Files
# =============================================================================
write_files:
  # Networking modules for Kubernetes
  - path: /etc/modules-load.d/networking.conf
    content: |
      overlay
      br_netfilter
    permissions: '0644'

  # Storage modules for OpenEBS Mayastor
  - path: /etc/modules-load.d/replicated-pv.conf
    content: |
      nvme-tcp
      ext4
    permissions: '0644'

  # Network sysctl configuration for Kubernetes
  - path: /etc/sysctl.d/99-networking.conf
    content: |
      net.bridge.bridge-nf-call-iptables = 1
      net.bridge.bridge-nf-call-ip6tables = 1
      net.ipv4.ip_forward = 1
    permissions: '0644'

  # Hugepages sysctl configuration
  - path: /etc/sysctl.d/99-hugepages.conf
    content: |
      vm.nr_hugepages = 1024
    permissions: '0644'

  # GRUB configuration for hugepages
  - path: /etc/default/grub.d/hugepages.cfg
    content: |
      GRUB_CMDLINE_LINUX_DEFAULT="$GRUB_CMDLINE_LINUX_DEFAULT hugepagesz=2M hugepages=1024"
    permissions: '0644'

  # GRUB configuration for NVMe multipath
  - path: /etc/default/grub.d/nvme-multipath.cfg
    content: |
      GRUB_CMDLINE_LINUX_DEFAULT="$GRUB_CMDLINE_LINUX_DEFAULT nvme_core.multipath=Y"
    permissions: '0644'

# =============================================================================
# Boot Commands
# =============================================================================
bootcmd:
  # Load networking modules immediately
  - modprobe overlay
  - modprobe br_netfilter
  # Load storage modules immediately
  - modprobe nvme-tcp
  - modprobe ext4

# =============================================================================
# Run Commands
# =============================================================================
runcmd:
  # Apply sysctl settings
  - sysctl --system
  # Configure hugepages immediately
  - echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages
  # Update GRUB with new configuration
  - update-grub
  # Enable SSH service
  - systemctl enable ssh
  - systemctl start ssh
  # Log verification information
  - echo 'Worker node setup completed successfully' > /var/log/cloud-init-setup.log
  - echo 'Hugepages:' >> /var/log/cloud-init-setup.log
  - grep HugePages /proc/meminfo >> /var/log/cloud-init-setup.log
  - echo 'Storage modules:' >> /var/log/cloud-init-setup.log
  - lsmod | grep -E '(nvme|ext4)' >> /var/log/cloud-init-setup.log || echo 'Storage modules loaded' >> /var/log/cloud-init-setup.log

# =============================================================================
# Power State - Reboot Required
# =============================================================================
# Reboot to apply GRUB changes (hugepages, nvme multipath)
power_state:
  mode: reboot
  message: "Rebooting to apply GRUB changes (hugepages, nvme multipath)"
  timeout: 30
  condition: true

final_message: "Worker node setup completed successfully - rebooting for GRUB changes"
