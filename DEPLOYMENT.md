# KibaShip Operator Deployment Guide

This guide provides comprehensive instructions for deploying the KibaShip Operator to GitHub Container Registry and Kubernetes clusters.

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.24+)
- kubectl configured for your cluster
- Docker or compatible container runtime

### One-Command Deployment

#### Option 1: Helm (Recommended)

```bash
helm install kibaship-operator deploy/helm/kibaship-operator \
  --set operator.domain=your-apps.example.com \
  --create-namespace \
  --namespace kibaship-operator
```

#### Option 2: kubectl

Deploy the latest stable release:

```bash
kubectl apply -f https://github.com/kibamail/kibaship-operator/releases/latest/download/install.yaml
```

### Configure Operator Domain

After deployment, configure your operator domain:

```bash
kubectl set env deployment/kibaship-operator-controller-manager \
  -n kibaship-operator \
  KIBASHIP_OPERATOR_DOMAIN=your-apps.example.com \
  KIBASHIP_OPERATOR_DEFAULT_PORT=3000
```

## Building and Pushing Images

### Build Configuration

The operator is configured to use GitHub Container Registry (`ghcr.io/kibamail/kibaship-operator`).

### Manual Build and Push

```bash
# Set version (defaults to 0.0.1)
export VERSION=0.1.0

# Build the image
make docker-build IMG=ghcr.io/kibamail/kibaship-operator:v${VERSION}

# Push to GitHub Container Registry (requires authentication)
make docker-push IMG=ghcr.io/kibamail/kibaship-operator:v${VERSION}

# Generate deployment manifests
make build-installer IMG=ghcr.io/kibamail/kibaship-operator:v${VERSION}
```

### Cross-Platform Build

Build for multiple architectures:

```bash
make docker-buildx IMG=ghcr.io/kibamail/kibaship-operator:v${VERSION}
```

## GitHub Container Registry Setup

### Authentication

1. Create a Personal Access Token with `write:packages` permission
2. Log in to GitHub Container Registry:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```

### Automated Builds

The operator includes GitHub Actions workflows that automatically:

- Build and test on every PR
- Build and push images on main branch commits
- Create releases with deployment manifests for version tags

#### Triggering Releases

Create a new release by pushing a version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Deployment Methods

### Method 1: Helm Chart (Recommended)

```bash
# Install from source
helm install kibaship-operator deploy/helm/kibaship-operator \
  --set operator.domain=your-apps.example.com \
  --create-namespace \
  --namespace kibaship-operator

# With custom values file
helm install kibaship-operator deploy/helm/kibaship-operator \
  -f custom-values.yaml \
  --create-namespace \
  --namespace kibaship-operator
```

See [HELM_INSTALL.md](./HELM_INSTALL.md) for detailed Helm installation instructions.

### Method 2: Direct Manifest Application

```bash
# Deploy latest release
kubectl apply -f https://github.com/kibamail/kibaship-operator/releases/latest/download/install.yaml

# Or deploy specific version
kubectl apply -f https://github.com/kibamail/kibaship-operator/releases/download/v0.1.0/install.yaml
```

### Method 2: Using Local Manifests

```bash
# Clone the repository
git clone https://github.com/kibamail/kibaship-operator.git
cd kibaship-operator

# Build manifests with custom image
make build-installer IMG=ghcr.io/kibamail/kibaship-operator:v0.1.0

# Deploy
kubectl apply -f dist/install.yaml
```

### Method 3: Development Deployment

```bash
# Deploy directly from source (for development)
make deploy IMG=ghcr.io/kibamail/kibaship-operator:latest
```

## Configuration

### Required Environment Variables

Set these in the operator deployment:

```yaml
env:
- name: KIBASHIP_OPERATOR_DOMAIN
  value: "your-apps.example.com"  # Your operator subdomain
- name: KIBASHIP_OPERATOR_DEFAULT_PORT
  value: "3000"                   # Default port for applications
```

### Configuration Methods

#### Method 1: kubectl set env

```bash
kubectl set env deployment/kibaship-operator-controller-manager \
  -n kibaship-operator \
  KIBASHIP_OPERATOR_DOMAIN=your-apps.example.com \
  KIBASHIP_OPERATOR_DEFAULT_PORT=3000
```

#### Method 2: ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kibaship-operator-config
  namespace: kibaship-operator
data:
  KIBASHIP_OPERATOR_DOMAIN: "your-apps.example.com"
  KIBASHIP_OPERATOR_DEFAULT_PORT: "3000"
---
# Then reference in deployment
spec:
  template:
    spec:
      containers:
      - name: manager
        envFrom:
        - configMapRef:
            name: kibaship-operator-config
```

