#!/bin/bash
set -e

# Kibaship Image Build, Load, and Deploy Script
# Builds all Kibaship images, loads them into the kind cluster,
# generates e2e manifests, applies them, and restarts deployments

echo "=========================================="
echo "Kibaship Image Build and Deployment"
echo "=========================================="

# Color output helpers
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

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

# Get script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "${SCRIPT_DIR}")"

# Configuration
CLUSTER_NAME="${CLUSTER_NAME:-kibaship}"
VERSION="${VERSION:-1.0.0}"
IMAGE_TAG_BASE="kibamail/kibaship"

# Check prerequisites
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v kind &> /dev/null; then
    log_error "kind is not installed. Please install kind first."
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    log_error "kubectl is not installed. Please install kubectl first."
    exit 1
fi

# Check if cluster exists
if ! kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    log_error "Cluster '${CLUSTER_NAME}' does not exist."
    log_info "Run ./kind/deploy.sh to create the cluster first"
    exit 1
fi

# Check if cert-manager is installed (required for webhook certificates)
log_info "Checking prerequisites..."
if ! kubectl get crd certificates.cert-manager.io >/dev/null 2>&1; then
    log_error "cert-manager CRDs not found"
    log_error "The cluster infrastructure is not fully set up"
    log_info "Please run ./kind/deploy.sh first to install:"
    echo "  - Gateway API CRDs"
    echo "  - Cilium CNI"
    echo "  - cert-manager"
    echo "  - Prometheus Operator"
    echo "  - Tekton Pipelines"
    echo "  - Valkey Operator"
    echo "  - MySQL Operator"
    echo ""
    exit 1
fi

log_success "Prerequisites check passed"

cd "${PROJECT_ROOT}"

################################################################################
# BUILD IMAGES IN PARALLEL
################################################################################

log_info "Building Kibaship images in parallel..."
log_info "This will be much faster than sequential builds..."
echo ""

# Create temp directory for build logs
BUILD_LOG_DIR=$(mktemp -d)
BUILD_FAILED=0

# Function to build an image
build_image() {
    local name=$1
    local tag=$2
    local dockerfile=$3
    local context=$4
    local log_file="${BUILD_LOG_DIR}/${name}.log"

    log_info "Building ${name}..."
    if docker build -t "${tag}" -f "${dockerfile}" "${context}" > "${log_file}" 2>&1; then
        log_success "Built: ${tag}"
        return 0
    else
        log_error "Failed to build: ${tag}"
        log_error "See log: ${log_file}"
        return 1
    fi
}

# Build all images in parallel
build_image "operator" "${IMAGE_TAG_BASE}:v${VERSION}" "Dockerfile" "." &
PID1=$!

build_image "apiserver" "${IMAGE_TAG_BASE}-apiserver:v${VERSION}" "Dockerfile.apiserver" "." &
PID2=$!

build_image "registry-auth" "${IMAGE_TAG_BASE}-registry-auth:v${VERSION}" "Dockerfile.registry-auth" "." &
PID3=$!

build_image "cert-manager-webhook" "${IMAGE_TAG_BASE}-cert-manager-webhook:v${VERSION}" \
    "webhooks/cert-manager-kibaship-webhook/Dockerfile" "webhooks/cert-manager-kibaship-webhook" &
PID4=$!

build_image "railpack-cli" "kibamail/kibaship-railpack-cli:v${VERSION}" \
    "build/railpack-cli/Dockerfile" "build/railpack-cli" &
PID5=$!

build_image "railpack-build" "kibamail/kibaship-railpack-build:v${VERSION}" \
    "build/railpack-build/Dockerfile" "build/railpack-build" &
PID6=$!

# Wait for all builds to complete
log_info "Waiting for all builds to complete..."

wait $PID1 || BUILD_FAILED=1
wait $PID2 || BUILD_FAILED=1
wait $PID3 || BUILD_FAILED=1
wait $PID4 || BUILD_FAILED=1
wait $PID5 || BUILD_FAILED=1
wait $PID6 || BUILD_FAILED=1

