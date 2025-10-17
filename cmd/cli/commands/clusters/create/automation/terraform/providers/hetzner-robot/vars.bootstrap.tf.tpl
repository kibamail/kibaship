# Server-specific Storage Disk Variables
# These are used to configure Longhorn disk annotations on nodes
{{range .HetznerRobot.SelectedServers}}
variable "server_{{.ID}}_storage_disks" {
  description = "Storage disks (non-installation disks) for server {{.Name}} ({{.ID}})"
  type = list(object({
    name = string
    path = string
  }))
  default = []
}

{{end}}
