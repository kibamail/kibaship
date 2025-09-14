# 🚀 KibaShip Operator - Deployment Ready!

## ✅ What's Been Configured

### 1. GitHub Container Registry Setup
- **Image Registry**: `ghcr.io/kibamail/kibaship-operator`
- **Versioning**: Semantic versioning (v0.0.1, v0.1.0, etc.)
- **Multi-architecture**: Supports ARM64 and AMD64

### 2. Automated CI/CD Pipeline
- **GitHub Actions**: `.github/workflows/build-and-push.yml`
- **Triggers**:
  - Push to `main` branch → builds and pushes `latest`
  - Version tags (e.g., `v0.1.0`) → builds, pushes, and creates GitHub releases
  - Pull requests → builds and tests (no push)

### 3. Build Scripts
- **Docker Build**: `make docker-build`
- **Cross-platform**: `make docker-buildx`
- **Generate Manifests**: `make build-installer`
- **Full Deploy**: `make deploy`

### 4. Deployment Manifests
- **Generated**: `dist/install.yaml` (ready for kubectl apply)
- **Includes**: CRDs, RBAC, Deployment, Services, Webhooks
- **ApplicationDomain**: Fully integrated with validation webhooks

## 📋 Quick Deployment Instructions

### For End Users

```bash
# 1. Deploy the operator
kubectl apply -f https://github.com/kibamail/kibaship-operator/releases/latest/download/install.yaml

# 2. Configure your domain
kubectl set env deployment/kibaship-operator-controller-manager \
  -n kibaship-operator-system \
  KIBASHIP_OPERATOR_DOMAIN=your-apps.example.com \
  KIBASHIP_OPERATOR_DEFAULT_PORT=3000

# 3. Verify installation
kubectl get deployment -n kibaship-operator-system
```

### For Developers

```bash
# Build and test locally
make test
make docker-build
make build-installer

# Deploy to development cluster
make deploy
```

## 🔧 Release Process

### Creating a New Release

1. **Update version** (if needed):
   ```bash
   export VERSION=0.1.0
   # Update VERSION in Makefile if desired
   ```

2. **Create and push tag**:
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

3. **GitHub Actions automatically**:
   - Builds multi-arch images
   - Pushes to `ghcr.io/kibamail/kibaship-operator:v0.1.0`
   - Generates `dist/install.yaml`
   - Creates GitHub release with deployment manifests

### Manual Build and Push

```bash
# Set your version
VERSION=0.1.0

# Build and push
make docker-build IMG=ghcr.io/kibamail/kibaship-operator:v${VERSION}
make docker-push IMG=ghcr.io/kibamail/kibaship-operator:v${VERSION}

# Generate deployment manifests
make build-installer IMG=ghcr.io/kibamail/kibaship-operator:v${VERSION}
```

## 📦 What Gets Built

### Container Image
- **Base**: `gcr.io/distroless/static:nonroot` (minimal, secure)
- **Architecture**: Multi-arch (linux/amd64, linux/arm64)
- **Binary**: Statically linked Go binary
- **Size**: ~30MB (optimized)

### Kubernetes Manifests
- **CRDs**: Project, Application, Deployment, ApplicationDomain
- **RBAC**: ClusterRole, ClusterRoleBinding, ServiceAccount
- **Deployment**: Operator controller manager
- **Webhooks**: Validation webhooks for ApplicationDomain
- **Service**: Webhook service endpoints

## 🔍 Verification Commands

```bash
# Check image exists
docker pull ghcr.io/kibamail/kibaship-operator:latest

# Verify manifests
kubectl apply --dry-run=client -f dist/install.yaml

# Test deployment
kubectl apply -f dist/install.yaml
kubectl get pods -n kibaship-operator-system
```

## 🚦 Next Steps

1. **Test the CI/CD**: Push a tag to trigger automated release
2. **Update Documentation**: Add any specific deployment requirements
3. **Monitor**: Set up monitoring/alerting for the operator
4. **Security**: Review RBAC permissions and security policies

## 📞 Support

- **Build Issues**: Check GitHub Actions logs
- **Registry Access**: Ensure GITHUB_TOKEN has packages:write permission
- **Deployment Problems**: See `DEPLOYMENT.md` for troubleshooting

The KibaShip Operator is now fully configured for production deployment! 🎉