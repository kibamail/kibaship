# KibaShip Operator - Simple Helm Installation

The KibaShip Operator provides a **simple, opinionated** Helm chart with minimal configuration required.

## 🚀 Quick Start

### One-Command Installation

```bash
helm install kibaship-operator deploy/helm/kibaship-operator \
  --set operator.domain=your-apps.example.com \
  --create-namespace \
  --namespace kibaship-operator-system
```

That's it! Everything else is automatically configured.

## 📋 What You Need to Configure

### Required
- **`operator.domain`** - Your domain for ApplicationDomains (e.g., "myapps.example.com")

### Optional
- **`operator.defaultPort`** - Default port for applications (default: 3000)
- **`controllerManager.replicas`** - Number of replicas (default: 1)
- **`controllerManager.resources`** - CPU/Memory limits
- **`debug.enabled`** - Enable debug logging (default: false)

## 📦 What Gets Auto-Configured

✅ **RBAC** - Complete cluster permissions
✅ **Webhooks** - ApplicationDomain validation
✅ **Security** - Non-root containers, security contexts
✅ **Health Checks** - Liveness/readiness probes
✅ **Service Account** - Auto-created with permissions
✅ **CRDs** - All 4 custom resources installed
✅ **Namespace** - Auto-created

## 🎯 Installation Examples

### Basic Installation
```bash
helm install kibaship-operator deploy/helm/kibaship-operator \
  --set operator.domain=myapps.example.com \
  --create-namespace \
  --namespace kibaship-operator-system
```

### Production Setup
```bash
cat > production-values.yaml <<EOF
operator:
  domain: "apps.production.com"

controllerManager:
  replicas: 2
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
  --namespace kibaship-operator-system
```

### Development Setup
```bash
helm install kibaship-operator deploy/helm/kibaship-operator \
  --set operator.domain=dev.local \
  --set debug.enabled=true \
  --set debug.level=debug \
  --create-namespace \
  --namespace kibaship-operator-system
```

## ✅ Verification

```bash
# Check if everything is running
kubectl get pods -n kibaship-operator-system

# Test ApplicationDomain creation
kubectl apply -f - <<EOF
apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Project
metadata:
  name: test-project
  labels:
    platform.kibaship.com/uuid: test-123
spec: {}
---
apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: project-test-project-app-frontend-kibaship-com
  labels:
    platform.kibaship.com/uuid: app-456
    platform.kibaship.com/project-uuid: test-123
spec:
  type: GitRepository
  projectRef:
    name: test-project
  gitRepository:
    repository: https://github.com/example/app
EOF

# Check if ApplicationDomain was created
kubectl get applicationdomains
```

## 🔧 Management

### Upgrade
```bash
helm upgrade kibaship-operator deploy/helm/kibaship-operator \
  --set controllerManager.image.tag=v0.2.0
```

### Check Status
```bash
helm status kibaship-operator -n kibaship-operator-system
kubectl logs -f deployment/kibaship-operator-controller-manager -n kibaship-operator-system
```

### Uninstall
```bash
helm uninstall kibaship-operator -n kibaship-operator-system
```

## 🎯 Why Simple?

This chart follows a **"batteries included"** approach:

- 🎯 **Only Essential Options** - Configure what matters
- 🔒 **Secure by Default** - All security hardening included
- ✅ **Everything Auto-Enabled** - No complex feature toggles
- 📝 **Minimal Config** - Just set your domain and go
- 🚀 **One Command Install** - From zero to running instantly

## 🆚 Comparison

| Configuration | This Chart | Complex Charts |
|---------------|------------|----------------|
| **Required Config** | 1 field (domain) | 10+ fields |
| **Lines of YAML** | ~10 lines | 50+ lines |
| **Installation Time** | 30 seconds | 10+ minutes |
| **Maintenance** | Minimal | Complex |

## 🚨 Troubleshooting

**Problem**: "operator.domain must be set"
```bash
# Wrong
helm install kibaship-operator deploy/helm/kibaship-operator

# Right
helm install kibaship-operator deploy/helm/kibaship-operator \
  --set operator.domain=your-domain.com
```

**Problem**: Check if domain is configured
```bash
kubectl get deployment kibaship-operator-controller-manager \
  -n kibaship-operator-system -o yaml | grep KIBASHIP_OPERATOR_DOMAIN
```

For advanced configuration needs, see the [detailed chart documentation](deploy/helm/kibaship-operator/README.md) or use the kubectl deployment method.

The KibaShip Operator Helm chart - **Simple by design, powerful by default!** 🚀