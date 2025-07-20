# =============================================================================
# KibaShip Staging Infrastructure Configuration
# =============================================================================
# This configuration provisions a complete Kubernetes cluster infrastructure
# for the KibaShip staging environment on Hetzner Cloud, including:
# - Private networking infrastructure
# - Load balancers for Kubernetes API and application traffic
# - Ubuntu 24.04 Kubernetes cluster with kubeadm and Cilium CNI
# - Persistent storage volumes for worker nodes
# - OpenEBS Mayastor preparation
# - SSH key management for secure server access

terraform {
  # Remote state backend - temporarily disabled due to S3 credential issues
  # Uncomment and configure once Storj S3 credentials are working

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.51.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2.0"
    }
    time = {
      source  = "hashicorp/time"
      version = "~> 0.12.0"
    }

  }
}

# =============================================================================
# Variables
# =============================================================================

variable "hcloud_token" {
  description = "Hetzner Cloud API Token"
  type        = string
  sensitive   = true
}

variable "cluster_name" {
  description = "Name of the Kubernetes cluster"
  type        = string
  default     = "kibaship-staging"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "staging"
}

variable "location" {
  description = "Hetzner Cloud location"
  type        = string
  default     = "nbg1"
}

variable "server_type" {
  description = "Hetzner Cloud server type"
  type        = string
  default     = "cpx21"
}

variable "api_load_balancer_private_ip" {
  description = "Private IP address for the API load balancer"
  type        = string
  default     = "10.0.1.100"
}

variable "ubuntu_version" {
  description = "Ubuntu version to use"
  type        = string
  default     = "24.04"
}

variable "kube_pods_subnet" {
  description = "Kubernetes pod subnet CIDR"
  type        = string
  default     = "10.0.16.0/20"
}

variable "kube_service_addresses" {
  description = "Kubernetes service subnet CIDR"
  type        = string
  default     = "10.0.8.0/21"
}



variable "volume_size" {
  description = "Size of each storage volume in GB"
  type        = number
  default     = 40
}

variable "ssh_public_key_path" {
  description = "Path to store the generated SSH public key"
  type        = string
  default     = ".secrets/staging/id_ed25519.pub"
}

variable "ssh_private_key_path" {
  description = "Path to store the generated SSH private key"
  type        = string
  default     = ".secrets/staging/id_ed25519"
}

# =============================================================================
# Provider Configuration
# =============================================================================

provider "hcloud" {
  token = var.hcloud_token
}

provider "tls" {}

provider "local" {}

provider "null" {}

# kubectl provider configuration is handled in individual modules
# that need it, since the kubeconfig file doesn't exist at plan time

# =============================================================================
# SSH Key Management Module
# =============================================================================

module "ssh_keys" {
  source = "./modules/ssh-keys"

  cluster_name         = var.cluster_name
  environment          = var.environment
  ssh_public_key_path  = var.ssh_public_key_path
  ssh_private_key_path = var.ssh_private_key_path
}

# =============================================================================
# Networking Module
# =============================================================================

module "networking" {
  source = "./modules/networking"

  cluster_name      = var.cluster_name
  environment       = var.environment
  network_ip_range  = "10.0.0.0/16"
  subnet_ip_range   = "10.0.1.0/24"
  network_zone      = "eu-central"
}

# =============================================================================
# Jump Server Module
# =============================================================================

module "jump_server" {
  source = "./modules/jump-server"

  cluster_name     = var.cluster_name
  environment      = var.environment
  network_id       = module.networking.network_id
  server_type      = var.server_type
  location         = var.location
  ubuntu_version   = var.ubuntu_version
  ssh_key_id       = module.ssh_keys.ssh_key_id
  ssh_private_key  = module.ssh_keys.ssh_private_key
  ssh_public_key   = module.ssh_keys.ssh_public_key

  depends_on = [module.networking, module.ssh_keys]
}

# =============================================================================
# Load Balancers Module
# =============================================================================

module "load_balancers" {
  source = "./modules/load-balancers"

  cluster_name         = var.cluster_name
  environment          = var.environment
  network_id           = module.networking.network_id
  location             = var.location
  load_balancer_type   = "lb11"
  k8s_api_private_ip   = "10.0.1.100"
  app_private_ip       = "10.0.1.101"

  depends_on = [module.networking]
}

# =============================================================================
# Servers Module
# =============================================================================

module "servers" {
  source = "./modules/servers"

