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

	"github.com/kibamail/kibaship/pkg/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// issuerName is the cert-manager ClusterIssuer for ACME certificates
	issuerName = "certmanager-acme-issuer"
)

// EnsureStorageClasses creates Longhorn-backed StorageClasses used by the platform.
// It is safe to call repeatedly; it only creates them if missing.
func EnsureStorageClasses(ctx context.Context, c client.Client) error {
	log := ctrl.Log.WithName("bootstrap").WithName("storage-classes")
	log.Info("Starting storage classes provisioning")

	log.Info("Ensuring storage class", "name", config.StorageClassReplica1, "replicas", "1")
	if err := ensureStorageClass(ctx, c, config.StorageClassReplica1, "1"); err != nil {
		log.Error(err, "Failed to ensure storage class", "name", config.StorageClassReplica1)
		return err
	}
	log.Info("Storage class ensured successfully", "name", config.StorageClassReplica1)

	log.Info("Ensuring storage class", "name", config.StorageClassReplica2, "replicas", "2")
	if err := ensureStorageClass(ctx, c, config.StorageClassReplica2, "2"); err != nil {
		log.Error(err, "Failed to ensure storage class", "name", config.StorageClassReplica2)
		return err
	}
	log.Info("Storage class ensured successfully", "name", config.StorageClassReplica2)

	log.Info("Storage classes provisioning completed successfully")
	return nil
}

func ensureStorageClass(ctx context.Context, c client.Client, name, replicas string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("storage-class")
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "storage.k8s.io", Version: "v1", Kind: "StorageClass"})
	obj.SetName(name)

	log.Info("Checking if storage class exists", "name", name)
	if err := c.Get(ctx, client.ObjectKey{Name: name}, obj); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to check storage class", "name", name)
			return err
		}
		log.Info("Storage class not found, creating", "name", name, "replicas", replicas)
		obj.Object["provisioner"] = "driver.longhorn.io"
		obj.Object["allowVolumeExpansion"] = true
		obj.Object["volumeBindingMode"] = "WaitForFirstConsumer"
		obj.Object["parameters"] = map[string]any{
			"numberOfReplicas": replicas,
		}
		if err := c.Create(ctx, obj); err != nil {
			log.Error(err, "Failed to create storage class", "name", name)
			return err
		}
		log.Info("Storage class created successfully", "name", name)
	} else {
		log.Info("Storage class already exists", "name", name)
	}
	return nil
}

