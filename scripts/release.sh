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

# Run linting first (fast feedback)
log_info "Running linting..."
if ! make lint > /tmp/release-lint.log 2>&1; then
    log_error "Linting failed. Check /tmp/release-lint.log for details"
    tail -20 /tmp/release-lint.log
    exit 1
fi
log_success "Linting passed"

# Run unit tests
log_info "Running unit tests..."
if ! make test > /tmp/release-test.log 2>&1; then
    log_error "Unit tests failed. Check /tmp/release-test.log for details"
    tail -20 /tmp/release-test.log
    exit 1
fi
log_success "All unit tests passed"

# Run E2E tests
log_info "Running E2E tests (this may take several minutes)..."
if ! make test-e2e > /tmp/release-test-e2e.log 2>&1; then
    log_error "E2E tests failed. Check /tmp/release-test-e2e.log for details"
    tail -30 /tmp/release-test-e2e.log
    exit 1
fi
log_success "All E2E tests passed"

# Build project
log_info "Building project..."
if ! make build > /tmp/release-build.log 2>&1; then
    log_error "Build failed. Check /tmp/release-build.log for details"
    tail -20 /tmp/release-build.log
    exit 1
fi
log_success "Build successful"

# Validate all 6 Docker image builds
log_info "Validating Docker builds for all 6 images..."

log_info "  [1/6] Building operator image..."
if ! make docker-build > /tmp/release-docker-build-operator.log 2>&1; then
    log_error "Operator Docker build validation failed. Check /tmp/release-docker-build-operator.log"
    tail -20 /tmp/release-docker-build-operator.log
    exit 1
fi
log_success "  [1/6] Operator image build successful"

log_info "  [2/6] Building API server image..."
if ! make build-apiserver > /tmp/release-docker-build-apiserver.log 2>&1; then
    log_error "API server Docker build validation failed. Check /tmp/release-docker-build-apiserver.log"
    tail -20 /tmp/release-docker-build-apiserver.log
    exit 1
fi
log_success "  [2/6] API server image build successful"

log_info "  [3/6] Building cert-manager webhook image..."
if ! make docker-build-cert-manager-webhook > /tmp/release-docker-build-webhook.log 2>&1; then
    log_error "Cert-manager webhook Docker build validation failed. Check /tmp/release-docker-build-webhook.log"
    tail -20 /tmp/release-docker-build-webhook.log
    exit 1
fi
log_success "  [3/6] Cert-manager webhook image build successful"

log_info "  [4/6] Building registry-auth image..."
if ! make docker-build-registry-auth > /tmp/release-docker-build-registry-auth.log 2>&1; then
    log_error "Registry-auth Docker build validation failed. Check /tmp/release-docker-build-registry-auth.log"
    tail -20 /tmp/release-docker-build-registry-auth.log
    exit 1
fi
log_success "  [4/6] Registry-auth image build successful"

log_info "  [5/6] Building railpack-cli image..."
if ! make docker-build-railpack-cli > /tmp/release-docker-build-railpack-cli.log 2>&1; then
    log_error "Railpack CLI Docker build validation failed. Check /tmp/release-docker-build-railpack-cli.log"
    tail -20 /tmp/release-docker-build-railpack-cli.log
    exit 1
fi
log_success "  [5/6] Railpack CLI image build successful"

log_info "  [6/6] Building railpack-build image..."
if ! make docker-build-railpack-build > /tmp/release-docker-build-railpack-build.log 2>&1; then
    log_error "Railpack build Docker build validation failed. Check /tmp/release-docker-build-railpack-build.log"
    tail -20 /tmp/release-docker-build-railpack-build.log
    exit 1
fi
log_success "  [6/6] Railpack build image build successful"

log_success "All 6 Docker image builds validated successfully"

# Update version in files
log_info "Updating version references..."

# Update Makefile
sed -i.bak "s/^VERSION ?= .*/VERSION ?= $NEW_VERSION/" Makefile
log_success "Updated Makefile"

