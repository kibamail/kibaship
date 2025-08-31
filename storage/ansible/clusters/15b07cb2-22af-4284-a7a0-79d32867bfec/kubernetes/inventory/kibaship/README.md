# Kibaship Dynamic Inventory

This directory contains the Edge template that Kibaship.com uses to dynamically generate Ansible inventory files for Kubernetes cluster provisioning.

## Files

- **`inventory.ini.edge`** - Edge template for generating inventory files
- **`inventory.ini`** - Generated inventory file (created at runtime, not in source control)

## Template Variables

The Edge template expects a `cluster` object with the following structure:

```javascript
{
  name: "production-k8s-cluster",
  
  sshUser: "root",
  sshKeyPath: "~/.ssh/kibaship-cluster-key",
  
  controlPlanes: [
    {
      name: "k8s-cp-1",
      publicIP: "203.0.113.11", 
      privateIP: "10.0.1.11"
    },
    {
      name: "k8s-cp-2",
      publicIP: "203.0.113.12",
      privateIP: "10.0.1.12" 
    }
  ],
  
  workers: [
    {
      name: "k8s-worker-1", 
      publicIP: "203.0.113.21",
      privateIP: "10.0.1.21"
    }
  ],
  
  loadBalancer: {
    domain: "k8s-api.company.com",
    port: 6443,
    publicIP: "203.0.113.100",
    privateIP: "10.0.1.100" // optional
  },
  
  network: {
    serviceSubnet: "10.96.0.0/12",
    podSubnet: "10.244.0.0/16", 
    dnsDomain: "cluster.local"
  },
  
  cloudProvider: {
    name: "hetzner", // or "digitalocean"
    region: "nbg1",
    projectId: "12345"
  },
  
  provisionedAt: "2024-01-15T10:30:00Z"
}
```

## Usage in Application

In your AdonisJS application, render the template like this:

```typescript
import { EdgeRenderer } from 'edge.js'

const edge = new EdgeRenderer()
const inventoryContent = await edge.render('inventory/kibaship/inventory.ini.edge', {
  cluster: clusterConfig,
  kibashipVersion: '1.0.0'
})

// Write to inventory/kibaship/inventory.ini
await fs.writeFile(inventoryPath, inventoryContent)
```

## Generated Inventory Structure

The template generates a clean INI format inventory:

```ini
[kube_control_plane]
k8s-cp-1 ansible_host=203.0.113.11 ip=10.0.1.11 access_ip=203.0.113.11
k8s-cp-2 ansible_host=203.0.113.12 ip=10.0.1.12 access_ip=203.0.113.12

[kube_node]
k8s-worker-1 ansible_host=203.0.113.21 ip=10.0.1.21 access_ip=203.0.113.21

[k8s_cluster:children]
kube_control_plane
kube_node

[all:vars]
ansible_user=root
ansible_ssh_private_key_file=~/.ssh/kibaship-cluster-key
apiserver_loadbalancer_domain_name=k8s-api.company.com
loadbalancer_apiserver_port=6443
cluster_name=production-k8s-cluster
kube_service_addresses=10.96.0.0/12
kube_pods_subnet=10.244.0.0/16
dns_domain=cluster.local
```

## Architecture Decisions Made by Ansible

The inventory template does NOT configure:

- **Kubernetes versions** - Fixed in `group_vars/all/versions.yml` (v1.34.0)
- **Container runtime** - Fixed to containerd 2.1.4
- **CNI plugin** - Fixed to Cilium v1.81.1 with kube-proxy replacement
- **Certificate SANs** - Automatically handled by the playbook
- **Default values** - All values must be provided by the application

These are architectural decisions made by the Ansible playbook, not runtime configuration.

## Benefits

- **Pure inventory generation** - Only defines hosts and minimal required variables
- **No version management** - Versions are controlled centrally in Ansible
- **Single responsibility** - Template only handles dynamic node inventory
- **Type safety** - Template enforces consistent structure
- **Cloud provider agnostic** - Works with any infrastructure provider