// ProvisionIngressAndCertificates ensures dynamic resources are present based on configured domain/email.
// It is idempotent and safe to call on every manager start.
//
// This function orchestrates the provisioning of:
//   - ClusterIssuer for ACME certificates
//   - Ingress resources (Gateway, certificates, routes) via ProvisionIngress
func ProvisionIngressAndCertificates(ctx context.Context, c client.Client, baseDomain, acmeEmail string) error {
	if baseDomain == "" {
		return nil // nothing to do without a domain
	}

	// 1) ClusterIssuer (requires cert-manager CRDs to exist)
	if acmeEmail != "" {
		if err := ensureClusterIssuer(ctx, c, acmeEmail); err != nil {
			return fmt.Errorf("ensure ClusterIssuer: %w", err)
		}
	}

	// 2) Ingress provisioning (wildcard certificate, Gateway, ReferenceGrant)
	// This is handled in provision-ingress.go
	if err := ProvisionIngress(ctx, c, baseDomain, acmeEmail); err != nil {
		return fmt.Errorf("provision ingress: %w", err)
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
							"acmeDNS": map[string]any{
								"host": "http://acme-dns.kibaship.svc.cluster.local",
								"accountSecretRef": map[string]any{
									"name": "acme-dns-account",
									"key":  "acmedns.json",
								},
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

// EnsureRegistryCredentials provisions the registry-registry-auth secret in the registry namespace.
// This secret contains a randomly generated HTTP secret for the Docker registry.
// The init container in the registry deployment waits for this secret before starting.
func EnsureRegistryCredentials(ctx context.Context, c client.Client) error {
	log := ctrl.Log.WithName("bootstrap").WithName("registry-credentials")
	log.Info("Starting registry credentials provisioning")

	const (
		registryNS    = "registry"
		secretName    = "registry-registry-auth"
		httpSecretKey = "http-secret"
	)

	// Ensure registry namespace exists (create if missing)
	log.Info("Ensuring registry namespace exists", "namespace", registryNS)
	if err := ensureRegistryNamespace(ctx, c, registryNS); err != nil {
		log.Error(err, "Failed to ensure registry namespace", "namespace", registryNS)
		return fmt.Errorf("failed to ensure registry namespace: %w", err)
	}
	log.Info("Registry namespace ensured successfully", "namespace", registryNS)

	// Check if secret already exists
	log.Info("Checking if registry credentials secret exists", "secret", secretName, "namespace", registryNS)
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{Namespace: registryNS, Name: secretName}
	if err := c.Get(ctx, secretKey, secret); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to check registry secret", "secret", secretName, "namespace", registryNS)
			return fmt.Errorf("failed to check registry secret: %w", err)
		}

		log.Info("Registry credentials secret not found, creating", "secret", secretName, "namespace", registryNS)
		// Generate random HTTP secret (32 bytes, base64 encoded)
		log.Info("Generating random HTTP secret")
		httpSecretBytes := make([]byte, 32)
		if _, err := rand.Read(httpSecretBytes); err != nil {
			log.Error(err, "Failed to generate HTTP secret")
			return fmt.Errorf("failed to generate http secret: %w", err)
		}
		httpSecret := base64.StdEncoding.EncodeToString(httpSecretBytes)
		log.Info("HTTP secret generated successfully", "length", len(httpSecret))

		// Create secret
		log.Info("Creating registry credentials secret", "secret", secretName, "namespace", registryNS)
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: registryNS,
				Labels: map[string]string{
					"app":                          "registry",
					"app.kubernetes.io/managed-by": "kibaship",
				},
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				httpSecretKey: httpSecret,
			},
		}

		if err := c.Create(ctx, secret); err != nil {
			log.Error(err, "Failed to create registry credentials secret", "secret", secretName, "namespace", registryNS)
			return fmt.Errorf("failed to create registry secret: %w", err)
		}
		log.Info("Registry credentials secret created successfully", "secret", secretName, "namespace", registryNS)
	} else {
		log.Info("Registry credentials secret already exists", "secret", secretName, "namespace", registryNS)
	}

	log.Info("Registry credentials provisioning completed successfully")
	return nil
}

