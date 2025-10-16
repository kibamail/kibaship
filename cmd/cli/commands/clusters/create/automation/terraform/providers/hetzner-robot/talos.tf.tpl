terraform {
  required_providers {
    talos = {
      source = "siderolabs/talos"
      version = "0.9.0"
    }
  }

  backend "s3" {
    bucket = "{{.TerraformState.S3Bucket}}"
    key    = "clusters/{{.Name}}/bare-metal-talos-bootstrap/terraform.tfstate"
    region = "{{.TerraformState.S3Region}}"
    encrypt = true
  }
}

# Talos machine secrets for cluster bootstrapping
resource "talos_machine_secrets" "machine_secrets" {}

locals {
  # Base cluster configuration shared by all nodes
  cluster_config = {
    network = {
      cni = {
        name = "none"
      }
    }
    proxy = {
      disabled = true
    }
  }

  # Control plane specific cluster configuration
  control_plane_cluster_config = merge(local.cluster_config, {
    allowSchedulingOnControlPlanes = true
    apiServer = {
      admissionControl = [
        {
          name = "PodSecurity"
          configuration = {
            apiVersion = "pod-security.admission.config.k8s.io/v1beta1"
            kind = "PodSecurityConfiguration"
            exemptions = {
              namespaces = [
                "longhorn",
                "observability"
              ]
            }
          }
        }
      ]
    }
  })

  worker_machine_config = {
    machine = {
      nodeLabels = {
        "ingress.kibaship.com/ready" = "true"
      }
      features = {
        hostDNS = {
          enabled = true
          forwardKubeDNSToHost = false
        }
      }
    }
  }
}

# Generate control plane machine configurations with VLAN networking
{{$controlPlaneIndex := 0}}
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "control-plane"}}
data "talos_machine_configuration" "machineconfig_cp_{{.ID}}" {
  cluster_name     = var.cluster_name
  cluster_endpoint = var.cluster_endpoint
  machine_type     = "controlplane"
  machine_secrets  = talos_machine_secrets.machine_secrets.machine_secrets

  config_patches = concat([
    yamlencode({
      cluster = local.control_plane_cluster_config
    }),
    yamlencode({
      machine = {
        kubelet = {
          nodeIP = {
            validSubnets = [
              var.vswitch_subnet_ip_range
            ]
          }
        }
        network = {
          hostname = "cp-{{$controlPlaneIndex}}"
          interfaces = [
            {
              interface = var.server_{{.ID}}_public_network_interface
              addresses = [
                var.server_{{.ID}}_public_address_subnet
              ]
              routes = [
                {
                  network = "0.0.0.0/0"
                  gateway = var.server_{{.ID}}_public_ipv4_gateway
                }
              ]
              vlans = [
                {
                  vlanId = var.vlan_id
                  addresses = [
                    var.server_{{.ID}}_private_address_subnet
                  ]
                  mtu = 1400
                  routes = [
                    {
                      network = var.cluster_network_ip_range
                      gateway = var.server_{{.ID}}_private_ipv4_gateway
                    }
                  ]
                  vip = {
                    ip = var.vip_ip
                  }
                }
              ]
            }
          ]
        }
      }
    })
  ], [
    for disk in var.server_{{.ID}}_storage_disks :
    yamlencode({
      apiVersion = "v1alpha1"
      kind = "UserVolumeConfig"
      name = disk.name
      provisioning = {
        diskSelector = {
          match = format("\"%s\" in disk.symlinks", disk.path)
        }
      }
      filesystem = {
        type = "xfs"
      }
    })
  ])
}
{{$controlPlaneIndex = add $controlPlaneIndex 1}}
{{end}}
{{end}}

# Generate worker machine configurations with VLAN networking
{{$workerIndex := 0}}
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "worker"}}
data "talos_machine_configuration" "machineconfig_worker_{{.ID}}" {
  cluster_name     = var.cluster_name
  cluster_endpoint = var.cluster_endpoint
  machine_type     = "worker"
  machine_secrets  = talos_machine_secrets.machine_secrets.machine_secrets

  config_patches = concat([
    yamlencode(merge(
      {
        cluster = local.cluster_config
      },
      local.worker_machine_config
    )),
    yamlencode({
      machine = {
        kubelet = {
          nodeIP = {
            validSubnets = [
              var.vswitch_subnet_ip_range
            ]
          }
        }
        network = {
          hostname = "worker-{{$workerIndex}}"
          interfaces = [
            {
              interface = var.server_{{.ID}}_public_network_interface
              addresses = [
                var.server_{{.ID}}_public_address_subnet
              ]
              routes = [
                {
                  network = "0.0.0.0/0"
                  gateway = var.server_{{.ID}}_public_ipv4_gateway
                }
              ]
              vlans = [
                {
                  vlanId = var.vlan_id
                  addresses = [
                    var.server_{{.ID}}_private_address_subnet
                  ]
                  mtu = 1400
                  routes = [
                    {
                      network = var.cluster_network_ip_range
                      gateway = var.server_{{.ID}}_private_ipv4_gateway
                    }
                  ]
                }
              ]
            }
          ]
        }
      }
    })
  ], [
    for disk in var.server_{{.ID}}_storage_disks :
    yamlencode({
      apiVersion = "v1alpha1"
      kind = "UserVolumeConfig"
      name = disk.name
      provisioning = {
        diskSelector = {
          match = format("\"%s\" in disk.symlinks", disk.path)
        }
      }
      filesystem = {
        type = "xfs"
      }
    })
  ])
}
{{$workerIndex = add $workerIndex 1}}
{{end}}
{{end}}

