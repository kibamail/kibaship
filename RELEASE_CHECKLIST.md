# KibaShip Operator Release Checklist

This checklist ensures that all necessary steps are completed before releasing a new version of the KibaShip Operator.

## Pre-Release Preparation

### 🔍 Version Configuration
- [ ] **Makefile VERSION**: Update `VERSION ?= 0.0.1` in Makefile to new version
- [ ] **Helm Chart appVersion**: Update `appVersion: "0.0.1"` in `deploy/helm/kibaship-operator/Chart.yaml`
- [ ] **Helm Chart version**: Update `version: 0.0.1` in `deploy/helm/kibaship-operator/Chart.yaml` (matches appVersion)
- [ ] **Values.yaml**: Update `tag: "v0.0.1"` in `deploy/helm/kibaship-operator/values.yaml`
- [ ] **Version Consistency**: Ensure all version references match across the project (all should be the same)

### 🧪 Quality Assurance
- [ ] **All Tests Pass**: Run `make test` - all tests must pass
- [ ] **Linting Clean**: Run `make lint` - no linting errors
- [ ] **Build Success**: Run `make build` - successful compilation
- [ ] **Docker Build**: Run `make docker-build` - successful container image build
- [ ] **Installation Manifest**: Run `make build-installer` - generates clean `dist/install.yaml`

### 📋 Documentation
- [ ] **README Updated**: Version references and installation instructions current
- [ ] **CHANGELOG**: Document new features, fixes, and breaking changes
- [ ] **API Documentation**: Ensure CRD documentation is up-to-date
- [ ] **Examples**: Sample configurations work with new version

### 🛡️ Security & Dependencies
- [ ] **Dependency Check**: Run `go mod tidy` and verify no vulnerable dependencies
- [ ] **RBAC Review**: Verify RBAC permissions are minimal and correct
- [ ] **Container Security**: Ensure base images are up-to-date and secure
- [ ] **Secrets Management**: No hardcoded secrets or credentials

### 🏗️ Functionality Verification
- [ ] **Core Features**: All CRDs (Project, Application, Deployment, ApplicationDomain) work correctly
- [ ] **Webhooks**: Validation webhooks function properly
- [ ] **Controllers**: All controllers reconcile resources correctly
- [ ] **Tekton Integration**: Git clone task deploys and works with pipelines
- [ ] **Resource Cleanup**: Finalizers and cleanup logic work correctly

### 📦 Container & Registry
- [ ] **Image Registry**: Verify access to `ghcr.io/kibamail/kibaship-operator`
- [ ] **Multi-arch Build**: Test on target architectures (if applicable)
- [ ] **Image Scanning**: Run security scan on container image
- [ ] **Image Tags**: Ensure proper semantic versioning (v1.0.0 format)

### 🎯 Helm Chart Verification
- [ ] **Chart Lint**: Run `helm lint deploy/helm/kibaship-operator/`
- [ ] **Template Render**: Run `helm template` to verify templates render correctly
- [ ] **Chart Dependencies**: All chart dependencies are available and compatible
- [ ] **Values Validation**: Default values work in different environments

### 🔧 Environment Testing
- [ ] **Local Development**: Works with local Kubernetes (kind/minikube)
- [ ] **CI Pipeline**: All CI checks passing on main branch
- [ ] **Staging Environment**: Deployed and tested in staging cluster (if applicable)
- [ ] **Upgrade Path**: If applicable, test upgrade from previous version
- [ ] **Rollback Plan**: Document rollback procedure if issues arise

### 🤖 CI/CD Pipeline
- [ ] **GitHub Actions**: Release workflow is configured and functional
- [ ] **Registry Access**: GHCR permissions are set up for publishing
- [ ] **Branch Protection**: Main branch is protected and requires PR reviews
- [ ] **Secrets**: No hardcoded secrets in workflows or code

## Release Process Steps

### 1. Pre-Release Validation (Local)
```bash
# Run comprehensive test suite
make test
make lint
make build

# Verify Tekton resources
./bin/kustomize build config/tekton-resources

# Test Helm chart
helm lint deploy/helm/kibaship-operator/
helm template test-release deploy/helm/kibaship-operator/ --debug

# Validate Docker build (optional - CI will do official build)
make docker-build
```

