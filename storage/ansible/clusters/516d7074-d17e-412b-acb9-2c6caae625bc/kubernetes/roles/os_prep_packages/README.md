# os_prep_packages

Install essential system packages for Kubernetes cluster preparation on Ubuntu 24.04.

## Description

This role installs all essential system packages required for Kubernetes cluster provisioning, following Kubespray's bootstrap patterns. It's specifically optimized for Ubuntu 24.04 LTS without any OS detection overhead.

## Requirements

- Ubuntu 24.04 LTS
- Root privileges (become: true)
- Ansible 2.18+
- Internet connectivity for package downloads

## Essential Packages Installed

### Core System Utilities
- `apt-transport-https` - Secure HTTPS repository access
- `ca-certificates` - SSL/TLS certificate authorities
- `software-properties-common` - Repository management tools

### HTTP Clients & Cryptography
- `curl` - Primary HTTP client for downloads
- `wget` - Alternative HTTP client
- `openssl` - Cryptographic operations and certificates

### Package Verification & GPG
- `gnupg` - GPG for package signature verification
- `dirmngr` - GPG network certificate management daemon

### Python Ecosystem (Ansible Requirements)
- `python3` - Python interpreter for Ansible operations
- `python3-apt` - Python APT library for ansible apt module

### File Operations
- `rsync` - File synchronization utility
- `unzip` - Archive extraction utility

## Features

- **Ubuntu 24.04 Optimized**: No OS detection - pure Ubuntu 24.04 focus
- **Retry Logic**: Robust package installation with retry mechanisms
- **Cache Management**: Efficient apt cache handling with validity checks
- **GPG Keyring Setup**: Prepares `/etc/apt/keyrings` for repository keys
- **Idempotent**: Safe to run multiple times

## Dependencies

None

## Example Playbook

```yaml
- hosts: kubernetes_nodes
  become: true
  roles:
    - kibaship.os_prep_packages
```

## Testing

This role includes comprehensive Molecule tests:

```bash
molecule test
```

**Note**: Container testing has limitations with package management operations. The role is designed for real Ubuntu 24.04 systems where full package management is available.

## Production Usage

On real Ubuntu 24.04 systems, this role will:
1. Update the apt package cache
2. Install all essential packages with retry logic
3. Set up GPG keyring directories for future repository additions
4. Prepare the system foundation for Kubernetes components

This role provides the essential foundation that all subsequent Kubernetes preparation roles depend on.

## License

MIT

## Author Information

Created by Claude-SRE for Kibaship Kubernetes cluster provisioning.