// EnsureRegistryJWKS provisions the registry-auth-keys-jwks secret in the registry namespace.
// This secret contains the JWKS (JSON Web Key Set) file required by Docker Registry v3.0.0
// for JWT token validation. The JWKS is generated from the registry-auth-keys certificate.
// This function also ensures all prerequisite resources are created.
func EnsureRegistryJWKS(ctx context.Context, c client.Client) error {
	log := ctrl.Log.WithName("bootstrap").WithName("registry-jwks")
	log.Info("Starting registry JWKS provisioning")

	const (
		registryNS     = "registry"
		certSecretName = "registry-auth-keys"
		jwksSecretName = "registry-auth-keys-jwks"
		jwksKeyID      = "registry-auth-jwt-signer"
	)

	// 1. Ensure registry namespace exists
	log.Info("Step 1: Ensuring registry namespace exists", "namespace", registryNS)
	if err := ensureRegistryNamespace(ctx, c, registryNS); err != nil {
		log.Error(err, "Failed to ensure registry namespace", "namespace", registryNS)
		return fmt.Errorf("failed to ensure registry namespace: %w", err)
	}
	log.Info("Step 1: Registry namespace ensured successfully", "namespace", registryNS)

	// 2. Ensure registry self-signed issuer exists
	log.Info("Step 2: Ensuring registry self-signed issuer exists", "namespace", registryNS)
	if err := ensureRegistrySelfSignedIssuer(ctx, c, registryNS); err != nil {
		log.Error(err, "Failed to ensure registry self-signed issuer", "namespace", registryNS)
		return fmt.Errorf("failed to ensure registry self-signed issuer: %w", err)
	}
	log.Info("Step 2: Registry self-signed issuer ensured successfully", "namespace", registryNS)

	// 3. Ensure registry-auth-keys certificate exists
	log.Info("Step 3: Ensuring registry-auth-keys certificate exists", "namespace", registryNS)
	if err := ensureRegistryAuthKeysCertificate(ctx, c, registryNS); err != nil {
		log.Error(err, "Failed to ensure registry auth keys certificate", "namespace", registryNS)
		return fmt.Errorf("failed to ensure registry auth keys certificate: %w", err)
	}
	log.Info("Step 3: Registry-auth-keys certificate ensured successfully", "namespace", registryNS)

	// 4. Ensure registry-tls certificate exists
	log.Info("Step 4: Ensuring registry-tls certificate exists", "namespace", registryNS)
	if err := ensureRegistryTLSCertificate(ctx, c, registryNS); err != nil {
		log.Error(err, "Failed to ensure registry TLS certificate", "namespace", registryNS)
		return fmt.Errorf("failed to ensure registry TLS certificate: %w", err)
	}
	log.Info("Step 4: Registry-tls certificate ensured successfully", "namespace", registryNS)

	// Check if JWKS secret already exists
	log.Info("Step 5: Checking if JWKS secret already exists", "secret", jwksSecretName, "namespace", registryNS)
	jwksSecret := &corev1.Secret{}
	jwksSecretKey := client.ObjectKey{Namespace: registryNS, Name: jwksSecretName}
	if err := c.Get(ctx, jwksSecretKey, jwksSecret); err == nil {
		// Secret already exists, nothing to do
		log.Info("JWKS secret already exists, skipping creation", "secret", jwksSecretName, "namespace", registryNS)
		log.Info("Registry JWKS provisioning completed successfully (secret already existed)")
		return nil
	} else if !errors.IsNotFound(err) {
		log.Error(err, "Failed to check JWKS secret", "secret", jwksSecretName, "namespace", registryNS)
		return fmt.Errorf("failed to check jwks secret: %w", err)
	}
	log.Info("JWKS secret not found, will create after certificate is ready", "secret", jwksSecretName, "namespace", registryNS)

	// Wait for registry-auth-keys certificate to be ready (with timeout)
	log.Info("Step 6: Waiting for registry-auth-keys certificate to be ready", "certificate", certSecretName, "namespace", registryNS, "timeout", "2m")
	certSecret := &corev1.Secret{}
	certSecretKey := client.ObjectKey{Namespace: registryNS, Name: certSecretName}

	// Wait up to 2 minutes for the certificate to be ready
	deadline := time.Now().Add(2 * time.Minute)
	for {
		err := c.Get(ctx, certSecretKey, certSecret)
		if err == nil {
			// Secret exists, check if it has the tls.crt key
			if _, ok := certSecret.Data["tls.crt"]; ok {
				// Certificate is ready, proceed
				log.Info("Registry-auth-keys certificate is ready", "certificate", certSecretName, "namespace", registryNS)
				break
			}
			log.Info("Registry-auth-keys certificate secret exists but tls.crt key not ready yet", "certificate", certSecretName, "namespace", registryNS)
		} else if !errors.IsNotFound(err) {
			log.Error(err, "Failed to check certificate secret", "certificate", certSecretName, "namespace", registryNS)
			return fmt.Errorf("failed to check certificate secret: %w", err)
		} else {
			log.Info("Registry-auth-keys certificate secret not found yet", "certificate", certSecretName, "namespace", registryNS)
		}

		// Check if we've exceeded deadline
		if time.Now().After(deadline) {
			log.Error(nil, "Registry-auth-keys certificate did not become ready within timeout", "certificate", certSecretName, "namespace", registryNS, "timeout", "2m")
			return fmt.Errorf("registry-auth-keys certificate did not become ready within 2 minutes")
		}

		// Wait 5 seconds before retrying
		log.Info("Waiting for certificate to be ready", "certificate", certSecretName, "namespace", registryNS, "retryIn", "5s")
		time.Sleep(5 * time.Second)
	}

	// Extract public key from certificate
	log.Info("Step 7: Extracting public key from certificate", "certificate", certSecretName, "namespace", registryNS)
	certPEM := certSecret.Data["tls.crt"]
	block, _ := pem.Decode(certPEM)
	if block == nil {
		log.Error(nil, "Failed to decode PEM block from certificate", "certificate", certSecretName, "namespace", registryNS)
		return fmt.Errorf("failed to decode PEM block from certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Error(err, "Failed to parse certificate", "certificate", certSecretName, "namespace", registryNS)
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	rsaPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		log.Error(nil, "Certificate does not contain RSA public key", "certificate", certSecretName, "namespace", registryNS)
		return fmt.Errorf("certificate does not contain RSA public key")
	}
	log.Info("Public key extracted successfully from certificate", "certificate", certSecretName, "namespace", registryNS, "keySize", rsaPubKey.Size()*8)

	// Generate JWKS JSON
	log.Info("Step 8: Generating JWKS JSON", "keyID", jwksKeyID)
	jwksJSON, err := generateJWKS(rsaPubKey, jwksKeyID)
	if err != nil {
		log.Error(err, "Failed to generate JWKS", "keyID", jwksKeyID)
		return fmt.Errorf("failed to generate JWKS: %w", err)
	}
	log.Info("JWKS JSON generated successfully", "keyID", jwksKeyID, "size", len(jwksJSON))

	// Create JWKS secret
	log.Info("Step 9: Creating JWKS secret", "secret", jwksSecretName, "namespace", registryNS)
	jwksSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jwksSecretName,
			Namespace: registryNS,
			Labels: map[string]string{
				"app":                          "registry",
				"app.kubernetes.io/managed-by": "kibaship",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"jwks.json": jwksJSON,
		},
	}

	if err := c.Create(ctx, jwksSecret); err != nil {
		log.Error(err, "Failed to create JWKS secret", "secret", jwksSecretName, "namespace", registryNS)
		return fmt.Errorf("failed to create jwks secret: %w", err)
	}
	log.Info("JWKS secret created successfully", "secret", jwksSecretName, "namespace", registryNS)

	log.Info("Registry JWKS provisioning completed successfully")
	return nil
}