# Clean up temp directory
rm -rf "${BUILD_LOG_DIR}"

if [ $BUILD_FAILED -eq 1 ]; then
    log_error "One or more builds failed!"
    exit 1
fi

echo ""
log_success "All images built successfully!"

################################################################################
# LOAD IMAGES INTO KIND IN PARALLEL
################################################################################

echo ""
log_info "Loading images into kind cluster '${CLUSTER_NAME}' in parallel..."
echo ""

# Array of images to load
IMAGES=(
    "${IMAGE_TAG_BASE}:v${VERSION}"
    "${IMAGE_TAG_BASE}-apiserver:v${VERSION}"
    "${IMAGE_TAG_BASE}-registry-auth:v${VERSION}"
    "${IMAGE_TAG_BASE}-cert-manager-webhook:v${VERSION}"
    "kibamail/kibaship-railpack-cli:v${VERSION}"
    "kibamail/kibaship-railpack-build:v${VERSION}"
)

# Function to load an image
load_image() {
    local image=$1
    log_info "Loading: ${image}"
    if kind load docker-image "${image}" --name "${CLUSTER_NAME}" 2>/dev/null; then
        log_success "Loaded: ${image}"
        return 0
    else
        log_error "Failed to load: ${image}"
        return 1
    fi
}

# Load all images in parallel
LOAD_FAILED=0
PIDS=()

for image in "${IMAGES[@]}"; do
    load_image "${image}" &
    PIDS+=($!)
done

# Wait for all loads to complete
for pid in "${PIDS[@]}"; do
    wait $pid || LOAD_FAILED=1
done

if [ $LOAD_FAILED -eq 1 ]; then
    log_error "One or more image loads failed!"
    exit 1
fi

echo ""
log_success "All images loaded successfully!"

################################################################################
# BUILD E2E MANIFESTS
################################################################################

echo ""
log_info "Building e2e manifests with v${VERSION} images..."
echo ""

if ! make build-e2e-installers > /dev/null 2>&1; then
    log_error "Failed to build e2e manifests"
    exit 1
fi

log_success "E2E manifests built successfully"

################################################################################
# APPLY E2E MANIFESTS
################################################################################

echo ""
log_info "Applying e2e manifests in correct order..."
echo ""

MANIFESTS_DIR="${PROJECT_ROOT}/dist/e2e/manifests"

# Apply manifests in order based on e2e test suite
log_info "Applying operator (CRDs + operator deployment)..."
kubectl apply -f "${MANIFESTS_DIR}/operator.yaml"

log_info "Applying buildkit..."
kubectl apply -f "${MANIFESTS_DIR}/buildkit.yaml"

log_info "Applying tekton resources..."
kubectl apply -f "${MANIFESTS_DIR}/tekton-resources.yaml"

log_info "Applying registry..."
kubectl apply -f "${MANIFESTS_DIR}/registry.yaml"

log_info "Applying registry-auth..."
kubectl apply -f "${MANIFESTS_DIR}/registry-auth.yaml"

log_info "Applying api-server..."
kubectl apply -f "${MANIFESTS_DIR}/api-server.yaml"

log_info "Applying cert-manager-webhook..."
kubectl apply -f "${MANIFESTS_DIR}/cert-manager-webhook.yaml"

log_info "Applying cert-manager-webhook-kube-system..."
kubectl apply -f "${MANIFESTS_DIR}/cert-manager-webhook-kube-system.yaml"

log_success "All manifests applied successfully"

################################################################################
# REDEPLOY COMPONENTS (DELETE + APPLY)
################################################################################

echo ""
log_info "Redeploying components (delete + apply) to pick up new images..."
echo ""

# Delete manifests in reverse order (except operator CRDs)
log_info "Deleting cert-manager-webhook-kube-system..."
kubectl delete -f "${MANIFESTS_DIR}/cert-manager-webhook-kube-system.yaml" --ignore-not-found=true

