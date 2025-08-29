# Read SSH public key
locals {
  ssh_public_key = file(var.ssh_key_path)
  timestamp      = formatdate("YYYY-MM-DD-hhmm", timestamp())
}

# Create SSH key on DigitalOcean with unique identifier
resource "digitalocean_ssh_key" "kibaship_e2e" {
  name       = "${var.project_name}-key-${local.timestamp}-${substr(sha256(local.ssh_public_key), 0, 8)}"
  public_key = local.ssh_public_key
}

# Create droplet for E2E testing
resource "digitalocean_droplet" "kibaship_e2e" {
  image     = var.droplet_image
  name      = "${var.project_name}-droplet-${local.timestamp}"
  region    = var.droplet_region
  size      = var.droplet_size
  
  ssh_keys = [digitalocean_ssh_key.kibaship_e2e.id]
  
  # User data to prepare the system for Ansible
  user_data = <<-EOF
    #cloud-config
    package_update: true
    package_upgrade: true
    
    packages:
      - python3
      - python3-pip
      - curl
      - wget
      - git
      - unzip
    
    # Configure SSH for root access (needed for some Ansible roles)
    ssh_pwauth: false
    disable_root: false
    
    runcmd:
      - systemctl enable ssh
      - systemctl start ssh
      - echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config
      - systemctl restart ssh
      - mkdir -p /root/.ssh
      - chmod 700 /root/.ssh
  EOF

  tags = [
    "${var.project_name}",
    "e2e-testing",
    "ansible-target"
  ]
}