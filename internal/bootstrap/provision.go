package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/kibamail/kibaship-operator/pkg/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	certificatesNS   = "certificates"
	ingressGatewayNS = "ingress-gateway"
	issuerName       = "certmanager-acme-issuer"
	wildcardCertName = "tenant-wildcard-certificate"
	deploymentNFName = "deployment-not-found"
	gatewayName      = "ingress-gateway"
)

// EnsureStorageClasses creates Longhorn-backed StorageClasses used by the platform.
// It is safe to call repeatedly; it only creates them if missing.
func EnsureStorageClasses(ctx context.Context, c client.Client) error {
	if err := ensureStorageClass(ctx, c, config.StorageClassReplica1, "1"); err != nil {
		return err
	}
	if err := ensureStorageClass(ctx, c, config.StorageClassReplica2, "2"); err != nil {
		return err
	}
	return nil
}

func ensureStorageClass(ctx context.Context, c client.Client, name, replicas string) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "storage.k8s.io", Version: "v1", Kind: "StorageClass"})
	obj.SetName(name)

	if err := c.Get(ctx, client.ObjectKey{Name: name}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		obj.Object["provisioner"] = "driver.longhorn.io"
		obj.Object["allowVolumeExpansion"] = true
		obj.Object["volumeBindingMode"] = "WaitForFirstConsumer"
		obj.Object["parameters"] = map[string]any{
			"numberOfReplicas": replicas,
		}
		return c.Create(ctx, obj)
	}
	return nil
}

// ProvisionIngressAndCertificates ensures dynamic resources are present based on configured domain/email.
// It is idempotent and safe to call on every manager start.
func ProvisionIngressAndCertificates(ctx context.Context, c client.Client, baseDomain, acmeEmail string) error {
	if baseDomain == "" {
		return nil // nothing to do without a domain
	}

	// 1) Ensure namespaces exist (in case installer omitted them)
	for _, ns := range []string{certificatesNS, ingressGatewayNS} {
		if err := ensureNamespace(ctx, c, ns); err != nil {
			return fmt.Errorf("ensure namespace %s: %w", ns, err)
		}
	}

	// 2) ClusterIssuer (requires cert-manager CRDs to exist)
	if acmeEmail != "" {
		if err := ensureClusterIssuer(ctx, c, acmeEmail); err != nil {
			return fmt.Errorf("ensure ClusterIssuer: %w", err)
		}
	}

	// 3) Wildcard Certificate (depends on issuer and domain)
	if acmeEmail != "" { // only create certificate if issuer/email provided
		if err := ensureWildcardCertificate(ctx, c, baseDomain); err != nil {
			return fmt.Errorf("ensure wildcard Certificate: %w", err)
		}
	}

	// 4) HTTPRoutes with hostnames derived from domain
	if err := ensureHTTPRedirectRoute(ctx, c, baseDomain); err != nil {
		return fmt.Errorf("ensure HTTP redirect HTTPRoute: %w", err)
	}
	if err := ensureHTTPSRoute(ctx, c, baseDomain); err != nil {
		return fmt.Errorf("ensure HTTPS HTTPRoute: %w", err)
	}
	if err := ensureDeploymentNotFoundRoute(ctx, c, baseDomain); err != nil {
		return fmt.Errorf("ensure 404 HTTPRoute: %w", err)
	}

	// 5) Ensure static app (Deployment/Service) exists in case not installed via manifests
	if err := ensureDeploymentNotFoundWorkload(ctx, c); err != nil {
		return fmt.Errorf("ensure 404 workload: %w", err)
	}

	return nil
}

func ensureNamespace(ctx context.Context, c client.Client, name string) error {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := c.Get(ctx, client.ObjectKey{Name: name}, ns); err != nil {
		if errors.IsNotFound(err) {
			return c.Create(ctx, ns)
		}
		return err
	}
	return nil
}

func ensureClusterIssuer(ctx context.Context, c client.Client, email string) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "ClusterIssuer"})
	obj.SetName(issuerName)

	if err := c.Get(ctx, client.ObjectKey{Name: issuerName}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		obj.Object["spec"] = map[string]any{
			"acme": map[string]any{
				"email":               email,
				"server":              "https://acme-v02.api.letsencrypt.org/directory",
				"privateKeySecretRef": map[string]any{"name": "acme-certificates-private-key"},
				"solvers": []any{
					map[string]any{
						"dns01": map[string]any{
							"webhook": map[string]any{
								"groupName":  "dns.kibaship.com",
								"solverName": "kibaship",
							},
						},
					},
				},
			},
		}
		return c.Create(ctx, obj)
	}
	return nil
}

