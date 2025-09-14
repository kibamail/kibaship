#!/bin/bash

set -e

echo "🔍 Validating KibaShip Operator Helm Chart..."

CHART_DIR="deploy/helm/kibaship-operator"
RELEASE_NAME="test-kibaship-operator"
TEST_DOMAIN="test.example.com"
TEST_PORT="8080"

# Check if helm is installed
if ! command -v helm &> /dev/null; then
    echo "❌ Helm is not installed. Please install Helm first."
    exit 1
fi

echo "✅ Helm is installed: $(helm version --short)"

# Validate chart syntax
echo "🔍 Validating chart syntax..."
if helm lint "$CHART_DIR"; then
    echo "✅ Chart syntax is valid"
else
    echo "❌ Chart syntax validation failed"
    exit 1
fi

# Test template rendering
echo "🔍 Testing template rendering..."
if helm template "$RELEASE_NAME" "$CHART_DIR" \
    --set operator.domain="$TEST_DOMAIN" \
    --set operator.defaultPort="$TEST_PORT" \
    --dry-run > /tmp/helm-template-output.yaml; then
    echo "✅ Template rendering successful"
else
    echo "❌ Template rendering failed"
    exit 1
fi

# Check if required resources are generated
echo "🔍 Checking generated resources..."

required_resources=(
    "Namespace"
    "ServiceAccount"
    "ClusterRole"
    "ClusterRoleBinding"
    "Deployment"
    "Service"
    "ValidatingAdmissionWebhook"
    "CustomResourceDefinition"
)

for resource in "${required_resources[@]}"; do
    if grep -q "kind: $resource" /tmp/helm-template-output.yaml; then
        echo "✅ $resource found in generated manifests"
    else
        echo "❌ $resource missing from generated manifests"
        exit 1
    fi
done

# Check if values are properly interpolated
echo "🔍 Checking value interpolation..."

if grep -q "value: \"$TEST_DOMAIN\"" /tmp/helm-template-output.yaml; then
    echo "✅ Domain value properly interpolated"
else
    echo "❌ Domain value not found in manifests"
    exit 1
fi

if grep -q "value: \"$TEST_PORT\"" /tmp/helm-template-output.yaml; then
    echo "✅ Port value properly interpolated"
else
    echo "❌ Port value not found in manifests"
    exit 1
fi

# Check CRD count
crd_count=$(grep -c "kind: CustomResourceDefinition" /tmp/helm-template-output.yaml)
expected_crds=4  # Project, Application, Deployment, ApplicationDomain

if [ "$crd_count" -eq "$expected_crds" ]; then
    echo "✅ All $expected_crds CRDs found"
else
    echo "❌ Expected $expected_crds CRDs, found $crd_count"
    exit 1
fi

# Test with different configurations
echo "🔍 Testing different configurations..."

# Test with debug enabled
if helm template "$RELEASE_NAME" "$CHART_DIR" \
    --set operator.domain="$TEST_DOMAIN" \
    --set debug.enabled=true \
    --set debug.level=debug \
    --dry-run > /tmp/helm-debug.yaml; then
    echo "✅ Template works with debug enabled"
else
    echo "❌ Template failed with debug enabled"
    exit 1
fi

# Verify debug environment variable is present when enabled
if grep -q "LOG_LEVEL" /tmp/helm-debug.yaml; then
    echo "✅ Debug environment variable correctly included when enabled"
else
    echo "❌ Debug environment variable missing when enabled"
    exit 1
fi

# Clean up
rm -f /tmp/helm-template-output.yaml /tmp/helm-debug.yaml

echo "🎉 All Helm chart validations passed!"
echo ""
echo "📋 Installation Commands:"
echo ""
echo "  # Basic installation:"
echo "  helm install kibaship-operator $CHART_DIR \\"
echo "    --set operator.domain=your-domain.com \\"
echo "    --create-namespace \\"
echo "    --namespace kibaship-operator"
echo ""
echo "  # With custom values:"
echo "  helm install kibaship-operator $CHART_DIR \\"
echo "    -f your-values.yaml \\"
echo "    --create-namespace \\"
echo "    --namespace kibaship-operator"
echo ""
echo "📚 For more information, see:"
echo "  - HELM_INSTALL.md"
echo "  - deploy/helm/kibaship-operator/README.md"