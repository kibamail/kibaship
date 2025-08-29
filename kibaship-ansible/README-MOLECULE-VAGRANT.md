# Kibaship Molecule Testing with Vagrant

This project now uses **Vagrant with Bento Ubuntu 24.04** instead of Docker for Molecule testing. This provides a more realistic testing environment that closely mirrors production systems.

## 🎯 Why Vagrant Instead of Docker?

- **Real OS Environment**: Full Ubuntu 24.04 with real kernel, systemd, and networking
- **Container Runtime Testing**: Proper testing of containerd, runc, and other container components
- **No Environment Conditionals**: Eliminates the need for `when: not molecule_test` conditions
- **Production Parity**: Testing environment matches production Ubuntu 24.04 systems

## 🛠 Prerequisites

### Required Software
```bash
# Install VirtualBox (default provider)
brew install --cask virtualbox

# Install Vagrant
brew install vagrant

# Install Molecule with Vagrant driver
pip install molecule[vagrant]

# Install required Vagrant plugins (done automatically)
vagrant plugin install vagrant-vbguest
```

### Alternative Providers
- **VMware Desktop**: `brew install --cask vmware-fusion` (requires license)
- **Parallels**: `brew install --cask parallels` (requires license)

## 📁 File Structure

```
kibaship-ansible/
├── Vagrantfile                      # Main Vagrant configuration
├── molecule-vagrant-template.yml    # Template for role configurations
├── update-molecule-configs.sh       # Script to update all roles
└── roles/
    └── [role-name]/
        └── molecule/
            └── default/
                ├── molecule.yml      # Updated to use Vagrant
                ├── converge.yml      # Test playbook (unchanged)
                └── verify.yml        # Test assertions (unchanged)
```

## 🚀 Usage

### Test a Single Role
```bash
# Navigate to a role directory
cd roles/container_prep_containerd

# Run full Molecule test cycle
molecule test

# Individual test phases
molecule create     # Create and provision VM
molecule converge   # Run the role
molecule verify     # Run verification tests
molecule destroy    # Clean up VM
```

### Test All Roles
```bash
# Run tests for all roles (from project root)
find roles -name molecule.yml -exec dirname {} \; | \
  while read role_path; do
    echo "Testing $(basename $(dirname $(dirname $role_path)))..."
    cd "$role_path/.."
    molecule test
    cd - > /dev/null
  done
```

### Development Workflow
```bash
# Create VM for iterative development
molecule create
molecule converge

# Make changes to your role, then test
molecule converge   # Apply changes
molecule verify     # Test assertions

# When done
molecule destroy
```

## ⚙️ Configuration Details

### VM Specifications
- **OS**: Bento Ubuntu 24.04 LTS
- **Memory**: 4GB
- **CPUs**: 2 cores
- **Nested Virtualization**: Enabled (for container runtime testing)
- **Provider**: VirtualBox (default), VMware, or Parallels

### Ansible Configuration
```yaml
provisioner:
  name: ansible
  config_options:
    defaults:
      host_key_checking: false
      callbacks_enabled: profile_tasks
      stdout_callback: yaml
  inventory:
    host_vars:
      molecule-instance:
        ansible_user: vagrant
        ansible_become: yes
        ansible_become_method: sudo
```

## 🔧 Customization

### Change VM Resources
Edit `molecule.yml` in individual roles:
```yaml
platforms:
  - name: molecule-instance
    box: bento/ubuntu-24.04
    memory: 8192    # Increase memory
    cpus: 4         # Increase CPU cores
```

### Use Different Provider
```yaml
platforms:
  - name: molecule-instance
    box: bento/ubuntu-24.04
    provider:
      name: vmware_desktop  # or parallels
      type: vmware_desktop
```

### Test Multiple OS Versions
```yaml
platforms:
  - name: ubuntu-24-04
    box: bento/ubuntu-24.04
  - name: ubuntu-22-04
    box: bento/ubuntu-22.04
```

## 🐛 Troubleshooting

### Common Issues

**Molecule/Vagrant not found:**
```bash
# Ensure you're in the Python virtual environment
source venv/bin/activate
pip install molecule[vagrant]
```

**VirtualBox issues:**
```bash
# Check VirtualBox is running
VBoxManage --version

# Reset VirtualBox if needed
sudo /Library/Application\ Support/VirtualBox/LaunchDaemons/VirtualBoxStartup.sh restart
```

**Vagrant plugin issues:**
```bash
# Reinstall vagrant plugins
vagrant plugin uninstall vagrant-vbguest
vagrant plugin install vagrant-vbguest
```

**VM creation fails:**
```bash
# Check available Vagrant boxes
vagrant box list

# Update Vagrant box
vagrant box update --box bento/ubuntu-24.04
```

### Performance Tips

1. **Increase VM resources** for roles that need more power:
   ```yaml
   memory: 8192
   cpus: 4
   ```

2. **Use SSD storage** for better VM performance

3. **Enable hardware acceleration** in your VM provider settings

4. **Close unnecessary applications** while running tests

## 🔄 Migration from Docker

The migration script has:
- ✅ Updated all `molecule.yml` files to use Vagrant
- ✅ Created backup files (`*.docker-backup`)
- ✅ Configured proper VM specifications
- ✅ Set up Ansible provisioner settings

### Rollback (if needed)
```bash
# Restore Docker configurations
find roles -name "*.docker-backup" -exec bash -c 'mv "$1" "${1%.docker-backup}"' _ {} \;

# Remove Vagrant configurations  
rm -f Vagrantfile molecule-vagrant-template.yml
```

## 📊 Performance Comparison

| Aspect | Docker | Vagrant |
|--------|--------|---------|
| **Startup** | ~10s | ~60s |
| **Reality** | Container | Real VM |
| **systemd** | Limited | Full |
| **Networking** | Emulated | Real |
| **Storage** | Overlay | Real filesystem |
| **Container Runtime** | Nested issues | Native support |

## 🎯 Best Practices

1. **Always destroy VMs** after testing to free resources
2. **Use `molecule test`** for full cycle testing
3. **Use `molecule converge`** for iterative development
4. **Monitor system resources** when running multiple tests
5. **Keep VM configs consistent** across roles

## 🔗 Useful Commands

```bash
# Check Vagrant status
vagrant status

# SSH into test VM (when created manually)
vagrant ssh

# View Molecule scenarios
molecule list

# Check VM resource usage
VBoxManage list runningvms

# Clean up all Vagrant VMs
vagrant global-status --prune
```

## 📝 Next Steps

1. Test the new Vagrant-based workflow with a simple role
2. Update any role-specific test configurations as needed
3. Remove Docker backup files once confirmed working: `find roles -name '*.docker-backup' -delete`
4. Consider updating CI/CD pipelines to use Vagrant (if applicable)