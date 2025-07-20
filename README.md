# Staging Infrastructure Setup

## Infrastructure Provisioning

In the `terraform/staging` folder, run `terraform apply`. This will provision:

### **Networking**
- Private network (10.0.0.0/16)
- Subnet for servers (10.0.1.0/24)
- Load balancer for API access (port 6443)
- Load balancer for application traffic (ports 30080/30443)

### **Servers**
- 3 control plane nodes running Ubuntu 24.04
- 3 worker nodes running Ubuntu 24.04
- SSH key generation and configuration
- Basic system preparation (networking, swap disabled, etc.)

### **Storage**
- 40GB storage volumes attached to each worker node
- Raw block devices ready for storage configuration
- Consistent device paths across nodes

## Server Configuration

Each server is prepared with:
- **Operating System**: Ubuntu 24.04 LTS
- **SSH Access**: Configured with generated SSH keys
- **Networking**: Basic kernel modules loaded, IP forwarding enabled
- **System**: Swap disabled, basic packages installed
- **Storage**: Raw block devices attached (workers only)

## Access Information

After deployment:
- **SSH Key**: `.secrets/staging/ssh_key`
- **Control Plane IPs**: Listed in Terraform output
- **Worker IPs**: Listed in Terraform output
- **Load Balancer Endpoints**: Available for API and application traffic

## Next Steps

The infrastructure is ready for application deployment. You can:
- Deploy container orchestration platforms
- Configure storage solutions
- Set up monitoring and logging
- Deploy applications and services


# How to approve kubelet-serving CSRs

```bash
kubectl --kubeconfig=.secrets/staging/kubeconfig get csr -o name | xargs kubectl --kubeconfig=.secrets/staging/kubeconfig certificate approve
```

# Cilium CNI Setup

Cilium CNI is automatically installed during cluster bootstrap with the following features:
- Native routing mode for optimal performance
- Gateway API support enabled
- Load balancer acceleration
- Kubernetes IPAM mode

The installation includes Gateway API CRDs and is optimized for Ubuntu 24.04 nodes.

To verify Cilium installation:

```bash
kubectl --kubeconfig=.secrets/staging/kubeconfig get pods -n kube-system -l k8s-app=cilium
```

# How to label cilium test namespace to allow running connectivity tests

```bash
kubectl --kubeconfig=.secrets/staging/kubeconfig create namespace cilium-test-1

kubectl --kubeconfig=.secrets/staging/kubeconfig label namespace cilium-test-1 pod-security.kubernetes.io/enforce=privileged
```

# SSH Access to Nodes

To access cluster nodes via SSH:

```bash
# Access control plane nodes
ssh -i .secrets/staging/ssh_key ubuntu@<control-plane-ip>

# Access worker nodes
ssh -i .secrets/staging/ssh_key ubuntu@<worker-ip>
```

Node IPs can be found in the Terraform output or Hetzner Cloud console.

# How to install OpenEBS replicated storage with Mayastor

First add the OpenEBS Helm repository:

```bash
helm repo add openebs https://openebs.github.io/openebs
helm repo update
```

Then install OpenEBS with Mayastor enabled, optimized for Ubuntu nodes:

```bash
helm install openebs openebs/openebs \
  --namespace openebs --create-namespace \
  --set mayastor.enabled=true \
  -f terraform/staging/mayastor/values.yaml \
  --kubeconfig=.secrets/staging/kubeconfig
```

After installation, create DiskPools using the configured storage volumes:

1. Update `terraform/staging/mayastor/diskpools.yaml` with actual node names and volume device paths
2. Apply the DiskPool configuration:

```bash
kubectl --kubeconfig=.secrets/staging/kubeconfig apply -f terraform/staging/mayastor/diskpools.yaml
```

Verify Mayastor installation:

```bash
kubectl --kubeconfig=.secrets/staging/kubeconfig get pods -n openebs
kubectl --kubeconfig=.secrets/staging/kubeconfig get diskpools -n openebs
```

