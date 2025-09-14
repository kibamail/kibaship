# KibaShip Operator Helm Chart

A simple and opinionated Helm chart for deploying the KibaShip Operator with automatic ApplicationDomain management.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.8+

## Quick Installation

### 1. Basic Installation

```bash
helm install kibaship-operator deploy/helm/kibaship-operator \
  --set operator.domain=your-apps.example.com \
  --create-namespace \
  --namespace kibaship-operator
```

### 2. Production Installation

```bash
# Create values file for production
cat > production-values.yaml <<EOF
operator:
  domain: "apps.production.com"
  defaultPort: 3000

controllerManager:
  replicas: 2
  image:
    tag: "v0.1.0"
  resources:
    limits:
      cpu: 1000m
      memory: 256Mi
    requests:
      cpu: 100m
      memory: 128Mi
EOF

helm install kibaship-operator deploy/helm/kibaship-operator \
  -f production-values.yaml \
  --create-namespace \
  --namespace kibaship-operator
```

### 3. Development Installation

```bash
helm install kibaship-operator deploy/helm/kibaship-operator \
  --set operator.domain=dev.local \
  --set debug.enabled=true \
  --set debug.level=debug \
  --create-namespace \
  --namespace kibaship-operator
```

## Configuration

### Required Configuration

| Parameter | Description | Example |
|-----------|-------------|---------|
| `operator.domain` | **REQUIRED** - Domain for auto-generated ApplicationDomains | `"myapps.example.com"` |

### Optional Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `operator.defaultPort` | Default port for applications | `3000` |
| `controllerManager.replicas` | Number of controller replicas | `1` |
| `controllerManager.image.tag` | Container image tag | `"v0.0.1"` |
| `controllerManager.resources.limits.cpu` | CPU limit | `500m` |
| `controllerManager.resources.limits.memory` | Memory limit | `128Mi` |
| `debug.enabled` | Enable debug logging | `false` |
| `debug.level` | Log level (info, debug, error) | `"info"` |

### Auto-Configured Features

The following are **automatically enabled** with sensible defaults:

- ✅ **RBAC** - Complete permissions for operator functionality
- ✅ **Webhooks** - ApplicationDomain validation webhooks
- ✅ **Security** - Non-root containers with security contexts
- ✅ **Health Checks** - Liveness and readiness probes
- ✅ **Service Account** - Auto-created with proper permissions
- ✅ **CRDs** - All 4 custom resources (Project, Application, Deployment, ApplicationDomain)
- ✅ **Namespace** - Auto-created as `<release>-system`

## Examples

### Minimal Configuration

```yaml
# values-minimal.yaml
operator:
  domain: "myapps.example.com"
```

```bash
helm install kibaship-operator deploy/helm/kibaship-operator -f values-minimal.yaml
```

### Complete Configuration

```yaml
# values-complete.yaml
operator:
  domain: "apps.production.com"
  defaultPort: 8080

controllerManager:
  replicas: 3
  image:
    repository: ghcr.io/kibamail/kibaship-operator
    tag: "v0.2.0"
    pullPolicy: Always
  resources:
    limits:
      cpu: 1000m
      memory: 512Mi
    requests:
      cpu: 200m
      memory: 256Mi

debug:
  enabled: false
  level: "info"
```

```bash
helm install kibaship-operator deploy/helm/kibaship-operator -f values-complete.yaml
```

## Verification

Check if the installation was successful:

```bash
# Check deployment status
kubectl get deployment -n kibaship-operator

# Check pods are running
kubectl get pods -n kibaship-operator

# Check CRDs are installed
kubectl get crd | grep platform.operator.kibaship.com

# View logs
kubectl logs -f deployment/kibaship-operator-controller-manager -n kibaship-operator
```

## Testing

Create a test application to verify automatic ApplicationDomain creation:

```bash
kubectl apply -f - <<EOF
apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Project
metadata:
  name: test-project
  labels:
    platform.kibaship.com/uuid: test-project-123
spec: {}
---
apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: project-test-project-app-frontend-kibaship-com
  labels:
    platform.kibaship.com/uuid: test-app-456
    platform.kibaship.com/project-uuid: test-project-123
spec:
  type: GitRepository
  projectRef:
    name: test-project
  gitRepository:
    repository: https://github.com/example/app
EOF

# Check if ApplicationDomain was created automatically
kubectl get applicationdomains
```

## Management

### Upgrade

```bash
# Upgrade to new version
helm upgrade kibaship-operator deploy/helm/kibaship-operator \
  --set controllerManager.image.tag=v0.2.0

# Upgrade with new values
helm upgrade kibaship-operator deploy/helm/kibaship-operator \
  -f new-values.yaml
```

### Status

```bash
# Check Helm release status
helm status kibaship-operator -n kibaship-operator

# View current values
helm get values kibaship-operator -n kibaship-operator
```

### Uninstall

```bash
# Remove the operator
helm uninstall kibaship-operator -n kibaship-operator

# Optionally remove namespace (will delete all resources)
kubectl delete namespace kibaship-operator
```

## Troubleshooting

### Common Issues

**1. Domain not configured:**
```bash
# This will fail - domain is required
helm install kibaship-operator deploy/helm/kibaship-operator
# Error: operator.domain must be set

# Correct way
helm install kibaship-operator deploy/helm/kibaship-operator \
  --set operator.domain=your-domain.com
```

**2. Check operator configuration:**
```bash
kubectl get deployment kibaship-operator-controller-manager \
  -n kibaship-operator -o yaml | grep KIBASHIP_OPERATOR_DOMAIN
```

**3. Check logs for errors:**
```bash
kubectl logs -f deployment/kibaship-operator-controller-manager \
  -n kibaship-operator
```

## Why This Chart is Simple

This Helm chart follows an **opinionated approach**:

- 🎯 **Essential Options Only** - Only expose what you actually need to configure
- 🔒 **Secure by Default** - All security settings are pre-configured
- ✅ **Everything Auto-Enabled** - RBAC, webhooks, health checks just work
- 📝 **Minimal Configuration** - Only requires your domain name
- 🚀 **Quick Setup** - From zero to running in one command

For advanced users who need full control, modify the templates directly or use the kubectl deployment method.

## Support

- GitHub Issues: https://github.com/kibamail/kibaship-operator/issues
- Documentation: https://github.com/kibamail/kibaship-operator/tree/main/docs