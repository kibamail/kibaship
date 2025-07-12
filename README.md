# KibaCloud - Production-Grade PaaS on Kubernetes

KibaCloud is a production-grade Platform-as-a-Service (PaaS) built on Kubernetes using GitOps principles with ArgoCD. This repository contains both infrastructure provisioning and application deployment configurations.

## Repository Structure

```
├── infrastructure/          # Terraform infrastructure configurations
│   └── staging/            # Staging environment infrastructure
├── clusters/               # Kubernetes manifests and GitOps configurations
│   ├── argocd/            # ArgoCD installation and configuration
│   ├── operators/         # Platform operators (Istio, monitoring, etc.)
│   ├── applicationsets/   # ArgoCD ApplicationSets
│   └── bootstrap/         # Bootstrap configurations
└── scripts/               # Deployment and utility scripts
```

## Infrastructure Overview

### Staging Environment

The staging environment consists of:
- **3 Control Plane Servers**: Minimal Hetzner Cloud servers (cx22) for Kubernetes masters
- **3 Worker Nodes**: Minimal Hetzner Cloud servers (cx22) for application workloads
- **2 Load Balancers**: 
  - Kubernetes API load balancer with TLS termination
  - Application traffic load balancer with TLS termination
- **Private Network**: All servers communicate through a private network (10.0.0.0/16)
- **SSL Certificates**: Managed certificates for secure communication

### Key Features

- **GitOps Approach**: All deployments managed through ArgoCD
- **Service Mesh**: Istio for complete customer isolation and traffic management
- **Minimal Resources**: Cost-optimized for staging environment
- **Security**: Private network with firewall rules and TLS termination
- **Scalability**: Ready for production scaling with identical configurations

## Prerequisites

1. **Hetzner Cloud Account**: Create an account at [console.hetzner.cloud](https://console.hetzner.cloud)
2. **Hetzner Cloud API Token**: Generate from your project's API tokens section
3. **Terraform**: Install Terraform >= 1.0
4. **SSH Key**: Ensure you have an SSH key pair at `~/.ssh/id_rsa` and `~/.ssh/id_rsa.pub`
5. **DNS Configuration**: Configure DNS zone in Hetzner DNS for SSL certificate generation

## Quick Start

### 1. Environment Setup

```bash
# Clone the repository
git clone https://github.com/kibamail/kibacloud.git
cd kibacloud

# Copy environment configuration
cp .env.example .env

# Edit .env and add your Hetzner Cloud API token
export HCLOUD_TOKEN="your-hetzner-cloud-api-token"
```

### 2. Infrastructure Deployment

```bash
# Navigate to staging infrastructure
cd infrastructure/staging

# Initialize Terraform
terraform init

# Review the planned infrastructure
terraform plan

# Deploy the infrastructure
terraform apply
```

### 3. Verify Deployment

```bash
# View infrastructure outputs
terraform output

# SSH to control plane node
ssh root@$(terraform output -raw cluster_summary | jq -r '.control_plane_ips[0]')
```

## Infrastructure Configuration

### Environment Variables

Set the following environment variable before running Terraform:

```bash
export HCLOUD_TOKEN="your-hetzner-cloud-api-token"
```

### Default Configuration

The Terraform configuration uses sensible defaults:

- **Cluster Name**: `kibaship-staging`
- **Location**: `nbg1` (Nuremberg)
- **Network Zone**: `eu-central`
- **Domain**: `staging.k8s.kibaship.com`

All defaults can be overridden by setting Terraform variables if needed.

### DNS Configuration

For SSL certificate generation, ensure your domain is configured in Hetzner DNS:

1. Create a DNS zone in [Hetzner DNS](https://dns.hetzner.com)
2. Add NS records in your primary DNS provider pointing to Hetzner nameservers
3. The Terraform configuration will automatically generate SSL certificates

## Server Specifications

### Staging Environment

| Component | Server Type | vCPU | RAM | Purpose |
|-----------|-------------|------|-----|---------|
| Control Plane | cx22 | 2 | 4 GB | Kubernetes masters |
| Worker Nodes | cx22 | 2 | 4 GB | Application workloads |
| Load Balancers | lb11 | - | - | Traffic distribution |

### Network Configuration

- **Private Network**: 10.0.0.0/16
- **Subnet**: 10.0.1.0/24
- **Control Plane IPs**: 10.0.1.10-12
- **Worker IPs**: 10.0.1.20-22
- **K8s API LB**: 10.0.1.100
- **App LB**: 10.0.1.101

## Load Balancer Configuration

### Kubernetes API Load Balancer
- **Purpose**: Routes traffic to Kubernetes API servers
- **Port**: 6443 (HTTPS)
- **Health Check**: TCP on port 6443
- **Domain**: staging.k8s.kibaship.com

### Application Load Balancer
- **Purpose**: Routes application traffic to worker nodes
- **Ports**: 80 (HTTP) → 443 (HTTPS redirect), 443 (HTTPS)
- **NodePorts**: 30080 (HTTP), 30443 (HTTPS)
- **Health Check**: HTTP on port 30080
- **SSL**: Wildcard certificate for staging subdomains

## Security Features

- **Private Network**: All inter-server communication through private IPs
- **TLS Termination**: SSL certificates managed by Hetzner Cloud
- **SSH Access**: Key-based authentication only

## Outputs

After deployment, Terraform provides detailed outputs including:
- Server public and private IP addresses
- Load balancer configurations
- Network details
- SSL certificate information
- SSH connection examples

## Next Steps

After infrastructure deployment:

1. **Install Kubernetes**: Use kubespray or similar tool to install Kubernetes
2. **Deploy ArgoCD**: Apply the ArgoCD bootstrap configuration
3. **Configure GitOps**: Set up ApplicationSets for automated deployments
4. **Install Operators**: Deploy Istio and other platform operators

## Troubleshooting

### Common Issues

1. **SSL Certificate Generation Fails**
   - Verify DNS zone configuration in Hetzner DNS
   - Check NS record delegation from primary DNS provider
   - Ensure domain ownership verification

2. **Server Creation Fails**
   - Verify Hetzner Cloud API token permissions
   - Check server type availability in selected location
   - Ensure SSH key exists at specified path

3. **Load Balancer Health Checks Fail**
   - Verify firewall rules allow health check traffic
   - Check NodePort services are running on worker nodes
   - Confirm network connectivity between load balancer and targets

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