log_info "Deleting cert-manager-webhook..."
kubectl delete -f "${MANIFESTS_DIR}/cert-manager-webhook.yaml" --ignore-not-found=true

log_info "Deleting api-server..."
kubectl delete -f "${MANIFESTS_DIR}/api-server.yaml" --ignore-not-found=true

log_info "Deleting registry-auth..."
kubectl delete -f "${MANIFESTS_DIR}/registry-auth.yaml" --ignore-not-found=true

log_info "Deleting registry..."
kubectl delete -f "${MANIFESTS_DIR}/registry.yaml" --ignore-not-found=true

log_info "Deleting tekton resources..."
kubectl delete -f "${MANIFESTS_DIR}/tekton-resources.yaml" --ignore-not-found=true

log_info "Deleting buildkit..."
kubectl delete -f "${MANIFESTS_DIR}/buildkit.yaml" --ignore-not-found=true

log_info "Deleting operator deployments (keeping CRDs)..."
kubectl delete deployment,service,serviceaccount,role,rolebinding,clusterrole,clusterrolebinding -n kibaship --all --ignore-not-found=true

log_success "All components deleted"

echo ""
log_info "Waiting for pods to terminate..."
sleep 5

echo ""
log_info "Reapplying manifests in correct order..."
echo ""

# Reapply manifests in order
log_info "Applying operator (CRDs + operator deployment)..."
kubectl apply -f "${MANIFESTS_DIR}/operator.yaml"

log_info "Applying buildkit..."
kubectl apply -f "${MANIFESTS_DIR}/buildkit.yaml"

log_info "Applying tekton resources..."
kubectl apply -f "${MANIFESTS_DIR}/tekton-resources.yaml"

log_info "Applying registry..."
kubectl apply -f "${MANIFESTS_DIR}/registry.yaml"

log_info "Applying registry-auth..."
kubectl apply -f "${MANIFESTS_DIR}/registry-auth.yaml"

log_info "Applying api-server..."
kubectl apply -f "${MANIFESTS_DIR}/api-server.yaml"

log_info "Applying cert-manager-webhook..."
kubectl apply -f "${MANIFESTS_DIR}/cert-manager-webhook.yaml"

log_info "Applying cert-manager-webhook-kube-system..."
kubectl apply -f "${MANIFESTS_DIR}/cert-manager-webhook-kube-system.yaml"

log_success "All manifests reapplied successfully"

echo ""
log_info "Waiting for deployments to be ready..."
echo ""

# Wait for kibaship deployments
log_info "Waiting for kibaship namespace deployments..."
kubectl wait --for=condition=available deployment --all -n kibaship --timeout=5m 2>/dev/null || log_warning "Some kibaship deployments may not be ready yet"

# Wait for registry deployments
log_info "Waiting for registry namespace deployments..."
kubectl wait --for=condition=available deployment --all -n registry --timeout=5m 2>/dev/null || log_warning "Some registry deployments may not be ready yet"

log_success "All deployments ready"

################################################################################
# SUMMARY
################################################################################

echo ""
log_success "=========================================="
log_success "All Images Deployed Successfully!"
log_success "=========================================="
echo ""
log_info "Cluster: ${CLUSTER_NAME}"
log_info "Version: v${VERSION}"
echo ""
log_info "Deployed images:"
for image in "${IMAGES[@]}"; do
    echo "  ✓ ${image}"
done
echo ""
log_info "Components deployed:"
echo "  ✓ Kibaship Operator"
echo "  ✓ Kibaship API Server"
echo "  ✓ Cert Manager Webhook"
echo "  ✓ Registry Auth Service"
echo "  ✓ Docker Registry"
echo "  ✓ BuildKit"
echo "  ✓ Tekton Resources"
echo ""
log_info "To verify deployments:"
echo "  kubectl get pods -n kibaship"
echo "  kubectl get pods -n registry"
echo ""
