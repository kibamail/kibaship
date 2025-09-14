# Quick Deployment Guide

## 🚀 Deploy KibaShip Operator

### One-Command Deployment

```bash
kubectl apply -f https://github.com/kibamail/kibaship-operator/releases/latest/download/install.yaml
```

### Configure Your Domain

```bash
kubectl set env deployment/kibaship-operator-controller-manager \
  -n kibaship-operator-system \
  KIBASHIP_OPERATOR_DOMAIN=your-apps.example.com \
  KIBASHIP_OPERATOR_DEFAULT_PORT=3000
```

### Verify Installation

```bash
# Check operator status
kubectl get deployment -n kibaship-operator-system

# Test with a sample application
kubectl apply -f - <<EOF
apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Project
metadata:
  name: demo-project
  labels:
    platform.kibaship.com/uuid: demo-project-123
spec: {}
---
apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: project-demo-project-app-frontend-kibaship-com
  labels:
    platform.kibaship.com/uuid: demo-app-456
    platform.kibaship.com/project-uuid: demo-project-123
spec:
  type: GitRepository
  projectRef:
    name: demo-project
  gitRepository:
    repository: https://github.com/example/app
EOF

# Check if ApplicationDomain was created automatically
kubectl get applicationdomains
```

## 📋 What Gets Created

- **ApplicationDomain**: Automatically created for GitRepository applications
- **Domain Format**: `<app-slug>-<random>.your-apps.example.com`
- **Default Port**: 3000 (configurable)
- **TLS**: Enabled by default

## 🔧 Build Your Own

```bash
# Clone and build
git clone https://github.com/kibamail/kibaship-operator.git
cd kibaship-operator

# Build image
make docker-build IMG=ghcr.io/kibamail/kibaship-operator:latest

# Deploy
make deploy IMG=ghcr.io/kibamail/kibaship-operator:latest
```

For detailed instructions, see [DEPLOYMENT.md](./DEPLOYMENT.md).