  cluster_name         = var.cluster_name
  environment          = var.environment
  network_id           = module.networking.network_id
  cluster_endpoint     = module.load_balancers.k8s_api_endpoint
  k8s_api_public_ip    = module.load_balancers.k8s_api_public_ip
  k8s_api_private_ip   = var.api_load_balancer_private_ip
  ubuntu_version       = var.ubuntu_version
  server_type          = var.server_type
  location             = var.location
  control_plane_count  = 3
  worker_count         = 3
  ssh_key_id           = module.ssh_keys.ssh_key_id
  ssh_private_key      = module.ssh_keys.ssh_private_key
  ssh_public_key       = module.ssh_keys.ssh_public_key
  jump_server_public_ip = module.jump_server.jump_server_public_ip

  depends_on = [module.load_balancers, module.ssh_keys, module.jump_server]
}

# =============================================================================
# Storage Module
# =============================================================================

module "storage" {
  source = "./modules/storage"

  cluster_name          = var.cluster_name
  environment           = var.environment
  worker_servers        = module.servers.worker_servers
  volume_size           = var.volume_size
  volume_type           = "network-ssd"
  location              = var.location
  ssh_private_key       = module.ssh_keys.ssh_private_key
  jump_server_public_ip = module.jump_server.jump_server_public_ip

  depends_on = [module.servers, module.jump_server]
}

# =============================================================================
# Infrastructure Complete
# =============================================================================
# This configuration provisions the complete infrastructure including:
# - Servers, networking, load balancers, and storage
# - Servers are prepared with basic system configuration
# - Ready for application deployment

# =============================================================================
# Load Balancer Targets (After Servers)
# =============================================================================

resource "hcloud_load_balancer_target" "k8s_api_targets" {
  type             = "label_selector"
  load_balancer_id = module.load_balancers.k8s_api_load_balancer.id
  label_selector   = "role=control-plane"
  use_private_ip   = true

  depends_on = [module.servers]
}

resource "hcloud_load_balancer_target" "app_targets" {
  type             = "label_selector"
  load_balancer_id = module.load_balancers.app_load_balancer.id
  label_selector   = "role=worker"
  use_private_ip   = true

  depends_on = [module.servers]
}

# =============================================================================
# Kubespray Inventory Generation
# =============================================================================

locals {
  # Generate connection strings for control plane nodes
  connection_strings_master = join("\n", [
    for name, server in module.servers.control_plane_servers :
    "${replace(name, var.cluster_name, "default")} ansible_user=ubuntu ansible_host=${server.private_ip} ip=${server.private_ip} etcd_member_name=etcd${index(keys(module.servers.control_plane_servers), name) + 1}"
  ])

  # Generate connection strings for worker nodes
  connection_strings_worker = join("\n", [
    for name, server in module.servers.worker_servers :
    "${replace(name, var.cluster_name, "default")} ansible_user=ubuntu ansible_host=${server.private_ip} ip=${server.private_ip}"
  ])

  # Generate list of control plane node names
  list_master = join("\n", [
    for name, server in module.servers.control_plane_servers :
    replace(name, var.cluster_name, "default")
  ])

  # Generate list of worker node names
  list_worker = join("\n", [
    for name, server in module.servers.worker_servers :
    replace(name, var.cluster_name, "default")
  ])
}

# Ensure secrets and group_vars directories exist
resource "null_resource" "create_secrets_dir" {
  provisioner "local-exec" {
    command = "mkdir -p ${path.module}/.secrets/staging/group_vars/all ${path.module}/.secrets/staging/group_vars/k8s_cluster"
  }
}

# Generate inventory file from template
resource "local_file" "kubespray_inventory" {
  content = templatefile("${path.module}/templates/inventory.ini.tpl", {
    connection_strings_master = local.connection_strings_master
    connection_strings_worker = local.connection_strings_worker
    list_master              = local.list_master
    list_worker              = local.list_worker
  })
  filename = "${path.module}/.secrets/staging/inventory.ini"

  depends_on = [module.servers, null_resource.create_secrets_dir]
}

# Generate Kubespray group_vars configuration files
resource "local_file" "kubespray_all_config" {
  content = templatefile("${path.module}/templates/group_vars/all/all.yml.tpl", {
    k8s_api_public_ip       = module.load_balancers.k8s_api_public_ip
    cluster_name            = var.cluster_name
    kube_pods_subnet        = var.kube_pods_subnet
    kube_service_addresses  = var.kube_service_addresses
  })
  filename = "${path.module}/.secrets/staging/group_vars/all/all.yml"

  depends_on = [module.load_balancers, null_resource.create_secrets_dir]
}

resource "local_file" "kubespray_cloud_config" {
  content  = file("${path.module}/templates/group_vars/all/cloud.yml.tpl")
  filename = "${path.module}/.secrets/staging/group_vars/all/cloud.yml"

  depends_on = [null_resource.create_secrets_dir]
}

