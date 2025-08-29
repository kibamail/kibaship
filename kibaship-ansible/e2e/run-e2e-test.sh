#!/bin/bash

# Kibaship E2E Testing Script with DigitalOcean
# Provisions multi-node infrastructure, runs Ansible playbooks, and cleans up

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TERRAFORM_DIR="$SCRIPT_DIR/terraform"
SSH_KEY_PATH="$SCRIPT_DIR/.ssh/kibaship-e2e"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

usage() {
    cat << EOF
Kibaship E2E Testing Script - Multi-Node HA Cluster

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    up          Provision infrastructure and run E2E tests
    provision   Only provision the infrastructure
    test        Run tests on existing infrastructure  
    destroy     Destroy infrastructure
    status      Show infrastructure status
    ssh         SSH into a cluster node
    
Options:
    --skip-tests    Skip running Ansible playbooks (provision only)
    --keep          Don't destroy infrastructure after tests
    --help          Show this help message

Examples:
    $0 up                    # Full E2E test cycle (3 CP + 3 workers + 2 LBs)
    $0 provision             # Just create the cluster infrastructure
    $0 test                  # Run tests on existing cluster
    $0 destroy               # Clean up everything
    $0 ssh                   # SSH into first control plane node

EOF
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check for required tools
    local missing_tools=()
    
    command -v terraform >/dev/null 2>&1 || missing_tools+=("terraform")
    command -v ssh >/dev/null 2>&1 || missing_tools+=("ssh")
    command -v jq >/dev/null 2>&1 || missing_tools+=("jq")
    
    # Check for Ansible in virtual environment
    if [ -f "$PROJECT_ROOT/venv/bin/ansible" ]; then
        log "Found Ansible in virtual environment"
    else
        missing_tools+=("ansible (virtual environment)")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        error "Missing required tools: ${missing_tools[*]}"
        echo "Please install the missing tools and try again."
        exit 1
    fi
    
    # Check for SSH key
    if [ ! -f "$SSH_KEY_PATH" ]; then
        error "SSH key not found at $SSH_KEY_PATH"
        echo "Run: ssh-keygen -t ed25519 -f $SSH_KEY_PATH -N '' -C 'kibaship-e2e'"
        exit 1
    fi
    
    # Check for terraform.tfvars
    if [ ! -f "$TERRAFORM_DIR/terraform.tfvars" ]; then
        error "Terraform variables file not found"
        echo "Please copy terraform.tfvars.example to terraform.tfvars and add your DigitalOcean token:"
        echo "  cp $TERRAFORM_DIR/terraform.tfvars.example $TERRAFORM_DIR/terraform.tfvars"
        echo "  # Edit terraform.tfvars and add your DO token"
        exit 1
    fi
    
    success "Prerequisites check passed"
}

provision_infrastructure() {
    log "Provisioning DigitalOcean multi-node cluster infrastructure..."
    cd "$TERRAFORM_DIR"
    
    # Initialize Terraform
    log "Initializing Terraform..."
    terraform init
    
    # Plan the deployment
    log "Planning infrastructure (3 control planes + 3 workers + 2 load balancers)..."
    terraform plan
    
    # Apply the configuration
    log "Creating infrastructure..."
    if terraform apply -auto-approve; then
        success "Infrastructure provisioned successfully"
        
        # Get cluster info
        log "Cluster Infrastructure Summary:"
        echo "Control Planes: $(terraform output -json control_plane_ips | jq -r '.[]' | wc -l)"
        echo "Workers: $(terraform output -json worker_ips | jq -r '.[]' | wc -l)"
        echo "Kube API LB: $(terraform output -raw kube_api_lb_ip)"
        echo "Ingress LB: $(terraform output -raw ingress_lb_ip)"
        
        # Wait for all nodes to be SSH ready
        log "Waiting for all nodes to be SSH ready..."
        local all_ips=(
            $(terraform output -json control_plane_ips | jq -r '.[]')
            $(terraform output -json worker_ips | jq -r '.[]')
        )
        
        for ip in "${all_ips[@]}"; do
            log "Waiting for SSH on $ip..."
            for i in {1..30}; do
                if ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no -i "$SSH_KEY_PATH" root@"$ip" echo "SSH Ready" >/dev/null 2>&1; then
                    success "SSH connection established to $ip"
                    break
                fi
                echo -n "."
                sleep 10
            done
            
            if [ $i -eq 30 ]; then
                error "SSH connection timeout for $ip"
                exit 1
            fi
        done
        
    else
        error "Failed to provision infrastructure"
        exit 1
    fi
}

