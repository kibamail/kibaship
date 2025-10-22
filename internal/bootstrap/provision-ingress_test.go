package bootstrap

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureIngressGatewayWithoutCertificate(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	// Create fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Test parameters
	gatewayClassName := "cilium"
	baseDomain := "example.com"

	// Call ensureIngressGateway (certificate doesn't exist, so should create minimal gateway)
	err := ensureIngressGateway(ctx, fakeClient, gatewayClassName, baseDomain)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify Gateway was created
	gateway := &unstructured.Unstructured{}
	gateway.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	})
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: KibashipNamespace,
		Name:      IngressGatewayName,
	}, gateway)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify gateway has minimal listeners (HTTP and DNS only)
	spec, found, err := unstructured.NestedMap(gateway.Object, "spec")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())

	listeners, found, err := unstructured.NestedSlice(spec, "listeners")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(listeners).To(HaveLen(1)) // HTTP only

	// Verify listener names
	listenerNames := make([]string, 0, len(listeners))
	for _, listener := range listeners {
		if listenerMap, ok := listener.(map[string]any); ok {
			if name, ok := listenerMap["name"].(string); ok {
				listenerNames = append(listenerNames, name)
			}
		}
	}
	g.Expect(listenerNames).To(ContainElements("http"))

	// Verify ACME-DNS routes were created
	httpRoute := &unstructured.Unstructured{}
	httpRoute.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	})
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: KibashipNamespace,
		Name:      "acme-dns-api",
	}, httpRoute)
	g.Expect(err).NotTo(HaveOccurred())


}

func TestEnsureIngressGatewayWithCertificate(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	// Create fake client with certificate secret
	certificateSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      IngressWildcardCertName,
			Namespace: KibashipNamespace,
		},
		Data: map[string][]byte{
			"tls.crt": []byte("fake-cert"),
			"tls.key": []byte("fake-key"),
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(certificateSecret).Build()

	// Test parameters
	gatewayClassName := "cilium"
	baseDomain := "example.com"

	// Call ensureIngressGateway (certificate exists, so should create full gateway)
	err := ensureIngressGateway(ctx, fakeClient, gatewayClassName, baseDomain)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify Gateway was created
	gateway := &unstructured.Unstructured{}
	gateway.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	})
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: KibashipNamespace,
		Name:      IngressGatewayName,
	}, gateway)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify gateway has all listeners
	spec, found, err := unstructured.NestedMap(gateway.Object, "spec")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())

	listeners, found, err := unstructured.NestedSlice(spec, "listeners")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(listeners).To(HaveLen(5)) // All listeners: HTTP, HTTPS, MySQL, Valkey, PostgreSQL

	// Verify listener names
	listenerNames := make([]string, 0, len(listeners))
	for _, listener := range listeners {
		if listenerMap, ok := listener.(map[string]any); ok {
			if name, ok := listenerMap["name"].(string); ok {
				listenerNames = append(listenerNames, name)
			}
		}
	}
	g.Expect(listenerNames).To(ContainElements("http", "https", "mysql-tls", "valkey-tls", "postgres-tls"))
}

func TestEnsureIngressGatewayPatchesExistingGateway(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	// Create existing gateway with minimal listeners
	existingGateway := &unstructured.Unstructured{}
	existingGateway.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	})
	existingGateway.SetNamespace(KibashipNamespace)
	existingGateway.SetName(IngressGatewayName)
	existingGateway.Object["spec"] = map[string]any{
		"gatewayClassName": "cilium",
		"listeners": []any{
			map[string]any{
				"name":     "http",
				"protocol": "HTTP",
				"port":     int64(80),
			},
		},
	}

	// Create certificate secret
	certificateSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      IngressWildcardCertName,
			Namespace: KibashipNamespace,
		},
		Data: map[string][]byte{
			"tls.crt": []byte("fake-cert"),
			"tls.key": []byte("fake-key"),
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingGateway, certificateSecret).Build()

	// Test parameters
	gatewayClassName := "cilium"
	baseDomain := "example.com"

	// Call ensureIngressGateway (should patch existing gateway)
	err := ensureIngressGateway(ctx, fakeClient, gatewayClassName, baseDomain)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify Gateway was patched
	gateway := &unstructured.Unstructured{}
	gateway.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	})
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: KibashipNamespace,
		Name:      IngressGatewayName,
	}, gateway)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify gateway now has all listeners
	spec, found, err := unstructured.NestedMap(gateway.Object, "spec")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())

	listeners, found, err := unstructured.NestedSlice(spec, "listeners")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(listeners).To(HaveLen(5)) // All listeners after patching

	// Verify listener names
	listenerNames := make([]string, 0, len(listeners))
	for _, listener := range listeners {
		if listenerMap, ok := listener.(map[string]any); ok {
			if name, ok := listenerMap["name"].(string); ok {
				listenerNames = append(listenerNames, name)
			}
		}
	}
	g.Expect(listenerNames).To(ContainElements("http", "https", "mysql-tls", "valkey-tls", "postgres-tls"))
}

func TestProvisionIngressIdempotent(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	// Create fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Test parameters
	baseDomain := "example.com"
	acmeEmail := "test@example.com"
	gatewayClassName := "cilium"

	// Call ProvisionIngress twice
	err := ProvisionIngress(ctx, fakeClient, baseDomain, acmeEmail, gatewayClassName)
	g.Expect(err).NotTo(HaveOccurred())

	err = ProvisionIngress(ctx, fakeClient, baseDomain, acmeEmail, gatewayClassName)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify only one gateway exists
	gatewayList := &unstructured.UnstructuredList{}
	gatewayList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	})
	err = fakeClient.List(ctx, gatewayList, client.InNamespace(KibashipNamespace))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gatewayList.Items).To(HaveLen(1))
}
