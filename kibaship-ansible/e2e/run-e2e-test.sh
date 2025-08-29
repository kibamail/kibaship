#!/bin/bash

# Kibaship E2E Testing Script with DigitalOcean
# Provisions infrastructure, runs Ansible playbooks, and cleans up

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
Kibaship E2E Testing Script

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    up          Provision infrastructure and run E2E tests
    provision   Only provision the infrastructure
    test        Run tests on existing infrastructure  
    destroy     Destroy infrastructure
    status      Show infrastructure status
    ssh         SSH into the test server
    
Options:
    --skip-tests    Skip running Ansible playbooks (provision only)
    --keep          Don't destroy infrastructure after tests
    --help          Show this help message

Examples:
    $0 up                    # Full E2E test cycle
    $0 provision             # Just create the server
    $0 test                  # Run tests on existing server
    $0 destroy               # Clean up everything
    $0 ssh                   # SSH into test server

EOF
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check for required tools
    local missing_tools=()
    
    command -v terraform >/dev/null 2>&1 || missing_tools+=("terraform")
    command -v ssh >/dev/null 2>&1 || missing_tools+=("ssh")
    
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
    log "Provisioning DigitalOcean infrastructure..."
    cd "$TERRAFORM_DIR"
    
    # Initialize Terraform
    log "Initializing Terraform..."
    terraform init
    
    # Plan the deployment
    log "Planning infrastructure..."
    terraform plan
    
    # Apply the configuration
    log "Creating infrastructure..."
    if terraform apply -auto-approve; then
        success "Infrastructure provisioned successfully"
        
        # Get droplet IP
        DROPLET_IP=$(terraform output -raw droplet_ip)
        log "Droplet IP: $DROPLET_IP"
        
        # Wait for SSH to be ready
        log "Waiting for SSH to be ready..."
        for i in {1..30}; do
            if ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no -i "$SSH_KEY_PATH" root@"$DROPLET_IP" echo "SSH Ready" >/dev/null 2>&1; then
                success "SSH connection established"
                break
            fi
            echo -n "."
            sleep 10
        done
        
        if [ $i -eq 30 ]; then
            error "SSH connection timeout"
            exit 1
        fi
        
    else
        error "Failed to provision infrastructure"
        exit 1
    fi
}

run_ansible_tests() {
    log "Running Ansible E2E tests..."
    cd "$PROJECT_ROOT"
    
    # Get droplet IP from terraform
    DROPLET_IP=$(cd "$TERRAFORM_DIR" && terraform output -raw droplet_ip)
    
    if [ -z "$DROPLET_IP" ]; then
        error "Could not get droplet IP from Terraform"
        exit 1
    fi
    
    # Generate dynamic inventory
    cat > "$SCRIPT_DIR/inventory.yml" << EOF
---
all:
  hosts:
    kibaship-e2e:
      ansible_host: $DROPLET_IP
      ansible_user: root
      ansible_ssh_private_key_file: $SSH_KEY_PATH
      ansible_ssh_common_args: '-o StrictHostKeyChecking=no'
  children:
    k8s_cluster:
      hosts:
        kibaship-e2e:
    kube_control_plane:
      hosts:
        kibaship-e2e:
    kube_node:
      hosts:
        kibaship-e2e:
    etcd:
      hosts:
        kibaship-e2e:
EOF
    
    log "Generated inventory for $DROPLET_IP"
    
    # Activate Python virtual environment
    source "$PROJECT_ROOT/venv/bin/activate"
    
    # Test connectivity
    log "Testing Ansible connectivity..."
    if ansible -i "$SCRIPT_DIR/inventory.yml" all -m ping; then
        success "Ansible connectivity confirmed"
    else
        error "Ansible connectivity failed"
        exit 1
    fi
    
    # Run all our roles in sequence with explicit roles path
    log "Running complete E2E test suite..."
    cd "$PROJECT_ROOT"
    ANSIBLE_ROLES_PATH="./roles" ansible-playbook -i "$SCRIPT_DIR/inventory.yml" -e "target_host=kibaship-e2e" "$SCRIPT_DIR/e2e-playbook.yml"
    
    if [ $? -eq 0 ]; then
        success "All Ansible roles executed successfully!"
    else
        error "Ansible playbook execution failed"
        exit 1
    fi
}

destroy_infrastructure() {
    log "Destroying infrastructure..."
    cd "$TERRAFORM_DIR"
    
    if terraform destroy -auto-approve; then
        success "Infrastructure destroyed successfully"
        
        # Clean up generated files
        rm -f "$SCRIPT_DIR/inventory.yml"
    else
        error "Failed to destroy infrastructure"
        exit 1
    fi
}

show_status() {
    log "Infrastructure Status:"
    cd "$TERRAFORM_DIR"
    
    if terraform show >/dev/null 2>&1; then
        terraform show
    else
        warning "No infrastructure found or Terraform not initialized"
    fi
}

ssh_to_server() {
    cd "$TERRAFORM_DIR"
    DROPLET_IP=$(terraform output -raw droplet_ip 2>/dev/null)
    
    if [ -z "$DROPLET_IP" ]; then
        error "No infrastructure found or droplet IP unavailable"
        exit 1
    fi
    
    log "Connecting to $DROPLET_IP..."
    ssh -i "$SSH_KEY_PATH" -o StrictHostKeyChecking=no root@"$DROPLET_IP"
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