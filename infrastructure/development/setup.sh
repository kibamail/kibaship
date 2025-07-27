#!/bin/bash

# =============================================================================
# KibaShip Development Environment Setup Script
# =============================================================================
# This script creates a Kind cluster and installs Cilium 1.17.5 with
# configuration matching the staging environment.

set -euo pipefail

# =============================================================================
# Configuration Variables
# =============================================================================
CLUSTER_NAME="kibaship-dev"
CILIUM_VERSION="1.17.5"
KIND_CONFIG_FILE="kind-config.yaml"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# =============================================================================
# Color Output Functions
# =============================================================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# =============================================================================
# Dependency Check Functions
# =============================================================================
check_dependencies() {
    log_info "Checking dependencies..."
    
    local missing_deps=()
    
    if ! command -v docker &> /dev/null; then
        missing_deps+=("docker")
    fi
    
    if ! command -v kubectl &> /dev/null; then
        missing_deps+=("kubectl")
    fi
    
    if ! command -v helm &> /dev/null; then
        missing_deps+=("helm")
    fi
    
    if ! command -v kind &> /dev/null; then
        missing_deps+=("kind")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        log_error "Please install the missing dependencies and try again."
        exit 1
    fi
    
    log_success "All dependencies are installed"
}

# =============================================================================
# Kind Configuration Check
# =============================================================================
check_kind_config() {
    if [ ! -f "${SCRIPT_DIR}/${KIND_CONFIG_FILE}" ]; then
        log_error "Kind configuration file not found: ${KIND_CONFIG_FILE}"
        log_error "Please ensure ${KIND_CONFIG_FILE} exists in the same directory as this script."
        exit 1
    fi
    log_success "Kind configuration file found"
}

# =============================================================================
# Cluster Management Functions
# =============================================================================
create_kind_cluster() {
    log_info "Creating Kind cluster: ${CLUSTER_NAME}"
    
    # Check if cluster already exists
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_warning "Cluster ${CLUSTER_NAME} already exists"
        read -p "Do you want to delete and recreate it? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "Deleting existing cluster..."
            kind delete cluster --name="${CLUSTER_NAME}"
        else
            log_info "Using existing cluster"
            return 0
        fi
    fi
    
    # Create the cluster
    kind create cluster --name="${CLUSTER_NAME}" --config="${SCRIPT_DIR}/${KIND_CONFIG_FILE}"
    
    # Set kubectl context
    kubectl cluster-info --context "kind-${CLUSTER_NAME}"
    
    log_success "Kind cluster created successfully"
}

# =============================================================================
# Cilium Installation Functions
# =============================================================================
install_cilium() {
    log_info "Installing Cilium ${CILIUM_VERSION}..."
    
    # Add Helm repository
    log_info "Adding Cilium Helm repository..."
    helm repo add cilium https://helm.cilium.io/
    helm repo update
    
    # Preload Cilium image
    log_info "Preloading Cilium image into Kind nodes..."
    docker pull "quay.io/cilium/cilium:v${CILIUM_VERSION}"
    kind load docker-image "quay.io/cilium/cilium:v${CILIUM_VERSION}" --name="${CLUSTER_NAME}"
    
    # Install Cilium with configuration matching staging environment
    log_info "Installing gateway CRDs ..."
    kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_gatewayclasses.yaml --context kind-kibaship-dev
    kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_gateways.yaml --context kind-kibaship-dev
    kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_httproutes.yaml --context kind-kibaship-dev
    kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_referencegrants.yaml --context kind-kibaship-dev
    kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_grpcroutes.yaml --context kind-kibaship-dev

    log_info "Labelling worker nodes for cilium gateway..."
    kubectl label nodes --selector='node-role.kubernetes.io/worker=' ingress-ready=true --context kind-kibaship-dev
    log_info "Installing Cilium with staging-compatible configuration..."
    helm install cilium cilium/cilium \
        --version "${CILIUM_VERSION}" \
        --namespace kube-system \
        --set image.pullPolicy=IfNotPresent \
        --set ipam.mode=cluster-pool \
        --set tunnelProtocol=vxlan \
        --set gatewayAPI.enabled=true \
        --set hubble.enabled=true \
        --set hubble.relay.enabled=true \
        --set hubble.ui.enabled=true \
        --set hubble.metrics.enabled="{dns,drop,tcp,flow,icmp,http}" \
        --set prometheus.enabled=true \
        --set operator.replicas=1 \
        --set bpf.masquerade=true \
        --set kubeProxyReplacement=true \
        --set nodePort.enabled=true
    
    log_success "Cilium installation completed"
}

# =============================================================================
# Main Execution
# =============================================================================
main() {
    log_info "Starting KibaShip development environment setup..."
    
    check_dependencies
    check_kind_config
    create_kind_cluster
    install_cilium

    log_success "Development environment setup completed!"
    log_info "Cluster name: ${CLUSTER_NAME}"
    log_info "Kubectl context: kind-${CLUSTER_NAME}"
    log_info "Cluster configuration: 3 control-plane + 3 worker nodes (HA setup)"
    log_info ""
    log_info "Next steps:"
    log_info "1. Deploy your applications using kubectl or ArgoCD"
    log_info "2. Access Hubble UI: kubectl port-forward -n kube-system svc/hubble-ui 12000:80"
    log_info "3. Test connectivity: kubectl apply -f https://raw.githubusercontent.com/cilium/cilium/v${CILIUM_VERSION}/examples/kubernetes/connectivity-check/connectivity-check.yaml"
    log_info "4. Applications will be accessible at http://localhost:30080 and https://localhost:30443"
}

# Run main function
main "$@"
