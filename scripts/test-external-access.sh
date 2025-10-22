#!/bin/bash
set -e

# Script to test external LoadBalancer access in Kind cluster with MetalLB
# This script helps you easily test the external LoadBalancer created by the Gateway

CLUSTER_NAME="${KIND_CLUSTER:-kibaship}"
NAMESPACE="kibaship"
GATEWAY_NAME="ingress-kibaship-gateway"

echo "🔍 Testing external LoadBalancer access for Kibaship Gateway..."

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed or not in PATH"
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    echo "❌ Cannot connect to Kubernetes cluster"
    echo "   Make sure your kubeconfig is set up correctly"
    exit 1
fi

# Check if Gateway exists
echo "📋 Checking Gateway status..."
if ! kubectl get gateway -n "$NAMESPACE" "$GATEWAY_NAME" &> /dev/null; then
    echo "❌ Gateway $GATEWAY_NAME not found in namespace $NAMESPACE"
    echo "   Make sure the Kibaship operator is deployed"
    exit 1
fi

# Get Gateway status
GATEWAY_STATUS=$(kubectl get gateway -n "$NAMESPACE" "$GATEWAY_NAME" -o jsonpath='{.status.conditions[?(@.type=="Programmed")].status}' 2>/dev/null || echo "Unknown")
echo "   Gateway Status: $GATEWAY_STATUS"

# Find the LoadBalancer service created by Cilium
echo "🔍 Finding LoadBalancer service..."
SERVICE_NAME=$(kubectl get svc -n "$NAMESPACE" -l "io.cilium.gateway/owning-gateway=$GATEWAY_NAME" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$SERVICE_NAME" ]; then
    echo "❌ LoadBalancer service not found"
    echo "   The Gateway API implementation may not have created the service yet"
    echo "   Available services in $NAMESPACE namespace:"
    kubectl get svc -n "$NAMESPACE"
    exit 1
fi

echo "   Found LoadBalancer service: $SERVICE_NAME"

# Get LoadBalancer external IP
echo "🌐 Getting LoadBalancer external IP..."
EXTERNAL_IP=$(kubectl get svc -n "$NAMESPACE" "$SERVICE_NAME" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")

if [ -z "$EXTERNAL_IP" ] || [ "$EXTERNAL_IP" = "<none>" ]; then
    echo "⏳ LoadBalancer external IP not assigned yet"
    echo "   Waiting for MetalLB to assign an IP address..."
    
    # Wait for external IP assignment
    for i in {1..30}; do
        EXTERNAL_IP=$(kubectl get svc -n "$NAMESPACE" "$SERVICE_NAME" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
        if [ -n "$EXTERNAL_IP" ] && [ "$EXTERNAL_IP" != "<none>" ]; then
            break
        fi
        echo "   Attempt $i/30: Still waiting for external IP..."
        sleep 10
    done
    
    if [ -z "$EXTERNAL_IP" ] || [ "$EXTERNAL_IP" = "<none>" ]; then
        echo "❌ LoadBalancer external IP not assigned after 5 minutes"
        echo "   Check MetalLB status:"
        kubectl get pods -n metallb-system
        exit 1
    fi
fi

echo "   External IP: $EXTERNAL_IP"

# Get service ports
echo "📋 LoadBalancer service details:"
kubectl get svc -n "$NAMESPACE" "$SERVICE_NAME" -o wide

# Test connectivity to each port
echo ""
echo "🧪 Testing connectivity to LoadBalancer ports..."

# Test HTTP (port 80)
echo "   Testing HTTP (port 80)..."
if curl -s --connect-timeout 5 --max-time 10 "http://$EXTERNAL_IP:80" > /dev/null 2>&1; then
    echo "   ✅ HTTP port 80: Reachable"
elif curl -v --connect-timeout 5 --max-time 10 "http://$EXTERNAL_IP:80" 2>&1 | grep -q "Connection refused\|Empty reply\|HTTP/\|404\|502\|503"; then
    echo "   ✅ HTTP port 80: Reachable (Gateway responding)"
else
    echo "   ❌ HTTP port 80: Not reachable"
fi

# Test HTTPS (port 443)
echo "   Testing HTTPS (port 443)..."
if curl -s -k --connect-timeout 5 --max-time 10 "https://$EXTERNAL_IP:443" > /dev/null 2>&1; then
    echo "   ✅ HTTPS port 443: Reachable"
elif curl -v -k --connect-timeout 5 --max-time 10 "https://$EXTERNAL_IP:443" 2>&1 | grep -q "Connection refused\|Empty reply\|HTTP/\|404\|502\|503\|SSL"; then
    echo "   ✅ HTTPS port 443: Reachable (Gateway responding)"
else
    echo "   ❌ HTTPS port 443: Not reachable"
fi

# Test database ports with netcat/telnet if available
for port in 3306 6379 5432; do
    service_name=""
    case $port in
        3306) service_name="MySQL" ;;
        6379) service_name="Valkey/Redis" ;;
        5432) service_name="PostgreSQL" ;;
    esac
    
    echo "   Testing $service_name (port $port)..."
    if command -v nc &> /dev/null; then
        if timeout 5 nc -z "$EXTERNAL_IP" "$port" 2>/dev/null; then
            echo "   ✅ $service_name port $port: Reachable"
        else
            echo "   ❌ $service_name port $port: Not reachable"
        fi
    elif command -v telnet &> /dev/null; then
        if timeout 5 bash -c "echo | telnet $EXTERNAL_IP $port" 2>/dev/null | grep -q "Connected"; then
            echo "   ✅ $service_name port $port: Reachable"
        else
            echo "   ❌ $service_name port $port: Not reachable"
        fi
    else
        echo "   ⚠️  $service_name port $port: Cannot test (nc/telnet not available)"
    fi