// ensureRegistryNamespace creates the registry namespace if it doesn't exist
func ensureRegistryNamespace(ctx context.Context, c client.Client, registryNS string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("registry-namespace")
	log.Info("Checking if registry namespace exists", "namespace", registryNS)

	ns := &corev1.Namespace{}
	err := c.Get(ctx, client.ObjectKey{Name: registryNS}, ns)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Registry namespace not found, creating", "namespace", registryNS)
			// Create the namespace
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: registryNS,
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "kibaship",
						"app.kubernetes.io/component":  "registry",
					},
				},
			}
			if err := c.Create(ctx, ns); err != nil {
				log.Error(err, "Failed to create registry namespace", "namespace", registryNS)
				return fmt.Errorf("failed to create registry namespace: %w", err)
			}
			log.Info("Registry namespace created successfully", "namespace", registryNS)
		} else {
			log.Error(err, "Failed to check registry namespace", "namespace", registryNS)
			return fmt.Errorf("failed to check registry namespace: %w", err)
		}
	} else {
		log.Info("Registry namespace already exists", "namespace", registryNS)
	}
	return nil
}

// ensureRegistrySelfSignedIssuer creates the registry self-signed issuer if it doesn't exist
func ensureRegistrySelfSignedIssuer(ctx context.Context, c client.Client, registryNS string) error {
	const issuerName = "registry-selfsigned-issuer"

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Issuer"})
	obj.SetNamespace(registryNS)
	obj.SetName(issuerName)

	err := c.Get(ctx, client.ObjectKey{Namespace: registryNS, Name: issuerName}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the issuer
			obj.Object["spec"] = map[string]any{
				"selfSigned": map[string]any{},
			}
			if err := c.Create(ctx, obj); err != nil {
				return fmt.Errorf("failed to create registry self-signed issuer: %w", err)
			}
		} else {
			return fmt.Errorf("failed to check registry self-signed issuer: %w", err)
		}
	}
	return nil
}

