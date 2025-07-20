#cloud-config

# =============================================================================
# Jump Server Cloud-Init Configuration
# =============================================================================
# This cloud-init configuration sets up the ubuntu user with SSH keys
# for accessing all cluster nodes via their private IPs.

write_files:
  # SSH private key for accessing cluster nodes
  - path: /home/ubuntu/.ssh/id_ed25519
    content: |
      ${indent(6, ssh_private_key)}
    permissions: '0600'
    owner: ubuntu:ubuntu

  # SSH public key
  - path: /home/ubuntu/.ssh/id_ed25519.pub
    content: ${ssh_public_key}
    permissions: '0644'
    owner: ubuntu:ubuntu

  # SSH config for cluster nodes
  - path: /home/ubuntu/.ssh/config
    content: |
      Host 10.0.1.*
          StrictHostKeyChecking no
          UserKnownHostsFile=/dev/null
          IdentityFile ~/.ssh/id_ed25519
    permissions: '0600'
    owner: ubuntu:ubuntu

# Ensure SSH service is enabled and started
runcmd:
  - systemctl enable ssh
  - systemctl start ssh
  - echo 'Jump server setup completed successfully' > /var/log/jump-server-setup.log

final_message: "Jump server setup completed successfully"
