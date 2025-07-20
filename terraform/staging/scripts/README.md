# Server Setup Scripts

This directory contains YAML-based configuration files for server setup scripts used by Terraform remote-exec provisioners.

## 📁 File Structure

```
scripts/
├── common-setup.yaml           # Shared setup steps for all servers
├── worker-specific.yaml        # Worker-only configuration (Mayastor requirements)
├── post-reboot-verification.yaml  # Post-reboot verification steps
└── README.md                   # This documentation
```

## 🔧 Script Components

### **common-setup.yaml**
Contains setup steps shared between control plane and worker nodes:
- **User Management**: Create ubuntu user with sudo access
- **SSH Key Setup**: Configure SSH key authentication
- **System Update**: Update packages and system
- **Package Installation**: Install required packages including chrony
- **Networking**: Configure kernel modules and sysctl settings
- **Time Sync**: Configure chrony for NTP synchronization
- **Helm Installation**: Install Helm v3.x
- **SSH Security**: Disable password auth, enable key-only access

### **worker-specific.yaml**
Contains worker-only configuration for Replicated PV requirements:
- **Storage Modules**: Load nvme-tcp and ext4 kernel modules
- **Hugepages**: Configure 2GiB hugepages for Mayastor
- **GRUB Configuration**: Persistent hugepages and nvme multipath settings
- **Verification**: Check hugepages and storage modules

### **post-reboot-verification.yaml**
Contains verification steps after server reboot:
- **Connection Test**: Verify SSH connectivity as ubuntu user
- **Hugepages Check**: Verify hugepages on worker nodes
- **Modules Check**: Verify kernel modules loaded
- **Time Sync Check**: Verify chrony synchronization
- **Final Verification**: Confirm server readiness

## 🚀 Usage in Terraform

The YAML files are loaded and processed in `modules/servers/main.tf`:

```hcl
locals {
  # Load YAML configurations
  common_setup_yaml = yamldecode(file("${path.module}/../../scripts/common-setup.yaml"))
  worker_specific_yaml = yamldecode(file("${path.module}/../../scripts/worker-specific.yaml"))
  post_reboot_yaml = yamldecode(file("${path.module}/../../scripts/post-reboot-verification.yaml"))

  # Build script arrays and filter out empty strings
  common_setup_steps = [for cmd in flatten([...]) : cmd if cmd != ""]
  worker_specific_steps = [for cmd in flatten([...]) : cmd if cmd != ""]
  post_reboot_steps = [for cmd in flatten([...]) : cmd if cmd != ""]

  # Complete script arrays
  control_plane_script = concat(local.common_setup_steps, ["reboot"])
  worker_script = concat(local.common_setup_steps, local.worker_specific_steps, ["reboot"])
}
```

## ✅ Benefits

### **Clean Separation**
- ✅ **Common logic reused** between control plane and workers
- ✅ **Worker-specific requirements** isolated and clearly defined
- ✅ **Easy to maintain** - scripts organized by purpose

### **Readable Configuration**
- ✅ **YAML format** - human-readable and well-structured
- ✅ **Logical grouping** - related commands grouped together
- ✅ **Clear documentation** - each section has a clear purpose

### **Maintainable**
- ✅ **Single source of truth** for each script component
- ✅ **Easy to modify** - change YAML files without touching Terraform
- ✅ **Version controlled** - scripts tracked alongside infrastructure

## 🔄 Execution Flow

1. **Server Provisioning** → Hetzner Cloud servers created
2. **Setup Scripts** → Remote-exec runs YAML-based scripts as root
3. **Reboot** → Servers reboot to apply GRUB changes
4. **60s Delay** → Terraform waits for servers to come back online
5. **Post-Reboot Verification** → Verify configuration as ubuntu user
6. **Ready** → Infrastructure marked as ready for use

## 📝 Modifying Scripts

To modify server setup:

1. **Edit YAML files** in this directory
2. **Run terraform plan** to see changes
3. **Run terraform apply** to apply changes

The modular structure makes it easy to:
- Add new packages to `common-setup.yaml`
- Modify worker requirements in `worker-specific.yaml`
- Add verification steps to `post-reboot-verification.yaml`