generate_inventory() {
    log "Generating dynamic inventory for multi-node cluster..."
    cd "$TERRAFORM_DIR"
    
    # Get cluster info from Terraform
    local cluster_info=$(terraform output -json cluster_info)
    local cp_ips=($(terraform output -json control_plane_ips | jq -r '.[]'))
    local cp_private_ips=($(terraform output -json control_plane_private_ips | jq -r '.[]'))
    local cp_names=($(terraform output -json control_plane_names | jq -r '.[]'))
    local worker_ips=($(terraform output -json worker_ips | jq -r '.[]'))
    local worker_private_ips=($(terraform output -json worker_private_ips | jq -r '.[]'))
    local worker_names=($(terraform output -json worker_names | jq -r '.[]'))
    local kube_api_lb_ip=$(terraform output -raw kube_api_lb_ip)
    local ingress_lb_ip=$(terraform output -raw ingress_lb_ip)
    
    # Generate inventory from template
    cp "$PROJECT_ROOT/inventory/e2e/hosts.yml" "$PROJECT_ROOT/inventory/e2e/hosts-active.yml"
    
    # Replace placeholders for control planes
    for i in {0..2}; do
        local cp_num=$((i + 1))
        sed -i "s/CP${cp_num}_PUBLIC_IP_PLACEHOLDER/${cp_ips[$i]}/g" "$PROJECT_ROOT/inventory/e2e/hosts-active.yml"
        sed -i "s/CP${cp_num}_PRIVATE_IP_PLACEHOLDER/${cp_private_ips[$i]}/g" "$PROJECT_ROOT/inventory/e2e/hosts-active.yml"
    done
    
    # Replace placeholders for workers
    for i in {0..2}; do
        local w_num=$((i + 1))
        sed -i "s/W${w_num}_PUBLIC_IP_PLACEHOLDER/${worker_ips[$i]}/g" "$PROJECT_ROOT/inventory/e2e/hosts-active.yml"
        sed -i "s/W${w_num}_PRIVATE_IP_PLACEHOLDER/${worker_private_ips[$i]}/g" "$PROJECT_ROOT/inventory/e2e/hosts-active.yml"
    done
    
    # Replace load balancer placeholders
    sed -i "s/KUBE_API_LB_IP_PLACEHOLDER/$kube_api_lb_ip/g" "$PROJECT_ROOT/inventory/e2e/hosts-active.yml"
    sed -i "s/INGRESS_LB_IP_PLACEHOLDER/$ingress_lb_ip/g" "$PROJECT_ROOT/inventory/e2e/hosts-active.yml"
    
    # Replace timestamp in hostnames
    local timestamp=$(echo "${cp_names[0]}" | grep -o '[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}-[0-9]\{4\}')
    sed -i "s/TIMESTAMP/$timestamp/g" "$PROJECT_ROOT/inventory/e2e/hosts-active.yml"
    
    success "Dynamic inventory generated successfully"
    log "Control Plane Load Balancer: $kube_api_lb_ip:6443"
    log "Ingress Load Balancer: $ingress_lb_ip"
}