done

# Test ACME-DNS LoadBalancer service separately
echo ""
echo "🔍 Testing ACME-DNS LoadBalancer service..."

# Find the ACME-DNS LoadBalancer service
ACME_DNS_SERVICE_NAME=$(kubectl get svc -n "$NAMESPACE" -l "service-type=dns" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$ACME_DNS_SERVICE_NAME" ]; then
    echo "⚠️  ACME-DNS LoadBalancer service not found"
    echo "   This is expected if ACME-DNS is not configured or base domain is not set"
else
    echo "   Found ACME-DNS LoadBalancer service: $ACME_DNS_SERVICE_NAME"

    # Get ACME-DNS LoadBalancer external IP
    ACME_DNS_EXTERNAL_IP=$(kubectl get svc -n "$NAMESPACE" "$ACME_DNS_SERVICE_NAME" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")

    if [ -z "$ACME_DNS_EXTERNAL_IP" ] || [ "$ACME_DNS_EXTERNAL_IP" = "<none>" ]; then
        echo "   ⚠️  ACME-DNS LoadBalancer external IP not assigned yet"
    else
        echo "   ACME-DNS External IP: $ACME_DNS_EXTERNAL_IP"

        # Test DNS port 53 (UDP)
        echo "   Testing DNS UDP (port 53)..."
        if command -v dig &> /dev/null; then
            if timeout 5 dig @"$ACME_DNS_EXTERNAL_IP" -p 53 +short example.com 2>/dev/null | grep -q ".*"; then
                echo "   ✅ DNS UDP port 53: Reachable (DNS responding)"
            else
                echo "   ⚠️  DNS UDP port 53: No response (may be normal for ACME-DNS)"
            fi
        elif command -v nslookup &> /dev/null; then
            if timeout 5 nslookup example.com "$ACME_DNS_EXTERNAL_IP" 2>/dev/null | grep -q "Address"; then
                echo "   ✅ DNS UDP port 53: Reachable (DNS responding)"
            else
                echo "   ⚠️  DNS UDP port 53: No response (may be normal for ACME-DNS)"
            fi
        else
            echo "   ⚠️  DNS UDP port 53: Cannot test (dig/nslookup not available)"
        fi

        # Test DNS port 53 (TCP)
        echo "   Testing DNS TCP (port 53)..."
        if command -v nc &> /dev/null; then
            if timeout 5 nc -z "$ACME_DNS_EXTERNAL_IP" 53 2>/dev/null; then
                echo "   ✅ DNS TCP port 53: Reachable"
            else
                echo "   ❌ DNS TCP port 53: Not reachable"
            fi
        else
            echo "   ⚠️  DNS TCP port 53: Cannot test (nc not available)"
        fi
    fi
fi

echo ""
echo "🎉 External LoadBalancer testing complete!"
echo ""
echo "📋 Summary:"
echo "   Cluster: $CLUSTER_NAME"
echo "   Gateway: $GATEWAY_NAME"
echo "   LoadBalancer Service: $SERVICE_NAME"
echo "   External IP: $EXTERNAL_IP"
if [ -n "$ACME_DNS_SERVICE_NAME" ] && [ -n "$ACME_DNS_EXTERNAL_IP" ]; then
    echo "   ACME-DNS Service: $ACME_DNS_SERVICE_NAME"
    echo "   ACME-DNS External IP: $ACME_DNS_EXTERNAL_IP"
fi
echo ""
echo "🧪 Manual testing commands:"
echo "   # Test HTTP"
echo "   curl -v http://$EXTERNAL_IP:80"
echo ""
echo "   # Test HTTPS (with self-signed cert)"
echo "   curl -v -k https://$EXTERNAL_IP:443"
echo ""
echo "   # Test database ports"
echo "   telnet $EXTERNAL_IP 3306  # MySQL"
echo "   telnet $EXTERNAL_IP 6379  # Valkey/Redis"
echo "   telnet $EXTERNAL_IP 5432  # PostgreSQL"
if [ -n "$ACME_DNS_EXTERNAL_IP" ]; then
    echo ""
    echo "   # Test ACME-DNS"
    echo "   dig @$ACME_DNS_EXTERNAL_IP example.com  # DNS UDP"
    echo "   nslookup example.com $ACME_DNS_EXTERNAL_IP  # DNS UDP"
    echo "   telnet $ACME_DNS_EXTERNAL_IP 53  # DNS TCP"
fi
echo ""
echo "💡 Note: The Gateway will route traffic based on SNI (Server Name Indication)"
echo "   For actual application routing, you'll need to:"
echo "   1. Deploy applications with the Kibaship operator"
echo "   2. Use proper hostnames in your requests"
echo "   3. Configure DNS or /etc/hosts entries"