resource "local_file" "kubespray_addons_config" {
  content  = file("${path.module}/templates/group_vars/k8s_cluster/addons.yml.tpl")
  filename = "${path.module}/.secrets/staging/group_vars/k8s_cluster/addons.yml"

  depends_on = [null_resource.create_secrets_dir]
}

resource "local_file" "kubespray_cilium_config" {
  content  = file("${path.module}/templates/group_vars/k8s_cluster/k8s-net-cilium.yml.tpl")
  filename = "${path.module}/.secrets/staging/group_vars/k8s_cluster/k8s-net-cilium.yml"

  depends_on = [null_resource.create_secrets_dir]
}

# Copy Kubespray configuration files to jump server
resource "null_resource" "copy_kubespray_config_to_jump_server" {
  connection {
    type        = "ssh"
    user        = "ubuntu"
    private_key = module.ssh_keys.ssh_private_key
    host        = module.jump_server.jump_server_public_ip
    timeout     = "5m"
  }

  # Wait for Kubespray to be cloned and set up
  provisioner "remote-exec" {
    inline = [
      "while [ ! -d /home/ubuntu/kubespray ]; do echo 'Waiting for Kubespray clone...'; sleep 5; done",
      "echo 'Kubespray directory found, proceeding with configuration copy'"
    ]
  }

  # Create Kubespray inventory directory structure
  provisioner "remote-exec" {
    inline = [
      "mkdir -p /home/ubuntu/kubespray/inventory/kibaship-staging/group_vars/all",
      "mkdir -p /home/ubuntu/kubespray/inventory/kibaship-staging/group_vars/k8s_cluster"
    ]
  }

  # Copy inventory file to Kubespray directory
  provisioner "file" {
    source      = local_file.kubespray_inventory.filename
    destination = "/home/ubuntu/kubespray/inventory/kibaship-staging/inventory.ini"
  }

  # Copy group_vars configuration files to Kubespray directory
  provisioner "file" {
    source      = local_file.kubespray_all_config.filename
    destination = "/home/ubuntu/kubespray/inventory/kibaship-staging/group_vars/all/all.yml"
  }

  provisioner "file" {
    source      = local_file.kubespray_cloud_config.filename
    destination = "/home/ubuntu/kubespray/inventory/kibaship-staging/group_vars/all/cloud.yml"
  }

  provisioner "file" {
    source      = local_file.kubespray_addons_config.filename
    destination = "/home/ubuntu/kubespray/inventory/kibaship-staging/group_vars/k8s_cluster/addons.yml"
  }

  provisioner "file" {
    source      = local_file.kubespray_cilium_config.filename
    destination = "/home/ubuntu/kubespray/inventory/kibaship-staging/group_vars/k8s_cluster/k8s-net-cilium.yml"
  }

  # Set proper permissions and log completion
  provisioner "remote-exec" {
    inline = [
      # Set file permissions for Kubespray inventory
      "find /home/ubuntu/kubespray/inventory/kibaship-staging -type f -exec chmod 644 {} \\;",
      "find /home/ubuntu/kubespray/inventory/kibaship-staging -type d -exec chmod 755 {} \\;",

      # Log completion to user's home directory
      "echo 'Kubespray configuration files copied successfully' >> /home/ubuntu/setup.log",
      "echo 'Files available at:' >> /home/ubuntu/setup.log",
      "echo '  - /home/ubuntu/kubespray/inventory/kibaship-staging/inventory.ini' >> /home/ubuntu/setup.log",
      "echo '  - /home/ubuntu/kubespray/inventory/kibaship-staging/group_vars/' >> /home/ubuntu/setup.log",
      "echo 'Kubespray cloned to: /home/ubuntu/kubespray' >> /home/ubuntu/setup.log",
      "echo 'Setup completed at: $(date)' >> /home/ubuntu/setup.log"
    ]
  }

  depends_on = [
    local_file.kubespray_inventory,
    local_file.kubespray_all_config,
    local_file.kubespray_cloud_config,
    local_file.kubespray_addons_config,
    local_file.kubespray_cilium_config,
    module.jump_server,
    module.servers
  ]
}



# =============================================================================
# Outputs
# =============================================================================

output "cluster_info" {
  description = "Complete infrastructure information"
  value = {
    name           = var.cluster_name
    environment    = var.environment
    endpoint       = module.load_balancers.k8s_api_endpoint
    ubuntu_version = var.ubuntu_version
  }
}

output "network" {
  description = "Network infrastructure details"
  value       = module.networking.network
}

