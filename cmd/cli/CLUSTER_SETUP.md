# Kibaship Cluster Setup Process

This document outlines the complete end-to-end cluster setup process based on the test suite analysis. This is exactly what the `clusters create` command will implement.

## Overview

The cluster creation process involves 9 major phases with 32 detailed steps, mirroring the exact setup used in the e2e test suite. Currently implemented: Phase 1-4.

## Phase Breakdown

### Phase 1: Prerequisites & Cluster Creation
1. **Check if Kind is installed** (`command -v kind`)
2. **Check if Helm is installed** (`helm version --short`)
3. **Check if kubectl is installed**
4. **Generate Kind cluster config** (`kind.config.yaml`):
   - Disable default CNI (`disableDefaultCNI: true`)
   - Disable kube-proxy (`kubeProxyMode: none`)
   - Add control-plane node with ingress labels
5. **Create Kind cluster** (`kind create cluster --name <name> --config kind.config.yaml`)

### Phase 2: Core Infrastructure (CNI & Gateway API)
6. **Install Gateway API CRDs** (v1.3.0 - Custom Kibaship CRDs):
   - `backendtlspolicies.gateway.networking.k8s.io`
   - `gatewayclasses.gateway.networking.k8s.io`
   - `gateways.gateway.networking.k8s.io`
   - `grpcroutes.gateway.networking.k8s.io`
   - `httproutes.gateway.networking.k8s.io`
   - `referencegrants.gateway.networking.k8s.io`
   - `tcproutes.gateway.networking.k8s.io`
   - `tlsroutes.gateway.networking.k8s.io`
   - `udproutes.gateway.networking.k8s.io`
7. **Install Cilium CNI via Helm** (v1.18.0):
   - Add cilium helm repo
   - Install with Gateway API enabled
   - Configure hostNetwork and node selectors
   - Wait for DaemonSet and operator to be ready
8. **Label all nodes** with `"ingress.kibaship.com/ready=true"`

### Phase 3: Storage & Certificate Management
9. **Install cert-manager** (v1.18.2):
   - Helm-based installation with HA configuration
   - 3 controller replicas, 2 webhook replicas
   - Prometheus monitoring enabled
   - Automatic CRD installation

10. **Install Longhorn Storage** (v1.10.0):
    - Distributed block storage for Kubernetes
    - Automatic StorageClass creation
    - Volume snapshots and backup support
    - Web UI for management

### Phase 4: Build & CI/CD Infrastructure
11. **Install Tekton Pipelines** (v1.4.0):
    - Apply 75 Tekton manifests in order
    - Wait for controller, webhook, events, and resolvers deployments
    - Verify all CRDs are established

12. **Install Valkey Operator** (v0.0.59):
    - Apply Valkey operator manifests
    - Wait for controller manager deployment
    - Verify CRDs are established

### Phase 5: Operator Configuration & Installation
13. **Install Kibaship Operator Configuration**:
    - Create kibaship-operator namespace
    - Create operator ConfigMap with required environment variables
    - Validate configuration parameters

14. **Install Kibaship Operator** (matches CLI version):
    - Apply operator manifests from official release (version matches CLI)
    - For development builds: Uses `KIBASHIP_VERSION` env var or defaults to v0.1.3
    - Wait for controller manager deployment
    - Verify operator is running and ready
12. **Install BuildKit shared daemon**:
    - Create buildkit namespace
    - Deploy BuildKit daemon (3 replicas)
    - Configure registry CA certificates
13. **Apply Tekton custom tasks**:
    - git-clone task
    - railpack-prepare task
    - railpack-build task

### Phase 5: Database & Storage
14. **Install Valkey Operator** (v0.0.59):
    - Apply valkey-operator manifests
    - Wait for controller-manager deployment
15. **Create storage classes**:
    - `storage-replica-1` (Longhorn with 1 replica)
    - `storage-replica-2` (Longhorn with 2 replicas)

