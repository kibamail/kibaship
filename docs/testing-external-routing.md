# Testing External Routing with Kind Clusters

This document explains how to test SNI-based routing and external access for the Kibaship operator using Kind clusters.

## Overview

The Kibaship operator now supports configurable Gateway API implementations through the `KIBASHIP_GATEWAY_CLASS_NAME` configuration. This enables testing with different Gateway API providers like Cilium, Istio, or others.

## Configuration Changes

### 1. Gateway Class Name Configuration

The operator now requires a `KIBASHIP_GATEWAY_CLASS_NAME` configuration:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kibaship-config
  namespace: kibaship
data:
  KIBASHIP_OPERATOR_DOMAIN: "example.com"
  KIBASHIP_GATEWAY_CLASS_NAME: "cilium"  # Required: Gateway API class
  WEBHOOK_TARGET_URL: "https://webhook.example.com/kibaship"
  KIBASHIP_ACME_EMAIL: "admin@example.com"  # Optional
```

### 2. Sample Configurations

- **Cilium**: Use `config/samples/kibaship-config-cilium.yaml`
- **Istio**: Use `config/samples/kibaship-config-istio.yaml`

## Testing External Access on Kind

### Option 1: Kind with extraPortMapping (Recommended)

Create a Kind cluster with port forwarding for LoadBalancer services:

```yaml
# kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  # HTTP traffic
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  # HTTPS traffic  
  - containerPort: 443
    hostPort: 443
    protocol: TCP
  # MySQL TLS (for database applications)
  - containerPort: 3306
    hostPort: 3306
    protocol: TCP
  # Valkey/Redis TLS
  - containerPort: 6379
    hostPort: 6379
    protocol: TCP
  # PostgreSQL TLS
  - containerPort: 5432
    hostPort: 5432
    protocol: TCP
```

Create the cluster:
```bash
kind create cluster --config kind-config.yaml --name kibaship-external-test
```

### Option 2: MetalLB for LoadBalancer Services

Install MetalLB to provide LoadBalancer service support:

```bash
# Install MetalLB
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.12/config/manifests/metallb-native.yaml

# Wait for MetalLB to be ready
kubectl wait --namespace metallb-system \
  --for=condition=ready pod \
  --selector=app=metallb \
  --timeout=90s

# Configure IP address pool
kubectl apply -f - <<EOF
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: example
  namespace: metallb-system
spec:
  addresses:
  - 172.19.255.200-172.19.255.250
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: empty
  namespace: metallb-system
EOF
```

### Option 3: Cilium with LoadBalancer Support

Install Cilium with LoadBalancer support:

```bash
# Install Cilium with LoadBalancer support
helm repo add cilium https://helm.cilium.io/
helm install cilium cilium/cilium \
  --namespace kube-system \
  --set kubeProxyReplacement=strict \
  --set k8sServiceHost=kind-control-plane \
  --set k8sServicePort=6443 \
  --set gatewayAPI.enabled=true \
  --set loadBalancer.mode=dsr
```

## Testing SNI-Based Routing

### 1. Deploy Test Applications

Create test applications with different domains:

```bash
# Create test applications
kubectl apply -f - <<EOF
apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: app1
  namespace: default
spec:
  name: "Test App 1"
  type: ImageFromRegistry
  imageFromRegistry:
    image: "nginx:latest"
  environmentVariables:
    - name: "APP_NAME"
      value: "app1"
---
apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Application
metadata:
  name: app2
  namespace: default
spec:
  name: "Test App 2"
  type: ImageFromRegistry
  imageFromRegistry:
    image: "nginx:latest"
  environmentVariables:
    - name: "APP_NAME"
      value: "app2"
EOF
```

### 2. Verify Gateway and HTTPRoute Creation

Check that the Gateway and HTTPRoutes are created:

```bash
# Check Gateway
kubectl get gateway -n kibaship

# Check HTTPRoutes (once implemented)
kubectl get httproute -A

# Check ApplicationDomains
kubectl get applicationdomains -A
```

### 3. Test External Access

#### With extraPortMapping:
```bash
# Add entries to /etc/hosts
echo "127.0.0.1 app1-slug.apps.example.com" >> /etc/hosts
echo "127.0.0.1 app2-slug.apps.example.com" >> /etc/hosts

# Test HTTP access
curl -H "Host: app1-slug.apps.example.com" http://localhost
curl -H "Host: app2-slug.apps.example.com" http://localhost

