terraform {
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.0"
    }
  }

  backend "s3" {
    bucket = "{{.TerraformState.S3Bucket}}"
    key    = "clusters/{{.Name}}/bootstrap/terraform.tfstate"
    region = "{{.TerraformState.S3Region}}"
    encrypt = true
  }
}

# Read disk discovery results from provision stage
{{range .HetznerRobot.SelectedServers}}
data "terraform_remote_state" "provision_{{.ID}}" {
  backend = "s3"
  config = {
    bucket = "{{$.TerraformState.S3Bucket}}"
    key    = "clusters/{{$.Name}}/provision/terraform.tfstate"
    region = "{{$.TerraformState.S3Region}}"
  }
}

{{end}}

# Bootstrap configuration placeholder
# This will be used for Talos cluster configuration and Cilium/cert-manager installation
resource "null_resource" "bootstrap_placeholder" {
  provisioner "local-exec" {
    command = <<-EOF
      echo "Bootstrap stage for Hetzner Robot cluster: {{.Name}}"
      echo "Selected servers:"
{{range .HetznerRobot.SelectedServers}}
      echo "  - {{.Name}} ({{.ID}}) at {{.IP}}"
{{end}}
      echo "Disk discovery results available from provision stage"
      echo "TODO: Implement Talos cluster bootstrap and Cilium/cert-manager installation"
    EOF
  }

  triggers = {
    always_run = "${timestamp()}"
  }
}

# Output disk discovery results for reference
{{range .HetznerRobot.SelectedServers}}
output "server_{{.ID}}_disk_info" {
  description = "Disk information for server {{.Name}} (ID: {{.ID}})"
  value       = data.terraform_remote_state.provision_{{.ID}}.outputs.server_{{.ID}}_disk_discovery
}

{{end}}

# Output cluster information
output "cluster_info" {
  description = "Hetzner Robot cluster information"
  value = {
    name = "{{.Name}}"
    email = "{{.Email}}"
    paas_features = "{{.PaaSFeatures}}"
    servers = {
{{range .HetznerRobot.SelectedServers}}
      "{{.ID}}" = {
        name = "{{.Name}}"
        ip   = "{{.IP}}"
        product = "{{.Product}}"
        dc   = "{{.DC}}"
      }
{{end}}
    }
  }
}
