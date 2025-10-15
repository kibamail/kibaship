package bootstrap

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ingress configuration constants
// All ingress-related resources are prefixed with "ingress-" for consistency
const (
	// KibashipNamespace is the namespace where the operator and ingress resources run
	KibashipNamespace = "kibaship"

	// CertificatesNamespace is where TLS certificates are stored
	CertificatesNamespace = "certificates"

	// IngressIssuerName is the cert-manager ClusterIssuer for ACME certificates
	IngressIssuerName = "certmanager-acme-issuer"

	// IngressWildcardCertName is the wildcard certificate for all application domains
	IngressWildcardCertName = "ingress-kibaship-certificate"

	// IngressGatewayName is the Gateway API Gateway resource name
	IngressGatewayName = "ingress-kibaship-gateway"

	// IngressReferenceGrantName is the ReferenceGrant for cross-namespace access
	IngressReferenceGrantName = "ingress-certificates-access"
)

// ProvisionIngress ensures ingress resources are present based on configured domain/email.
// It is idempotent and safe to call on every manager start.
//
// Prerequisites:
//   - certificates namespace must exist
//   - kibaship namespace must exist (operator namespace)
//   - cert-manager must be installed and ready
//   - Cilium Gateway API support must be enabled
//
// Resources created (in order):
//   1. Wildcard Certificate for all app domains (*.apps, *.valkey, *.mysql, *.postgres)
//   2. ReferenceGrant for Gateway to access certificates
//   3. Gateway resource with multi-protocol listeners (only if certificate exists)
func ProvisionIngress(ctx context.Context, c client.Client, baseDomain, acmeEmail string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("ingress")

	if baseDomain == "" {
		log.Info("No base domain configured, skipping ingress provisioning")
		return nil
	}

	// 1. Ensure certificates namespace exists
	log.Info("Step 1: Ensuring certificates namespace exists")
	if err := ensureNamespace(ctx, c, CertificatesNamespace); err != nil {
		return fmt.Errorf("ensure certificates namespace: %w", err)
	}

	// 2. Ensure kibaship namespace exists (should already exist, but check anyway)
	log.Info("Step 2: Ensuring kibaship namespace exists")
	if err := ensureNamespace(ctx, c, KibashipNamespace); err != nil {
		return fmt.Errorf("ensure kibaship namespace: %w", err)
	}

	// 3. Wildcard Certificate for all app domains (depends on issuer and domain)
	if acmeEmail != "" {
		log.Info("Step 3: Ensuring wildcard certificate for all app domains")
		if err := ensureIngressWildcardCertificate(ctx, c, baseDomain); err != nil {
			return fmt.Errorf("ensure wildcard certificate: %w", err)
		}
	}

	// 4. Check if wildcard certificate is ready before creating Gateway resources
	log.Info("Step 4: Checking if wildcard certificate is ready")
	certReady, err := isWildcardCertificateReady(ctx, c)
	if err != nil {
		log.Info("Failed to check wildcard certificate status, will retry on next reconciliation", "error", err)
		return fmt.Errorf("check wildcard certificate: %w", err)
	}

	if !certReady {
		log.Info("Wildcard certificate not ready yet, skipping Gateway resource creation")
		return nil
	}

	log.Info("Wildcard certificate is ready, proceeding with Gateway resource creation")

	// 5. ReferenceGrant for Gateway to access certificates (in kibaship namespace)
	log.Info("Step 5: Ensuring ReferenceGrant for certificate access")
	if err := ensureIngressReferenceGrant(ctx, c); err != nil {
		return fmt.Errorf("ensure ReferenceGrant: %w", err)
	}

	// 6. Gateway resource with multi-protocol listeners (in kibaship namespace)
	log.Info("Step 6: Ensuring Gateway resource")
	if err := ensureIngressGateway(ctx, c); err != nil {
		return fmt.Errorf("ensure Gateway: %w", err)
	}

	log.Info("Ingress provisioning completed successfully")
	return nil
}