# Test HTTPS access (with proper certificates)
curl -H "Host: app1-slug.apps.example.com" https://localhost -k
curl -H "Host: app2-slug.apps.example.com" https://localhost -k
```

#### With MetalLB:
```bash
# Get LoadBalancer IP
LB_IP=$(kubectl get svc -n kibaship kibaship-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Add entries to /etc/hosts
echo "$LB_IP app1-slug.apps.example.com" >> /etc/hosts
echo "$LB_IP app2-slug.apps.example.com" >> /etc/hosts

# Test access
curl https://app1-slug.apps.example.com -k
curl https://app2-slug.apps.example.com -k
```

## LoadBalancer TLS Passthrough Configuration

### ✅ Automatic Cloud Provider Annotations

The Gateway resource is created with annotations that ensure the external LoadBalancer uses TLS passthrough:

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: ingress-kibaship-gateway
  namespace: kibaship
  annotations:
    # DigitalOcean LoadBalancer annotations
    # Ref: https://docs.digitalocean.com/products/kubernetes/how-to/configure-load-balancers/
    service.beta.kubernetes.io/do-loadbalancer-tls-passthrough: "true"

    # AWS LoadBalancer annotations
    # Ref: https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.3/guide/service/annotations/
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: "tcp"

    # Azure LoadBalancer annotations
    # Ref: https://learn.microsoft.com/en-us/azure/aks/load-balancer-standard
    service.beta.kubernetes.io/azure-load-balancer-tcp-idle-timeout: "4"
```

### 🌐 Traffic Flow with TLS Passthrough

```
Client (HTTPS) → Cloud LoadBalancer (TLS Passthrough) → Gateway (TLS Termination) → Application
     ↑ TLS encrypted        ↑ TLS passthrough           ↑ TLS encrypted    ↑ cert-manager certs    ↑ Plain HTTP
```

**Benefits:**
- **🔒 End-to-end encryption** until Gateway termination
- **🎯 cert-manager integration** for automatic certificate management
- **⚡ Cloud LoadBalancer efficiency** (no decrypt/re-encrypt overhead)
- **🌍 Multi-cloud compatibility** with provider-specific annotations

## Current Implementation Status

### ✅ Completed
- Gateway class name configuration system
- Dynamic Gateway creation with configurable class
- **LoadBalancer TLS passthrough annotations** for all major cloud providers
- Certificate provisioning for wildcard domains
- ApplicationDomain CR creation

### 🚧 Next Steps (HTTPRoute Implementation)
- HTTPRoute creation in ApplicationDomainReconciler
- ReferenceGrant for cross-namespace service access
- HTTP→HTTPS redirect routes
- Status updates for ingress readiness

## Testing External LoadBalancer with MetalLB

### ✅ Automated Testing

The e2e tests now include MetalLB installation and external LoadBalancer testing:

```bash
# Run e2e tests with MetalLB and external LoadBalancer testing
make test-e2e
```

The tests will verify:
- Gateway creation with correct gateway class
- **MetalLB installation and configuration**
- **LoadBalancer service creation by Cilium**
- **External IP assignment by MetalLB**
- **External connectivity to all ports (80, 443, 3306, 6379, 5432)**
- Certificate provisioning
- ApplicationDomain creation
- Service and Deployment creation

### 🧪 Manual Testing

After running the e2e tests, you can manually test external access:

```bash
# Use the automated testing script
./scripts/test-external-access.sh
```

This script will:
1. **Find the LoadBalancer service** created by Cilium
2. **Get the external IP** assigned by MetalLB
3. **Test connectivity** to all ports (HTTP, HTTPS, MySQL, Valkey, PostgreSQL)
4. **Provide manual testing commands** for further verification

### 🌐 Easy Local Testing

Once the LoadBalancer is working, you can test from your local machine:

```bash
# Get the external IP
EXTERNAL_IP=$(kubectl get svc -n kibaship -l "io.cilium.gateway/owning-gateway=ingress-kibaship-gateway" -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}')

# Test HTTP (should get Gateway response)
curl -v http://$EXTERNAL_IP:80

# Test HTTPS (should get Gateway TLS response)
curl -v -k https://$EXTERNAL_IP:443

# Test database ports (should connect to Gateway listeners)
telnet $EXTERNAL_IP 3306  # MySQL
telnet $EXTERNAL_IP 6379  # Valkey/Redis
telnet $EXTERNAL_IP 5432  # PostgreSQL
```

### 📋 What You'll See

#### **✅ Successful LoadBalancer Setup:**
```bash
$ ./scripts/test-external-access.sh
🔍 Testing external LoadBalancer access for Kibaship Gateway...
📋 Checking Gateway status...
   Gateway Status: True
🔍 Finding LoadBalancer service...
   Found LoadBalancer service: cilium-gateway-ingress-kibaship-gateway
🌐 Getting LoadBalancer external IP...
   External IP: 172.18.200.1

🧪 Testing connectivity to LoadBalancer ports...
   Testing HTTP (port 80)...
   ✅ HTTP port 80: Reachable (Gateway responding)
   Testing HTTPS (port 443)...
   ✅ HTTPS port 443: Reachable (Gateway responding)
   Testing MySQL (port 3306)...
   ✅ MySQL port 3306: Reachable
   Testing Valkey/Redis (port 6379)...
   ✅ Valkey/Redis port 6379: Reachable
   Testing PostgreSQL (port 5432)...
   ✅ PostgreSQL port 5432: Reachable

🎉 External LoadBalancer testing complete!
```

#### **🎯 Key Benefits:**
- **✅ Real LoadBalancer IPs** - MetalLB provides actual external IPs
- **✅ Multi-protocol support** - HTTP, HTTPS, and database protocols
- **✅ Local testing** - Easy to test from your development machine
- **✅ Production-like** - Mimics cloud LoadBalancer behavior
- **✅ Automated verification** - E2E tests ensure everything works

**Note**: For actual application routing, you'll need HTTPRoute implementation (next phase). The current setup verifies that external LoadBalancer connectivity is working end-to-end.