#### Method 3: Kustomize Overlay

Create `config/environments/production/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../../default

patches:
- patch: |-
    - op: add
      path: /spec/template/spec/containers/0/env
      value:
      - name: KIBASHIP_OPERATOR_DOMAIN
        value: your-apps.example.com
      - name: KIBASHIP_OPERATOR_DEFAULT_PORT
        value: "3000"
  target:
    kind: Deployment
    name: controller-manager
```

Then deploy:

```bash
kubectl apply -k config/environments/production
```

## Verification

### Check Deployment Status

```bash
# Check operator deployment
kubectl get deployment -n kibaship-operator

# Check operator logs
kubectl logs -f deployment/kibaship-operator-controller-manager -n kibaship-operator

# Verify CRDs are installed
kubectl get crd | grep platform.operator.kibaship.com
```

### Test Application Domain Creation

```bash
# Apply test resources
kubectl apply -f config/samples/demo_automatic_domain_creation.yaml

# Check if ApplicationDomain was created automatically
kubectl get applicationdomains -n default

# Check the generated domain
kubectl get applicationdomain -n default -o yaml
```

## Troubleshooting

### Common Issues

#### 1. Image Pull Errors

```bash
# Ensure the image exists and is accessible
docker pull ghcr.io/kibamail/kibaship-operator:v0.1.0

# Check if the image is public or if auth is needed
kubectl get events -n kibaship-operator
```

#### 2. Webhook Failures

```bash
# Check webhook service
kubectl get svc -n kibaship-operator

# Check certificate status
kubectl get certificates -n kibaship-operator

# Check webhook configuration
kubectl get validatingwebhookconfigurations
```

#### 3. Missing Environment Variables

```bash
# Check current environment variables
kubectl get deployment kibaship-operator-controller-manager -n kibaship-operator -o yaml | grep -A 10 env:

# Validate configuration
kubectl logs deployment/kibaship-operator-controller-manager -n kibaship-operator | grep -i "operator configuration"
```

### Debug Commands

```bash
# Get all resources in operator namespace
kubectl get all -n kibaship-operator

# Describe problematic pods
kubectl describe pod -l control-plane=controller-manager -n kibaship-operator

# Check RBAC permissions
kubectl auth can-i create applications --as=system:serviceaccount:kibaship-operator:kibaship-operator-controller-manager
```

## Cleanup

### Uninstall Operator

```bash
# Using the uninstall manifest (if available)
kubectl delete -f dist/install.yaml

# Or remove components individually
kubectl delete namespace kibaship-operator
kubectl delete crd projects.platform.operator.kibaship.com
kubectl delete crd applications.platform.operator.kibaship.com
kubectl delete crd deployments.platform.operator.kibaship.com
kubectl delete crd applicationdomains.platform.operator.kibaship.com

# Clean up webhook configurations
kubectl delete validatingwebhookconfigurations kibaship-operator-validating-webhook-configuration
kubectl delete mutatingwebhookconfigurations kibaship-operator-mutating-webhook-configuration

# Clean up RBAC
kubectl delete clusterrole kibaship-operator-manager-role
kubectl delete clusterrolebinding kibaship-operator-manager-rolebinding
```

## Monitoring and Maintenance

### Health Checks

The operator includes health check endpoints:

- Liveness: `/healthz` on port 8081
- Readiness: `/readyz` on port 8081

### Upgrading

1. Check release notes for breaking changes
2. Backup your resources (optional but recommended):
   ```bash
   kubectl get projects,applications,deployments,applicationdomains -A -o yaml > backup.yaml
   ```
3. Apply the new version:
   ```bash
   kubectl apply -f https://github.com/kibamail/kibaship-operator/releases/download/v0.2.0/install.yaml
   ```
4. Verify the upgrade:
   ```bash
   kubectl rollout status deployment/kibaship-operator-controller-manager -n kibaship-operator
   ```

## Development Setup

For development deployments:

```bash
# Build and deploy from source
make build-installer
kubectl apply -f dist/install.yaml

# Or use the development target
make deploy
```

## Support

- GitHub Issues: https://github.com/kibamail/kibaship-operator/issues
- Documentation: https://github.com/kibamail/kibaship-operator/tree/main/docs
- Container Images: https://github.com/kibamail/kibaship-operator/pkgs/container/kibaship-operator