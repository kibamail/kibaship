# Output droplet information for Ansible inventory
output "droplet_ip" {
  description = "Public IP address of the droplet"
  value       = digitalocean_droplet.kibaship_e2e.ipv4_address
}

output "droplet_id" {
  description = "ID of the droplet"
  value       = digitalocean_droplet.kibaship_e2e.id
}

output "droplet_name" {
  description = "Name of the droplet"
  value       = digitalocean_droplet.kibaship_e2e.name
}

output "ssh_key_fingerprint" {
  description = "SSH key fingerprint"
  value       = digitalocean_ssh_key.kibaship_e2e.fingerprint
}

output "ssh_connection_command" {
  description = "SSH command to connect to the droplet"
  value       = "ssh -i ../.ssh/kibaship-e2e root@${digitalocean_droplet.kibaship_e2e.ipv4_address}"
}