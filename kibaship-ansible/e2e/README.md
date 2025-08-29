# Kibaship E2E Testing with DigitalOcean

End-to-end testing infrastructure that provisions real Ubuntu 24.04 servers on DigitalOcean to test all Ansible roles in a production-like environment.

## Overview

This E2E testing system:
1. **Provisions** a DigitalOcean droplet (s-2vcpu-4gb, Ubuntu 24.04)
2. **Configures** SSH access with generated keys
3. **Runs** all Kibaship Ansible roles in sequence
4. **Verifies** successful execution
5. **Destroys** infrastructure (unless `--keep` is used)

## Quick Start

### 1. Prerequisites

```bash
# Install required tools
brew install terraform  # or your preferred method
# Ansible should already be installed in ../venv

# Ensure you're in the project root
cd /path/to/kibaship-ansible
```

### 2. Configure DigitalOcean API Token

```bash
# Copy the example configuration
cp e2e/terraform/terraform.tfvars.example e2e/terraform/terraform.tfvars

# Edit the file and add your DigitalOcean API token
# Get your token from: https://cloud.digitalocean.com/account/api/tokens
nano e2e/terraform/terraform.tfvars
```

### 3. Run Complete E2E Test

```bash
# Full test cycle (provision → test → destroy)
./e2e/run-e2e-test.sh up

# Keep infrastructure for debugging
./e2e/run-e2e-test.sh up --keep

# Only provision (skip tests)  
./e2e/run-e2e-test.sh up --skip-tests
```

## Detailed Usage

### Commands

| Command | Description |
|---------|-------------|
| `up` | Complete E2E cycle (provision → test → destroy) |
| `provision` | Create infrastructure only |
| `test` | Run Ansible tests on existing infrastructure |
| `destroy` | Clean up all resources |
| `status` | Show current infrastructure status |
| `ssh` | SSH into the test server |

### Options

| Option | Description |
|--------|-------------|
| `--skip-tests` | Skip Ansible playbook execution |
| `--keep` | Don't destroy infrastructure after tests |
| `--help` | Show help message |

### Examples

```bash
# Development workflow
./e2e/run-e2e-test.sh provision     # Create server
./e2e/run-e2e-test.sh ssh           # SSH in to debug
./e2e/run-e2e-test.sh test          # Run tests
./e2e/run-e2e-test.sh destroy       # Clean up

# Quick test specific roles
./e2e/run-e2e-test.sh up --keep     # Keep server running
# Edit run-e2e-test.sh to test specific roles
./e2e/run-e2e-test.sh test          # Re-run tests
./e2e/run-e2e-test.sh destroy       # Clean up when done
```

## Architecture

### Infrastructure Components

- **Droplet**: s-2vcpu-4gb Ubuntu 24.04 server
- **SSH Key**: Auto-generated ED25519 key pair
- **Security**: SSH key-based authentication only
- **Region**: NYC3 (configurable)
- **Tags**: Proper resource tagging for management

### Test Flow

1. **OS Preparation**
   - `os_prep_swap` - Disable swap
   - `os_prep_directories` - Create directories
   - `os_prep_kernel` - Configure kernel parameters  
   - `os_prep_packages` - Install system packages

2. **Container Runtime**
   - `container_prep_runc` - Install runc OCI runtime
   - `container_prep_cni` - Install CNI plugins
   - `container_prep_containerd` - Install containerd CRI
   - `container_prep_tools` - Install crictl and nerdctl

3. **Kubernetes Components**
   - `k8s_binaries_download` - Install Kubernetes binaries
   - `k8s_node_networking` - Configure node networking

4. **Storage**
   - `etcd_install` - Install and configure etcd

### Directory Structure

```
e2e/
├── .ssh/
│   ├── kibaship-e2e      # Private SSH key
│   └── kibaship-e2e.pub  # Public SSH key
├── terraform/
│   ├── versions.tf       # Terraform and provider versions
│   ├── variables.tf      # Variable definitions
│   ├── main.tf           # Main infrastructure resources
│   ├── outputs.tf        # Output values
│   ├── terraform.tfvars.example  # Configuration template
│   └── terraform.tfvars  # Your actual config (git-ignored)
├── run-e2e-test.sh       # Main test script
├── inventory.yml         # Generated Ansible inventory
└── README.md            # This file
```

## Configuration

### DigitalOcean Settings

Edit `e2e/terraform/terraform.tfvars`:

```hcl
# Required
do_token = "your_digitalocean_api_token_here"

# Optional overrides
droplet_region = "nyc3"      # or sgp1, lon1, fra1, etc.
droplet_size = "s-2vcpu-4gb" # or s-1vcpu-2gb, s-4vcpu-8gb, etc.
project_name = "kibaship-e2e"
```

### Cost Considerations

- **Droplet**: s-2vcpu-4gb costs ~$0.036/hour
- **Typical test duration**: 10-15 minutes  
- **Estimated cost per test**: ~$0.01
- **Auto-cleanup**: Infrastructure destroyed after tests

## Troubleshooting

### SSH Connection Issues

```bash
# Check if droplet is running
./e2e/run-e2e-test.sh status

# Manual SSH test
./e2e/run-e2e-test.sh ssh

# Check SSH key permissions
ls -la e2e/.ssh/
chmod 600 e2e/.ssh/kibaship-e2e
```

### Terraform Issues

```bash
# Re-initialize Terraform
cd e2e/terraform
terraform init -upgrade

# Check state
terraform show

# Force destroy if needed
terraform destroy -auto-approve
```

### Ansible Issues

```bash
# Test connectivity
source venv/bin/activate
ansible -i e2e/inventory.yml all -m ping

# Run with verbose output
ansible-playbook -i e2e/inventory.yml -vvv [playbook]
```

## Security Notes

- SSH keys are generated locally and not committed to git
- DigitalOcean API token should be kept secure
- Droplets are automatically destroyed after tests
- All resources are properly tagged for identification

## Integration with CI/CD

Example GitHub Actions integration:

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  e2e-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3
      - name: Run E2E Tests
        env:
          DO_TOKEN: ${{ secrets.DIGITALOCEAN_TOKEN }}
        run: |
          echo "do_token = \"$DO_TOKEN\"" > e2e/terraform/terraform.tfvars
          ./e2e/run-e2e-test.sh up
```

## Next Steps

After successful E2E testing, this infrastructure can be extended for:
- Multi-node cluster testing
- Kubernetes cluster initialization testing
- Application deployment testing
- Performance testing
- Upgrade testing