func ensureWildcardCertificate(ctx context.Context, c client.Client, baseDomain string) error {
	name := wildcardCertName
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"})
	obj.SetNamespace(certificatesNS)
	obj.SetName(name)

	if err := c.Get(ctx, client.ObjectKey{Namespace: certificatesNS, Name: name}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		obj.Object["spec"] = map[string]any{
			"secretName": wildcardCertName,
			"issuerRef":  map[string]any{"name": issuerName, "kind": "ClusterIssuer"},
			"dnsNames":   []any{fmt.Sprintf("*.%s", baseDomain)},
		}
		return c.Create(ctx, obj)
	}
	return nil
}

func ensureHTTPRedirectRoute(ctx context.Context, c client.Client, baseDomain string) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRoute"})
	obj.SetNamespace(ingressGatewayNS)
	obj.SetName("ingress")
	if err := c.Get(ctx, client.ObjectKey{Namespace: ingressGatewayNS, Name: "ingress"}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		obj.Object["spec"] = map[string]any{
			"parentRefs": []any{map[string]any{"name": gatewayName, "sectionName": "http"}},
			"hostnames":  []any{fmt.Sprintf("*.%s", baseDomain)},
			"rules": []any{
				map[string]any{
					"filters": []any{map[string]any{
						"type":            "RequestRedirect",
						"requestRedirect": map[string]any{"scheme": "https", "statusCode": int64(301)},
					}},
				},
			},
		}
		return c.Create(ctx, obj)
	}
	return nil
}

func ensureHTTPSRoute(ctx context.Context, c client.Client, baseDomain string) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRoute"})
	obj.SetNamespace(ingressGatewayNS)
	obj.SetName("ingress-https")
	if err := c.Get(ctx, client.ObjectKey{Namespace: ingressGatewayNS, Name: "ingress-https"}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		obj.Object["spec"] = map[string]any{
			"parentRefs": []any{map[string]any{"name": gatewayName, "sectionName": "https"}},
			"hostnames":  []any{fmt.Sprintf("*.%s", baseDomain)},
		}
		return c.Create(ctx, obj)
	}
	return nil
}

func ensureDeploymentNotFoundRoute(ctx context.Context, c client.Client, baseDomain string) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRoute"})
	obj.SetNamespace(ingressGatewayNS)
	obj.SetName(deploymentNFName)
	if err := c.Get(ctx, client.ObjectKey{Namespace: ingressGatewayNS, Name: deploymentNFName}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		obj.Object["spec"] = map[string]any{
			"parentRefs": []any{map[string]any{"name": gatewayName, "namespace": ingressGatewayNS}},
			"hostnames":  []any{fmt.Sprintf("*.%s", baseDomain)},
			"rules": []any{
				map[string]any{
					"matches":     []any{map[string]any{"path": map[string]any{"type": "PathPrefix", "value": "/"}}},
					"backendRefs": []any{map[string]any{"name": deploymentNFName, "namespace": ingressGatewayNS, "port": int64(80)}},
				},
			},
		}
		return c.Create(ctx, obj)
	}
	return nil
}

func ensureDeploymentNotFoundWorkload(ctx context.Context, c client.Client) error {
	// Deployment
	dep := &appsv1.Deployment{}
	key := client.ObjectKey{Namespace: ingressGatewayNS, Name: deploymentNFName}
	if err := c.Get(ctx, key, dep); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		dep = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: deploymentNFName, Namespace: ingressGatewayNS},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(3),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": deploymentNFName}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": deploymentNFName}},
					Spec: corev1.PodSpec{Containers: []corev1.Container{{
						Name:      deploymentNFName,
						Image:     "ghcr.io/kibamail/kibaship-404-deployment-not-found:latest",
						Ports:     []corev1.ContainerPort{{ContainerPort: 3000}},
						Resources: corev1.ResourceRequirements{},
					}}},
				},
			},
		}
		if err := c.Create(ctx, dep); err != nil {
			return err
		}
	}
	// Service
	svc := &corev1.Service{}
	if err := c.Get(ctx, key, svc); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		svc = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: deploymentNFName, Namespace: ingressGatewayNS},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app": deploymentNFName},
				Ports:    []corev1.ServicePort{{Port: 80, TargetPort: intstr.FromInt(3000)}},
			},
		}
		if err := c.Create(ctx, svc); err != nil {
			return err
		}
	}
	return nil
}

func int32Ptr(i int32) *int32 { return &i }

