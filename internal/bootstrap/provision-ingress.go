package bootstrap

import (
	"context"
	"fmt"

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

	// CertificatesNamespace is where TLS certificates are stored (same as operator namespace)
	CertificatesNamespace = "kibaship"

	// IngressIssuerName is the cert-manager ClusterIssuer for ACME certificates
	IngressIssuerName = "certmanager-acme-issuer"

	// IngressWildcardCertName is the wildcard certificate for all application domains
	IngressWildcardCertName = "ingress-kibaship-certificate"

	// IngressGatewayName is the Gateway API Gateway resource name
	IngressGatewayName = "ingress-kibaship-gateway"
)

// ProvisionIngress ensures ingress resources are present based on configured domain/email.
// It is idempotent and safe to call on every manager start.
//
// Prerequisites:
//   - kibaship namespace must exist (operator namespace)
//   - cert-manager must be installed and ready
//   - Gateway API support must be enabled (e.g., Cilium, Istio, etc.)
//
// Resources created (in order):
//  1. Wildcard Certificate for web app domains (*.apps.{domain} only)
//  2. Gateway resource with certificate-based listener provisioning:
//     - If certificate not ready: HTTP and DNS listeners only (for ACME challenges)
//     - If certificate ready: All listeners (HTTP, HTTPS, MySQL, Valkey, PostgreSQL, DNS)
//  3. ACME-DNS routes (HTTPRoute and UDPRoute) when certificate not ready
//
// Note: Database certificates (*.valkey, *.mysql, *.postgres) will be provisioned separately
func ProvisionIngress(ctx context.Context, c client.Client, baseDomain, acmeEmail, gatewayClassName string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("ingress")

	if baseDomain == "" {
		log.Info("No base domain configured, skipping ingress provisioning")
		return nil
	}

	// 1. Ensure kibaship namespace exists (should already exist, but check anyway)
	log.Info("Step 1: Ensuring kibaship namespace exists")
	if err := ensureNamespace(ctx, c, KibashipNamespace); err != nil {
		return fmt.Errorf("ensure kibaship namespace: %w", err)
	}

	// 2. Wildcard Certificate for all app domains (depends on issuer and domain)
	if acmeEmail != "" {
		log.Info("Step 2: Ensuring wildcard certificate for all app domains")
		if err := ensureIngressWildcardCertificate(ctx, c, baseDomain); err != nil {
			return fmt.Errorf("ensure wildcard certificate: %w", err)
		}
	}

	// 3. Gateway resource creation (handles certificate readiness internally)
	log.Info("Step 3: Proceeding with Gateway resource creation")

	// 3. Gateway resource with multi-protocol listeners (in kibaship namespace)
	log.Info("Step 3: Ensuring Gateway resource")
	if err := ensureIngressGateway(ctx, c, gatewayClassName, baseDomain); err != nil {
		return fmt.Errorf("ensure Gateway: %w", err)
	}

	log.Info("Ingress provisioning completed successfully")
	return nil
}

// ensureIngressWildcardCertificate creates the wildcard certificate for web application domains.
// The certificate covers:
//   - *.apps.{domain} - Web applications (GitRepository, DockerImage)
//
// Note: Database certificates (*.valkey, *.mysql, *.postgres) will be requested separately
func ensureIngressWildcardCertificate(ctx context.Context, c client.Client, baseDomain string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("wildcard-cert")

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cert-manager.io",
		Version: "v1",
		Kind:    "Certificate",
	})
	obj.SetNamespace(KibashipNamespace)
	obj.SetName(IngressWildcardCertName)

	if err := c.Get(ctx, client.ObjectKey{
		Namespace: KibashipNamespace,
		Name:      IngressWildcardCertName,
	}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating wildcard certificate", "name", IngressWildcardCertName)

		// Create wildcard certificate covering web application subdomain only
		obj.Object["spec"] = map[string]any{
			"secretName": IngressWildcardCertName,
			"issuerRef": map[string]any{
				"name": IngressIssuerName,
				"kind": "ClusterIssuer",
			},
			"dnsNames": []any{
				fmt.Sprintf("*.apps.%s", baseDomain),
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
// Returns true only if the Certificate resource has Ready=True condition.
func isWildcardCertificateReady(ctx context.Context, c client.Client) (bool, error) {
	log := ctrl.Log.WithName("bootstrap").WithName("cert-check")

	// Check Certificate resource status instead of Secret
	cert := &unstructured.Unstructured{}
	cert.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cert-manager.io",
		Version: "v1",
		Kind:    "Certificate",
	})

	err := c.Get(ctx, client.ObjectKey{
		Namespace: KibashipNamespace,
		Name:      IngressWildcardCertName,
	}, cert)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Wildcard certificate not found yet", "name", IngressWildcardCertName)
			return false, nil
		}
		return false, err
	}

	// Check Certificate status conditions for Ready=True
	status, found, err := unstructured.NestedMap(cert.Object, "status")
	if err != nil || !found {
		log.Info("Certificate status not available yet", "name", IngressWildcardCertName)
		return false, nil
	}

	conditions, found, err := unstructured.NestedSlice(status, "conditions")
	if err != nil || !found {
		log.Info("Certificate conditions not available yet", "name", IngressWildcardCertName)
		return false, nil
	}

	// Look for Ready=True condition
	for _, condition := range conditions {
		if condMap, ok := condition.(map[string]interface{}); ok {
			if condType, ok := condMap["type"].(string); ok && condType == "Ready" {
				if condStatus, ok := condMap["status"].(string); ok && condStatus == "True" {
					log.Info("Wildcard certificate is ready", "name", IngressWildcardCertName)
					return true, nil
				}
			}
		}
	}

	log.Info("Wildcard certificate not ready yet", "name", IngressWildcardCertName)
	return false, nil
}

