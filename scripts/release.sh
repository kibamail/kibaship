#!/bin/bash

# KibaShip Operator Release Script
# Usage: ./scripts/release.sh [patch|minor|major]
# Example: ./scripts/release.sh minor

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to get the latest git tag
get_latest_version() {
    # Get the latest tag, fallback to 0.0.0 if no tags exist
    local latest_tag=$(git tag -l | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1)
    if [ -z "$latest_tag" ]; then
        echo "0.0.0"
    else
        echo "${latest_tag#v}" # Remove 'v' prefix
    fi
}

# Function to increment version based on release type
increment_version() {
    local version="$1"
    local release_type="$2"

    local major=$(echo "$version" | cut -d. -f1)
    local minor=$(echo "$version" | cut -d. -f2)
    local patch=$(echo "$version" | cut -d. -f3)

    case "$release_type" in
        "patch")
            echo "$major.$minor.$((patch + 1))"
            ;;
        "minor")
            echo "$major.$((minor + 1)).0"
            ;;
        "major")
            echo "$((major + 1)).0.0"
            ;;
        *)
            log_error "Invalid release type: $release_type"
            exit 1
            ;;
    esac
}

# Show usage if no arguments or help requested
if [ $# -eq 0 ] || [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    echo "KibaShip Operator Release Script"
    echo ""
    echo "Usage: $0 [patch|minor|major]"
    echo ""
    echo "Release types:"
    echo "  patch  - Bug fixes and small improvements (x.x.X)"
    echo "  minor  - New features, backward compatible (x.X.0)"
    echo "  major  - Breaking changes, API changes (X.0.0)"
    echo ""
    echo "Examples:"
    echo "  $0 patch  # 0.1.0 -> 0.1.1"
    echo "  $0 minor  # 0.1.0 -> 0.2.0"
    echo "  $0 major  # 0.1.0 -> 1.0.0"
    exit 0
fi

RELEASE_TYPE="$1"

# Validate release type
if [[ ! "$RELEASE_TYPE" =~ ^(patch|minor|major)$ ]]; then
    log_error "Invalid release type: $RELEASE_TYPE"
    echo "Valid options: patch, minor, major"
    exit 1
fi

# Get current version and calculate new version
CURRENT_VERSION=$(get_latest_version)
NEW_VERSION=$(increment_version "$CURRENT_VERSION" "$RELEASE_TYPE")

log_info "Current version: v$CURRENT_VERSION"
log_info "Release type: $RELEASE_TYPE"
log_info "New version: v$NEW_VERSION"

echo ""
log_warning "About to create $RELEASE_TYPE release: v$CURRENT_VERSION → v$NEW_VERSION"
read -p "Continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_info "Release cancelled"
    exit 0
fi

log_info "Starting release process for version v$NEW_VERSION"

# Ensure we're on the main branch and up to date
log_info "Checking git status..."
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "main" ]; then
    log_error "Must be on main branch to create release. Current branch: $CURRENT_BRANCH"
    exit 1
fi

# Check for uncommitted changes
if ! git diff --quiet || ! git diff --cached --quiet; then
    log_error "There are uncommitted changes. Please commit or stash them first."
    git status --porcelain
    exit 1
fi

# Pull latest changes
log_info "Pulling latest changes..."
git pull origin main

# Check if tag already exists
if git tag -l | grep -q "^v$NEW_VERSION$"; then
    log_error "Tag v$NEW_VERSION already exists"
    exit 1
fi

# Run tests and validation
log_info "Running test suite..."
if ! make test > /tmp/release-test.log 2>&1; then
    log_error "Tests failed. Check /tmp/release-test.log for details"
    exit 1
fi
log_success "All tests passed"

# Run linting
log_info "Running linting..."
if ! make lint > /tmp/release-lint.log 2>&1; then
    log_error "Linting failed. Check /tmp/release-lint.log for details"
    exit 1
fi
log_success "Linting passed"

