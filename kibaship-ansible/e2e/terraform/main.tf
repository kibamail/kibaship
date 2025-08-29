# Read SSH public key
locals {
  ssh_public_key = file(var.ssh_key_path)
  timestamp      = formatdate("YYYY-MM-DD-hhmm", timestamp())
}

# Create VPC for private networking
resource "digitalocean_vpc" "kibaship_cluster" {
  name     = "${var.project_name}-vpc-${local.timestamp}"
  region   = var.droplet_region
  ip_range = "10.108.0.0/20"
}

# Create SSH key on DigitalOcean with unique identifier
resource "digitalocean_ssh_key" "kibaship_e2e" {
  name       = "${var.project_name}-key-${local.timestamp}-${substr(sha256(local.ssh_public_key), 0, 8)}"
  public_key = local.ssh_public_key
}

# Control Plane nodes (3)
resource "digitalocean_droplet" "control_plane" {
  count  = 3
  image  = var.droplet_image
  name   = "${var.project_name}-cp-${count.index + 1}-${local.timestamp}"
  region = var.droplet_region
  size   = var.control_plane_size

  ssh_keys = [digitalocean_ssh_key.kibaship_e2e.id]
  vpc_uuid = digitalocean_vpc.kibaship_cluster.id

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
    "ansible-target",
    "control-plane"
  ]
}

# Worker nodes (3)
resource "digitalocean_droplet" "worker" {
  count  = 3
  image  = var.droplet_image
  name   = "${var.project_name}-worker-${count.index + 1}-${local.timestamp}"
  region = var.droplet_region
  size   = var.worker_size

  ssh_keys = [digitalocean_ssh_key.kibaship_e2e.id]
  vpc_uuid = digitalocean_vpc.kibaship_cluster.id

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
    "ansible-target",
    "worker-node"
  ]
}

# Kubernetes API Load Balancer
resource "digitalocean_loadbalancer" "kube_api" {
  name   = "${var.project_name}-kube-api-lb-${local.timestamp}"
  region = var.droplet_region
  size   = "lb-small"

  vpc_uuid = digitalocean_vpc.kibaship_cluster.id

  # Target control plane nodes on port 6443 (kube-apiserver)
  forwarding_rule {
    entry_protocol  = "tcp"
    entry_port      = 6443
    target_protocol = "tcp"
    target_port     = 6443
  }

  # Health check for API server
  healthcheck {
    protocol                 = "tcp"
    port                     = 6443
    check_interval_seconds   = 10
    response_timeout_seconds = 5
    unhealthy_threshold      = 3
    healthy_threshold        = 3
  }

  # Target all control plane nodes using their private IPs
  droplet_ids = digitalocean_droplet.control_plane[*].id
}

# Ingress Load Balancer
resource "digitalocean_loadbalancer" "ingress" {
  name   = "${var.project_name}-ingress-lb-${local.timestamp}"
  region = var.droplet_region
  size   = "lb-small"

  vpc_uuid = digitalocean_vpc.kibaship_cluster.id

  # HTTP traffic
  forwarding_rule {
    entry_protocol  = "http"
    entry_port      = 80
    target_protocol = "http"
    target_port     = 30080
  }

  # HTTPS traffic (transparent SSL passthrough)
  forwarding_rule {
    entry_protocol  = "https"
    entry_port      = 443
    target_protocol = "https"
    target_port     = 30443
    tls_passthrough = true
  }

  # Health check for ingress
  healthcheck {
    protocol                 = "http"
    port                     = 30080
    path                     = "/healthz"
    check_interval_seconds   = 10
    response_timeout_seconds = 5
    unhealthy_threshold      = 3
    healthy_threshold        = 3
  }

  # Enable PROXY protocol for real client IP preservation
  enable_proxy_protocol = true
  
  # Target all nodes (control plane + workers) for ingress traffic
  droplet_ids = concat(
    digitalocean_droplet.control_plane[*].id,
    digitalocean_droplet.worker[*].id
  )
}