output "load_balancers" {
  description = "Load balancer details"
  value = {
    k8s_api = module.load_balancers.k8s_api_load_balancer
    app     = module.load_balancers.app_load_balancer
  }
}

output "jump_server" {
  description = "Jump server details"
  value       = module.jump_server.jump_server
}

output "servers" {
  description = "Server details"
  value = {
    jump_server    = module.jump_server.jump_server
    control_planes = module.servers.control_plane_servers
    workers        = module.servers.worker_servers
  }
}

output "storage" {
  description = "Storage configuration details"
  value = module.storage.storage_summary
}

output "servers_ready" {
  description = "Server deployment status"
  value       = module.servers.servers_ready
}

output "ssh_private_key" {
  description = "SSH private key for server access"
  value       = module.ssh_keys.ssh_private_key
  sensitive   = true
}

output "ssh_public_key" {
  description = "SSH public key for server access"
  value       = module.ssh_keys.ssh_public_key
}

output "kubespray_config" {
  description = "Kubespray configuration information"
  value = {
    inventory_local_path     = local_file.kubespray_inventory.filename
    inventory_jump_server_path = "/home/ubuntu/inventory.ini"
    group_vars_local_path    = "${path.module}/.secrets/staging/group_vars/"
    group_vars_jump_server_path = "/home/ubuntu/kibaship-staging/group_vars/"
    jump_server_ip          = module.jump_server.jump_server_public_ip
    setup_instructions = [
      "1. SSH to jump server: ssh -i .secrets/staging/id_ed25519 ubuntu@${module.jump_server.jump_server_public_ip}",
      "2. Clone Kubespray: git clone https://github.com/kubernetes-sigs/kubespray.git",
      "3. Copy config: cp -r /home/ubuntu/kibaship-staging/group_vars/* kubespray/inventory/mycluster/",
      "4. Copy inventory: cp /home/ubuntu/inventory.ini kubespray/inventory/mycluster/",
      "5. Install deps: cd kubespray && pip3 install -r requirements.txt",
      "6. Deploy cluster: ansible-playbook -i inventory/mycluster/inventory.ini cluster.yml -b"
    ]
  }
}

output "control_plane_ips" {
  description = "Private IP addresses of control plane nodes"
  value       = module.servers.control_plane_ips
}

output "worker_ips" {
  description = "Private IP addresses of worker nodes"
  value       = module.servers.worker_ips
}

output "deployment_info" {
  description = "Infrastructure deployment information"
  value = <<-EOT
    Infrastructure provisioned successfully!

    Jump Server (Bastion Host):
    - Public IP: ${module.jump_server.jump_server_public_ip}
    - SSH Access: ssh -i .secrets/staging/id_ed25519 ubuntu@${module.jump_server.jump_server_public_ip}

    Server Details (Private IPs - accessible via jump server):
    - Control Plane IPs: ${join(", ", module.servers.control_plane_ips)}
    - Worker IPs: ${join(", ", module.servers.worker_ips)}
    - SSH Key: .secrets/staging/id_ed25519

    Kubespray Setup:
    - Inventory file: /home/ubuntu/inventory.ini
    - Configuration files: /home/ubuntu/kibaship-staging/group_vars/
    - Local files: ${path.module}/.secrets/staging/
    - Ready for Kubernetes deployment with Cilium CNI and minimal configuration

    Infrastructure is ready for Kubernetes cluster deployment via Kubespray.
    All cluster nodes are accessible only through the jump server.
  EOT
}

output "cluster_summary" {
  description = "Complete cluster deployment summary"
  value = {
    cluster = {
      name        = var.cluster_name
      environment = var.environment
      endpoint    = module.load_balancers.k8s_api_endpoint
      network     = module.networking.network
    }
    load_balancers = {
      k8s_api_ip = module.load_balancers.k8s_api_public_ip
      app_ip     = module.load_balancers.app_public_ip
    }
    servers = {
      control_planes = length(module.servers.control_plane_servers)
      workers        = length(module.servers.worker_servers)
    }
    storage = {
      total_volumes = module.storage.storage_summary.total_volumes
      total_storage = module.storage.storage_summary.total_storage
    }
    access = {
      jump_server    = "SSH to jump server: ssh -i .secrets/staging/id_ed25519 ubuntu@${module.jump_server.jump_server_public_ip}"
      kubernetes_api = "Use load balancer IP: ${module.load_balancers.k8s_api_public_ip}:6443"
      applications   = "Use load balancer IP: ${module.load_balancers.app_public_ip} (ports 80/443)"
      note          = "All cluster nodes accessible only through jump server"
    }
  }
}