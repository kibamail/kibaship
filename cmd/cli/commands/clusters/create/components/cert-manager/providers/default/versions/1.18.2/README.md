# Cert-Manager v1.18.2 - Default Provider Manifest Generation

This directory contains the cert-manager v1.18.2 manifest for the default provider configuration. Cert-manager is a native Kubernetes certificate management controller that automates the management and issuance of TLS certificates from various issuing sources.

## Overview

Cert-manager provides:
- **Automatic Certificate Management** - Automated certificate provisioning and renewal
- **Multiple Certificate Authorities** - Support for Let's Encrypt, HashiCorp Vault, Venafi, and more
- **Kubernetes Native** - CRDs for Certificate, Issuer, and ClusterIssuer resources
- **High Availability** - Multiple replicas for production deployments
- **Monitoring Integration** - Prometheus metrics and monitoring support

## Default Provider Configuration

This manifest is configured for production use with the following optimizations:

### High Availability Configuration
- **3 Controller Replicas** - High availability for the main cert-manager controller
- **2 Webhook Replicas** - Redundancy for admission webhook validation
- **1 CA Injector Replica** - Certificate authority injection component

### Feature Configuration
- **CRDs Enabled** - Custom Resource Definitions included in manifest
- **Prometheus Enabled** - Metrics collection and monitoring support
- **Production Ready** - Optimized for production workloads

## Manifest Generation Process

### Prerequisites

1. **Helm installed** - Required for manifest generation
2. **Jetstack Helm repository** - Added to your Helm repositories

### Step 1: Add Jetstack Helm Repository

```bash
helm repo add jetstack https://charts.jetstack.io
helm repo update jetstack
```

### Step 2: Generate Cert-Manager Manifest

The manifest was generated using the following command:

```bash
helm template cert-manager jetstack/cert-manager \
  --version v1.18.2 \
  --namespace cert-manager \
  --set replicaCount=3 \
  --set crds.enabled=true \
  --set prometheus.enabled=true \
  --set webhook.replicaCount=2 \
  > manifest.yaml
```

## Configuration Options Explained

### High Availability Settings

| Option | Value | Description |
|--------|-------|-------------|
| `replicaCount` | `3` | Number of cert-manager controller replicas for high availability |
| `webhook.replicaCount` | `2` | Number of webhook replicas for admission control redundancy |

### Feature Settings

| Option | Value | Description |
|--------|-------|-------------|
| `crds.enabled` | `true` | Include Custom Resource Definitions in the manifest |
| `prometheus.enabled` | `true` | Enable Prometheus metrics collection and monitoring |

## Custom Resource Definitions (CRDs)

The manifest includes the following CRDs:

1. **Certificate** - Represents a certificate request
2. **Issuer** - Defines a certificate issuer (namespace-scoped)
3. **ClusterIssuer** - Defines a certificate issuer (cluster-scoped)
4. **CertificateRequest** - Represents a certificate signing request
5. **Order** - ACME order resource for Let's Encrypt
6. **Challenge** - ACME challenge resource for domain validation

## Components Deployed

### Core Components

1. **cert-manager-controller** (3 replicas)
   - Main certificate management logic
   - Watches Certificate resources
   - Manages certificate lifecycle

2. **cert-manager-webhook** (2 replicas)
   - Admission webhook for validation
   - Ensures resource integrity
   - Validates certificate requests

3. **cert-manager-cainjector** (1 replica)
   - CA certificate injection
   - Updates webhook configurations
   - Manages CA bundles

### Supporting Resources

- **ServiceAccounts** - RBAC for each component
- **ClusterRoles** - Permissions for cluster-wide operations
- **ClusterRoleBindings** - Bind roles to service accounts
- **Services** - Network access to components
- **Deployments** - Pod management and scaling
- **ConfigMaps** - Configuration data
- **Secrets** - Sensitive configuration

## Deployment Process

### 1. Create Namespace

```bash
kubectl create namespace cert-manager
```

### 2. Apply the Manifest

```bash
kubectl apply -f manifest.yaml
```

### 3. Verify Installation

```bash
# Check all cert-manager pods
kubectl get pods -n cert-manager

# Verify CRDs are installed
kubectl get crd | grep cert-manager

# Check cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager

# Verify webhook is working
kubectl logs -n cert-manager -l app=webhook
```

### 4. Test Installation

