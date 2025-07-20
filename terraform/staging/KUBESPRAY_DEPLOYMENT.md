# KibaShip Kubespray Deployment Guide

## Overview

This guide walks you through deploying a Kubernetes cluster using Kubespray with the pre-configured settings optimized for the KibaShip staging environment.

## Pre-configured Features

- **CNI**: Cilium with Gateway API and Hubble observability
- **Cloud Provider**: Disabled (no automatic cloud resource management)
- **Load Balancer**: External Hetzner Cloud load balancer (managed by Terraform)
- **Storage**: Ready for OpenEBS Mayastor (configured via Terraform)
- **Addons**: Minimal set (only metrics-server and DNS autoscaler)

## Quick Deployment

### 1. Connect to Jump Server

```bash
ssh -i .secrets/staging/ssh_key ubuntu@<JUMP_SERVER_PUBLIC_IP>
```

### 2. Clone Kubespray

```bash
git clone https://github.com/kubernetes-sigs/kubespray.git
cd kubespray
```

### 3. Set Up Configuration

```bash
# Create inventory directory
mkdir -p inventory/mycluster

# Copy pre-generated configuration
cp -r /home/ubuntu/kubespray-config/group_vars/* inventory/mycluster/
cp /home/ubuntu/inventory.ini inventory/mycluster/

# Install Python dependencies
pip3 install -r requirements.txt
```

### 4. Deploy Cluster

```bash
# Deploy the Kubernetes cluster
ansible-playbook -i inventory/mycluster/inventory.ini cluster.yml -b
```

## Configuration Details

### Load Balancer Configuration
- **API Server**: External load balancer at `<K8S_API_PUBLIC_IP>:6443`
- **Applications**: External load balancer for ingress traffic
- **No Cloud LB**: Kubernetes won't create cloud load balancers automatically

### Network Configuration
- **CNI**: Cilium with native routing (no tunneling)
- **Pod CIDR**: `10.0.16.0/20`
- **Service CIDR**: `10.0.8.0/21`
- **Gateway API**: Enabled for advanced traffic management
- **Hubble**: Enabled for network observability

### Security Features
- **Network Policies**: Enabled via Cilium
- **Encryption**: Disabled for staging (can be enabled for production)
- **Cloud Integration**: Completely disabled

### Disabled Components
- All cloud provider integrations
- Automatic load balancer provisioning
- Cloud storage provisioners
- Most Kubespray addons (ingress, monitoring, etc.)

## Post-Deployment

### 1. Verify Cluster

```bash
# Copy kubeconfig from control plane
scp ubuntu@10.0.1.10:/etc/kubernetes/admin.conf ~/.kube/config

# Check cluster status
kubectl get nodes
kubectl get pods --all-namespaces
```

### 2. Verify Cilium

```bash
# Check Cilium status
kubectl get pods -n kube-system -l k8s-app=cilium

# Check Hubble UI (if enabled)
kubectl get pods -n kube-system -l k8s-app=hubble-ui
```

### 3. Test Connectivity

```bash
# Test DNS resolution
kubectl run test-pod --image=busybox --rm -it -- nslookup kubernetes.default

# Test pod-to-pod communication
kubectl create deployment nginx --image=nginx
kubectl expose deployment nginx --port=80
kubectl run test --image=busybox --rm -it -- wget -qO- nginx
```

## Troubleshooting

### Common Issues

1. **SSH Connection Failures**
   - Verify jump server access
   - Check private key permissions
   - Ensure all nodes are accessible from jump server

2. **Ansible Failures**
   - Check Python dependencies: `pip3 install -r requirements.txt`
   - Verify inventory file format
   - Check node connectivity: `ansible -i inventory/mycluster/inventory.ini all -m ping`

3. **Network Issues**
   - Verify Cilium pods are running
   - Check node-to-node connectivity
   - Ensure private network is properly configured

### Logs and Debugging

```bash
# Check Ansible logs during deployment
ansible-playbook -i inventory/mycluster/inventory.ini cluster.yml -b -vv

# Check Cilium logs
kubectl logs -n kube-system -l k8s-app=cilium

# Check kubelet logs on nodes
journalctl -u kubelet -f
```

## Advanced Configuration

### Enable Additional Features

To enable additional features, modify the configuration files in `/home/ubuntu/kubespray-config/group_vars/`:

```bash
# Enable ingress controller
echo "ingress_nginx_enabled: true" >> kubespray-config/group_vars/k8s_cluster/addons.yml

# Enable monitoring
echo "prometheus_enabled: true" >> kubespray-config/group_vars/k8s_cluster/addons.yml
echo "grafana_enabled: true" >> kubespray-config/group_vars/k8s_cluster/addons.yml
```

### Scale the Cluster

To add more nodes:
1. Update Terraform configuration
2. Run `terraform apply`
3. Update inventory file
4. Run Kubespray scale playbook: `ansible-playbook -i inventory/mycluster/inventory.ini scale.yml -b`

## Next Steps

1. **Configure Storage**: Set up OpenEBS Mayastor for persistent volumes
2. **Deploy Ingress**: Configure ingress controllers for application access
3. **Set up Monitoring**: Deploy Prometheus and Grafana
4. **Configure Backup**: Set up etcd and persistent volume backups
5. **Security Hardening**: Implement additional security measures

## Support Resources

- **Kubespray Documentation**: https://kubespray.io/
- **Cilium Documentation**: https://docs.cilium.io/
- **Kubernetes Documentation**: https://kubernetes.io/docs/
- **Configuration Files**: Located in `/home/ubuntu/kubespray-config/group_vars/`
