# KibaShip Staging Infrastructure

This directory contains the complete Terraform configuration for the KibaShip staging environment, combining networking, load balancers, servers, and storage into a single modular project.

## Prerequisites

- Hetzner Cloud API token
- Terraform >= 1.0
- Storj S3-compatible storage for remote state
- Required providers will be automatically installed

## Environment Variables

Set these environment variables before running Terraform:

```bash
export TF_VAR_hcloud_token="your-hetzner-cloud-token"
export AWS_ACCESS_KEY_ID="your-s3-access-key"
export AWS_SECRET_ACCESS_KEY="your-s3-secret-key"
```

## Staged Deployment Process

**IMPORTANT**: Deploy in this exact order to avoid dependency issues.

### Phase 1: Initialize and Plan
```bash
# Initialize Terraform with remote state
terraform init

# Plan the entire deployment
terraform plan
```

### Phase 2: Core Infrastructure
```bash
# Deploy networking first
terraform apply -target=module.networking

# Deploy load balancers (without targets)
terraform apply -target=module.load_balancers
```

### Phase 3: Kubernetes Cluster with Cilium CNI
```bash
# Deploy servers, bootstrap cluster, and install Cilium CNI
terraform apply -target=module.servers

# Wait for cluster to be ready (check manually)
# This can take 5-10 minutes
```

### Phase 4: Load Balancer Targets
```bash
# Connect load balancers to servers
terraform apply -target=hcloud_load_balancer_target.k8s_api_targets
terraform apply -target=hcloud_load_balancer_target.app_targets
```

### Phase 5: Storage
```bash
# Deploy storage volumes
terraform apply -target=module.storage
```

### Phase 6: Complete Deployment
```bash
# Apply any remaining resources
terraform apply
```

## Verification Commands

After each phase, you can verify the deployment:

### After Phase 2 (Infrastructure)
```bash
# Check networking
terraform output network

# Check load balancers
terraform output load_balancers
```

### After Phase 3 (Cluster)
```bash
# Get kubeconfig
terraform output -raw kubeconfig > ~/.kube/config-staging

# Check cluster status
kubectl --kubeconfig ~/.kube/config-staging get nodes

# Check if all nodes are ready
kubectl --kubeconfig ~/.kube/config-staging get nodes -o wide
```

### After Phase 4 (Load Balancer Targets)
```bash
# Check load balancer health in Hetzner Cloud console
# API load balancer should show healthy targets on port 6443
# App load balancer should show healthy targets on ports 30080/30443
```

### After Phase 5 (Storage)
```bash
# Check storage volumes
terraform output storage

# Verify volumes are attached
kubectl --kubeconfig ~/.kube/config-staging get pv
```

## Troubleshooting

### Common Issues

1. **Load balancer targets fail**: Ensure servers are fully ready before Phase 4
2. **Cluster bootstrap timeout**: Wait longer, Talos can take 5-10 minutes
3. **Storage attachment fails**: Ensure worker nodes are ready and labeled correctly

### Manual Checks

```bash
# Check Talos cluster status
talosctl --talosconfig <(terraform output -raw talosconfig) health

# Check Cilium status
kubectl --kubeconfig ~/.kube/config-staging -n kube-system get pods -l k8s-app=cilium

# Check KubePrism status
kubectl --kubeconfig ~/.kube/config-staging get nodes -o wide
```

## Architecture Overview

- **Networking**: Private network `10.0.0.0/16` with subnet `10.0.1.0/24`
- **Load Balancers**: Kubernetes API (port 6443) and Application (ports 80/443)
- **Cluster**: 3 control plane + 3 worker nodes (cx22 servers)
- **Storage**: 40GB volumes per worker node, OpenEBS Mayastor ready
- **CNI**: Cilium with Gateway API enabled, KubePrism for HA

## Remote State

This configuration uses Storj S3-compatible storage for Terraform state:
- Bucket: `kibaship-terraform-state`
- Key: `staging/terraform.tfstate`
- Endpoint: `https://gateway.storjshare.io`

## Next Steps

After successful deployment:
1. Configure DNS records for `*.staging.kibaship.app`
2. Generate SSL certificates for the wildcard domain
3. Create Gateway and HTTPRoute resources for applications
4. Deploy your PaaS applications
