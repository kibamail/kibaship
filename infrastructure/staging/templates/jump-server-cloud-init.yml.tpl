#cloud-config

# =============================================================================
# Jump Server Cloud-Init Configuration
# =============================================================================
# This cloud-init configuration sets up the ubuntu user with SSH keys
# for accessing all cluster nodes via their private IPs.

# =============================================================================
# User Configuration
# =============================================================================
users:
  - name: ubuntu
    groups: [adm, sudo]
    shell: /bin/bash
    sudo: ["ALL=(ALL) NOPASSWD:ALL"]
    ssh_authorized_keys:
      - ${ssh_public_key}

# =============================================================================
# File Configuration
# =============================================================================
write_files:
  - path: /home/ubuntu/.ssh/id_ed25519
    content: ${base64encode(ssh_private_key)}
    permissions: "0600"
    encoding: b64

  - path: /home/ubuntu/.ssh/id_ed25519.pub
    content: ${ssh_public_key}
    permissions: "0644"

  - path: /home/ubuntu/.ssh/config
    content: |
      Host 10.0.1.*
          StrictHostKeyChecking no
          UserKnownHostsFile=/dev/null
          IdentityFile ~/.ssh/id_ed25519
    permissions: "0600"

runcmd:
  - chown -R ubuntu:ubuntu /home/ubuntu
  - chmod 755 /home/ubuntu
  - chmod 700 /home/ubuntu/.ssh
  - chown -R ubuntu:ubuntu /home/ubuntu/.ssh
  - systemctl enable ssh
  - systemctl start ssh

  # Install Python3 and pip3
  - apt-get update
  - apt-get install -y python3 python3-pip git

  # Clone Kubespray and checkout specific version
  - cd /home/ubuntu
  - git clone https://github.com/kubernetes-sigs/kubespray.git
  - cd kubespray
  - git checkout v2.28.0
  - chown -R ubuntu:ubuntu /home/ubuntu/kubespray

  # Install Kubespray requirements
  - cd /home/ubuntu/kubespray 
  - echo 'Jump server setup completed successfully' > /var/log/jump-server-setup.log

final_message: "Jump server setup completed successfully"
