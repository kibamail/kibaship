# KibaShip Staging Storage Configuration

This directory contains the Terraform configuration for provisioning persistent storage volumes for the KibaShip staging Kubernetes cluster worker nodes.

## Overview

The configuration provisions:
- **3 x 40GB SSD volumes** (one per worker node)
- **Automatic volume attachment** to worker nodes using label selectors
- **ext4 filesystem formatting** for immediate use
- **OpenEBS Mayastor-ready configuration** for persistent storage

## Prerequisites

1. **Staging cluster must be deployed** - The servers module must be applied first
2. **Hetzner Cloud API token** - Set as `HCLOUD_TOKEN` environment variable
3. **Talos configuration** - Available from the servers module deployment

## Deployment

```bash
# Navigate to storage directory
cd infrastructure/staging/storage

# Initialize Terraform
terraform init

# Review the planned changes
terraform plan

# Apply the configuration
terraform apply
```

## Manual Health Check Instructions

After deploying the storage volumes, perform these manual health checks to verify volume accessibility:

### 1. Get Talos Configuration

First, ensure you have the Talos configuration from the servers deployment:

```bash
# Copy talosconfig from servers module
cp ../servers/talosconfig ./talosconfig

# Or export from servers module output
cd ../servers
terraform output -raw talosconfig > ../storage/talosconfig
cd ../storage
```

### 2. Get Worker Node IPs

```bash
# Get worker node public IPs
terraform output storage_volumes
```

### 3. Verify Volume Attachment

For each worker node, verify the volume is attached and accessible:

```bash
# Replace <WORKER_IP> with actual worker node IP
# Replace <VOLUME_ID> with actual volume ID from terraform output

# Check if volume is visible in the system
talosctl --nodes <WORKER_IP> \
         --endpoints <WORKER_IP> \
         --talosconfig ./talosconfig \
         list /dev/disk/by-id/ | grep "scsi-0HC_Volume_<VOLUME_ID>"

# Example for worker-1:
talosctl --nodes 10.0.1.20 \
         --endpoints 10.0.1.20 \
         --talosconfig ./talosconfig \
         list /dev/disk/by-id/ | grep "scsi-0HC_Volume_"
```

### 4. Check Volume Mount Status

```bash
# Check mounted filesystems
talosctl --nodes <WORKER_IP> \
         --endpoints <WORKER_IP> \
         --talosconfig ./talosconfig \
         df

# Check disk usage
talosctl --nodes <WORKER_IP> \
         --endpoints <WORKER_IP> \
         --talosconfig ./talosconfig \
         ls -la /dev/disk/by-id/

# Verify consistent mount paths
talosctl --nodes <WORKER_IP> \
         --endpoints <WORKER_IP> \
         --talosconfig ./talosconfig \
         ls -la /mnt/
```

### 5. Verify All Worker Nodes

Run the checks for all three worker nodes:

```bash
# Get all worker IPs from terraform output
terraform output -json storage_volumes | jq -r '.[] | .worker_node + ": " + .device_path'

# For each worker, run the verification commands above
```

### 6. Expected Results

✅ **Successful health check should show:**
- Volume device appears in `/dev/disk/by-id/` with format `scsi-0HC_Volume_<volume_id>`
- Volume is mounted at consistent path `/mnt/HC_Volume_<volume_id>`
- Filesystem shows available space (approximately 40GB)
- All worker nodes have the same mount path pattern

❌ **Troubleshooting common issues:**
- **Volume not found**: Wait 30-60 seconds for attachment to complete, then retry
- **Permission denied**: Ensure talosconfig has proper permissions
- **Connection refused**: Verify worker node IP and network connectivity

## Volume Configuration Details

### Volume Specifications
- **Size**: 40GB per volume
- **Type**: network-ssd (high performance)
- **Filesystem**: ext4
- **Auto-mount**: Enabled
- **Location**: Same as worker nodes (nbg1)

### Device Paths and Mount Points
Volumes are accessible at predictable paths:

**Device paths:**
```
/dev/disk/by-id/scsi-0HC_Volume_<volume_id>
```

**Mount paths (consistent across all workers):**
```
/mnt/HC_Volume_<volume_id>
```

### Labels
Each volume is labeled for easy identification:
```
environment = "staging"
cluster     = "kibaship-staging"
role        = "worker-storage"
worker_node = "kibaship-staging-worker-<N>"
```

## OpenEBS Mayastor Integration

The volumes are pre-configured for OpenEBS Mayastor storage engine:

### DiskPool Configuration
Use the device paths in your Mayastor DiskPool configuration:

```yaml
apiVersion: openebs.io/v1alpha1
kind: DiskPool
metadata:
  name: pool-worker-1
  namespace: openebs
spec:
  node: kibaship-staging-worker-1
  disks: ["/dev/disk/by-id/scsi-0HC_Volume_<volume_id>"]
```

### Next Steps
1. Deploy OpenEBS Mayastor operator
2. Create DiskPools using the volume device paths
3. Configure StorageClasses for persistent volumes

## Outputs

The configuration provides detailed outputs including:
- **storage_volumes**: Complete volume details with device paths
- **volume_attachments**: Attachment status for each volume
- **storage_summary**: Overview of total storage capacity and Mayastor readiness

## Cleanup

To remove all storage volumes:

```bash
terraform destroy
```

⚠️ **Warning**: This will permanently delete all data on the volumes. Ensure you have backups before destroying.