// ensureRegistryAuthKeysCertificate creates the registry-auth-keys certificate if it doesn't exist
func ensureRegistryAuthKeysCertificate(ctx context.Context, c client.Client, registryNS string) error {
	const certName = "registry-auth-keys"

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"})
	obj.SetNamespace(registryNS)
	obj.SetName(certName)

	err := c.Get(ctx, client.ObjectKey{Namespace: registryNS, Name: certName}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the certificate
			obj.Object["spec"] = map[string]any{
				"secretName":  certName,
				"duration":    "87600h", // 10 years
				"renewBefore": "720h",   // 30 days before expiry
				"subject": map[string]any{
					"organizations": []any{"kibaship"},
				},
				"commonName": "registry-auth-jwt-signer",
				"usages": []any{
					"digital signature",
					"key encipherment",
				},
				"privateKey": map[string]any{
					"algorithm": "RSA",
					"size":      4096,
				},
				"issuerRef": map[string]any{
					"name": "registry-selfsigned-issuer",
					"kind": "Issuer",
				},
			}
			if err := c.Create(ctx, obj); err != nil {
				return fmt.Errorf("failed to create registry auth keys certificate: %w", err)
			}
		} else {
			return fmt.Errorf("failed to check registry auth keys certificate: %w", err)
		}
	}
	return nil
}

