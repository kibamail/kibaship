terraform {
  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "~> 3.0"
    }
  }

  backend "s3" {
    bucket = "{{.TerraformState.S3Bucket}}"
    key    = "clusters/{{.Name}}/provision/terraform.tfstate"
    region = "{{.TerraformState.S3Region}}"
    encrypt = true
  }
}

{{range .HetznerRobot.SelectedServers}}
# Discover disks for server {{.Name}} (ID: {{.ID}})
resource "null_resource" "disk_discovery_{{.ID}}" {
  provisioner "remote-exec" {
    connection {
      type     = "ssh"
      user     = "root"
      password = var.server_{{.ID}}_password
      host     = "{{.IP}}"
      timeout  = "5m"
    }

    inline = [
      # Ensure required tools are installed and run disk discovery
      <<-EOF
        #!/bin/bash
        set -e
        # 0. install jq
        apt-get update && apt-get install -y jq

        # 1. Get all block devices (exclude md/raid devices)
        DEVICES=$(lsblk -J -b -d -n -o NAME,SIZE,TYPE | jq -c '.blockdevices[] | select(.type=="disk" and (.name | startswith("md") | not))')

        # 2. Wipe all block devices
        echo "$DEVICES" | jq -r '.name' | while read -r dev; do
            wipefs -a /dev/$dev 2>/dev/null || true
            sgdisk --zap-all /dev/$dev 2>/dev/null || true
        done

        # 3. Add disk_by_id to each device
        DEVICES_WITH_ID=$(echo "$DEVICES" | while read -r device; do
            DEV_NAME=$(echo "$device" | jq -r '.name')

            # Get disk-by-id with model+serial preference
            DISK_BY_ID=$(ls -l /dev/disk/by-id/ | \
                grep -w "$DEV_NAME\$" | \
                grep -v "part" | \
                grep -v "eui\." | \
                grep -v "_1\$" | \
                head -n1 | \
                awk '{print $9}')

            # Fallback to any valid ID if model+serial not found
            if [ -z "$DISK_BY_ID" ]; then
                DISK_BY_ID=$(ls -l /dev/disk/by-id/ | \
                    grep -w "$DEV_NAME\$" | \
                    grep -v "part" | \
                    head -n1 | \
                    awk '{print $9}')
            fi

            echo "$device" | jq --arg disk_by_id "$DISK_BY_ID" '. + {disk_by_id: $disk_by_id}'
        done | jq -s '.')

        # 4. Find smallest device
        SMALLEST=$(echo "$DEVICES_WITH_ID" | jq 'sort_by(.size) | .[0]')
        SMALLEST_NAME=$(echo "$SMALLEST" | jq -r '.name')
        SMALLEST_DISK_BY_ID=$(echo "$SMALLEST" | jq -r '.disk_by_id')

        # 5. Create JSON response and save to file
        jq -n \
          --argjson devices "$DEVICES_WITH_ID" \
          --arg talos_device "$SMALLEST_NAME" \
          --arg talos_disk_by_id "$SMALLEST_DISK_BY_ID" \
          '{
            "all_devices": $devices,
            "talos_installation": {
              "device": $talos_device,
              "disk_by_id": $talos_disk_by_id,
              "full_path": ("/dev/" + $talos_device),
              "disk_by_id_path": ("/dev/disk/by-id/" + $talos_disk_by_id)
            }
          }' > /tmp/disk_discovery_{{.ID}}.json

        # Output the result
        cat /tmp/disk_discovery_{{.ID}}.json

        # 6. Detect architecture
        ARCH=$(uname -m)
        TALOS_VERSION="v1.11.2"
        if [ "$ARCH" = "x86_64" ]; then
          TALOS_ARCH="amd64"
        elif [ "$ARCH" = "aarch64" ]; then
          TALOS_ARCH="arm64"
        else
          echo "Unsupported architecture: $ARCH"
          exit 1
        fi

        # 7. Download Talos image
        echo "Downloading Talos image for $TALOS_ARCH..."
        wget -O /tmp/metal-$TALOS_ARCH.raw.zst https://factory.talos.dev/image/3df38c5e5faf43879e6ff0f13c6b0ba02aaa0eb5f9291f28749c4056c1974e7b/$TALOS_VERSION/metal-$TALOS_ARCH.raw.zst

        # 8. Extract Talos image
        echo "Extracting Talos image..."
        apt-get install -y zstd
        zstd -d /tmp/metal-$TALOS_ARCH.raw.zst -o /tmp/metal-$TALOS_ARCH.raw

        # 9. Write Talos image to target disk
        TARGET_DISK="/dev/disk/by-id/$SMALLEST_DISK_BY_ID"
        echo "Writing Talos image to $TARGET_DISK..."
        dd if=/tmp/metal-$TALOS_ARCH.raw of=$TARGET_DISK bs=1M status=progress

        # 10. Sync to ensure all writes are flushed
        echo "Syncing..."
        sync

        echo "Talos image installation complete on $TARGET_DISK"
      EOF
    ]
  }

  provisioner "local-exec" {
    command = <<-EOF
      set -e

      # Create temporary expect script for SSH with password
      EXPECT_SCRIPT=$(mktemp)
      trap "rm -f $EXPECT_SCRIPT" EXIT

      cat > $EXPECT_SCRIPT << 'EXPECT_EOF'
#!/usr/bin/expect -f
log_user 0
set timeout 30
spawn ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@{{.IP}} "cat /tmp/disk_discovery_{{.ID}}.json"
expect {
    "password:" {
        send "$env(SSH_PASS)\r"
        log_user 1
        expect eof
    }
    eof
}
EXPECT_EOF

      chmod +x $EXPECT_SCRIPT

      # Run expect script and save JSON to local file
      export SSH_PASS='${var.server_{{.ID}}_password}'
      $EXPECT_SCRIPT > ${path.module}/disk_discovery_{{.ID}}.json 2>/dev/null
    EOF
  }

  provisioner "remote-exec" {
    connection {
      type     = "ssh"
      user     = "root"
      password = var.server_{{.ID}}_password
      host     = "{{.IP}}"
      timeout  = "2m"
    }

    inline = [
      "echo 'Rebooting server to boot into Talos...'",
      "reboot"
    ]

    on_failure = continue
  }

  triggers = {
    always_run = "${timestamp()}"
  }
}

{{end}}

# Read local disk discovery files
{{range .HetznerRobot.SelectedServers}}
data "local_file" "disk_discovery_{{.ID}}" {
  depends_on = [null_resource.disk_discovery_{{.ID}}]
  filename   = "${path.module}/disk_discovery_{{.ID}}.json"
}

{{end}}

{{range .HetznerRobot.SelectedServers}}
variable "server_{{.ID}}_password" {
  description = "Root password for server {{.Name}} (ID: {{.ID}})"
  type        = string
  sensitive   = true
}

{{end}}

{{range .HetznerRobot.SelectedServers}}
output "server_{{.ID}}_disk_discovery" {
  description = "Disk discovery results for server {{.Name}} (ID: {{.ID}})"
  value       = jsondecode(data.local_file.disk_discovery_{{.ID}}.content)
}

{{end}}