// ensureIngressGateway creates the Gateway API Gateway resource in the kibaship namespace.
// It implements certificate-based provisioning logic:
//   - If wildcard certificate doesn't exist: creates gateway with HTTP and DNS listeners only
//   - If wildcard certificate exists: creates gateway with all listeners (HTTP, HTTPS, MySQL, Valkey, PostgreSQL, DNS)
//   - If gateway exists but missing listeners: patches gateway to add missing listeners
func ensureIngressGateway(ctx context.Context, c client.Client, gatewayClassName, baseDomain string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("gateway")

	// Check if wildcard certificate is ready
	certReady, err := isWildcardCertificateReady(ctx, c)
	if err != nil {
		return fmt.Errorf("check wildcard certificate: %w", err)
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	})
	obj.SetNamespace(KibashipNamespace)
	obj.SetName(IngressGatewayName)

	// Set annotations for LoadBalancer service configuration
	// These annotations will be propagated to the LoadBalancer service
	// created by the Gateway API implementation (Cilium/Istio)
	obj.SetAnnotations(map[string]string{
		// DigitalOcean LoadBalancer annotations
		// Ref: https://docs.digitalocean.com/products/kubernetes/how-to/configure-load-balancers/
		"service.beta.kubernetes.io/do-loadbalancer-tls-passthrough": "true",

		// AWS LoadBalancer annotations
		// Ref: https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.3/guide/service/annotations/
		"service.beta.kubernetes.io/aws-load-balancer-backend-protocol": "tcp",

		// Azure LoadBalancer annotations
		// Ref: https://learn.microsoft.com/en-us/azure/aks/load-balancer-standard
		"service.beta.kubernetes.io/azure-load-balancer-tcp-idle-timeout": "4",
	})

	// Try to get existing Gateway
	existingGateway := &unstructured.Unstructured{}
	existingGateway.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())

	err = c.Get(ctx, client.ObjectKey{
		Namespace: KibashipNamespace,
		Name:      IngressGatewayName,
	}, existingGateway)

	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		// Gateway doesn't exist, create it based on certificate status
		log.Info("Creating Gateway resource", "name", IngressGatewayName, "namespace", KibashipNamespace, "certReady", certReady)

		var listeners []any
		if certReady {
			// Certificate is ready, create gateway with HTTP and HTTPS listeners
			listeners = createHTTPAndHTTPSListeners()
			log.Info("Creating Gateway with HTTP and HTTPS listeners (certificate ready)")
		} else {
			// Certificate not ready, create gateway with HTTP listener only
			listeners = createHTTPOnlyListeners()
			log.Info("Creating Gateway with HTTP listener only (certificate not ready)")
		}

		obj.Object["spec"] = map[string]any{
			"gatewayClassName": gatewayClassName,
			"listeners":        listeners,
		}

		if err := c.Create(ctx, obj); err != nil {
			log.Error(err, "Failed to create Gateway resource")
			return err
		}

		log.Info("Gateway resource created successfully", "name", IngressGatewayName, "namespace", KibashipNamespace)

		// Create routes for ACME-DNS if certificate is not ready
		if !certReady && baseDomain != "" {
			if err := ensureAcmeDNSRoutes(ctx, c, baseDomain); err != nil {
				log.Error(err, "Failed to create ACME-DNS routes")
				return fmt.Errorf("ensure ACME-DNS routes: %w", err)
			}
		}
	} else {
		// Gateway exists, check if it needs to be updated
		log.Info("Gateway resource already exists", "name", IngressGatewayName, "namespace", KibashipNamespace)

		if certReady {
			// Certificate is ready, ensure gateway has HTTPS listener
			if err := ensureGatewayHasHTTPSListener(ctx, c, existingGateway); err != nil {
				return fmt.Errorf("ensure gateway has HTTPS listener: %w", err)
			}
		}
	}

	return nil
}