// ensureRegistryTLSCertificate creates the registry-tls certificate if it doesn't exist
func ensureRegistryTLSCertificate(ctx context.Context, c client.Client, registryNS string) error {
	const certName = "registry-tls"

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"})
	obj.SetNamespace(registryNS)
	obj.SetName(certName)

	err := c.Get(ctx, client.ObjectKey{Namespace: registryNS, Name: certName}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the certificate
			obj.Object["spec"] = map[string]any{
				"secretName":  certName,
				"duration":    "87600h", // 10 years
				"renewBefore": "720h",   // 30 days before expiry
				"subject": map[string]any{
					"organizations": []any{"kibaship"},
				},
				"commonName": "registry.registry.svc.cluster.local",
				"dnsNames": []any{
					"registry.registry.svc.cluster.local",
					"registry.registry.svc",
					"registry.registry",
					"registry",
				},
				"privateKey": map[string]any{
					"algorithm": "RSA",
					"size":      4096,
				},
				"usages": []any{
					"digital signature",
					"key encipherment",
					"server auth",
				},
				"issuerRef": map[string]any{
					"name": "registry-selfsigned-issuer",
					"kind": "Issuer",
				},
			}
			if err := c.Create(ctx, obj); err != nil {
				return fmt.Errorf("failed to create registry TLS certificate: %w", err)
			}
		} else {
			return fmt.Errorf("failed to check registry TLS certificate: %w", err)
		}
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
	log := ctrl.Log.WithName("bootstrap").WithName("registry-ca-buildkit")
	log.Info("Starting registry CA certificate provisioning in buildkit namespace")

	const (
		registryNS     = "registry"
		buildkitNS     = "buildkit"
		registrySecret = "registry-tls"
		buildkitSecret = "registry-ca-cert"
	)

	// Check if registry namespace exists (non-blocking)
	log.Info("Step 1: Checking if registry namespace exists", "namespace", registryNS)
	ns := &corev1.Namespace{}
	err := c.Get(ctx, client.ObjectKey{Name: registryNS}, ns)
	if err != nil {
		if errors.IsNotFound(err) {
			// Registry namespace doesn't exist yet, skip for now
			log.Info("Registry namespace doesn't exist yet, skipping CA certificate provisioning", "namespace", registryNS)
			return nil
		}
		log.Error(err, "Failed to check registry namespace", "namespace", registryNS)
		return fmt.Errorf("failed to check registry namespace: %w", err)
	}
	log.Info("Registry namespace exists", "namespace", registryNS)

	// Check if buildkit namespace exists (non-blocking)
	log.Info("Step 2: Checking if buildkit namespace exists", "namespace", buildkitNS)
	err = c.Get(ctx, client.ObjectKey{Name: buildkitNS}, ns)
	if err != nil {
		if errors.IsNotFound(err) {
			// Buildkit namespace doesn't exist yet, skip for now
			log.Info("Buildkit namespace doesn't exist yet, skipping CA certificate provisioning", "namespace", buildkitNS)
			return nil
		}
		log.Error(err, "Failed to check buildkit namespace", "namespace", buildkitNS)
		return fmt.Errorf("failed to check buildkit namespace: %w", err)
	}
	log.Info("Buildkit namespace exists", "namespace", buildkitNS)

	// Check if CA certificate secret already exists in buildkit namespace
	log.Info("Step 3: Checking if CA certificate secret already exists in buildkit namespace", "secret", buildkitSecret, "namespace", buildkitNS)
	existingSecret := &corev1.Secret{}
	buildkitSecretKey := client.ObjectKey{Namespace: buildkitNS, Name: buildkitSecret}
	if err := c.Get(ctx, buildkitSecretKey, existingSecret); err == nil {
		// Secret already exists, nothing to do
		log.Info("CA certificate secret already exists in buildkit namespace, skipping", "secret", buildkitSecret, "namespace", buildkitNS)
		log.Info("Registry CA certificate provisioning in buildkit namespace completed successfully (secret already existed)")
		return nil
	} else if !errors.IsNotFound(err) {
		log.Error(err, "Failed to check buildkit CA secret", "secret", buildkitSecret, "namespace", buildkitNS)
		return fmt.Errorf("failed to check buildkit CA secret: %w", err)
	}
	log.Info("CA certificate secret not found in buildkit namespace, will create", "secret", buildkitSecret, "namespace", buildkitNS)

	// Wait for registry-tls secret to be ready (with timeout)
	log.Info("Step 4: Waiting for registry-tls secret to be ready", "secret", registrySecret, "namespace", registryNS, "timeout", "2m")
	registryTLSSecret := &corev1.Secret{}
	registrySecretKey := client.ObjectKey{Namespace: registryNS, Name: registrySecret}

	// Wait up to 2 minutes for the certificate to be ready
	deadline := time.Now().Add(2 * time.Minute)
	for {
		err := c.Get(ctx, registrySecretKey, registryTLSSecret)
		if err == nil {
			// Secret exists, check if it has the ca.crt key
			if _, ok := registryTLSSecret.Data["ca.crt"]; ok {
				// Certificate is ready, proceed
				log.Info("Registry-tls secret is ready", "secret", registrySecret, "namespace", registryNS)
				break
			}
			log.Info("Registry-tls secret exists but ca.crt key not ready yet", "secret", registrySecret, "namespace", registryNS)
		} else if !errors.IsNotFound(err) {
			log.Error(err, "Failed to check registry-tls secret", "secret", registrySecret, "namespace", registryNS)
			return fmt.Errorf("failed to check registry-tls secret: %w", err)
		} else {
			log.Info("Registry-tls secret not found yet", "secret", registrySecret, "namespace", registryNS)
		}

		// Check if we've exceeded deadline
		if time.Now().After(deadline) {
			log.Error(nil, "Registry-tls certificate did not become ready within timeout", "secret", registrySecret, "namespace", registryNS, "timeout", "2m")
			return fmt.Errorf("registry-tls certificate did not become ready within 2 minutes")
		}

		// Wait 5 seconds before retrying
		log.Info("Waiting for registry-tls certificate to be ready", "secret", registrySecret, "namespace", registryNS, "retryIn", "5s")
		time.Sleep(5 * time.Second)
	}

	// Extract CA certificate
	log.Info("Step 5: Extracting CA certificate from registry-tls secret", "secret", registrySecret, "namespace", registryNS)
	caCert, ok := registryTLSSecret.Data["ca.crt"]
	if !ok {
		log.Error(nil, "Registry-tls secret does not contain ca.crt key", "secret", registrySecret, "namespace", registryNS)
		return fmt.Errorf("registry-tls secret does not contain ca.crt")
	}
	log.Info("CA certificate extracted successfully", "secret", registrySecret, "namespace", registryNS, "size", len(caCert))

	// Create CA certificate secret in buildkit namespace
	caSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      buildkitSecret,
			Namespace: buildkitNS,
			Labels: map[string]string{
				"app":                          "buildkitd",
				"app.kubernetes.io/managed-by": "kibaship",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"ca.crt": caCert,
		},
	}

	if err := c.Create(ctx, caSecret); err != nil {
		log.Error(err, "Failed to create buildkit CA secret", "secret", buildkitSecret, "namespace", buildkitNS)
		return fmt.Errorf("failed to create buildkit CA secret: %w", err)
	}
	log.Info("CA certificate secret created successfully in buildkit namespace", "secret", buildkitSecret, "namespace", buildkitNS)

	// Restart buildkit deployment to pick up the new secret
	log.Info("Step 7: Restarting buildkit deployment to pick up new CA certificate", "deployment", "buildkitd", "namespace", buildkitNS)
	// We trigger a rollout restart by updating the deployment's restart annotation
	buildkitDeployment := &appsv1.Deployment{}
	deploymentKey := client.ObjectKey{Namespace: buildkitNS, Name: "buildkitd"}
	if err := c.Get(ctx, deploymentKey, buildkitDeployment); err != nil {
		if errors.IsNotFound(err) {
			// Deployment doesn't exist yet, skip restart
			log.Info("Buildkit deployment doesn't exist yet, skipping restart", "deployment", "buildkitd", "namespace", buildkitNS)
			log.Info("Registry CA certificate provisioning in buildkit namespace completed successfully")
			return nil
		}
		log.Error(err, "Failed to get buildkit deployment", "deployment", "buildkitd", "namespace", buildkitNS)
		return fmt.Errorf("failed to get buildkit deployment: %w", err)
	}

	// Add/update restart annotation to trigger rollout
	log.Info("Adding restart annotation to buildkit deployment", "deployment", "buildkitd", "namespace", buildkitNS)
	if buildkitDeployment.Spec.Template.Annotations == nil {
		buildkitDeployment.Spec.Template.Annotations = make(map[string]string)
	}
	restartTime := time.Now().Format(time.RFC3339)
	buildkitDeployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartTime
	log.Info("Restart annotation added", "deployment", "buildkitd", "namespace", buildkitNS, "restartedAt", restartTime)

	if err := c.Update(ctx, buildkitDeployment); err != nil {
		log.Error(err, "Failed to restart buildkit deployment", "deployment", "buildkitd", "namespace", buildkitNS)
		return fmt.Errorf("failed to restart buildkit deployment: %w", err)
	}
	log.Info("Buildkit deployment restart triggered successfully", "deployment", "buildkitd", "namespace", buildkitNS)

	log.Info("Registry CA certificate provisioning in buildkit namespace completed successfully")
	return nil
}
