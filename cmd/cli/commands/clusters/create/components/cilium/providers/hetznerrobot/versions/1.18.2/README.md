# Cilium 1.18.2 for Hetzner Robot - Manifest Generation

This directory contains the Cilium 1.18.2 manifest specifically configured for Hetzner Robot dedicated servers. This configuration is optimized for bare metal environments with specific networking and security requirements.

## Overview

Cilium for Hetzner Robot provides:
- **Bare Metal Networking** - Optimized for dedicated server environments
- **VXLAN Tunneling** - Overlay networking for multi-server clusters
- **Kubernetes IPAM** - Native Kubernetes IP address management
- **SNAT Load Balancing** - Source NAT for external traffic
- **Enhanced Security** - Extended capabilities for bare metal
- **Gateway API Support** - Modern ingress capabilities

## Hetzner Robot Specific Configuration

This manifest is specifically tailored for Hetzner Robot dedicated servers with the following optimizations:

### Networking Configuration
- **Custom API Server** - `localhost:7445` for secure local communication
- **VXLAN Tunneling** - Overlay networking across dedicated servers
- **Kubernetes IPAM** - Native IP address management
- **SNAT Load Balancing** - Proper external traffic handling

### Security Configuration
- **Extended Capabilities** - Additional privileges for bare metal operations
- **Custom cgroup Mounting** - Optimized for dedicated server environments
- **BPF Masquerading** - Enhanced network address translation

## Manifest Generation Process

### Prerequisites

1. **Helm installed** - Required for manifest generation
2. **Cilium Helm repository** - Added to your Helm repositories

### Step 1: Add Cilium Helm Repository

```bash
helm repo add cilium https://helm.cilium.io/
helm repo update cilium
```

### Step 2: Generate Hetzner Robot Manifest

The manifest was generated using the following command:

```bash
helm template cilium cilium/cilium \
  --version 1.18.2 \
  --namespace kube-system \
  --set k8sServiceHost=localhost \
  --set k8sServicePort=7445 \
  --set kubeProxyReplacement=true \
  --set tunnelProtocol=vxlan \
  --set gatewayAPI.enabled=true \
  --set gatewayAPI.hostNetwork.enabled=true \
  --set gatewayAPI.enableAlpn=true \
  --set-string 'gatewayAPI.hostNetwork.nodes.matchLabels.ingress\.kibaship\.com/ready=true' \
  --set gatewayAPI.enableAppProtocol=true \
  --set ipam.mode=kubernetes \
  --set loadBalancer.mode=snat \
  --set operator.replicas=2 \
  --set bpf.masquerade=true \
  --set 'securityContext.capabilities.ciliumAgent={CHOWN,KILL,NET_ADMIN,NET_RAW,IPC_LOCK,SYS_ADMIN,SYS_RESOURCE,DAC_OVERRIDE,FOWNER,SETGID,SETUID}' \
  --set 'securityContext.capabilities.cleanCiliumState={NET_ADMIN,SYS_ADMIN,SYS_RESOURCE}' \
  --set cgroup.autoMount.enabled=false \
  --set cgroup.hostRoot=/sys/fs/cgroup \
  > manifest.yaml
```

## Configuration Options Explained

### Core Networking

| Option | Value | Description |
|--------|-------|-------------|
| `k8sServiceHost` | `localhost` | Kubernetes API server host for secure local communication |
| `k8sServicePort` | `7445` | Custom port for Kubernetes API server access |
| `kubeProxyReplacement` | `true` | Replace kube-proxy with Cilium's eBPF implementation |
| `tunnelProtocol` | `vxlan` | Use VXLAN for overlay networking across dedicated servers |

### IPAM and Load Balancing

| Option | Value | Description |
|--------|-------|-------------|
| `ipam.mode` | `kubernetes` | Use Kubernetes native IPAM for IP address management |
| `loadBalancer.mode` | `snat` | Source NAT mode for external load balancer traffic |
| `bpf.masquerade` | `true` | Enable BPF-based masquerading for better performance |

### Gateway API Configuration

| Option | Value | Description |
|--------|-------|-------------|
| `gatewayAPI.enabled` | `true` | Enable Kubernetes Gateway API support |
| `gatewayAPI.hostNetwork.enabled` | `true` | Allow Gateway pods to use host network |
| `gatewayAPI.hostNetwork.nodes.matchLabels.ingress.kibaship.com/ready` | `"true"` | Node selector for Gateway pod placement |
| `gatewayAPI.enableAlpn` | `true` | Enable Application-Layer Protocol Negotiation |
| `gatewayAPI.enableAppProtocol` | `true` | Enable application protocol detection |

### Security and System Configuration

| Option | Value | Description |
|--------|-------|-------------|
| `operator.replicas` | `2` | Run 2 operator replicas for high availability |
| `securityContext.capabilities.ciliumAgent` | `{CHOWN,KILL,NET_ADMIN,NET_RAW,IPC_LOCK,SYS_ADMIN,SYS_RESOURCE,DAC_OVERRIDE,FOWNER,SETGID,SETUID}` | Extended capabilities for bare metal operations |
| `securityContext.capabilities.cleanCiliumState` | `{NET_ADMIN,SYS_ADMIN,SYS_RESOURCE}` | Capabilities for state cleanup operations |
| `cgroup.autoMount.enabled` | `false` | Disable automatic cgroup mounting |
| `cgroup.hostRoot` | `/sys/fs/cgroup` | Custom cgroup root path for dedicated servers |

