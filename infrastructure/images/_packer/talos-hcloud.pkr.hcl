# hcloud.pkr.hcl
packer {
  required_plugins {
    hcloud = {
      version = "v1.6.1"
      source  = "github.com/hetznercloud/hcloud"
    }
  }
}

variable "talos_version" {
  type    = string
  default = "v1.10.6"
}

variable "image_id" {
  type    = string
  default = "376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba"
}

variable "server_location" {
  type    = string
  default = "fsn1"
}

locals {
  image_arm = "https://factory.talos.dev/image/${var.image_id}/${var.talos_version}/hcloud-arm64.raw.xz"
  image_x86 = "https://factory.talos.dev/image/${var.image_id}/${var.talos_version}/hcloud-amd64.raw.xz"

  download_image = "wget --timeout=5 --waitretry=5 --tries=5 --retry-connrefused --inet4-only -O /tmp/talos.raw.xz "

  write_image = <<-EOT
    set -ex
    echo 'Talos image loaded, writing to disk... '
    xz -d -c /tmp/talos.raw.xz | dd of=/dev/sda && sync
    echo 'done.'
  EOT

  clean_up = <<-EOT
    set -ex
    echo "Cleaning-up..."
    rm -rf /etc/ssh/ssh_host_*
  EOT
}

source "hcloud" "talos-arm" {
  rescue       = "linux64"
  image        = "debian-11"
  location     = "${var.server_location}"
  server_type  = "cax11"
  ssh_username = "root"

  snapshot_name   = "talos-linux-${var.talos_version}-arm"
  snapshot_labels = {
    os      = "talos",
    version = "${var.talos_version}",
    arch    = "arm",
  }
}

source "hcloud" "talos-x86" {
  rescue       = "linux64"
  image        = "debian-11"
  location     = "${var.server_location}"
  server_type  = "cx22"
  ssh_username = "root"

  snapshot_name   = "talos-linux-${var.talos_version}-x86"
  snapshot_labels = {
    os      = "talos",
    version = "${var.talos_version}",
    arch    = "x86",
  }
}

build {
  sources = ["source.hcloud.talos-arm"]

  provisioner "shell" {
    inline = ["${local.download_image}${local.image_arm}"]
  }

  provisioner "shell" {
    inline = [local.write_image]
  }

  provisioner "shell" {
    inline = [local.clean_up]
  }
}

build {
  sources = ["source.hcloud.talos-x86"]

  provisioner "shell" {
    inline = ["${local.download_image}${local.image_x86}"]
  }

  provisioner "shell" {
    inline = [local.write_image]
  }

  provisioner "shell" {
    inline = [local.clean_up]
  }
}