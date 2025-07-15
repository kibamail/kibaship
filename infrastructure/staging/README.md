# KibaShip Staging Infrastructure

This directory contains the Terraform configuration for the KibaShip staging environment infrastructure on Hetzner Cloud.

## Overview

The staging infrastructure is split into two main components:

1. **Load Balancers** (`load-balancers/`) - TCP passthrough load balancers for Kubernetes API and application traffic
2. **Servers** (`servers/`) - Kubernetes cluster nodes (control plane and worker nodes)

## Load Balancers

The load balancer configuration provisions two dedicated load balancers for different purposes:

### 1. Kubernetes API Load Balancer

- **Domain**: `staging.k8s.kibaship.com`
- **Purpose**: Routes traffic to Kubernetes API servers on control plane nodes
- **Configuration**:
  - **Protocol**: TCP passthrough
  - **Port**: 6443 (source) → 6443 (target)
  - **Health Check**: TCP on port 6443
  - **Targets**: Servers with label `role=control-plane`
  - **Load Balancer Type**: lb11
  - **Location**: nbg1 (Nuremberg)

### 2. Application Load Balancer

- **Domain**: `*.staging.kibaship.app`
- **Purpose**: Routes application traffic to worker nodes via NodePort services
- **Configuration**:
  - **Protocol**: TCP passthrough
  - **HTTP Port**: 80 (source) → 30080 (target)
  - **HTTPS Port**: 443 (source) → 30443 (target)
  - **Health Checks**: TCP on ports 30080 and 30443
  - **Targets**: Servers with label `role=worker`
  - **Load Balancer Type**: lb11
  - **Location**: nbg1 (Nuremberg)

## Architecture

```
Internet Traffic
       ↓
┌─────────────────────────────────────────────────────────────┐
│                    Hetzner Cloud                           │
│                                                             │
│  ┌─────────────────────┐    ┌─────────────────────────────┐ │
│  │   K8s API LB        │    │      App LB                 │ │
│  │ staging.k8s.        │    │ *.staging.kibaship.app      │ │
│  │ kibaship.com:6443   │    │ :80→30080, :443→30443       │ │
│  └─────────┬───────────┘    └─────────┬───────────────────┘ │
│            │                          │                     │
│            ▼                          ▼                     │
│  ┌─────────────────────┐    ┌─────────────────────────────┐ │
│  │  Control Plane      │    │     Worker Nodes            │ │
│  │  Nodes              │    │                             │ │
│  │  (role=control-     │    │  (role=worker)              │ │
│  │   plane)            │    │                             │ │
│  │                     │    │  ┌─────────────────────────┐ │ │
│  │  - K8s API Server   │    │  │    Istio Ingress        │ │ │
│  │  - etcd             │    │  │    Gateway              │ │ │
│  │  - Controller Mgr   │    │  │                         │ │ │
│  │  - Scheduler        │    │  │  NodePort 30080 (HTTP)  │ │ │
│  │                     │    │  │  NodePort 30443 (HTTPS) │ │ │
│  └─────────────────────┘    │  └─────────────────────────┘ │ │
│                             └─────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Target Selection

The load balancers use **label selectors** to automatically discover and target the appropriate servers:

- **Control Plane Targets**: `role=control-plane`
- **Worker Node Targets**: `role=worker`

This approach provides several benefits:
- **Automatic Discovery**: New servers with the correct labels are automatically added as targets
- **High Availability**: Traffic is distributed across all matching servers
- **Maintenance Friendly**: Servers can be added/removed without manual load balancer reconfiguration

## Health Checks

Both load balancers implement TCP health checks:

- **Kubernetes API LB**: TCP check on port 6443 (API server health)
- **Application LB**: TCP checks on ports 30080 and 30443 (Istio ingress gateway health)

Health check configuration:
- **Interval**: 10 seconds
- **Timeout**: 5 seconds
- **Protocol**: TCP

## Private Network Usage

The load balancers are configured to use private IP addresses (`use_private_ip = true`) for communication with target servers. This provides:

- **Enhanced Security**: Traffic between load balancers and servers stays within the private network
- **Better Performance**: Lower latency and higher throughput
- **Cost Efficiency**: No additional bandwidth charges for internal traffic

## Environment Variables

The configuration requires the following environment variable:

```bash
export HCLOUD_TOKEN="your-hetzner-cloud-api-token"
```

The token is marked as sensitive in the Terraform configuration to prevent accidental exposure in logs or state files.

## Deployment

### Prerequisites

1. **Hetzner Cloud Account** with API access
2. **Terraform** >= 1.0 installed
3. **HCLOUD_TOKEN** environment variable set

### Deploy Load Balancers

```bash
# Navigate to the load balancers directory
cd infrastructure/staging/load-balancers