// EnsureRegistryCredentials provisions the registry-registry-auth secret in the registry namespace.
// This secret contains a randomly generated HTTP secret for the Docker registry.
// The init container in the registry deployment waits for this secret before starting.
func EnsureRegistryCredentials(ctx context.Context, c client.Client) error {
	const (
		registryNS    = "registry"
		secretName    = "registry-registry-auth"
		httpSecretKey = "http-secret"
	)

	// Wait for registry namespace to exist (up to 5 minutes)
	ns := &corev1.Namespace{}
	deadline := time.Now().Add(5 * time.Minute)
	for {
		err := c.Get(ctx, client.ObjectKey{Name: registryNS}, ns)
		if err == nil {
			// Namespace exists, proceed
			break
		}
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check registry namespace: %w", err)
		}

		// Namespace not found, check if we've exceeded deadline
		if time.Now().After(deadline) {
			return fmt.Errorf("registry namespace did not appear within 5 minutes, operator cannot start")
		}

		// Wait 5 seconds before retrying
		time.Sleep(5 * time.Second)
	}

	// Check if secret already exists
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{Namespace: registryNS, Name: secretName}
	if err := c.Get(ctx, secretKey, secret); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check registry secret: %w", err)
		}

		// Generate random HTTP secret (32 bytes, base64 encoded)
		httpSecretBytes := make([]byte, 32)
		if _, err := rand.Read(httpSecretBytes); err != nil {
			return fmt.Errorf("failed to generate http secret: %w", err)
		}
		httpSecret := base64.StdEncoding.EncodeToString(httpSecretBytes)

		// Create secret
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: registryNS,
				Labels: map[string]string{
					"app":                          "registry",
					"app.kubernetes.io/managed-by": "kibaship-operator",
				},
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				httpSecretKey: httpSecret,
			},
		}

		if err := c.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create registry secret: %w", err)
		}
	}

	return nil
}

// EnsureRegistryJWKS provisions the registry-auth-keys-jwks secret in the registry namespace.
// This secret contains the JWKS (JSON Web Key Set) file required by Docker Registry v3.0.0
// for JWT token validation. The JWKS is generated from the registry-auth-keys certificate.
func EnsureRegistryJWKS(ctx context.Context, c client.Client) error {
	const (
		registryNS     = "registry"
		certSecretName = "registry-auth-keys"
		jwksSecretName = "registry-auth-keys-jwks"
		jwksKeyID      = "registry-auth-jwt-signer"
	)

	// Wait for registry namespace to exist (up to 5 minutes)
	ns := &corev1.Namespace{}
	deadline := time.Now().Add(5 * time.Minute)
	for {
		err := c.Get(ctx, client.ObjectKey{Name: registryNS}, ns)
		if err == nil {
			break
		}
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check registry namespace: %w", err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("registry namespace did not appear within 5 minutes, operator cannot start")
		}
		time.Sleep(5 * time.Second)
	}

	// Check if JWKS secret already exists
	jwksSecret := &corev1.Secret{}
	jwksSecretKey := client.ObjectKey{Namespace: registryNS, Name: jwksSecretName}
	if err := c.Get(ctx, jwksSecretKey, jwksSecret); err == nil {
		// Secret already exists, nothing to do
		return nil
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check jwks secret: %w", err)
	}

	// Wait for registry-auth-keys certificate to be ready (up to 5 minutes)
	certSecret := &corev1.Secret{}
	certSecretKey := client.ObjectKey{Namespace: registryNS, Name: certSecretName}
	deadline = time.Now().Add(5 * time.Minute)
	for {
		err := c.Get(ctx, certSecretKey, certSecret)
		if err == nil {
			// Secret exists, check if it has the tls.crt key
			if _, ok := certSecret.Data["tls.crt"]; ok {
				break
			}
		}
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check certificate secret: %w", err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("registry-auth-keys certificate did not become ready within 5 minutes")
		}
		time.Sleep(5 * time.Second)
	}

	// Extract public key from certificate
	certPEM := certSecret.Data["tls.crt"]
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block from certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	rsaPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificate does not contain RSA public key")
	}

	// Generate JWKS JSON
	jwksJSON, err := generateJWKS(rsaPubKey, jwksKeyID)
	if err != nil {
		return fmt.Errorf("failed to generate JWKS: %w", err)
	}

	// Create JWKS secret
	jwksSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jwksSecretName,
			Namespace: registryNS,
			Labels: map[string]string{
				"app":                          "registry",
				"app.kubernetes.io/managed-by": "kibaship-operator",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"jwks.json": jwksJSON,
		},
	}

	if err := c.Create(ctx, jwksSecret); err != nil {
		return fmt.Errorf("failed to create jwks secret: %w", err)
	}

	return nil
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"` // Key type (RSA)
	Kid string `json:"kid"` // Key ID
	Use string `json:"use"` // Public key use (sig = signature)
	Alg string `json:"alg"` // Algorithm (RS256)
	N   string `json:"n"`   // RSA modulus (base64url-encoded)
	E   string `json:"e"`   // RSA exponent (base64url-encoded)
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// generateJWKS creates a JWKS JSON from an RSA public key
func generateJWKS(pubKey *rsa.PublicKey, keyID string) ([]byte, error) {
	// Encode modulus as base64url (unpadded)
	nBytes := pubKey.N.Bytes()
	n := base64.RawURLEncoding.EncodeToString(nBytes)

	// Encode exponent as base64url (unpadded)
	eBytes := big.NewInt(int64(pubKey.E)).Bytes()
	e := base64.RawURLEncoding.EncodeToString(eBytes)

	jwks := JWKS{
		Keys: []JWK{
			{
				Kty: "RSA",
				Kid: keyID,
				Use: "sig",
				Alg: "RS256",
				N:   n,
				E:   e,
			},
		},
	}

	return json.MarshalIndent(jwks, "", "  ")
}

