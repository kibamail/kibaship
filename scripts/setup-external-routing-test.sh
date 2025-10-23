#!/bin/bash
set -e

# Setup script for testing external routing with Kind clusters
# This script creates a Kind cluster with external port mapping and installs
# the necessary components for testing SNI-based routing

CLUSTER_NAME="kibaship-external-test"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🚀 Setting up external routing test environment..."

# Check if kind is installed
if ! command -v kind &> /dev/null; then
    echo "❌ kind is not installed. Please install kind first:"
    echo "   https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
    exit 1
fi

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed. Please install kubectl first"
    exit 1
fi

# Check if docker is running
if ! docker info &> /dev/null; then
    echo "❌ Docker is not running. Please start Docker first"
    exit 1
fi

echo "✅ Prerequisites check passed"

# Delete existing cluster if it exists
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "🗑️  Deleting existing cluster: $CLUSTER_NAME"
    kind delete cluster --name "$CLUSTER_NAME"
fi

# Create Kind cluster with external port mapping
echo "🏗️  Creating Kind cluster with external port mapping..."
kind create cluster --config "$PROJECT_ROOT/config/kind/external-routing-cluster.yaml" --name "$CLUSTER_NAME"

# Wait for cluster to be ready
echo "⏳ Waiting for cluster to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=300s

echo "✅ Kind cluster created successfully"

# Install Gateway API CRDs
echo "📦 Installing Gateway API CRDs..."
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml

# Wait for Gateway API CRDs to be established
echo "⏳ Waiting for Gateway API CRDs to be ready..."
kubectl wait --for condition=established --timeout=60s crd/gateways.gateway.networking.k8s.io
kubectl wait --for condition=established --timeout=60s crd/httproutes.gateway.networking.k8s.io
kubectl wait --for condition=established --timeout=60s crd/referencegrants.gateway.networking.k8s.io

echo "✅ Gateway API CRDs installed"

# Option 1: Install Cilium with Gateway API support
echo "🔧 Installing Cilium with Gateway API support..."
if ! command -v helm &> /dev/null; then
    echo "⚠️  Helm not found. Installing Cilium via kubectl..."
    # Install Cilium via kubectl (basic installation)
    kubectl apply -f https://raw.githubusercontent.com/cilium/cilium/v1.14.5/install/kubernetes/quick-install.yaml
else
    # Install Cilium via Helm (preferred method)
    helm repo add cilium https://helm.cilium.io/ || true
    helm repo update
    
    # Get the Kind control plane IP
    KIND_CONTROL_PLANE_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "${CLUSTER_NAME}-control-plane")
    
    helm upgrade --install cilium cilium/cilium \
        --namespace kube-system \
        --set kubeProxyReplacement=strict \
        --set k8sServiceHost="${KIND_CONTROL_PLANE_IP}" \
        --set k8sServicePort=6443 \
        --set gatewayAPI.enabled=true \
        --set loadBalancer.mode=dsr \
        --set operator.replicas=1 \
        --wait
fi

# Wait for Cilium to be ready
echo "⏳ Waiting for Cilium to be ready..."
kubectl wait --namespace kube-system --for=condition=ready pod --selector=k8s-app=cilium --timeout=300s

echo "✅ Cilium installed with Gateway API support"

# Install cert-manager for certificate management
echo "📜 Installing cert-manager..."
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml

# Wait for cert-manager to be ready
echo "⏳ Waiting for cert-manager to be ready..."
kubectl wait --namespace cert-manager --for=condition=ready pod --selector=app.kubernetes.io/instance=cert-manager --timeout=300s

echo "✅ cert-manager installed"

# Install Tekton Pipelines
echo "🔧 Installing Tekton Pipelines..."
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.53.4/release.yaml

# Wait for Tekton to be ready
echo "⏳ Waiting for Tekton to be ready..."
kubectl wait --namespace tekton-pipelines --for=condition=ready pod --selector=app.kubernetes.io/part-of=tekton-pipelines --timeout=300s

echo "✅ Tekton Pipelines installed"

# Create kibaship namespace
echo "🏗️  Creating kibaship namespace..."
kubectl create namespace kibaship || true

# Create sample configuration for Cilium
echo "⚙️  Creating Kibaship configuration..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: kibaship-config
  namespace: kibaship
data:
  ingress.domain: "test.local"
  ingress.gateway_classname: "cilium"
  webhooks.url: "http://webhook.test.local/kibaship"
  certs.email: "admin@test.local"
EOF

echo "✅ Kibaship configuration created"

# Display cluster information
echo ""
echo "🎉 External routing test environment setup complete!"
echo ""
echo "📋 Cluster Information:"
echo "   Cluster Name: $CLUSTER_NAME"
echo "   Kubeconfig: $(kubectl config current-context)"
echo ""
echo "🌐 Port Mappings:"
echo "   HTTP:  localhost:8080  -> cluster:80"
echo "   HTTPS: localhost:8443  -> cluster:443"
echo "   MySQL: localhost:3306  -> cluster:3306"
echo "   Redis: localhost:6379  -> cluster:6379"
echo "   PostgreSQL: localhost:5432 -> cluster:5432"
echo ""
echo "🔧 Next Steps:"
echo "   1. Build and load the Kibaship operator image:"
echo "      make docker-build IMG=kibamail/kibaship:test"
echo "      kind load docker-image kibamail/kibaship:test --name $CLUSTER_NAME"
echo ""
echo "   2. Deploy the operator:"
echo "      make deploy IMG=kibamail/kibaship:test"
echo ""
echo "   3. Test external access (after HTTPRoute implementation):"
echo "      curl -H 'Host: app.apps.test.local' http://localhost:8080"
echo "      curl -H 'Host: app.apps.test.local' https://localhost:8443 -k"
echo ""
echo "🧹 Cleanup:"
echo "   kind delete cluster --name $CLUSTER_NAME"
