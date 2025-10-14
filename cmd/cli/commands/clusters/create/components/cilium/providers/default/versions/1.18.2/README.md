# Cilium 1.18.2 Manifest Generation

This directory contains the Cilium 1.18.2 manifest generated using Helm for Kibaship clusters. We use Helm to generate manifests rather than installing directly to maintain full control over the deployment process.

## Overview

Cilium is a modern CNI (Container Network Interface) that provides:

- Advanced networking capabilities
- Network security policies
- Load balancing
- Kubernetes Gateway API support
- Observability features

## Manifest Generation Process

### Prerequisites

1. **Helm installed** - Required for manifest generation
2. **Cilium Helm repository** - Added to your Helm repositories

### Step 1: Add Cilium Helm Repository

```bash
helm repo add cilium https://helm.cilium.io/
helm repo update cilium
```

### Step 2: Generate Manifest

The manifest was generated using the following command:

```bash
helm template cilium cilium/cilium \
  --version 1.18.2 \
  --namespace kube-system \
  --set gatewayAPI.enabled=true \
  --set gatewayAPI.hostNetwork.enabled=true \
  --set-string 'gatewayAPI.hostNetwork.nodes.matchLabels.ingress\.kibaship\.com/ready=true' \
  --set gatewayAPI.enableAlpn=true \
  --set gatewayAPI.enableAppProtocol=true \
  --set operator.replicas=2 \
  > cilium-v-1.18.2.yaml
```

## Configuration Options Explained

### Gateway API Configuration

| Option                                                                | Value    | Description                                                                                                  |
| --------------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------ |
| `gatewayAPI.enabled`                                                  | `true`   | Enables Kubernetes Gateway API support, a modern alternative to Ingress                                      |
| `gatewayAPI.hostNetwork.enabled`                                      | `true`   | Allows Gateway pods to use the host network, binding directly to node IPs                                    |
| `gatewayAPI.hostNetwork.nodes.matchLabels.ingress.kibaship.com/ready` | `"true"` | Only schedules Gateway pods on nodes with this label, giving control over which nodes handle ingress traffic |
| `gatewayAPI.enableAlpn`                                               | `true`   | Enables Application-Layer Protocol Negotiation for protocol detection                                        |
| `gatewayAPI.enableAppProtocol`                                        | `true`   | Enables application protocol detection for routing decisions                                                 |

### Operator Configuration

| Option              | Value | Description                                           |
| ------------------- | ----- | ----------------------------------------------------- |
| `operator.replicas` | `2`   | Runs 2 replicas of the Cilium operator for redundancy |

## Gateway API Benefits

The Gateway API configuration provides several advantages over traditional Ingress:

1. **Modern API Design** - More expressive and flexible than Ingress
2. **Protocol Support** - Native support for HTTP, HTTPS, TCP, and UDP
3. **Advanced Routing** - Header-based routing, traffic splitting, and more
4. **Security** - Built-in security policies and TLS management
5. **Extensibility** - Plugin architecture for custom functionality

## Node Label Requirements

For the Gateway API to work correctly, nodes that should handle ingress traffic must be labeled:

```bash
kubectl label node <node-name> ingress.kibaship.com/ready=true
```

This ensures that Gateway pods are only scheduled on designated nodes, providing:

- **Resource isolation** - Dedicated nodes for ingress traffic
- **Performance optimization** - Predictable resource allocation
- **Security boundaries** - Controlled access points

## Deployment Process

### 1. Apply the Manifest

```bash
kubectl apply -f manifest.yaml
```

### 2. Verify Installation

```bash
# Check Cilium pods
kubectl get pods -n kube-system -l k8s-app=cilium

# Check Cilium operator
kubectl get pods -n kube-system -l name=cilium-operator

# Verify Cilium status (if cilium CLI is installed)
cilium status
```

### 3. Label Nodes for Gateway API

```bash
# Label control plane node (for Kind clusters)
kubectl label node <cluster-name>-control-plane ingress.kibaship.com/ready=true

# Or label specific worker nodes
kubectl label node <worker-node-name> ingress.kibaship.com/ready=true
```

### 4. Verify Gateway API Resources

```bash
# Check if Gateway API CRDs are installed
kubectl get crd | grep gateway

# List available Gateway Classes
kubectl get gatewayclass
```

## Troubleshooting

### Common Issues

1. **Pods stuck in Pending** - Check node labels and resource availability
2. **Gateway API not working** - Verify CRDs are installed and nodes are labeled
3. **Connectivity issues** - Check Cilium agent logs

### Useful Commands

```bash
# Check Cilium agent logs
kubectl logs -n kube-system -l k8s-app=cilium

# Check Cilium operator logs
kubectl logs -n kube-system -l name=cilium-operator

# Cilium connectivity test (if cilium CLI is installed)
cilium connectivity test
```

## Integration with Kibaship

This Cilium configuration is specifically designed for Kibaship clusters:

1. **Kind Clusters** - Works with local development environments
2. **Cloud Clusters** - Compatible with DigitalOcean, AWS, etc.
3. **Gateway API** - Provides modern ingress capabilities
4. **High Availability** - 2 operator replicas for redundancy

## Version Information

- **Cilium Version**: 1.18.2
- **Kubernetes Compatibility**: 1.25+
- **Gateway API Version**: v1beta1
- **Generated**: October 2025

## Next Steps

After applying this manifest:

1. **Install Gateway API CRDs** (if not already present)
2. **Configure Gateway resources** for your applications
3. **Set up network policies** for security
4. **Enable Hubble** for observability (optional)

## References

- [Cilium Documentation](https://docs.cilium.io/)
- [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/)
- [Cilium Gateway API Guide](https://docs.cilium.io/en/stable/network/servicemesh/gateway-api/)
- [Kibaship Documentation](https://docs.kibaship.com/)