// EnsureRegistryCACertificateInBuildkit copies the registry CA certificate to the buildkit namespace.
// This allows BuildKit daemon to trust the self-signed registry certificate when pushing images.
func EnsureRegistryCACertificateInBuildkit(ctx context.Context, c client.Client) error {
	const (
		registryNS     = "registry"
		buildkitNS     = "buildkit"
		registrySecret = "registry-tls"
		buildkitSecret = "registry-ca-cert"
	)

	// Wait for registry namespace to exist (up to 5 minutes)
	ns := &corev1.Namespace{}
	deadline := time.Now().Add(5 * time.Minute)
	for {
		err := c.Get(ctx, client.ObjectKey{Name: registryNS}, ns)
		if err == nil {
			break
		}
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check registry namespace: %w", err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("registry namespace did not appear within 5 minutes, operator cannot start")
		}
		time.Sleep(5 * time.Second)
	}

	// Wait for buildkit namespace to exist (up to 5 minutes)
	deadline = time.Now().Add(5 * time.Minute)
	for {
		err := c.Get(ctx, client.ObjectKey{Name: buildkitNS}, ns)
		if err == nil {
			break
		}
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check buildkit namespace: %w", err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("buildkit namespace did not appear within 5 minutes, operator cannot start")
		}
		time.Sleep(5 * time.Second)
	}

	// Check if CA certificate secret already exists in buildkit namespace
	existingSecret := &corev1.Secret{}
	buildkitSecretKey := client.ObjectKey{Namespace: buildkitNS, Name: buildkitSecret}
	if err := c.Get(ctx, buildkitSecretKey, existingSecret); err == nil {
		// Secret already exists, nothing to do
		return nil
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check buildkit CA secret: %w", err)
	}

	// Wait for registry-tls secret to exist (up to 5 minutes)
	registryTLSSecret := &corev1.Secret{}
	registrySecretKey := client.ObjectKey{Namespace: registryNS, Name: registrySecret}
	deadline = time.Now().Add(5 * time.Minute)
	for {
		err := c.Get(ctx, registrySecretKey, registryTLSSecret)
		if err == nil {
			// Secret exists, check if it has the ca.crt key
			if _, ok := registryTLSSecret.Data["ca.crt"]; ok {
				break
			}
		}
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check registry-tls secret: %w", err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("registry-tls secret did not become ready within 5 minutes")
		}
		time.Sleep(5 * time.Second)
	}

	// Extract CA certificate
	caCert, ok := registryTLSSecret.Data["ca.crt"]
	if !ok {
		return fmt.Errorf("registry-tls secret does not contain ca.crt")
	}

	// Create CA certificate secret in buildkit namespace
	caSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildkitSecret,
			Namespace: buildkitNS,
			Labels: map[string]string{
				"app":                          "buildkitd",
				"app.kubernetes.io/managed-by": "kibaship-operator",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"ca.crt": caCert,
		},
	}

	if err := c.Create(ctx, caSecret); err != nil {
		return fmt.Errorf("failed to create buildkit CA secret: %w", err)
	}

	// Restart buildkit deployment to pick up the new secret
	// We trigger a rollout restart by updating the deployment's restart annotation
	buildkitDeployment := &appsv1.Deployment{}
	deploymentKey := client.ObjectKey{Namespace: buildkitNS, Name: "buildkitd"}
	if err := c.Get(ctx, deploymentKey, buildkitDeployment); err != nil {
		if errors.IsNotFound(err) {
			// Deployment doesn't exist yet, skip restart
			return nil
		}
		return fmt.Errorf("failed to get buildkit deployment: %w", err)
	}

	// Add/update restart annotation to trigger rollout
	if buildkitDeployment.Spec.Template.Annotations == nil {
		buildkitDeployment.Spec.Template.Annotations = make(map[string]string)
	}
	buildkitDeployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	if err := c.Update(ctx, buildkitDeployment); err != nil {
		return fmt.Errorf("failed to restart buildkit deployment: %w", err)
	}

	return nil
}