# Apply machine configurations to control plane nodes
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "control-plane"}}
resource "talos_machine_configuration_apply" "control_plane_{{.ID}}" {
  client_configuration        = talos_machine_secrets.machine_secrets.client_configuration
  machine_configuration_input = data.talos_machine_configuration.machineconfig_cp_{{.ID}}.machine_configuration
  node                        = "{{.IP}}"
  config_patches = [
    yamlencode({
      machine = {
        install = {
          disk = var.server_{{.ID}}_installation_disk
          wipe = true
        }
      }
    })
  ]
}
{{end}}
{{end}}

# Apply machine configurations to worker nodes
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "worker"}}
resource "talos_machine_configuration_apply" "worker_{{.ID}}" {
  client_configuration        = talos_machine_secrets.machine_secrets.client_configuration
  machine_configuration_input = data.talos_machine_configuration.machineconfig_worker_{{.ID}}.machine_configuration
  node                        = "{{.IP}}"
  config_patches = [
    yamlencode({
      machine = {
        install = {
          disk = var.server_{{.ID}}_installation_disk
          wipe = true
        }
      }
    })
  ]
}
{{end}}
{{end}}

# Bootstrap the cluster on the first control plane node
resource "talos_machine_bootstrap" "this" {
  depends_on = [
{{$first := true}}
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "control-plane"}}
{{if not $first}},{{end}}
    talos_machine_configuration_apply.control_plane_{{.ID}}
{{$first = false}}
{{end}}
{{end}}
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "worker"}}
    ,talos_machine_configuration_apply.worker_{{.ID}}
{{end}}
{{end}}
  ]
  node                 = "{{range .HetznerRobot.SelectedServers}}{{if eq .Role "control-plane"}}{{.IP}}{{break}}{{end}}{{end}}"
  client_configuration = talos_machine_secrets.machine_secrets.client_configuration
}

# Generate kubeconfig after cluster bootstrap
resource "talos_cluster_kubeconfig" "this" {
  depends_on = [
    talos_machine_bootstrap.this
  ]
  client_configuration = talos_machine_secrets.machine_secrets.client_configuration
  node                 = "{{range .HetznerRobot.SelectedServers}}{{if eq .Role "control-plane"}}{{.IP}}{{break}}{{end}}{{end}}"
}

# Generate Talos client configuration
data "talos_client_configuration" "talosconfig" {
  cluster_name         = var.cluster_name
  client_configuration = talos_machine_secrets.machine_secrets.client_configuration
  endpoints            = [
{{$first := true}}
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "control-plane"}}
{{if not $first}},{{end}}
    "{{.IP}}"
{{$first = false}}
{{end}}
{{end}}
  ]
}

# Talos Configuration Outputs
output "talos_config" {
  description = "Talos client configuration for managing the cluster"
  value       = data.talos_client_configuration.talosconfig.talos_config
  sensitive   = true
}

output "control_plane_machine_configurations" {
  description = "Map of control plane machine configurations keyed by server ID"
  value = {
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "control-plane"}}
    "{{.ID}}" = data.talos_machine_configuration.machineconfig_cp_{{.ID}}.machine_configuration
{{end}}
{{end}}
  }
  sensitive = true
}

output "worker_machine_configurations" {
  description = "Map of worker machine configurations keyed by server ID"
  value = {
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "worker"}}
    "{{.ID}}" = data.talos_machine_configuration.machineconfig_worker_{{.ID}}.machine_configuration
{{end}}
{{end}}
  }
  sensitive = true
}

output "kubeconfig" {
  description = "Kubeconfig for accessing the Kubernetes cluster"
  value       = talos_cluster_kubeconfig.this.kubeconfig_raw
  sensitive   = true
}

output "cluster_info" {
  description = "Talos cluster information"
  value = {
    cluster_name = var.cluster_name
    cluster_endpoint = var.cluster_endpoint
    control_plane_nodes = [
{{$first := true}}
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "control-plane"}}
{{if not $first}},{{end}}
      {
        id = "{{.ID}}"
        name = "{{.Name}}"
        ip = "{{.IP}}"
      }
{{$first = false}}
{{end}}
{{end}}
    ]
    worker_nodes = [
{{$first := true}}
{{range .HetznerRobot.SelectedServers}}
{{if eq .Role "worker"}}
{{if not $first}},{{end}}
      {
        id = "{{.ID}}"
        name = "{{.Name}}"
        ip = "{{.IP}}"
      }
{{$first = false}}
{{end}}
{{end}}
    ]
  }
}