// ensureIngressWildcardCertificate creates the wildcard certificate for all application domains.
// The certificate covers:
//   - *.apps.{domain} - Web applications (GitRepository, DockerImage)
//   - *.valkey.{domain} - Valkey/Redis databases
//   - *.mysql.{domain} - MySQL databases
//   - *.postgres.{domain} - PostgreSQL databases
func ensureIngressWildcardCertificate(ctx context.Context, c client.Client, baseDomain string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("wildcard-cert")

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cert-manager.io",
		Version: "v1",
		Kind:    "Certificate",
	})
	obj.SetNamespace(CertificatesNamespace)
	obj.SetName(IngressWildcardCertName)

	if err := c.Get(ctx, client.ObjectKey{
		Namespace: CertificatesNamespace,
		Name:      IngressWildcardCertName,
	}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating wildcard certificate", "name", IngressWildcardCertName)

		// Create wildcard certificate covering all application subdomains
		obj.Object["spec"] = map[string]any{
			"secretName": IngressWildcardCertName,
			"issuerRef": map[string]any{
				"name": IngressIssuerName,
				"kind": "ClusterIssuer",
			},
			"dnsNames": []any{
				fmt.Sprintf("*.apps.%s", baseDomain),
				fmt.Sprintf("*.valkey.%s", baseDomain),
				fmt.Sprintf("*.mysql.%s", baseDomain),
				fmt.Sprintf("*.postgres.%s", baseDomain),
			},
		}

		if err := c.Create(ctx, obj); err != nil {
			log.Error(err, "Failed to create wildcard certificate")
			return err
		}

		log.Info("Wildcard certificate created successfully", "name", IngressWildcardCertName)
	} else {
		log.Info("Wildcard certificate already exists", "name", IngressWildcardCertName)
	}

	return nil
}

// isWildcardCertificateReady checks if the wildcard certificate exists and is ready.
// Returns true only if the certificate Secret exists and has the tls.crt key.
func isWildcardCertificateReady(ctx context.Context, c client.Client) (bool, error) {
	log := ctrl.Log.WithName("bootstrap").WithName("cert-check")

	secret := &corev1.Secret{}
	err := c.Get(ctx, client.ObjectKey{
		Namespace: CertificatesNamespace,
		Name:      IngressWildcardCertName,
	}, secret)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Wildcard certificate secret not found yet", "name", IngressWildcardCertName)
			return false, nil
		}
		return false, err
	}

	// Check if the secret has the tls.crt key (indicates certificate is ready)
	if _, ok := secret.Data["tls.crt"]; !ok {
		log.Info("Wildcard certificate secret exists but tls.crt key not ready yet", "name", IngressWildcardCertName)
		return false, nil
	}

	log.Info("Wildcard certificate is ready", "name", IngressWildcardCertName)
	return true, nil
}

