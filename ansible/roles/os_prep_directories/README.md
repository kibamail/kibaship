# os_prep_directories

Creates essential Kubernetes system directories with proper permissions for cluster preparation.

## Description

This role creates the core directory structure required by Kubernetes components, following Kubespray patterns for proper filesystem preparation before cluster deployment.

## Requirements

- Ubuntu 24.04 LTS
- Root privileges (become: true)
- Ansible 2.18+

## Directories Created

- `/opt/cni/bin` - CNI plugin binaries
- `/etc/kubernetes` - Kubernetes configuration files
- `/etc/kubernetes/manifests` - Static pod manifests
- `/var/lib/kubelet` - Kubelet data directory

All directories are created with:
- Mode: 0755
- Owner: root:root

## Dependencies

None

## Example Playbook

```yaml
- hosts: kubernetes_nodes
  become: true
  roles:
    - kibaship.os_prep_directories
```

## Testing

```bash
molecule test
```

## License

MIT

## Author Information

Created by Claude-SRE for Kibaship Kubernetes cluster provisioning.