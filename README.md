# How to setup a staging cluster

1. Setup all infrastructure needed.

In the `infrastructure/staging` folder, run `terraform apply`. This will:

- Create a private network
- Create a load balancer for the Kubernetes API
- Create a load balancer for the application ingress
- Create 3 control plane nodes
- Create 3 worker nodes
- Bootstrap the Talos linux kubernetes cluster
- Install KubePrism
- Create storage volumes for the worker nodes
- Attach these volumes to worker nodes

2. Copy secret configuration files

Run the command `terraform output kubeconfig > /path/to/secret/store` to copy the Kubernetes configuration file to your local machine.
Run the command `terraform output talosconfig > /path/to/secret/store` to copy the Talos configuration file to your local machine.

3. Verify the cluster is up and running

Run the command `kubectl get nodes` to verify the cluster is up and running.


# How to approve kubelet-serving CSRs

```bash
kubectl get csr -o name | xargs kubectl certificate approve
```

# How to setup cilium CNI

Follow this guide to install cilium cli:

https://docs.cilium.io/en/v1.16/network/servicemesh/gateway-api/gateway-api/#gs-gateway-host-network-mode

Next, install the CRDs for k8s Gateway support:

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/standard/gateway.networking.k8s.io_gatewayclasses.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/standard/gateway.networking.k8s.io_gateways.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/standard/gateway.networking.k8s.io_httproutes.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/standard/gateway.networking.k8s.io_referencegrants.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/standard/gateway.networking.k8s.io_grpcroutes.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.1.0/config/crd/experimental/gateway.networking.k8s.io_tlsroutes.yaml
```

Finally, run the following command to install cilium:

```bash
cilium install \
    --set ipam.mode=kubernetes \
    --set kubeProxyReplacement=true \
    --set securityContext.capabilities.ciliumAgent="{CHOWN,KILL,NET_ADMIN,NET_RAW,IPC_LOCK,SYS_ADMIN,SYS_RESOURCE,DAC_OVERRIDE,FOWNER,SETGID,SETUID}" \
    --set securityContext.capabilities.cleanCiliumState="{NET_ADMIN,SYS_ADMIN,SYS_RESOURCE}" \
    --set cgroup.autoMount.enabled=false \
    --set cgroup.hostRoot=/sys/fs/cgroup \
    --set k8sServiceHost=localhost \
    --set k8sServicePort=7445 \
    --set gatewayAPI.enabled=true \
    --set gatewayAPI.enableAlpn=true \
    --set gatewayAPI.enableAppProtocol=true
```

# How to label cilium test namespace to allow running connectivity tests

```bash
kubectl create namespace cilium-test-1

kubectl label namespace cilium-test-1 pod-security.kubernetes.io/enforce=privileged
```

# How to setup fluxcd for gitops in staging cluster

1. Install fluxcd command line :

2. Set Github personal access token and user in command line:

```bash
export GITHUB_TOKEN=your-github-token-here
export GITHUB_USER=your-github-username-here
```

3. Run bootstrap command:

```bash
flux bootstrap github \
  --token-auth \
  --owner=kibamail \
  --repository=kibaship \
  --branch=main \
  --path=clusters/staging \
  --cluster-domain=kibaship.internal
```