```bash
# Create a test issuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: test-selfsigned
spec:
  selfSigned: {}
EOF

# Create a test certificate
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: test-cert
  namespace: default
spec:
  secretName: test-cert-tls
  issuerRef:
    name: test-selfsigned
    kind: ClusterIssuer
  commonName: test.example.com
EOF

# Check certificate status
kubectl describe certificate test-cert
```

## Monitoring and Observability

### Prometheus Metrics

Cert-manager exposes metrics on the following endpoints:

- **Controller metrics**: `:9402/metrics`
- **Webhook metrics**: `:10250/metrics`
- **CA Injector metrics**: `:9402/metrics`

### Key Metrics to Monitor

1. **Certificate Expiry** - `certmanager_certificate_expiration_timestamp_seconds`
2. **Certificate Renewal** - `certmanager_certificate_renewal_timestamp_seconds`
3. **ACME Orders** - `certmanager_acme_orders_total`
4. **Webhook Requests** - `certmanager_webhook_requests_total`

### Grafana Dashboard

Consider importing the official cert-manager Grafana dashboard:
- Dashboard ID: `11001`
- URL: https://grafana.com/grafana/dashboards/11001

## Common Use Cases

### 1. Let's Encrypt Integration

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
```

### 2. DNS Challenge with CloudFlare

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-dns
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-dns
    solvers:
    - dns01:
        cloudflare:
          email: admin@example.com
          apiTokenSecretRef:
            name: cloudflare-api-token
            key: api-token
```

### 3. Wildcard Certificates

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: wildcard-cert
  namespace: default
spec:
  secretName: wildcard-tls
  issuerRef:
    name: letsencrypt-dns
    kind: ClusterIssuer
  dnsNames:
  - "*.example.com"
  - "example.com"
```

## Troubleshooting

### Common Issues

1. **Webhook Timeout** - Check webhook pod logs and network policies
2. **Certificate Pending** - Verify issuer configuration and DNS/HTTP challenges
3. **ACME Rate Limits** - Use staging environment for testing
4. **CRD Conflicts** - Ensure clean installation without existing CRDs

### Useful Commands

```bash
# Check cert-manager controller logs
kubectl logs -n cert-manager -l app=cert-manager -f

# Check webhook logs
kubectl logs -n cert-manager -l app=webhook -f

# Check CA injector logs
kubectl logs -n cert-manager -l app=cainjector -f

# Describe certificate for troubleshooting
kubectl describe certificate <certificate-name>

# Check certificate events
kubectl get events --field-selector involvedObject.kind=Certificate

# Verify webhook configuration
kubectl get validatingwebhookconfigurations | grep cert-manager
```

### Debug Certificate Issues

```bash
# Check certificate request
kubectl get certificaterequests

# Check ACME orders (for Let's Encrypt)
kubectl get orders

# Check ACME challenges
kubectl get challenges

# Force certificate renewal
kubectl annotate certificate <cert-name> cert-manager.io/issue-temporary-certificate="true"
```

## Security Considerations

### RBAC Configuration

The manifest includes minimal required permissions:
- **Controller** - Manages certificates and secrets
- **Webhook** - Validates admission requests
- **CA Injector** - Updates webhook configurations

### Network Policies

Consider implementing network policies to restrict traffic:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: cert-manager-webhook
  namespace: cert-manager
spec:
  podSelector:
    matchLabels:
      app: webhook
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 10250
```

## Integration with Kibaship

This cert-manager configuration is designed for Kibaship clusters:

1. **High Availability** - Multiple replicas for production workloads
2. **Monitoring Ready** - Prometheus metrics enabled
3. **CRD Included** - No separate CRD installation required
4. **Production Optimized** - Suitable for production deployments

## Version Information

- **Cert-Manager Version**: v1.18.2
- **Kubernetes Compatibility**: 1.25+
- **Helm Chart Version**: v1.18.2
- **Generated**: October 2025

## Next Steps

After applying this manifest:

1. **Configure Issuers** - Set up ClusterIssuer for your certificate authority
2. **Create Certificates** - Define Certificate resources for your applications
3. **Set up Monitoring** - Configure Prometheus scraping and Grafana dashboards
4. **Test Renewal** - Verify automatic certificate renewal works
5. **Backup Configuration** - Backup issuer private keys and configurations

## References

- [Cert-Manager Documentation](https://cert-manager.io/docs/)
- [Jetstack Helm Charts](https://github.com/jetstack/cert-manager)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [Kubernetes Certificate Management](https://kubernetes.io/docs/concepts/cluster-administration/certificates/)
- [Kibaship Documentation](https://docs.kibaship.com/)
