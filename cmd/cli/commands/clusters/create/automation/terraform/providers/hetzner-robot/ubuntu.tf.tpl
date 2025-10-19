# ═══════════════════════════════════════════════════════════════════════════
# UBUNTU PREFLIGHT CONFIGURATION FOR MICROK8S ON HETZNER ROBOT SERVERS
# ═══════════════════════════════════════════════════════════════════════════
#
# PURPOSE:
# --------
# This Terraform configuration prepares freshly installed Ubuntu servers to
# run MicroK8s (Canonical's lightweight Kubernetes distribution). It handles
# all operating system-level prerequisites that must be in place BEFORE
# MicroK8s installation.
#
# EXECUTION PHASE:
# ----------------
# This runs as PHASE 2 of the cluster creation process, immediately after:
# - PHASE 1: OS Installation (Ubuntu via Hetzner Robot installimage)
#
# And before:
# - PHASE 3: MicroK8s installation and cluster bootstrap
#
# WHAT THIS CONFIGURATION DOES:
# ------------------------------
# For each server in the cluster, this configuration:
#
# 1. Installs system packages (snapd, networking tools, container prerequisites)
# 2. Disables swap memory (Kubernetes requirement)
# 3. Loads kernel modules for container networking (br_netfilter, overlay)
# 4. Tunes kernel parameters via sysctl (IP forwarding, netfilter, inotify, conntrack)
# 5. Enables snapd service (required for MicroK8s installation)
# 6. Configures VLAN networking for Hetzner vSwitch (private cluster network)
# 7. Sets up advanced policy routing for public subnet LoadBalancers (optional)
#
# DEPLOYMENT METHOD:
# ------------------
# Uses Terraform's null_resource with remote-exec provisioner to execute
# bash scripts over SSH. Each server is configured independently and in parallel.
#
# STATE MANAGEMENT:
# -----------------
# - Uses local backend to store state in terraform.tfstate
# - Reads provision phase outputs (SSH keys) via terraform_remote_state
# - Outputs include status for each server indicating successful configuration
#
# INFRASTRUCTURE DEPENDENCIES:
# -----------------------------
# Requires from previous phase:
# - SSH access to servers (port 22) with root user
# - SSH private key from provision phase
# - Server IPs and private network configuration
#
# IDEMPOTENCY:
# ------------
# All configuration steps are idempotent (safe to run multiple times).
# Most commands include '|| true' to prevent failures on reruns.
#
# ═══════════════════════════════════════════════════════════════════════════

terraform {
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }

  backend "local" {
    path = "terraform.tfstate"
  }
}

# ═══════════════════════════════════════════════════════════════════════════
# DATA SOURCE: PROVISION PHASE OUTPUTS
# ═══════════════════════════════════════════════════════════════════════════
# Read outputs from the provision phase to access:
# - SSH private key for server authentication
# - Network discovery results (if any)
# - Server configuration details
# ═══════════════════════════════════════════════════════════════════════════
data "terraform_remote_state" "provision" {
  backend = "local"
  config = {
    path = "${path.module}/../provision/terraform.tfstate"
  }
}

