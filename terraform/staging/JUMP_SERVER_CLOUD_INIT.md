# Jump Server Cloud-Init Configuration

## Overview

The jump server uses a minimal cloud-init configuration that only sets up the ubuntu user with SSH keys for accessing cluster nodes. No additional software is installed during the initial setup.

## Cloud-Init Configuration

The cloud-init configuration is stored in `templates/jump-server-cloud-init.yml.tpl` and includes:

### SSH Key Setup
- **Private Key**: `/home/ubuntu/.ssh/id_ed25519` (ED25519 format)
- **Public Key**: `/home/ubuntu/.ssh/id_ed25519.pub`
- **SSH Config**: Pre-configured for cluster nodes (10.0.1.*)

### SSH Configuration Features
- **StrictHostKeyChecking**: Disabled for cluster nodes
- **UserKnownHostsFile**: Set to /dev/null for cluster nodes
- **IdentityFile**: Points to the ED25519 private key

### Minimal Setup
- **SSH Service**: Enabled and started
- **No Package Installation**: No additional software installed
- **No Firewall Configuration**: Relies on Hetzner Cloud security groups

## SSH Key Format

The infrastructure uses **ED25519** SSH keys for better security and performance:
- **Private Key**: `.secrets/staging/id_ed25519`
- **Public Key**: `.secrets/staging/id_ed25519.pub`

## Usage

After Terraform deployment, connect to the jump server:

```bash
ssh -i .secrets/staging/id_ed25519 ubuntu@<JUMP_SERVER_PUBLIC_IP>
```

From the jump server, access any cluster node:

```bash
# Control plane nodes
ssh ubuntu@10.0.1.10
ssh ubuntu@10.0.1.11
ssh ubuntu@10.0.1.12

# Worker nodes
ssh ubuntu@10.0.1.20
ssh ubuntu@10.0.1.21
ssh ubuntu@10.0.1.22
```

## Post-Setup Software Installation

Since no software is pre-installed, you'll need to install required tools manually:

```bash
# Update package list
sudo apt update

# Install essential tools
sudo apt install -y curl wget git vim htop

# Install Python and Ansible for Kubespray
sudo apt install -y python3 python3-pip
pip3 install ansible-core

# Install Ansible collections for Kubernetes
ansible-galaxy collection install kubernetes.core
ansible-galaxy collection install ansible.posix
```

## Benefits of Minimal Setup

1. **Faster Boot Time**: No package installation during cloud-init
2. **Predictable State**: No dependency on package repositories during setup
3. **Flexibility**: Install only what you need when you need it
4. **Security**: Minimal attack surface with only essential services
5. **Reliability**: Less chance of cloud-init failures due to package issues

## Security Considerations

- SSH keys are properly secured with correct permissions (600 for private, 644 for public)
- SSH config is configured to trust cluster nodes automatically
- No additional services or packages that could introduce vulnerabilities
- Relies on Hetzner Cloud security groups for network-level protection
