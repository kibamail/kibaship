# os_prep_kernel

Configure essential kernel parameters for Kubernetes cluster preparation.

## Description

This role configures critical kernel networking parameters required by Kubernetes, following Kubespray patterns. It creates the necessary sysctl configuration files that will be applied at system boot.

## Requirements

- Ubuntu 24.04 LTS
- Root privileges (become: true)
- Ansible 2.18+

## Kernel Parameters Configured

- `net.ipv4.ip_forward = 1` - Enable IPv4 packet forwarding for pod networking
- `net.bridge.bridge-nf-call-iptables = 1` - Enable bridge netfilter for iptables
- `net.bridge.bridge-nf-call-ip6tables = 1` - Enable bridge netfilter for ip6tables

All parameters are written to `/etc/sysctl.d/99-kubernetes.conf` for persistence.

## Dependencies

None

## Example Playbook

```yaml
- hosts: kubernetes_nodes
  become: true
  roles:
    - kibaship.os_prep_kernel
```

## Testing

This role includes comprehensive Molecule tests:

```bash
molecule test
```

**Note**: Container testing has limitations with sysctl operations. The role creates the configuration files correctly, but kernel parameter application may be restricted in container environments.

## Production Usage

On real systems (non-containerized), the kernel parameters will be:
1. Written to `/etc/sysctl.d/99-kubernetes.conf`
2. Applied immediately via `sysctl --system`
3. Persistent across reboots

## License

MIT

## Author Information

Created by Claude-SRE for Kibaship Kubernetes cluster provisioning.