## Hetzner Robot Specific Benefits

### Bare Metal Optimization
1. **Direct Hardware Access** - Extended capabilities for bare metal operations
2. **Custom API Configuration** - Secure localhost communication
3. **VXLAN Networking** - Efficient overlay for dedicated server clusters
4. **Performance Tuning** - BPF masquerading and SNAT load balancing

### Security Enhancements
1. **Extended Capabilities** - Additional privileges for system-level operations
2. **Custom cgroup Management** - Optimized for dedicated server environments
3. **Secure Communication** - Localhost API server access
4. **Network Isolation** - VXLAN tunneling for secure inter-node communication

### High Availability
1. **Dual Operators** - 2 replicas for control plane redundancy
2. **Gateway API** - Modern ingress with host network support
3. **Load Balancing** - SNAT mode for external traffic handling
4. **Node Affinity** - Controlled Gateway pod placement

## Deployment Process

### 1. Prepare Hetzner Robot Servers

Ensure your Hetzner Robot servers have:
- **Kubernetes installed** - Control plane and worker nodes
- **Network connectivity** - Between all servers
- **Required ports open** - 7445 for API server, VXLAN ports
- **cgroup v2 support** - For modern eBPF features

### 2. Apply the Manifest

```bash
kubectl apply -f manifest.yaml
```

### 3. Label Nodes for Gateway API

```bash
# Label specific servers for ingress traffic
kubectl label node <server-hostname> ingress.kibaship.com/ready=true
```

### 4. Verify Installation

```bash
# Check Cilium pods
kubectl get pods -n kube-system -l k8s-app=cilium

# Check Cilium operator
kubectl get pods -n kube-system -l name=cilium-operator

# Verify Cilium status (if cilium CLI is installed)
cilium status

# Check VXLAN tunnels
cilium node list
```

### 5. Verify Gateway API Resources

```bash
# Check Gateway API CRDs
kubectl get crd | grep gateway

# List Gateway Classes
kubectl get gatewayclass

# Verify node labels
kubectl get nodes --show-labels | grep ingress.kibaship.com
```

## Troubleshooting

### Common Hetzner Robot Issues

1. **VXLAN Connectivity** - Ensure UDP ports 8472 are open between servers
2. **API Server Access** - Verify localhost:7445 is accessible
3. **cgroup Issues** - Check `/sys/fs/cgroup` is properly mounted
4. **Capabilities** - Ensure extended capabilities are allowed

### Useful Commands

```bash
# Check Cilium agent logs
kubectl logs -n kube-system -l k8s-app=cilium

# Check Cilium operator logs
kubectl logs -n kube-system -l name=cilium-operator

# Verify VXLAN tunnels
cilium bpf tunnel list

# Check node connectivity
cilium connectivity test

# Monitor BPF masquerading
cilium bpf nat list
```

### Network Debugging

```bash
# Check VXLAN interface
ip link show cilium_vxlan

# Verify routing
ip route show table cilium

# Check eBPF programs
cilium bpf fs show

# Monitor traffic
cilium monitor
```

## Integration with Kibaship

This Cilium configuration is specifically designed for Kibaship on Hetzner Robot:

1. **Dedicated Servers** - Optimized for bare metal environments
2. **Custom Networking** - VXLAN overlay for multi-server clusters
3. **Security Hardening** - Extended capabilities and secure communication
4. **Gateway API** - Modern ingress for dedicated server deployments
5. **High Performance** - BPF masquerading and SNAT load balancing

## Hetzner Robot Network Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Server 1      │    │   Server 2      │    │   Server 3      │
│  (Control)      │    │   (Worker)      │    │   (Worker)      │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ Cilium Agent    │    │ Cilium Agent    │    │ Cilium Agent    │
│ VXLAN: 10.0.1.1 │◄──►│ VXLAN: 10.0.1.2 │◄──►│ VXLAN: 10.0.1.3 │
│ API: :7445      │    │                 │    │                 │
│ Gateway: ✓      │    │                 │    │ Gateway: ✓      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Hetzner Network │
                    │ (Public IPs)    │
                    └─────────────────┘
```

## Version Information

- **Cilium Version**: 1.18.2
- **Target Platform**: Hetzner Robot Dedicated Servers
- **Kubernetes Compatibility**: 1.25+
- **Gateway API Version**: v1beta1
- **Generated**: October 2025

## Next Steps

After applying this manifest:

1. **Configure Gateway Resources** - Set up Gateway and HTTPRoute for your applications
2. **Set up Network Policies** - Implement security policies for bare metal
3. **Enable Monitoring** - Deploy Hubble for network observability
4. **Performance Tuning** - Optimize for your specific server configurations

## References

- [Cilium Documentation](https://docs.cilium.io/)
- [Hetzner Robot Documentation](https://docs.hetzner.com/robot/)
- [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/)
- [Cilium Bare Metal Guide](https://docs.cilium.io/en/stable/installation/k8s-install-default/)
- [Kibaship Documentation](https://docs.kibaship.com/)