{{range .HetznerRobot.SelectedServers}}
# ═══════════════════════════════════════════════════════════════════════════
# SERVER CONFIGURATION: {{.Name}} ({{.IP}})
# ═══════════════════════════════════════════════════════════════════════════
# Role: {{.Role}}
# Private IP: {{.PrivateIP}}
#
# Configures this server with all Ubuntu prerequisites for MicroK8s:
# - System packages and kernel configuration
# - Network setup (VLAN for private cluster communication)
# - Advanced routing (if public subnet is allocated)
# ═══════════════════════════════════════════════════════════════════════════
resource "null_resource" "ubuntu_setup_{{.ID}}" {
  provisioner "remote-exec" {
    connection {
      type        = "ssh"
      user        = "root"
      host        = "{{.IP}}"
      private_key = data.terraform_remote_state.provision.outputs.ssh_private_key
      timeout     = "5m"
    }

    inline = [
      <<-BASH
        #!/bin/bash
        set -eux

        # ═══════════════════════════════════════════════════════════════════════
        # STEP 1: INSTALL SYSTEM DEPENDENCIES
        # ═══════════════════════════════════════════════════════════════════════
        # Install essential packages required for Kubernetes/MicroK8s operation:
        #
        # - snapd:              Package manager for MicroK8s installation
        # - socat:              Required for port forwarding and kubectl port-forward
        # - conntrack:          Connection tracking for kube-proxy and service mesh
        # - ca-certificates:    SSL/TLS certificate validation
        # - curl:               HTTP client for downloading resources
        # - jq:                 JSON processor for parsing cluster configs
        # - iptables:           Firewall rules for pod networking and services
        # - iproute2:           Advanced routing (ip command) for network setup
        # - apt-transport-https: Allows apt to retrieve packages via HTTPS
        # - gnupg:              GPG for package signature verification
        # - lsb-release:        Linux Standard Base version info
        # ═══════════════════════════════════════════════════════════════════════
        echo "[*] Updating apt and installing required packages"
        export DEBIAN_FRONTEND=noninteractive
        apt-get update -y
        apt-get install -y --no-install-recommends \
          snapd socat conntrack ca-certificates curl jq iptables iproute2 \
          apt-transport-https gnupg lsb-release

        # ═══════════════════════════════════════════════════════════════════════
        # STEP 2: DISABLE SWAP MEMORY
        # ═══════════════════════════════════════════════════════════════════════
        # Kubernetes requires swap to be disabled for the following reasons:
        #
        # 1. MEMORY MANAGEMENT: The kubelet (Kubernetes node agent) needs predictable
        #    memory allocation. Swap can cause pods to use more memory than their
        #    limits, breaking resource guarantees and Quality of Service (QoS).
        #
        # 2. PERFORMANCE: Swapping pod memory to disk causes severe performance
        #    degradation and unpredictable latency, which is unacceptable for
        #    production workloads.
        #
        # 3. SCHEDULING: The Kubernetes scheduler makes placement decisions based
        #    on available memory. Swap would invalidate these calculations.
        #
        # This configuration:
        # - Immediately disables all active swap with 'swapoff -a'
        # - Comments out swap entries in /etc/fstab to prevent re-enabling on reboot
        # - Creates a backup of fstab as fstab.bak for safety
        # ═══════════════════════════════════════════════════════════════════════
        echo "[*] Disable swap"
        swapoff -a || true
        sed -i.bak -r 's/(^.*\sswap\s+swap\s+.*$)/# \1/g' /etc/fstab || true

        # ═══════════════════════════════════════════════════════════════════════
        # STEP 3: CONFIGURE KERNEL MODULES FOR CONTAINER NETWORKING
        # ═══════════════════════════════════════════════════════════════════════
        echo "[*] Configure kernel modules to load on boot"
        cat <<'MODEOF' > /etc/modules-load.d/k8s.conf
# ══════════════════════════════════════════════════════════════════════════════
# KERNEL MODULES FOR KUBERNETES/MICROK8S
# ══════════════════════════════════════════════════════════════════════════════
# File: /etc/modules-load.d/k8s.conf
# Auto-generated by: Kibaship Cluster Provisioning
# Purpose: Load kernel modules required for Kubernetes container networking
#
# These modules are loaded at boot by systemd-modules-load.service and are
# CRITICAL for proper Kubernetes operation. Do not remove unless you understand
# the consequences.
# ══════════════════════════════════════════════════════════════════════════════

# ─────────────────────────────────────────────────────────────────────────────
# MODULE: br_netfilter
# ─────────────────────────────────────────────────────────────────────────────
# Enables iptables to process traffic that traverses Linux network bridges.
#
# WHY KUBERNETES NEEDS THIS:
# - Pod-to-pod communication often uses Linux bridges (virtual switches)
# - CNI plugins (Calico, Cilium, Flannel) require iptables rules to work on
#   bridged traffic for NetworkPolicy enforcement and service load balancing
# - kube-proxy uses iptables for Service routing and load balancing
#
# WITHOUT THIS MODULE:
# - iptables rules won't apply to container network traffic
# - Pod networking will be broken
# - Kubernetes Services won't work
# - NetworkPolicies will fail to enforce
# ─────────────────────────────────────────────────────────────────────────────
br_netfilter

# ─────────────────────────────────────────────────────────────────────────────
# MODULE: overlay
# ─────────────────────────────────────────────────────────────────────────────
# Enables OverlayFS, a union filesystem for efficient container image storage.
#
# HOW CONTAINER RUNTIMES USE THIS:
# - Container images are built in layers (base OS, dependencies, application)
# - OverlayFS allows multiple containers to share common layers (saves space)
# - Copy-on-write: Containers can modify files without affecting base image
# - Fast container startup: No need to copy entire filesystem
#
# STORAGE BENEFITS:
# - 10 containers from same image = 1 copy of base layers on disk
# - Only unique data per container consumes additional space
# - Significantly reduces disk usage in multi-container environments
# ─────────────────────────────────────────────────────────────────────────────
overlay

# ══════════════════════════════════════════════════════════════════════════════
# VERIFICATION:
# ──────────────
# Check if modules are loaded:
#   lsmod | grep -E 'br_netfilter|overlay'
#
# Manually load modules (immediate effect, not persistent):
#   modprobe br_netfilter
#   modprobe overlay
#
# TROUBLESHOOTING:
# ────────────────
# If modules fail to load on boot, check:
#   systemctl status systemd-modules-load.service
#   journalctl -u systemd-modules-load.service
# ══════════════════════════════════════════════════════════════════════════════
MODEOF
        modprobe br_netfilter overlay || true

        # ═══════════════════════════════════════════════════════════════════════
        # STEP 4: CONFIGURE KERNEL NETWORK PARAMETERS (SYSCTL)
        # ═══════════════════════════════════════════════════════════════════════
        echo "[*] Configure Kubernetes-friendly sysctl settings"
        cat <<'SYSCTLEOF' > /etc/sysctl.d/99-k8s.conf
# ══════════════════════════════════════════════════════════════════════════════
# KERNEL TUNING PARAMETERS FOR KUBERNETES/MICROK8S
# ══════════════════════════════════════════════════════════════════════════════
# File: /etc/sysctl.d/99-k8s.conf
# Auto-generated by: Kibaship Cluster Provisioning
# Purpose: Kernel parameter tuning required for Kubernetes networking and operation
#
# These settings are applied at boot by systemd-sysctl.service and are CRITICAL
# for proper Kubernetes cluster operation. The high priority (99-) ensures these
# settings override any conflicting defaults.
#
# Apply changes immediately: sysctl --system
# View current values: sysctl -a | grep <parameter>
# ══════════════════════════════════════════════════════════════════════════════

# ─────────────────────────────────────────────────────────────────────────────
# NETWORKING: IP Forwarding
# ─────────────────────────────────────────────────────────────────────────────
# Enable IPv4 packet forwarding at the kernel level
# Default: 0 (disabled)
# Kubernetes requirement: CRITICAL
#
# WHY THIS IS NEEDED:
# - Allows the Linux kernel to forward packets between network interfaces
# - Required for pod-to-pod communication across different nodes
# - Enables NAT translation for pod egress traffic to external networks
# - Used by kube-proxy for Service routing and load balancing
#
# WITHOUT THIS:
# - Pods on different nodes cannot communicate
# - Services will not work
# - Cluster networking will be completely broken
# ─────────────────────────────────────────────────────────────────────────────
net.ipv4.ip_forward=1

# ─────────────────────────────────────────────────────────────────────────────
# NETWORKING: Bridge Netfilter (IPv4)
# ─────────────────────────────────────────────────────────────────────────────
# Force bridged IPv4 traffic to traverse iptables chains
# Default: Varies (often 0 or 1 depending on distribution)
# Kubernetes requirement: CRITICAL
#
# WHY THIS IS NEEDED:
# - Kubernetes networking relies on Linux bridges for pod communication
# - CNI plugins (Calico, Cilium, Flannel) use iptables for NetworkPolicy
# - kube-proxy uses iptables for Service load balancing
# - This ensures iptables rules apply to traffic crossing bridges
#
# WITHOUT THIS:
# - NetworkPolicies won't be enforced (security breach!)
# - Service load balancing may fail
# - Pod network isolation will be broken
# ─────────────────────────────────────────────────────────────────────────────
net.bridge.bridge-nf-call-iptables=1

# ─────────────────────────────────────────────────────────────────────────────
# NETWORKING: Bridge Netfilter (IPv6)
# ─────────────────────────────────────────────────────────────────────────────
# Force bridged IPv6 traffic to traverse ip6tables chains
# Default: Varies (often 0 or 1 depending on distribution)
# Kubernetes requirement: Required for IPv6 support
#
# Same as above but for IPv6 traffic. Even if you're not using IPv6 now,
# enabling this ensures the cluster is ready for dual-stack networking.
# ─────────────────────────────────────────────────────────────────────────────
net.bridge.bridge-nf-call-ip6tables=1

# ─────────────────────────────────────────────────────────────────────────────
# FILESYSTEM: Inotify Instance Limit
# ─────────────────────────────────────────────────────────────────────────────
# Maximum number of inotify instances (file watchers) per user
# Default: 128
# Kubernetes requirement: 8192+ (increased from default)
#
# WHY THIS IS NEEDED:
# - Kubernetes components watch many files/directories simultaneously:
#   * kubelet: watches ConfigMaps, Secrets, pod manifests
#   * controller-manager: watches resource definitions
#   * Custom controllers: watch CRDs and resources
# - Each watch requires an inotify instance
# - With hundreds of resources, default limit is quickly exhausted
#
# WITHOUT THIS:
# - Error: "too many open files" when creating new watches
# - ConfigMap/Secret updates won't propagate to pods
# - Controllers will fail to watch resources
# - Cluster functionality will degrade over time
# ─────────────────────────────────────────────────────────────────────────────
fs.inotify.max_user_instances=8192

# ─────────────────────────────────────────────────────────────────────────────
# FILESYSTEM: Inotify Watch Limit
# ─────────────────────────────────────────────────────────────────────────────
# Maximum number of files each inotify instance can watch
# Default: 8192
# Kubernetes requirement: 524288+ (64x increase from default)
#
# WHY THIS IS NEEDED:
# - Each mounted ConfigMap, Secret, or volume creates multiple watches
# - In a cluster with 100 pods × 5 mounted configs = 500+ watches minimum
# - Default limit is hit very quickly in production clusters
#
# REAL-WORLD EXAMPLE:
# - 200 pods across the cluster
# - Each pod mounts 3 ConfigMaps + 2 Secrets + service account token
# - 200 × 6 = 1200 watches just for configs (plus system watches)
# - Default 8192 limit: easily exceeded, causing failures
#
# WITHOUT THIS:
# - Error: "inotify watch limit reached"
# - New pods fail to mount ConfigMaps/Secrets
# - Existing pods won't see config updates
# - Deployments and rollouts will fail
# ─────────────────────────────────────────────────────────────────────────────
fs.inotify.max_user_watches=524288

# ─────────────────────────────────────────────────────────────────────────────
# NETWORKING: Connection Tracking Table Size
# ─────────────────────────────────────────────────────────────────────────────
# Maximum number of connections to track in the conntrack table
# Default: 65536 (varies by system memory)
# Kubernetes requirement: 524288+ (8x increase from typical default)
#
# WHY THIS IS NEEDED:
# - Linux connection tracking (conntrack) maintains state for all connections
# - In Kubernetes, conntrack tracks:
#   * Service connections (each client → Service connection)
#   * Pod-to-pod connections
#   * External ingress connections
#   * NAT translations for outbound traffic
# - A busy cluster generates thousands of concurrent connections
#
# REAL-WORLD SCENARIO:
# - 50 pods running HTTP services
# - Each service receives 100 requests/sec
# - Average connection duration: 2 seconds
# - Concurrent connections: 50 × 100 × 2 = 10,000+ (excluding pod-to-pod)
# - Default 65536: May be sufficient for small clusters, inadequate for production
#
# WITHOUT THIS (when table is full):
# - New connections are dropped silently
# - Services become unreachable intermittently
# - Error logs: "nf_conntrack: table full, dropping packet"
# - Application timeouts and failures
# ─────────────────────────────────────────────────────────────────────────────
net.netfilter.nf_conntrack_max=524288

# ══════════════════════════════════════════════════════════════════════════════
# VERIFICATION AND TROUBLESHOOTING
# ══════════════════════════════════════════════════════════════════════════════
#
# View current value of a parameter:
#   sysctl net.ipv4.ip_forward
#   sysctl -a | grep inotify
#
# Apply changes from this file immediately (without reboot):
#   sysctl --system
#   OR
#   sysctl -p /etc/sysctl.d/99-k8s.conf
#
# Temporarily change a value (lost on reboot):
#   sysctl -w net.ipv4.ip_forward=1
#
# Check if settings are applied at boot:
#   systemctl status systemd-sysctl.service
#   journalctl -u systemd-sysctl.service
#
# Monitor conntrack table usage (current/max):
#   cat /proc/sys/net/netfilter/nf_conntrack_count
#   cat /proc/sys/net/netfilter/nf_conntrack_max
#   # If count approaches max, consider increasing nf_conntrack_max
#
# Monitor inotify usage:
#   find /proc/*/fd -lname 'anon_inode:inotify' | wc -l
#   # Shows total inotify instances in use
#
# ══════════════════════════════════════════════════════════════════════════════
SYSCTLEOF
        sysctl --system || true

        # ═══════════════════════════════════════════════════════════════════════
        # STEP 5: ENABLE SNAPD SERVICE
        # ═══════════════════════════════════════════════════════════════════════
        # snapd is Ubuntu's package management system for snaps (containerized apps).
        # MicroK8s is distributed as a snap package, which provides:
        #
        # - Automatic updates with rollback capability
        # - Isolation from the host system
        # - Consistent installation across Ubuntu versions
        # - Easy installation of a complete Kubernetes cluster in one package
        #
        # The 'enable --now' command both enables snapd to start at boot AND
        # starts it immediately, ensuring it's ready for MicroK8s installation.
        # ═══════════════════════════════════════════════════════════════════════
        echo "[*] Ensure snapd is enabled"
        systemctl enable --now snapd || true

        # ═══════════════════════════════════════════════════════════════════════
        # STEP 6: CONFIGURE PRIVATE NETWORKING VIA VLAN (HETZNER VSWITCH)
        # ═══════════════════════════════════════════════════════════════════════
        # File: /etc/netplan/60-vswitch-vlan<VLAN_ID>.yaml
        #
        # Hetzner Robot servers use VLANs for private networking via vSwitch.
        # This configuration creates a VLAN interface for secure pod-to-pod
        # communication between cluster nodes without using the public internet.
        #
        # ARCHITECTURE OVERVIEW:
        # ----------------------
        # Physical NIC (e.g., enp5s0) ─┬─→ Public network (DHCP)
        #                               │
        #                               └─→ VLAN interface (e.g., enp5s0.4000)
        #                                   └─→ Private cluster network (static IP)
        #
        # WHY VLAN FOR KUBERNETES?
        # ------------------------
        # 1. SECURITY: Pod-to-pod traffic stays on private network, isolated from
        #    public internet. Reduces attack surface.
        #
        # 2. PERFORMANCE: Lower latency and higher throughput compared to routing
        #    through public network. No NAT overhead.
        #
        # 3. COST: Hetzner vSwitch traffic is free (no bandwidth charges), while
        #    public traffic may be metered.
        #
        # 4. NETWORK POLICIES: CNI plugins can enforce network policies on the
        #    private network without interfering with public access.
        #
        # MTU SETTING (1400 bytes):
        # -------------------------
        # VLAN adds 4 bytes of overhead to Ethernet frames. Standard MTU is 1500.
        # Setting VLAN MTU to 1400 ensures:
        # - No packet fragmentation (which hurts performance)
        # - Kubernetes pods can use 1400 MTU without issues
        # - Compatible with overlay networks (which add more encapsulation)
        #
        # NETPLAN CONFIGURATION DETAILS:
        # -------------------------------
        # - version 2: Uses modern netplan syntax
        # - renderer: networkd: Uses systemd-networkd (more reliable than NetworkManager)
        # - Primary interface keeps DHCP for public connectivity
        # - VLAN interface uses static IP from allocated private subnet
        # - Empty routes array: Prevents default route via private network
        # ═══════════════════════════════════════════════════════════════════════
        echo "[*] Configure private networking via netplan VLAN"
        VLAN_ID={{$.HetznerRobot.VLANID}}
        PRIV_IP="{{.PrivateIP}}"
        SUBNET_CIDR="{{$.HetznerRobot.NetworkConfig.ClusterVSwitchSubnetIPRange}}"

        if [ -n "$PRIV_IP" ] && [ -n "$SUBNET_CIDR" ] && [ -n "$VLAN_ID" ]; then
          PREFIXLEN=$(echo "$SUBNET_CIDR" | awk -F'/' '{print $2}')
          if [ -z "$PREFIXLEN" ]; then
            echo "[!] Could not parse prefix length from $SUBNET_CIDR; defaulting to 24"
            PREFIXLEN=24
          fi

          # Determine primary interface (public uplink) if not provided
          PRIMARY_IF=$(ip route 2>/dev/null | awk '/default/ {print $5; exit}')
          if [ -z "$PRIMARY_IF" ]; then
            PRIMARY_IF=$(ip -o link show | awk -F': ' '{print $2}' | grep -Ev '^(lo|docker|cni|flannel|kube|virbr|veth|br-)' | head -n1)
          fi

          if [ -z "$PRIMARY_IF" ]; then
            echo "[!] Unable to determine primary interface; skipping private network config"
          else
            echo "[*] Using primary interface: $PRIMARY_IF; VLAN: $VLAN_ID; IP: $PRIV_IP/$PREFIXLEN"

            mkdir -p /etc/netplan
            NETPLAN_FILE="/etc/netplan/60-vswitch-vlan$${VLAN_ID}.yaml"
            cat > "$NETPLAN_FILE" <<NETPLAN
# ══════════════════════════════════════════════════════════════════════════════
# HETZNER VSWITCH VLAN CONFIGURATION FOR KUBERNETES PRIVATE NETWORKING
# ══════════════════════════════════════════════════════════════════════════════
# File: /etc/netplan/60-vswitch-vlan$${VLAN_ID}.yaml
# Auto-generated by: Kibaship Cluster Provisioning
# Purpose: Configure VLAN interface for private cluster communication via Hetzner vSwitch
#
# NETWORK ARCHITECTURE:
# ─────────────────────
#                                    ┌─────────────────────────┐
#                                    │   Physical Server       │
#                                    │                         │
#   ┌────────────────────┐           │  ┌──────────────────┐   │
#   │  Public Internet   │◄──────────┼──┤  $${PRIMARY_IF}       │   │ (DHCP)
#   └────────────────────┘           │  │  (Primary NIC)   │   │
#                                    │  └────────┬─────────┘   │
#                                    │           │             │
#   ┌────────────────────┐           │  ┌────────▼─────────┐   │
#   │  Hetzner vSwitch   │◄──────────┼──┤ $${PRIMARY_IF}.$${VLAN_ID}  │   │ (Static: $${PRIV_IP})
#   │  (Private Network) │           │  │ (VLAN Interface) │   │
#   └────────────────────┘           │  └──────────────────┘   │
#         │                          └─────────────────────────┘
#         │
#         ├──► Other cluster nodes (private IPs)
#         ├──► Pod-to-pod traffic
#         └──► Internal Kubernetes communication
#
# WHY VLAN FOR KUBERNETES?
# ────────────────────────
# 1. SECURITY: Pod traffic never touches public internet (isolated network)
# 2. PERFORMANCE: Lower latency, higher throughput vs. public routing
# 3. COST: Hetzner vSwitch traffic is FREE (no bandwidth charges)
# 4. NETWORK POLICIES: CNI can enforce policies without affecting public access
#
# CONFIGURATION DETAILS:
# ──────────────────────
# - version 2: Modern netplan syntax
# - renderer: networkd: Uses systemd-networkd (reliable, lightweight)
# - Primary interface: Keeps DHCP for public connectivity (management, updates)
# - VLAN interface: Static IP from private subnet (Kubernetes networking)
# - MTU 1400: Accounts for VLAN overhead (4 bytes) + prevents fragmentation
# - Empty routes: No default gateway via VLAN (public interface handles internet)
#
# ══════════════════════════════════════════════════════════════════════════════

network:
  version: 2
  renderer: networkd

  # ─────────────────────────────────────────────────────────────────────────────
  # PRIMARY NETWORK INTERFACE (PUBLIC)
  # ─────────────────────────────────────────────────────────────────────────────
  # Maintains DHCP configuration for:
  # - Public IP assignment
  # - Default gateway for internet access
  # - DNS servers
  # - System management and updates
  # ─────────────────────────────────────────────────────────────────────────────
  ethernets:
    $${PRIMARY_IF}:
      dhcp4: true   # Obtain IPv4 address via DHCP from Hetzner
      dhcp6: false  # Disable IPv6 DHCP (not typically used with Hetzner Robot)

  # ─────────────────────────────────────────────────────────────────────────────
  # VLAN INTERFACE (PRIVATE KUBERNETES NETWORK)
  # ─────────────────────────────────────────────────────────────────────────────
  # Creates a VLAN sub-interface tagged with VLAN ID $${VLAN_ID}
  # This interface connects to Hetzner vSwitch for private cluster communication
  #
  # VLAN TAG: $${VLAN_ID}
  #   - Assigned by Hetzner vSwitch configuration
  #   - All servers in cluster use same VLAN ID
  #   - Traffic is isolated from other VLANs and public internet
  #
  # MTU 1400 EXPLANATION:
  #   - Standard Ethernet MTU: 1500 bytes
  #   - VLAN header overhead: 4 bytes (802.1Q tag)
  #   - Setting VLAN MTU to 1400 ensures:
  #     * No fragmentation of packets
  #     * Room for additional encapsulation (CNI overlay networks)
  #     * Optimal performance for Kubernetes pod networking
  #
  # STATIC IP: $${PRIV_IP}/$${PREFIXLEN}
  #   - Permanently assigned to this server for cluster membership
  #   - Used by Kubernetes for pod-to-pod and node-to-node communication
  #   - Must be unique across all cluster nodes
  #   - Subnet: All cluster nodes are in same private subnet
  #
  # ROUTES: [] (empty)
  #   - No default gateway configured on VLAN interface
  #   - Public interface handles default route (internet traffic)
  #   - Only direct subnet routes are used (auto-configured by Linux)
  #   - This prevents routing conflicts between public and private networks
  # ─────────────────────────────────────────────────────────────────────────────
  vlans:
    $${PRIMARY_IF}.$${VLAN_ID}:
      id: $${VLAN_ID}              # VLAN tag ID
      link: $${PRIMARY_IF}         # Parent interface (trunk port)
      mtu: 1400                # Reduced MTU for VLAN overhead
      addresses:
        - $${PRIV_IP}/$${PREFIXLEN}  # Static private IP for this node
      routes: []               # Empty: use public interface for default gateway

# ══════════════════════════════════════════════════════════════════════════════
# APPLYING CHANGES
# ══════════════════════════════════════════════════════════════════════════════
#
# Test configuration syntax (dry run):
#   netplan generate
#
# Apply configuration (brings up interface):
#   netplan apply
#
# Check interface status:
#   ip addr show $${PRIMARY_IF}.$${VLAN_ID}
#   ip -c addr show  # Colorized output
#
# Verify VLAN tagging:
#   cat /proc/net/vlan/$${PRIMARY_IF}.$${VLAN_ID}
#
# Test connectivity to other cluster nodes:
#   ping <other-node-private-ip>
#
# View routing table:
#   ip route show
#   # Should see direct route for private subnet via VLAN interface
#   # Should see default route (0.0.0.0/0) via public interface
#
# Troubleshooting:
#   journalctl -u systemd-networkd
#   networkctl status $${PRIMARY_IF}.$${VLAN_ID}
#
# ══════════════════════════════════════════════════════════════════════════════
NETPLAN

            echo "[*] Applying netplan configuration: $NETPLAN_FILE"
            netplan generate
            sudo netplan apply

            # Verify address assigned and display interface details (colored output if supported)
            sleep 1
            if ip -4 addr show dev $${PRIMARY_IF}.$${VLAN_ID} | grep -q "$${PRIV_IP}/"; then
              echo "[+] VLAN interface up with IP $${PRIV_IP}"
            else
              echo "[!] VLAN interface not found or IP not set yet"
            fi
            # Show details using colorized output
            ip -c a show $${PRIMARY_IF}.$${VLAN_ID} || ip -c a show enp5s0.$${VLAN_ID} || true
          fi
        else
          echo "[!] Missing parameters for private networking (VLAN_ID=$VLAN_ID, PRIV_IP=$PRIV_IP, SUBNET_CIDR=$SUBNET_CIDR); skipping"
        fi

        # ═══════════════════════════════════════════════════════════════════════
        # STEP 7: ADVANCED POLICY ROUTING FOR PUBLIC SUBNET (OPTIONAL)
        # ═══════════════════════════════════════════════════════════════════════
        # File: /etc/iproute2/rt_tables (table definition)
        #       + ip rules and routes (runtime configuration)
        #
        # This section configures Linux Policy-Based Routing (PBR) to handle
        # scenarios where you have a Hetzner public subnet allocated to your
        # vSwitch for Kubernetes LoadBalancer services.
        #
        # PROBLEM STATEMENT:
        # ------------------
        # By default, Linux uses a single routing table. When you have:
        # 1. A public IP on the physical interface (for management)
        # 2. A public subnet on the VLAN interface (for LoadBalancers)
        #
        # ...routing conflicts occur. Traffic to/from the public subnet may try
        # to use the wrong interface, causing asymmetric routing or packet loss.
        #
        # SOLUTION: POLICY-BASED ROUTING
        # -------------------------------
        # We create a separate routing table called 'vswitch' (ID: 1) that
        # contains routes specific to the public subnet on the VLAN interface.
        # Then we add policy rules that say:
        #
        # "If traffic is destined TO the public subnet, use the vswitch table"
        # "If traffic originates FROM the public subnet, use the vswitch table"
        #
        # ROUTING TABLE STRUCTURE:
        # ------------------------
        # Main table (default):
        #   - Default route via physical interface gateway (public internet)
        #   - Routes for physical interface subnet
        #
        # vSwitch table (ID: 1):
        #   - Default route via vSwitch gateway (first host in public subnet)
        #   - Used only for traffic matching policy rules
        #
        # GATEWAY CALCULATION:
        # --------------------
        # For a public subnet like 192.0.2.0/29, Hetzner configures:
        # - 192.0.2.1: Gateway (first usable host)
        # - 192.0.2.2-6: Usable IPs for your LoadBalancers
        # - 192.0.2.7: Broadcast
        #
        # The Python script calculates the first host IP (gateway) from the CIDR.
        #
        # POLICY RULES:
        # -------------
        # ip rule add to <PUBLIC_SUBNET> lookup vswitch
        #   → Routes packets DESTINED to the public subnet via vswitch table
        #
        # ip rule add from <PUBLIC_SUBNET> lookup vswitch
        #   → Routes packets ORIGINATING from public subnet IPs via vswitch table
        #
        # This ensures bidirectional routing consistency and prevents:
        # - Asymmetric routing (reply packets taking different path than requests)
        # - Source address validation failures
        # - LoadBalancer traffic being blocked by reverse path filtering
        #
        # WHY THIS MATTERS FOR KUBERNETES:
        # ---------------------------------
        # Kubernetes LoadBalancer services with external IPs from the public
        # subnet MUST have symmetric routing. Without policy routing:
        # - Incoming traffic arrives on VLAN interface
        # - Reply traffic might try to leave via physical interface (wrong!)
        # - Connection fails due to RPF (Reverse Path Filtering) or asymmetric routing
        #
        # With policy routing, all traffic to/from LoadBalancer IPs uses the
        # correct interface and gateway, ensuring proper operation.
        # ═══════════════════════════════════════════════════════════════════════
        echo "[*] Configure public subnet policy routing (if provided)"
        PUBLIC_SUBNET="{{$.HetznerRobot.PublicSubnet}}"
        if [ -n "$PUBLIC_SUBNET" ]; then
          VLAN_IF="$${PRIMARY_IF}.$${VLAN_ID}"

          echo "[*] Ensure routing table alias 'vswitch' exists"
          grep -qE '(^|\s)vswitch$' /etc/iproute2/rt_tables || echo "1 vswitch" >> /etc/iproute2/rt_tables

          echo "[*] Compute public subnet gateway (first host in $PUBLIC_SUBNET)"
          GATEWAY=$(python3 - <<PY
import ipaddress
net = ipaddress.ip_network("$PUBLIC_SUBNET", strict=False)
hosts = list(net.hosts())
print(str(hosts[0]) if hosts else "")
PY
)

          if [ -z "$GATEWAY" ]; then
            echo "[!] Could not determine gateway for $PUBLIC_SUBNET; skipping vswitch default route"
          else
            echo "[*] Setting default route in 'vswitch' table via $GATEWAY on $VLAN_IF"
            ip -4 route replace default via "$GATEWAY" dev "$VLAN_IF" table vswitch || true
          fi

          echo "[*] Add policy rules for traffic to/from $PUBLIC_SUBNET"
          ip -4 rule show | grep -q "to $PUBLIC_SUBNET .*lookup vswitch" || ip -4 rule add to "$PUBLIC_SUBNET" lookup vswitch || true
          ip -4 rule show | grep -q "from $PUBLIC_SUBNET .*lookup vswitch" || ip -4 rule add from "$PUBLIC_SUBNET" lookup vswitch || true

          echo "[*] Current policy rules and vswitch routes:"
          ip -4 rule show || true
          ip -4 route show table vswitch || true
        else
          echo "[!] No public subnet provided; skipping public subnet routing config"
        fi

        echo "[*] Preflight check complete for MicroK8s installation"
      BASH
    ]
  }

  triggers = {
    server_ip   = "{{.IP}}"
    server_name = "{{.Name}}"
  }
}
{{end}}