# Set your Hetzner Cloud API token
export HCLOUD_TOKEN="your-hetzner-cloud-api-token"

# Initialize Terraform
terraform init

# Review the planned infrastructure
terraform plan

# Deploy the load balancers
terraform apply
```

### Outputs

After deployment, Terraform provides detailed outputs including:

- Load balancer IDs and names
- Public IPv4 and IPv6 addresses
- Domain mappings
- Port configurations
- Target server label selectors

Example output:
```
k8s_api_load_balancer = {
  "domain" = "staging.k8s.kibaship.com"
  "id" = "123456"
  "ipv4" = "1.2.3.4"
  "name" = "kibaship-staging-k8s-api"
  "port" = 6443
  "targets" = "servers with label role=control-plane"
}

app_load_balancer = {
  "domain" = "*.staging.kibaship.app"
  "id" = "123457"
  "ipv4" = "1.2.3.5"
  "name" = "kibaship-staging-app"
  "ports" = "80 -> 30080, 443 -> 30443"
  "targets" = "servers with label role=worker"
}
```

## DNS Configuration

After deployment, configure your DNS to point to the load balancer IP addresses:

1. **Kubernetes API**: Create an A record for `staging.k8s.kibaship.com` pointing to the K8s API load balancer IP
2. **Applications**: Create a wildcard A record for `*.staging.kibaship.app` pointing to the application load balancer IP

## Integration with Kubernetes

The load balancers are designed to work with:

- **Kubernetes API Server**: Direct access via port 6443
- **Istio Ingress Gateway**: Application traffic via NodePort services (30080/30443)
- **Automatic Service Discovery**: Label-based server targeting

## OpenEBS Mayastor Preparation

The Kubernetes cluster is pre-configured for OpenEBS Mayastor integration:

### Control Plane Configuration
- **Pod Security Exemptions**: `openebs` namespace exempted from baseline pod security policies
- **API Server**: Configured with PodSecurity admission controller exemptions

### Worker Node Configuration
- **Huge Pages**: 2GiB (1024 x 2MiB pages) allocated for Mayastor performance
- **Node Labels**: `openebs.io/engine=mayastor` for automatic Mayastor pod scheduling
- **Extra Mounts**: `/var/local` mounted with `rshared` for volume management
- **System Tuning**: Optimized sysctls for storage workloads

### Storage Readiness
The cluster is prepared for:
- **Mayastor DiskPools**: Worker nodes ready for disk pool creation
- **Replicated Storage**: High-performance NVMe-over-Fabrics storage
- **Volume Snapshots**: Snapshot and restore capabilities
- **Storage Classes**: Dynamic provisioning with replication policies

## Security Considerations

- **TCP Passthrough**: No SSL termination at load balancer level (handled by Kubernetes/Istio)
- **Private Network**: Internal communication uses private IPs
- **Health Checks**: Continuous monitoring of target server health
- **Label-based Targeting**: Secure server selection based on metadata

## Troubleshooting

### Common Issues

1. **Load Balancer Creation Fails**
   - Verify HCLOUD_TOKEN is set and valid
   - Check Hetzner Cloud API permissions
   - Ensure lb11 load balancer type is available in nbg1

2. **No Target Servers Found**
   - Verify servers have the correct labels (`role=control-plane` or `role=worker`)
   - Check that servers are in the same location (nbg1)
   - Ensure servers are running and healthy

3. **Health Checks Failing**
   - Verify target ports (6443, 30080, 30443) are open and listening
   - Check server firewall configurations
   - Ensure Kubernetes/Istio services are running

### Monitoring

Monitor load balancer health through:
- Hetzner Cloud Console
- Terraform state and outputs
- Kubernetes cluster monitoring tools
- Application-level health checks

## Next Steps

After load balancer deployment:

1. **Deploy Servers**: Provision Kubernetes cluster nodes with appropriate labels
2. **Configure DNS**: Point domains to load balancer IP addresses
3. **Install Kubernetes**: Set up the cluster using the API load balancer endpoint
4. **Deploy Istio**: Configure ingress gateway to use NodePort services (30080/30443)
5. **Test Connectivity**: Verify both API and application traffic routing