### Phase 6: Container Registry
16. **Create registry namespace**
17. **Provision registry-auth Certificate** for JWT signing
18. **Deploy Docker Registry v3.0.0**:
    - Configure TLS and auth integration
    - Create PVC for storage
    - Wait for registry to be ready
19. **Wait for registry-auth Certificate** to be ready
20. **Verify registry-auth pods** are healthy

### Phase 7: Kibaship Operator Installation
21. **Build and load operator image** to Kind cluster
22. **Install Kibaship CRDs**:
    - `projects.platform.operator.kibaship.com`
    - `applications.platform.operator.kibaship.com`
    - `deployments.platform.operator.kibaship.com`
    - `applicationdomains.platform.operator.kibaship.com`
23. **Deploy Kibaship operator**:
    - Apply operator manifests
    - Configure `WEBHOOK_TARGET_URL` environment
    - Wait for operator to be ready
24. **Deploy API server**:
    - Build and load API server image
    - Deploy API server to operator namespace
25. **Deploy cert-manager webhook**:
    - Build and load webhook image
    - Deploy webhook to operator namespace

### Phase 8: Ingress & Networking
26. **Gateway API resources** (created dynamically by operator on startup):
    - gateway-api-system namespace
    - certificates namespace
    - Gateway with 5 listeners (HTTP, HTTPS, MySQL TLS, PostgreSQL TLS, Valkey TLS)
    - ReferenceGrant for cross-namespace certificate access
    - Wildcard certificate for web apps (*.apps.{domain})
    - Note: Database certificates created per-instance in project namespaces

### Phase 9: Verification & Health Checks
30. **Verify all components are healthy**:
    - Cilium CNI operational
    - Gateway API resources created
    - Tekton pipelines functional
    - BuildKit daemon responsive
    - Registry accepting pushes/pulls
    - Operator reconciling resources
    - API server responding
    - Webhook processing requests
31. **Configure kubectl context** to use new cluster
32. **Display cluster information** and next steps

## Key Configuration Details

### Kind Cluster Config
```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  disableDefaultCNI: true
  kubeProxyMode: none
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress.kibaship.com/ready=true"
```

### Cilium Helm Configuration
```bash
helm upgrade --install cilium cilium/cilium \
  --namespace kube-system --create-namespace \
  --version 1.18.0 \
  --set kubeProxyReplacement=true \
  --set tunnelProtocol=vxlan \
  --set gatewayAPI.enabled=true \
  --set gatewayAPI.hostNetwork.enabled=true \
  --set gatewayAPI.enableAlpn=true \
  --set gatewayAPI.hostNetwork.nodeLabelSelector=ingress.kibaship.com/ready=true \
  --set gatewayAPI.enableProxyProtocol=true \
  --set gatewayAPI.enableAppProtocol=true \
  --set ipam.mode=kubernetes \
  --set loadBalancer.mode=snat
```

### Component Versions
- **Gateway API**: v1.3.0 (Custom Kibaship CRDs)
- **Cilium**: v1.18.0
- **cert-manager**: v1.18.2
- **Longhorn**: v1.10.0
- **Tekton Pipelines**: v1.4.0
- **Valkey Operator**: v0.0.59
- **Kibaship Operator**: Automatically matches CLI version

## Implementation Notes

- Each step should have proper error handling and rollback capability
- Use the CLI styling functions for beautiful progress output
- Implement timeouts for all wait operations
- Provide detailed error messages with troubleshooting hints
- Support both interactive and non-interactive modes
- Allow resuming from failed steps
- Validate prerequisites before starting
- Clean up resources on failure

## File Structure for Implementation

```
cmd/cli/
├── clusters/
│   ├── create.go          # Main cluster creation logic
│   ├── prerequisites.go   # Check tools and dependencies
│   ├── kind.go           # Kind cluster management
│   ├── infrastructure.go # Install core components
│   ├── operator.go       # Kibaship operator installation
│   └── verify.go         # Health checks and verification
└── utils/
    ├── kubectl.go        # kubectl operations
    ├── helm.go          # Helm operations
    └── wait.go          # Wait and retry utilities
```