// ensureIngressGateway creates the Gateway API Gateway resource in the kibaship namespace.
// The Gateway has 5 listeners for different protocols:
//   - HTTP (port 80) - HTTP traffic, typically redirects to HTTPS
//   - HTTPS (port 443) - HTTPS traffic with TLS termination
//   - MySQL TLS (port 3306) - MySQL traffic with TLS passthrough (SNI-based routing)
//   - Valkey TLS (port 6379) - Valkey/Redis traffic with TLS passthrough (SNI-based routing)
//   - PostgreSQL TLS (port 5432) - PostgreSQL traffic with TLS passthrough (SNI-based routing)
func ensureIngressGateway(ctx context.Context, c client.Client) error {
	log := ctrl.Log.WithName("bootstrap").WithName("gateway")

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	})
	obj.SetNamespace(KibashipNamespace)
	obj.SetName(IngressGatewayName)

	if err := c.Get(ctx, client.ObjectKey{
		Namespace: KibashipNamespace,
		Name:      IngressGatewayName,
	}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating Gateway resource", "name", IngressGatewayName, "namespace", KibashipNamespace)

		// Create Gateway with multi-protocol listeners
		obj.Object["spec"] = map[string]any{
			"gatewayClassName": "cilium",
			"listeners": []any{
				// HTTP listener (port 80)
				map[string]any{
					"name":     "http",
					"protocol": "HTTP",
					"port":     int64(80),
					"allowedRoutes": map[string]any{
						"namespaces": map[string]any{"from": "All"},
					},
				},
				// HTTPS listener (port 443) with TLS termination
				map[string]any{
					"name":     "https",
					"protocol": "HTTPS",
					"port":     int64(443),
					"tls": map[string]any{
						"mode": "Terminate",
						"certificateRefs": []any{
							map[string]any{
								"name":      IngressWildcardCertName,
								"namespace": CertificatesNamespace,
								"kind":      "Secret",
							},
						},
					},
					"allowedRoutes": map[string]any{
						"namespaces": map[string]any{"from": "All"},
					},
				},
				// MySQL TLS listener (port 3306) with TLS passthrough
				map[string]any{
					"name":     "mysql-tls",
					"protocol": "TLS",
					"port":     int64(3306),
					"tls":      map[string]any{"mode": "Passthrough"},
					"allowedRoutes": map[string]any{
						"namespaces": map[string]any{"from": "All"},
						"kinds": []any{
							map[string]any{"kind": "TLSRoute"},
						},
					},
				},
				// Valkey TLS listener (port 6379) with TLS passthrough
				map[string]any{
					"name":     "valkey-tls",
					"protocol": "TLS",
					"port":     int64(6379),
					"tls":      map[string]any{"mode": "Passthrough"},
					"allowedRoutes": map[string]any{
						"namespaces": map[string]any{"from": "All"},
						"kinds": []any{
							map[string]any{"kind": "TLSRoute"},
						},
					},
				},
				// PostgreSQL TLS listener (port 5432) with TLS passthrough
				map[string]any{
					"name":     "postgres-tls",
					"protocol": "TLS",
					"port":     int64(5432),
					"tls":      map[string]any{"mode": "Passthrough"},
					"allowedRoutes": map[string]any{
						"namespaces": map[string]any{"from": "All"},
						"kinds": []any{
							map[string]any{"kind": "TLSRoute"},
						},
					},
				},
			},
		}

		if err := c.Create(ctx, obj); err != nil {
			log.Error(err, "Failed to create Gateway resource")
			return err
		}

		log.Info("Gateway resource created successfully", "name", IngressGatewayName, "namespace", KibashipNamespace)
	} else {
		log.Info("Gateway resource already exists", "name", IngressGatewayName, "namespace", KibashipNamespace)
	}

	return nil
}

// ensureIngressReferenceGrant creates the ReferenceGrant in the kibaship namespace.
// This allows the Gateway (in kibaship namespace) to reference the wildcard certificate
// Secret in the certificates namespace.
func ensureIngressReferenceGrant(ctx context.Context, c client.Client) error {
	log := ctrl.Log.WithName("bootstrap").WithName("referencegrant")

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1beta1",
		Kind:    "ReferenceGrant",
	})
	obj.SetNamespace(CertificatesNamespace)
	obj.SetName(IngressReferenceGrantName)

	if err := c.Get(ctx, client.ObjectKey{
		Namespace: CertificatesNamespace,
		Name:      IngressReferenceGrantName,
	}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating ReferenceGrant", "name", IngressReferenceGrantName, "namespace", CertificatesNamespace)

		obj.Object["spec"] = map[string]any{
			"from": []any{
				map[string]any{
					"group":     "gateway.networking.k8s.io",
					"kind":      "Gateway",
					"namespace": KibashipNamespace,
				},
			},
			"to": []any{
				map[string]any{
					"group": "",
					"kind":  "Secret",
				},
			},
		}

		if err := c.Create(ctx, obj); err != nil {
			log.Error(err, "Failed to create ReferenceGrant")
			return err
		}

		log.Info("ReferenceGrant created successfully", "name", IngressReferenceGrantName, "namespace", CertificatesNamespace)
	} else {
		log.Info("ReferenceGrant already exists", "name", IngressReferenceGrantName, "namespace", CertificatesNamespace)
	}

	return nil
}
