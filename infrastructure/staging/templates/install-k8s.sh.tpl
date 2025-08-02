#!/bin/bash

# =============================================================================
# KibaShip Kubernetes Cluster Installation Script
# =============================================================================
# This script installs a Kubernetes cluster using Kubespray on the staging
# environment. It sets up the Python virtual environment, installs dependencies,
# and runs the Ansible playbook to deploy the cluster.

set -euo pipefail

# =============================================================================
# Configuration
# =============================================================================

CLUSTER_NAME="${cluster_name}"
SCRIPT_DIR="$(cd "$(dirname "$${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/home/ubuntu/k8s-install.log"
KUBESPRAY_DIR="/home/ubuntu/kubespray"
INVENTORY_PATH="$${KUBESPRAY_DIR}/inventory/$${CLUSTER_NAME}/inventory.ini"

# =============================================================================
# Logging Functions
# =============================================================================

log() {
    local message="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$${timestamp}] $${message}" | tee -a "$${LOG_FILE}"
}

log_error() {
    local message="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$${timestamp}] ERROR: $${message}" | tee -a "$${LOG_FILE}" >&2
}

log_success() {
    local message="$1"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$${timestamp}] SUCCESS: $${message}" | tee -a "$${LOG_FILE}"
}

# =============================================================================
# Validation Functions
# =============================================================================

validate_environment() {
    log "Validating environment prerequisites..."

    if [[ ! -d "$${KUBESPRAY_DIR}" ]]; then
        log_error "Kubespray directory not found at $${KUBESPRAY_DIR}"
        exit 1
    fi

    if [[ ! -f "$${INVENTORY_PATH}" ]]; then
        log_error "Inventory file not found at $${INVENTORY_PATH}"
        exit 1
    fi

    if [[ ! -f "$${KUBESPRAY_DIR}/requirements.txt" ]]; then
        log_error "Kubespray requirements.txt not found"
        exit 1
    fi

    log_success "Environment validation completed"
}

# =============================================================================
# Installation Functions
# =============================================================================

install_python_dependencies() {
    log "Installing Python dependencies..."

    cd "$${KUBESPRAY_DIR}"

    log "Installing python3.12-venv package..."
    sudo apt update
    sudo apt install -y python3.12-venv

    log "Creating Python virtual environment..."
    python3 -m venv venv

    log "Activating virtual environment..."
    source venv/bin/activate

    log "Installing Python requirements..."
    pip3 install -r requirements.txt

    log_success "Python dependencies installed successfully"
}

deploy_kubernetes_cluster() {
    log "Starting Kubernetes cluster deployment..."

    cd "$${KUBESPRAY_DIR}"
    source venv/bin/activate

    log "Running Ansible playbook for cluster deployment..."
    log "Inventory: $${INVENTORY_PATH}"
    log "Cluster: $${CLUSTER_NAME}"

    # Force Ansible to output colors and preserve them while logging
    export ANSIBLE_FORCE_COLOR=true
    export ANSIBLE_STDOUT_CALLBACK=default

    # Use script command to preserve colors while logging
    script -q -c "ansible-playbook \
        -i '$${INVENTORY_PATH}' \
        cluster.yml \
        -b \
        -vv" /dev/null | tee -a "$${LOG_FILE}"

    if [[ $${PIPESTATUS[0]} -eq 0 ]]; then
        log_success "Kubernetes cluster deployment completed successfully"
    else
        log_error "Kubernetes cluster deployment failed"
        exit 1
    fi
}

# =============================================================================
# Main Execution
# =============================================================================

main() {
    log "Starting KibaShip Kubernetes installation for cluster: $${CLUSTER_NAME}"
    log "Log file: $${LOG_FILE}"
    log "Kubespray directory: $${KUBESPRAY_DIR}"

    validate_environment
    install_python_dependencies
    deploy_kubernetes_cluster

    log_success "KibaShip Kubernetes installation completed successfully!"
    log "Cluster '$${CLUSTER_NAME}' is now ready for use"
    log "Check the kubeconfig at: /home/ubuntu/.kube/config"
}

if [[ "$${BASH_SOURCE[0]}" == "$${0}" ]]; then
    main "$@"
fi
