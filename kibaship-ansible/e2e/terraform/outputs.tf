# VPC Information
output "vpc_id" {
  description = "VPC ID for the cluster"
  value       = digitalocean_vpc.kibaship_cluster.id
}

output "vpc_ip_range" {
  description = "IP range of the VPC"
  value       = digitalocean_vpc.kibaship_cluster.ip_range
}

# Control Plane Outputs
output "control_plane_ips" {
  description = "Public IP addresses of control plane nodes"
  value       = digitalocean_droplet.control_plane[*].ipv4_address
}

output "control_plane_private_ips" {
  description = "Private IP addresses of control plane nodes"
  value       = digitalocean_droplet.control_plane[*].ipv4_address_private
}

output "control_plane_names" {
  description = "Names of control plane nodes"
  value       = digitalocean_droplet.control_plane[*].name
}

output "control_plane_ids" {
  description = "IDs of control plane nodes"
  value       = digitalocean_droplet.control_plane[*].id
}

# Worker Node Outputs
output "worker_ips" {
  description = "Public IP addresses of worker nodes"
  value       = digitalocean_droplet.worker[*].ipv4_address
}

output "worker_private_ips" {
  description = "Private IP addresses of worker nodes"
  value       = digitalocean_droplet.worker[*].ipv4_address_private
}

output "worker_names" {
  description = "Names of worker nodes"
  value       = digitalocean_droplet.worker[*].name
}

output "worker_ids" {
  description = "IDs of worker nodes"
  value       = digitalocean_droplet.worker[*].id
}

# Load Balancer Outputs
output "kube_api_lb_ip" {
  description = "IP address of Kubernetes API load balancer"
  value       = digitalocean_loadbalancer.kube_api.ip
}

output "ingress_lb_ip" {
  description = "IP address of ingress load balancer"
  value       = digitalocean_loadbalancer.ingress.ip
}

# SSH Information
output "ssh_key_fingerprint" {
  description = "SSH key fingerprint"
  value       = digitalocean_ssh_key.kibaship_e2e.fingerprint
}

# Combined cluster information
output "cluster_info" {
  description = "Complete cluster information for inventory generation"
  value = {
    control_plane = {
      for i, node in digitalocean_droplet.control_plane : 
      node.name => {
        public_ip  = node.ipv4_address
        private_ip = node.ipv4_address_private
        id         = node.id
      }
    }
    workers = {
      for i, node in digitalocean_droplet.worker : 
      node.name => {
        public_ip  = node.ipv4_address
        private_ip = node.ipv4_address_private
        id         = node.id
      }
    }
    load_balancers = {
      kube_api = {
        ip = digitalocean_loadbalancer.kube_api.ip
      }
      ingress = {
        ip = digitalocean_loadbalancer.ingress.ip
      }
    }
  }
}