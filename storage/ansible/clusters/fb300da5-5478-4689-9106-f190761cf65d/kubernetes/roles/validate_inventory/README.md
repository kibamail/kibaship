# validate_inventory

This role validates the inventory configuration before Kubernetes cluster deployment to ensure all critical infrastructure requirements are met for HA setup.

## Purpose

Prevents deployment failures by validating:
- Load balancer configuration for HA Kubernetes API
- Network CIDR configurations
- Control plane node requirements
- Required variables are properly defined

## Required Variables

### Load Balancer Configuration (CRITICAL)

These variables **MUST** be defined in your inventory:

```yaml
# REQUIRED: Load balancer endpoint for Kubernetes API
kube_control_plane_endpoint: "10.0.1.100:6443"

# REQUIRED: Public IP of the Kube API load balancer
# This will be added to API server certificate SANs
kube_load_balancer_public_ip: "203.0.113.100"
```

### Optional Load Balancer Variables

```yaml
# OPTIONAL: Private IP of the Kube API load balancer
# Recommended for internal cluster communication
kube_load_balancer_private_ip: "10.0.1.100"

# OPTIONAL: Domain name for the Kube API load balancer
# Useful for certificate management and external access
kube_load_balancer_domain: "k8s-api.example.com"
```

## Load Balancer Requirements

Your infrastructure **must** have a load balancer configured outside of Kubernetes that:

1. **Routes to Control Planes**: Load balancer must forward traffic to all control plane nodes on port 6443
2. **Health Checks**: Should health-check the `/healthz` endpoint on each control plane
3. **High Availability**: Load balancer itself should be highly available
4. **Certificate SANs**: All IPs and domains will be added to API server certificates

### Example Load Balancer Setup

**HAProxy Configuration Example:**
```
frontend kubernetes-api
    bind *:6443
    default_backend kubernetes-control-planes

backend kubernetes-control-planes
    balance roundrobin
    option httpchk GET /healthz
    server cp1 10.0.1.11:6443 check check-ssl verify none
    server cp2 10.0.1.12:6443 check check-ssl verify none
    server cp3 10.0.1.13:6443 check check-ssl verify none
```

**NGINX Configuration Example:**
```
upstream kubernetes-control-planes {
    server 10.0.1.11:6443 max_fails=3 fail_timeout=10s;
    server 10.0.1.12:6443 max_fails=3 fail_timeout=10s;
    server 10.0.1.13:6443 max_fails=3 fail_timeout=10s;
}

server {
    listen 6443;
    proxy_pass kubernetes-control-planes;
    proxy_timeout 10s;
    proxy_connect_timeout 1s;
}
```

## Network Requirements

The following network CIDRs must be defined and non-overlapping:

```yaml
kube_service_addresses: "10.96.0.0/12"    # Kubernetes services
kube_pods_subnet: "10.244.0.0/16"         # Pod networking
```

## Node Requirements

All control plane nodes must define:

```yaml
your-control-plane-node:
  main_ip: "10.0.1.11"  # Internal IP for cluster communication
```

## HA Requirements

For production High Availability:
- Minimum 3 control plane nodes (recommended: odd number)
- External load balancer for API access
- Proper network segmentation

## Usage

This role runs automatically at the beginning of `kubernetes.yml`:

```bash
ansible-playbook -i inventory/your-inventory kubernetes.yml
```

The validation will fail immediately if any critical requirements are missing.

## Example Inventory

```yaml
# inventory/production/group_vars/all.yml
kube_control_plane_endpoint: "10.0.1.100:6443"
kube_load_balancer_public_ip: "203.0.113.100"
kube_load_balancer_private_ip: "10.0.1.100"
kube_load_balancer_domain: "k8s-api.company.com"

kube_service_addresses: "10.96.0.0/12"
kube_pods_subnet: "10.244.0.0/16"
```

```ini
# inventory/production/hosts.ini
[kube_control_plane]
cp1 ansible_host=203.0.113.11 main_ip=10.0.1.11
cp2 ansible_host=203.0.113.12 main_ip=10.0.1.12
cp3 ansible_host=203.0.113.13 main_ip=10.0.1.13
```

## Validation Output

Successful validation displays:
```
TASK [validate_inventory : Display validation success summary] 
ok: [localhost] => {
    "msg": [
        "============================================",
        "INVENTORY VALIDATION SUCCESSFUL ✓",
        "============================================",
        "",
        "Load Balancer Configuration:",
        "✓ Control Plane Endpoint: 10.0.1.100:6443",
        "✓ Load Balancer Public IP: 203.0.113.100",
        "✓ Load Balancer Private IP: 10.0.1.100",
        "",
        "Cluster Configuration:",
        "✓ Control Plane Nodes: 3",
        "✓ Worker Nodes: 3",
        "✓ Network Configuration: Validated"
    ]
}
```

Failed validation will stop playbook execution with detailed error messages.