# Build project
log_info "Building project..."
if ! make build > /tmp/release-build.log 2>&1; then
    log_error "Build failed. Check /tmp/release-build.log for details"
    exit 1
fi
log_success "Build successful"

# Update version in files
log_info "Updating version references..."

# Update Makefile
sed -i.bak "s/VERSION ?= .*/VERSION ?= $NEW_VERSION/" Makefile
log_success "Updated Makefile"

# Update Helm Chart appVersion to match operator version
sed -i.bak "s/appVersion: .*/appVersion: \"$NEW_VERSION\"/" deploy/helm/kibaship-operator/Chart.yaml
log_success "Updated Helm Chart appVersion to $NEW_VERSION"

# Update Helm Chart version to match operator version (keeping in sync)
sed -i.bak "s/version: .*/version: $NEW_VERSION/" deploy/helm/kibaship-operator/Chart.yaml
log_success "Updated Helm Chart version to $NEW_VERSION"

NEW_CHART_VERSION="$NEW_VERSION" # Keep them in sync

# Update Helm values.yaml
sed -i.bak "s/tag: .*/tag: \"v$NEW_VERSION\"/" deploy/helm/kibaship-operator/values.yaml
log_success "Updated Helm values.yaml"

# Clean up backup files
rm -f Makefile.bak deploy/helm/kibaship-operator/Chart.yaml.bak deploy/helm/kibaship-operator/values.yaml.bak

# Regenerate manifests
log_info "Regenerating manifests..."
make manifests
make build-installer
log_success "Manifests regenerated"

# Validate Helm chart
log_info "Validating Helm chart..."
if ! helm lint deploy/helm/kibaship-operator/ > /tmp/release-helm-lint.log 2>&1; then
    log_error "Helm chart validation failed. Check /tmp/release-helm-lint.log"
    exit 1
fi
log_success "Helm chart validation passed"

# Test Helm template rendering
log_info "Testing Helm template rendering..."
if ! helm template test-release deploy/helm/kibaship-operator/ > /tmp/release-helm-template.log 2>&1; then
    log_error "Helm template rendering failed. Check /tmp/release-helm-template.log"
    exit 1
fi
log_success "Helm template rendering successful"

# Validate Docker build (but don't actually build for release)
log_info "Validating Docker build..."
if ! make docker-build > /tmp/release-docker-build.log 2>&1; then
    log_error "Docker build validation failed. Check /tmp/release-docker-build.log"
    exit 1
fi
log_success "Docker build validation successful"

# Commit version changes
log_info "Committing version changes..."
git add -A
git commit -m "chore: bump version to v$NEW_VERSION

- Updated operator version to v$NEW_VERSION
- Updated Helm chart to v$NEW_CHART_VERSION
- Regenerated manifests and installation files"

# Create and push tag
log_info "Creating and pushing tag..."
git tag -a "v$NEW_VERSION" -m "Release v$NEW_VERSION

This release includes:
- Updated version to v$NEW_VERSION
- All tests passing
- Updated Helm chart
- Updated installation manifests

For installation instructions, see:
- kubectl apply -f https://github.com/kibamail/kibaship-operator/releases/download/v$NEW_VERSION/install.yaml
- helm install kibaship-operator deploy/helm/kibaship-operator/"

# Push changes first
log_info "Pushing version changes to remote..."
git push origin main

log_success "Version changes pushed successfully!"

echo ""
log_warning "Ready to create release tag v$NEW_VERSION"
echo "The tag will trigger CI/CD pipeline for:"
echo "- Docker image build and push"
echo "- GitHub release creation"
echo "- Asset attachment (install.yaml)"

