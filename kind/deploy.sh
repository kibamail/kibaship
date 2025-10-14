#!/bin/bash
set -e

# Kibaship kind Cluster Deployment Script
# Creates a kind cluster and deploys all infrastructure components

echo "=========================================="
echo "Kibaship kind Cluster Deployment"
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

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
MANIFESTS_DIR="${SCRIPT_DIR}/manifests"
CONFIG_FILE="${SCRIPT_DIR}/kind-cluster-config.yaml"

# Cluster configuration
CLUSTER_NAME="kibaship"

# Check prerequisites
if ! command -v kind &> /dev/null; then
    log_error "kind is not installed. Please install kind first."
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    log_error "kubectl is not installed. Please install kubectl first."
    exit 1
fi

if [ ! -d "${MANIFESTS_DIR}" ]; then
    log_error "Manifests directory not found: ${MANIFESTS_DIR}"
    log_info "Run ./kind/prepare.sh first to generate manifests"
    exit 1
fi

################################################################################
# DELETE EXISTING CLUSTER
################################################################################

log_info "Checking for existing cluster '${CLUSTER_NAME}'..."

if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    log_warning "Cluster '${CLUSTER_NAME}' already exists. Deleting..."
    kind delete cluster --name "${CLUSTER_NAME}"
    log_success "Cluster deleted"
else
    log_info "No existing cluster found"
fi

################################################################################
# CREATE KIND CLUSTER
################################################################################

log_info "Creating kind cluster '${CLUSTER_NAME}'..."

if [ -f "${CONFIG_FILE}" ]; then
    log_info "Using config file: ${CONFIG_FILE}"
    kind create cluster --name "${CLUSTER_NAME}" --config "${CONFIG_FILE}"
else
    log_warning "Config file not found, creating cluster with defaults"
    kind create cluster --name "${CLUSTER_NAME}"
fi

log_success "Cluster '${CLUSTER_NAME}' created successfully"

# Set kubectl context
kubectl config use-context "kind-${CLUSTER_NAME}"

################################################################################
# APPLY GATEWAY API CRDs (Required for Cilium)
################################################################################

log_info "Applying Gateway API CRDs..."

kubectl apply -f "${MANIFESTS_DIR}/gateway-crds-v-1.2.0.yaml"

log_success "Gateway API CRDs applied"

################################################################################
# APPLY CILIUM CNI (Required for cluster networking)
################################################################################

log_info "Applying Cilium CNI..."

kubectl apply -f "${MANIFESTS_DIR}/cilium-v-1.18.2.yaml"

log_success "Cilium manifests applied"

# Wait for Cilium to be ready
log_info "Waiting for Cilium pods to be ready..."
echo ""

# Wait for Cilium pods to be created
until kubectl get pods -n kube-system -l k8s-app=cilium 2>/dev/null | grep -q cilium; do
    sleep 2
done

# Now wait for them to be ready, showing status every 15 seconds
while true; do
    kubectl get pods -A
    echo ""

    # Check if Cilium pods are ready
    CILIUM_READY=$(kubectl get pods -n kube-system -l k8s-app=cilium --no-headers 2>/dev/null | grep "Running" | awk '{split($2,a,"/"); if(a[1]==a[2]) print}' | wc -l | tr -d ' ')
    CILIUM_TOTAL=$(kubectl get pods -n kube-system -l k8s-app=cilium --no-headers 2>/dev/null | wc -l | tr -d ' ')

    if [ "$CILIUM_READY" -gt 0 ] && [ "$CILIUM_READY" -eq "$CILIUM_TOTAL" ]; then
        log_success "Cilium is ready"
        break
    fi

    sleep 15
done

# Wait for nodes to be ready
log_info "Waiting for nodes to be ready..."
kubectl wait --for=condition=ready nodes --all --timeout=60s

log_success "All nodes are ready"

################################################################################
# APPLY REMAINING MANIFESTS
################################################################################

log_info "Applying remaining infrastructure manifests..."

# cert-manager
log_info "Applying cert-manager..."
kubectl apply -f "${MANIFESTS_DIR}/cert-manager-v-1.18.2.yaml"

# Prometheus Operator (use server-side apply for large CRDs)
log_info "Applying Prometheus Operator..."
kubectl apply --server-side -f "${MANIFESTS_DIR}/prometheus-operator-v-0.77.1.yaml" 2>&1 | grep -v "error" || true

# Tekton Pipelines
log_info "Applying Tekton Pipelines..."
kubectl apply -f "${MANIFESTS_DIR}/tekton-pipelines-v-1.4.0.yaml"

# Valkey Operator
log_info "Applying Valkey Operator..."
kubectl apply -f "${MANIFESTS_DIR}/valkey-operator-v-0.0.59.yaml"

# MySQL Operator (apply twice to handle CRD ordering)
log_info "Applying MySQL Operator..."
kubectl apply -f "${MANIFESTS_DIR}/mysql-operator-v-9.4.0-2.2.5.yaml" 2>&1 | grep -v "error: resource mapping not found" || true
sleep 2
kubectl apply -f "${MANIFESTS_DIR}/mysql-operator-v-9.4.0-2.2.5.yaml" > /dev/null 2>&1

# Set cluster domain for MySQL operator (required for DNS resolution)
kubectl set env deployment/mysql-operator -n mysql-operator MYSQL_OPERATOR_K8S_CLUSTER_DOMAIN=cluster.local > /dev/null 2>&1 || true

# Storage Classes
log_info "Applying Storage Classes..."
kubectl apply -f "${MANIFESTS_DIR}/storage-classes.yaml"

log_success "All manifests applied successfully"

################################################################################
# WAIT FOR ALL PODS TO BE READY
################################################################################

log_info "Waiting for all pods to be ready..."
echo ""

# Wait for all pods to be ready, showing status every 15 seconds
while true; do
    kubectl get pods -A
    echo ""

    # Check if all pods are Running (excluding Completed and Pending)
    NOT_READY=$(kubectl get pods -A --no-headers 2>/dev/null | grep -v "Completed" | grep -v "Pending" | grep -v "Running" | wc -l | tr -d ' ')

    # Also check that all Running pods have all containers ready (X/X format)
    NOT_READY_CONTAINERS=$(kubectl get pods -A --no-headers 2>/dev/null | grep "Running" | awk '{split($3,a,"/"); if(a[1]!=a[2]) print}' | wc -l | tr -d ' ')

    if [ "$NOT_READY" -eq 0 ] && [ "$NOT_READY_CONTAINERS" -eq 0 ]; then
        log_success "All pods are running!"
        break
    fi

    sleep 15
done

################################################################################
# SUMMARY
################################################################################

echo ""
log_success "=========================================="
log_success "Deployment Complete!"
log_success "=========================================="
echo ""
log_info "Cluster: ${CLUSTER_NAME}"
log_info "Context: kind-${CLUSTER_NAME}"
echo ""
log_info "Deployed components:"
echo "  ✓ Cilium CNI v1.18.2"
echo "  ✓ Gateway API v1.2.0"
echo "  ✓ cert-manager v1.18.2"
echo "  ✓ Prometheus Operator v0.77.1"
echo "  ✓ Tekton Pipelines v1.4.0"
echo "  ✓ Valkey Operator v0.0.59"
echo "  ✓ MySQL Operator v9.4.0-2.2.5"
echo "  ✓ Storage Classes"
echo ""
log_info "Final cluster status:"
kubectl get pods -A
echo ""
log_info "To delete the cluster:"
echo "  kind delete cluster --name ${CLUSTER_NAME}"
echo ""