{{range .HetznerRobot.SelectedServers}}
# ═══════════════════════════════════════════════════════════════════════════
# OUTPUT: UBUNTU SETUP STATUS FOR {{.Name}}
# ═══════════════════════════════════════════════════════════════════════════
# Provides confirmation that Ubuntu preflight configuration completed
# successfully for this server. The output can be queried via:
#   terraform output ubuntu_setup_{{.ID}}_status
# ═══════════════════════════════════════════════════════════════════════════
output "ubuntu_setup_{{.ID}}_status" {
  value       = {
    name = "{{.Name}}"
    ip   = "{{.IP}}"
    ok   = true
  }
  description = "Ubuntu preflight setup status for server {{.Name}}"
}
{{end}}

# ═══════════════════════════════════════════════════════════════════════════
# SUMMARY OF FILES CREATED/MODIFIED ON EACH SERVER
# ═══════════════════════════════════════════════════════════════════════════
#
# /etc/modules-load.d/k8s.conf
#   → Kernel modules to load at boot (br_netfilter, overlay)
#
# /etc/sysctl.d/99-k8s.conf
#   → Kernel parameters for Kubernetes networking and resource limits
#
# /etc/fstab (modified)
#   → Swap entries commented out to disable swap permanently
#
# /etc/netplan/60-vswitch-vlan<VLAN_ID>.yaml
#   → Network configuration for VLAN interface on Hetzner vSwitch
#
# /etc/iproute2/rt_tables (modified)
#   → Custom routing table definition for vswitch (if public subnet configured)
#
# Runtime configurations (not persisted to files):
#   → ip rules for policy-based routing
#   → ip routes in vswitch table
#   → Note: These need to be made persistent via netplan or systemd-networkd
#          in a production environment for survival across reboots
#
# ═══════════════════════════════════════════════════════════════════════════