run_ansible_tests() {
    log "Running Kibaship HA cluster deployment E2E test..."
    cd "$PROJECT_ROOT"
    
    # Generate inventory
    generate_inventory
    
    # Activate Python virtual environment
    source "$PROJECT_ROOT/venv/bin/activate"
    
    # Test connectivity to all nodes
    log "Testing Ansible connectivity to all cluster nodes..."
    if ansible -i "$PROJECT_ROOT/inventory/e2e/hosts-active.yml" all -m ping; then
        success "Ansible connectivity confirmed to all nodes"
    else
        error "Ansible connectivity failed"
        exit 1
    fi
    
    # Run production cluster deployment
    log "Deploying Kibaship Kubernetes HA cluster..."
    log "Cluster configuration: 3 control planes + 3 workers + HA load balancers"
    cd "$PROJECT_ROOT"
    ANSIBLE_ROLES_PATH="./roles" ansible-playbook -i "$PROJECT_ROOT/inventory/e2e/hosts-active.yml" cluster.yml
    
    if [ $? -eq 0 ]; then
        success "Kibaship Kubernetes HA cluster deployed successfully!"
        log "Production-ready cluster is now available"
        
        # Show cluster access information
        cd "$TERRAFORM_DIR"
        local kube_api_lb_ip=$(terraform output -raw kube_api_lb_ip)
        local ingress_lb_ip=$(terraform output -raw ingress_lb_ip)
        
        echo ""
        log "=== CLUSTER ACCESS INFORMATION ==="
        echo "Kubernetes API: https://$kube_api_lb_ip:6443"
        echo "Ingress Endpoint: http://$ingress_lb_ip"
        echo "SSH to first control plane: ssh -i $SSH_KEY_PATH root@$(terraform output -json control_plane_ips | jq -r '.[0]')"
        echo ""
    else
        error "HA cluster deployment failed"
        exit 1
    fi
}

destroy_infrastructure() {
    log "Destroying multi-node infrastructure..."
    cd "$TERRAFORM_DIR"
    
    if terraform destroy -auto-approve; then
        success "Infrastructure destroyed successfully"
        
        # Clean up generated files
        rm -f "$PROJECT_ROOT/inventory/e2e/hosts-active.yml"
    else
        error "Failed to destroy infrastructure"
        exit 1
    fi
}

show_status() {
    log "Multi-Node Cluster Status:"
    cd "$TERRAFORM_DIR"
    
    if terraform show >/dev/null 2>&1; then
        terraform show
        echo ""
        log "Quick Status Summary:"
        echo "Control Plane IPs: $(terraform output -json control_plane_ips 2>/dev/null | jq -r '.[]' | tr '\n' ' ' || echo 'N/A')"
        echo "Worker IPs: $(terraform output -json worker_ips 2>/dev/null | jq -r '.[]' | tr '\n' ' ' || echo 'N/A')"
        echo "Kube API LB: $(terraform output -raw kube_api_lb_ip 2>/dev/null || echo 'N/A')"
        echo "Ingress LB: $(terraform output -raw ingress_lb_ip 2>/dev/null || echo 'N/A')"
    else
        warning "No infrastructure found or Terraform not initialized"
    fi
}

ssh_to_server() {
    cd "$TERRAFORM_DIR"
    local cp_ips=($(terraform output -json control_plane_ips 2>/dev/null | jq -r '.[]'))
    
    if [ ${#cp_ips[@]} -eq 0 ]; then
        error "No infrastructure found or control plane IPs unavailable"
        exit 1
    fi
    
    local first_cp_ip="${cp_ips[0]}"
    log "Connecting to first control plane: $first_cp_ip..."
    ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@"$first_cp_ip"
}

# Parse command line arguments
COMMAND="$1"
SKIP_TESTS=false
KEEP_INFRA=false

shift

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        --keep)
            KEEP_INFRA=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Execute commands
case $COMMAND in
    up)
        check_prerequisites
        provision_infrastructure
        if [ "$SKIP_TESTS" = false ]; then
            run_ansible_tests
        fi
        if [ "$KEEP_INFRA" = false ]; then
            destroy_infrastructure
        else
            log "Infrastructure kept as requested (use '$0 destroy' to clean up)"
        fi
        ;;
    provision)
        check_prerequisites
        provision_infrastructure
        ;;
    test)
        check_prerequisites
        run_ansible_tests
        ;;
    destroy)
        destroy_infrastructure
        ;;
    status)
        show_status
        ;;
    ssh)
        ssh_to_server
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        error "Unknown command: $COMMAND"
        echo ""
        usage
        exit 1
        ;;
esac