// createHTTPOnlyListeners creates listeners for when certificate is not ready (HTTP only)
func createHTTPOnlyListeners() []any {
	return []any{
		// HTTP listener (port 80) - for ACME HTTP-01 challenges and web applications
		map[string]any{
			"name":     "http",
			"protocol": "HTTP",
			"port":     int64(80),
			"allowedRoutes": map[string]any{
				"namespaces": map[string]any{"from": "All"},
			},
		},
	}
}

// createHTTPAndHTTPSListeners creates listeners for when certificate is ready (HTTP + HTTPS)
func createHTTPAndHTTPSListeners() []any {
	return []any{
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
						"name": IngressWildcardCertName,
					},
				},
			},
			"allowedRoutes": map[string]any{
				"namespaces": map[string]any{"from": "All"},
			},
		},
	}
}

// ensureGatewayHasHTTPSListener checks if the gateway has HTTPS listener and patches it if needed
func ensureGatewayHasHTTPSListener(ctx context.Context, c client.Client, gateway *unstructured.Unstructured) error {
	log := ctrl.Log.WithName("bootstrap").WithName("gateway-patch")

	// Get current listeners
	spec, found, err := unstructured.NestedMap(gateway.Object, "spec")
	if err != nil || !found {
		return fmt.Errorf("failed to get gateway spec: %w", err)
	}

	currentListeners, found, err := unstructured.NestedSlice(spec, "listeners")
	if err != nil || !found {
		return fmt.Errorf("failed to get current listeners: %w", err)
	}

	// Check if we have HTTPS listener
	hasHTTPSListener := false
	for _, listener := range currentListeners {
		if listenerMap, ok := listener.(map[string]any); ok {
			if name, ok := listenerMap["name"].(string); ok && name == "https" {
				hasHTTPSListener = true
				break
			}
		}
	}

	if hasHTTPSListener {
		log.Info("Gateway already has HTTPS listener")
		return nil
	}

	// Patch gateway with HTTP and HTTPS listeners
	log.Info("Patching Gateway to add HTTPS listener")
	requiredListeners := createHTTPAndHTTPSListeners()

	if err := unstructured.SetNestedSlice(gateway.Object, requiredListeners, "spec", "listeners"); err != nil {
		return fmt.Errorf("failed to set listeners: %w", err)
	}

	if err := c.Update(ctx, gateway); err != nil {
		log.Error(err, "Failed to update Gateway with HTTPS listener")
		return err
	}

	log.Info("Gateway patched successfully with HTTPS listener")
	return nil
}

// ensureAcmeDNSRoutes creates HTTPRoute for ACME-DNS service
func ensureAcmeDNSRoutes(ctx context.Context, c client.Client, baseDomain string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns-routes")

	acmeDomain := fmt.Sprintf("acme.%s", baseDomain)

	// Create HTTPRoute for ACME-DNS API
	if err := ensureAcmeDNSHTTPRoute(ctx, c, acmeDomain); err != nil {
		return fmt.Errorf("ensure ACME-DNS HTTPRoute: %w", err)
	}

	log.Info("ACME-DNS routes created successfully")
	return nil
}

// ensureAcmeDNSHTTPRoute creates HTTPRoute for ACME-DNS HTTP API
func ensureAcmeDNSHTTPRoute(ctx context.Context, c client.Client, acmeDomain string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns-httproute")

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	})
	obj.SetNamespace(KibashipNamespace)
	obj.SetName("acme-dns-api")

	if err := c.Get(ctx, client.ObjectKey{
		Namespace: KibashipNamespace,
		Name:      "acme-dns-api",
	}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating ACME-DNS HTTPRoute", "domain", acmeDomain)

		obj.Object["spec"] = map[string]any{
			"parentRefs": []any{
				map[string]any{
					"name":        IngressGatewayName,
					"namespace":   KibashipNamespace,
					"sectionName": "http",
				},
			},
			"hostnames": []any{acmeDomain},
			"rules": []any{
				map[string]any{
					"matches": []any{
						map[string]any{
							"path": map[string]any{
								"type":  "PathPrefix",
								"value": "/",
							},
						},
					},
					"backendRefs": []any{
						map[string]any{
							"name":      "acme-dns",
							"namespace": KibashipNamespace,
							"port":      int64(80),
						},
					},
				},
			},
		}

		if err := c.Create(ctx, obj); err != nil {
			log.Error(err, "Failed to create ACME-DNS HTTPRoute")
			return err
		}

		log.Info("ACME-DNS HTTPRoute created successfully")
	} else {
		log.Info("ACME-DNS HTTPRoute already exists")
	}

	return nil
}