# Clean up backup files
rm -f Makefile.bak

# Regenerate manifests
log_info "Regenerating manifests..."
make manifests
make build-installer
log_success "Manifests regenerated"

# Commit version changes
log_info "Committing version changes..."
git add -A
git commit -m "chore: bump version to v$NEW_VERSION

- Updated operator version to v$NEW_VERSION
- Regenerated manifests and installation files
- All tests passing (unit, e2e, lint)
- All 6 Docker images validated"

# Create and push tag
log_info "Creating release tag..."
git tag -a "v$NEW_VERSION" -m "Release v$NEW_VERSION

This release includes:
- Updated version to v$NEW_VERSION
- All tests passing (unit, e2e, lint)
- All 6 container images validated
- Updated installation manifests

For installation instructions, see:
kubectl apply -f https://github.com/kibamail/kibaship/releases/download/v$NEW_VERSION/install.yaml"

# Push changes first
log_info "Pushing version changes to remote..."
git push origin main

log_success "Version changes pushed successfully!"

echo ""
log_warning "Ready to create release tag v$NEW_VERSION"
echo "The tag will trigger CI/CD pipeline for:"
echo "- Building and pushing all 6 Docker images"
echo "- Building CLI binaries for all platforms"
echo "- Creating GitHub release with assets"

read -p "Push tag v$NEW_VERSION to trigger CI/CD release? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "Pushing release tag..."

    git push origin "v$NEW_VERSION"

    log_success "Release tag v$NEW_VERSION pushed successfully!"
    echo ""
    echo "✅ CI/CD pipeline will now:"
    echo "   1. Build and push all 6 Docker images:"
    echo "      - kibamail/kibaship:v$NEW_VERSION"
    echo "      - kibamail/kibaship-apiserver:v$NEW_VERSION"
    echo "      - kibamail/kibaship-cert-manager-webhook:v$NEW_VERSION"
    echo "      - kibamail/kibaship-registry-auth:v$NEW_VERSION"
    echo "      - kibamail/kibaship-railpack-cli:v$NEW_VERSION"
    echo "      - kibamail/kibaship-railpack-build:v$NEW_VERSION"
    echo ""
    echo "   2. Build CLI binaries for 5 platforms:"
    echo "      - Linux (AMD64, ARM64)"
    echo "      - macOS (Intel, Apple Silicon)"
    echo "      - Windows (AMD64)"
    echo ""
    echo "   3. Create GitHub releases with manifests and binaries"
    echo ""
    echo "🔗 Monitor progress at:"
    echo "   - Actions: https://github.com/kibamail/kibaship/actions"
    echo "   - Releases: https://github.com/kibamail/kibaship/releases"
else
    log_warning "Release tag not pushed. To complete release later, run:"
    echo "   git push origin v$NEW_VERSION"
fi

echo ""
echo "Next steps:"
echo "1. Monitor CI/CD pipeline execution"
echo "2. Verify all 6 Docker images are available at docker.io"
echo "3. Verify CLI binaries are in GitHub release"
echo "4. Test installation:"
echo "   kubectl apply -f https://github.com/kibamail/kibaship/releases/download/v$NEW_VERSION/install.yaml"
echo "5. Update documentation if needed"

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
echo "kubectl apply -f https://github.com/kibamail/kibaship/releases/download/v$NEW_VERSION/install.yaml"
echo "\`\`\`"
echo ""
echo "### 🔧 Compatibility"
echo "- Kubernetes: 1.19+"
echo "- Tekton: 1.4.0+"
echo "- Cilium: 1.18.0+"
echo ""
if [ "$RELEASE_TYPE" = "major" ]; then
    echo "### ⚠️ Upgrade Notes"
    echo "This is a major release with breaking changes. Please review the migration guide before upgrading."
    echo ""
fi
echo "**Full Changelog**: https://github.com/kibamail/kibaship/compare/v$CURRENT_VERSION...v$NEW_VERSION"
echo "---"