read -p "Push tag v$NEW_VERSION to trigger CI/CD release? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "Pushing release tag..."
    git push origin "v$NEW_VERSION"

    log_success "Release tag v$NEW_VERSION pushed successfully!"
    echo ""
    echo "✅ CI/CD pipeline will now:"
    echo "   1. Build and push Docker image to ghcr.io/kibamail/kibaship-operator:v$NEW_VERSION"
    echo "   2. Create GitHub release with dist/install.yaml"
    echo "   3. Run additional release validation"
    echo ""
    echo "🔗 Monitor progress at:"
    echo "   - Actions: https://github.com/kibamail/kibaship-operator/actions"
    echo "   - Releases: https://github.com/kibamail/kibaship-operator/releases"
else
    log_warning "Release tag not pushed. To complete release later, run:"
    echo "   git push origin v$NEW_VERSION"
fi

echo ""
echo "Next steps:"
echo "1. Monitor CI/CD pipeline execution"
echo "2. Verify Docker image is available: ghcr.io/kibamail/kibaship-operator:v$NEW_VERSION"
echo "3. Test installations:"
echo "   kubectl: kubectl apply -f https://github.com/kibamail/kibaship-operator/releases/download/v$NEW_VERSION/install.yaml"
echo "   helm: helm install kibaship-operator https://github.com/kibamail/kibaship-operator/releases/download/v$NEW_VERSION/kibaship-operator-$NEW_CHART_VERSION.tgz"
echo "4. Update documentation if needed"

echo ""
echo "GitHub Release Notes Template for v$NEW_VERSION ($RELEASE_TYPE release):"
echo "---"

case "$RELEASE_TYPE" in
    "patch")
        echo "## 🐛 KibaShip Operator v$NEW_VERSION (Patch Release)"
        echo ""
        echo "### Bug Fixes"
        echo "- [List bug fixes and small improvements here]"
        echo ""
        echo "### Other Changes"
        echo "- [List other minor changes here]"
        ;;
    "minor")
        echo "## ✨ KibaShip Operator v$NEW_VERSION (Minor Release)"
        echo ""
        echo "### 🚀 New Features"
        echo "- [List new features here]"
        echo ""
        echo "### 🐛 Bug Fixes"
        echo "- [List bug fixes here]"
        echo ""
        echo "### 📈 Improvements"
        echo "- [List improvements here]"
        ;;
    "major")
        echo "## 🎉 KibaShip Operator v$NEW_VERSION (Major Release)"
        echo ""
        echo "### 💥 Breaking Changes"
        echo "- [List breaking changes here]"
        echo "- [Include migration guide if needed]"
        echo ""
        echo "### 🚀 New Features"
        echo "- [List new features here]"
        echo ""
        echo "### 📈 Improvements"
        echo "- [List improvements here]"
        echo ""
        echo "### 🐛 Bug Fixes"
        echo "- [List bug fixes here]"
        ;;
esac

echo ""
echo "### 📦 Installation"
echo ""
echo "**Using kubectl:**"
echo "\`\`\`bash"
echo "kubectl apply -f https://github.com/kibamail/kibaship-operator/releases/download/v$NEW_VERSION/install.yaml"
echo "\`\`\`"
echo ""
echo "**Using Helm:**"
echo "\`\`\`bash"
echo "# Direct from GitHub release"
echo "helm install kibaship-operator https://github.com/kibamail/kibaship-operator/releases/download/v$NEW_VERSION/kibaship-operator-$NEW_CHART_VERSION.tgz"
echo ""
echo "# Or if using a Helm repository (future)"
echo "# helm repo add kibaship https://helm.kibaship.com"
echo "# helm install kibaship-operator kibaship/kibaship-operator --version $NEW_CHART_VERSION"
echo "\`\`\`"
echo ""
echo "### 🔧 Compatibility"
echo "- Kubernetes: 1.19+"
echo "- Tekton: v0.47.0+"
echo ""
if [ "$RELEASE_TYPE" = "major" ]; then
    echo "### ⚠️ Upgrade Notes"
    echo "This is a major release with breaking changes. Please review the migration guide before upgrading."
    echo ""
fi
echo "**Full Changelog**: https://github.com/kibamail/kibaship-operator/compare/v$CURRENT_VERSION...v$NEW_VERSION"
echo "---"