### 2. Automated Release Process (Recommended)
```bash
# Use semantic versioning - script automatically determines next version
./scripts/release.sh patch   # For bug fixes (0.0.1 -> 0.0.2)
./scripts/release.sh minor   # For new features (0.1.0 -> 0.2.0)
./scripts/release.sh major   # For breaking changes (1.0.0 -> 2.0.0)

# The script will:
# - Automatically determine next version from git tags
# - Update all version references consistently
# - Run validation tests
# - Commit changes and create tag
# - Ask for confirmation before pushing tag
# - CI/CD will handle the rest
```

### 3. Manual Release Process (Alternative)
```bash
# Update VERSION in Makefile
export NEW_VERSION="0.1.0"  # Replace with actual version

# Update all version references
sed -i '' "s/VERSION ?= .*/VERSION ?= ${NEW_VERSION}/" Makefile
sed -i '' "s/appVersion: .*/appVersion: \"${NEW_VERSION}\"/" deploy/helm/kibaship-operator/Chart.yaml
sed -i '' "s/tag: .*/tag: \"v${NEW_VERSION}\"/" deploy/helm/kibaship-operator/values.yaml

# Regenerate manifests with new version
make manifests
make build-installer

# Commit version changes
git add -A
git commit -m "chore: bump version to v${NEW_VERSION}"
git push origin main

# Create and push git tag (triggers CI/CD)
git tag -a "v${NEW_VERSION}" -m "Release v${NEW_VERSION}"
git push origin "v${NEW_VERSION}"
```

### 4. CI/CD Automated Steps (Triggered by Tag Push)
The GitHub Actions workflow automatically handles:
- [ ] **Docker Build & Push**: Multi-arch image to ghcr.io
- [ ] **GitHub Release Creation**: With generated release notes
- [ ] **Asset Attachment**: `dist/install.yaml` attached automatically
- [ ] **Release Validation**: All tests and checks run in CI
- [ ] **Version Verification**: Ensures tag matches Makefile version

### 5. Post-Release Verification
- [ ] **CI/CD Success**: Verify GitHub Actions workflow completed successfully
- [ ] **Container Image**: Verify image exists at `ghcr.io/kibamail/kibaship-operator:vX.X.X`
- [ ] **GitHub Release**: Check release page has correct assets and notes
- [ ] **Installation Test**: Test installation from release assets
```bash
kubectl apply -f https://github.com/kibamail/kibaship-operator/releases/download/vX.X.X/install.yaml
```
- [ ] **Documentation Update**: Update any external documentation
- [ ] **Community Notice**: Announce release in relevant channels

## Version Numbering Guide

- **Patch Release** (0.0.X): Bug fixes, minor improvements
- **Minor Release** (0.X.0): New features, backward compatible
- **Major Release** (X.0.0): Breaking changes, API changes

## Rollback Plan

If issues are discovered post-release:

1. **Immediate**: Revert problematic changes via patch release
2. **Communication**: Notify users of known issues and workarounds
3. **Fix Forward**: Prepare hotfix release with resolution
4. **Tag Management**: Use `git tag -d` to remove problematic tags if necessary

## Files That Require Version Updates

- `Makefile` - VERSION variable (e.g., `VERSION ?= 0.1.0`)
- `deploy/helm/kibaship-operator/Chart.yaml` - version and appVersion (both match: `version: 0.1.0`, `appVersion: "0.1.0"`)
- `deploy/helm/kibaship-operator/values.yaml` - image.tag (e.g., `tag: "v0.1.0"`)
- `CHANGELOG.md` - Add release notes
- `README.md` - Installation instructions and version references

**Note**: All versions stay in sync - operator version, chart version, and appVersion are always the same.

## Post-Release Checklist

- [ ] **Smoke Test**: Verify release works in clean environment
- [ ] **Documentation**: All docs reflect new version
- [ ] **Next Version**: Prepare main branch for next development cycle
- [ ] **Metrics**: Monitor deployment metrics and error rates
- [ ] **User Feedback**: Monitor for issues and user feedback

---

**Release Manager**: ___________
**Release Date**: ___________
**Version**: ___________
**Approval**: ___________