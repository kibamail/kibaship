package bootstrap

import (
	"context"
	"fmt"

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
