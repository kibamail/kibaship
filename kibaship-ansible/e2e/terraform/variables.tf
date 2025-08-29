# DigitalOcean API Token
variable "do_token" {
  description = "DigitalOcean API Token"
  type        = string
  sensitive   = true
}

# SSH Key Configuration
variable "ssh_key_path" {
  description = "Path to SSH public key"
  type        = string
  default     = "../.ssh/kibaship-e2e.pub"
}

# Droplet Configuration
variable "droplet_image" {
  description = "Ubuntu image for the droplet"
  type        = string
  default     = "ubuntu-24-04-x64"
}

variable "droplet_size" {
  description = "Size of the droplet"
  type        = string
  default     = "s-2vcpu-4gb"
}

variable "droplet_region" {
  description = "Region for the droplet"
  type        = string
  default     = "nyc3"
}

variable "project_name" {
  description = "Project name for naming resources"
  type        = string
  default     = "kibaship-e2e"
}