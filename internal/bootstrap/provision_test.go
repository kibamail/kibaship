package bootstrap

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureClusterIssuerProduction(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()

	// Create fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Test production environment
	err := ensureClusterIssuer(ctx, fakeClient, "test@example.com", "production")
	g.Expect(err).NotTo(HaveOccurred())

	// Verify ClusterIssuer was created with production URL
	issuer := &unstructured.Unstructured{}
	issuer.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "ClusterIssuer"})
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "certmanager-acme-issuer"}, issuer)
	g.Expect(err).NotTo(HaveOccurred())

	// Check ACME server URL
	spec, found, err := unstructured.NestedMap(issuer.Object, "spec")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())

	acme, found, err := unstructured.NestedMap(spec, "acme")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())

	server, found, err := unstructured.NestedString(acme, "server")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(server).To(Equal("https://acme-v02.api.letsencrypt.org/directory"))

	// Check email
	email, found, err := unstructured.NestedString(acme, "email")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(email).To(Equal("test@example.com"))
}

func TestEnsureClusterIssuerStaging(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()

	// Create fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Test staging environment
	err := ensureClusterIssuer(ctx, fakeClient, "test@example.com", "staging")
	g.Expect(err).NotTo(HaveOccurred())

	// Verify ClusterIssuer was created with staging URL
	issuer := &unstructured.Unstructured{}
	issuer.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "ClusterIssuer"})
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "certmanager-acme-issuer"}, issuer)
	g.Expect(err).NotTo(HaveOccurred())

	// Check ACME server URL
	spec, found, err := unstructured.NestedMap(issuer.Object, "spec")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())

	acme, found, err := unstructured.NestedMap(spec, "acme")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())

	server, found, err := unstructured.NestedString(acme, "server")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(server).To(Equal("https://acme-staging-v02.api.letsencrypt.org/directory"))

	// Check email
	email, found, err := unstructured.NestedString(acme, "email")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(email).To(Equal("test@example.com"))
}

func TestEnsureClusterIssuerIdempotent(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()

	// Create fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create ClusterIssuer first time
	err := ensureClusterIssuer(ctx, fakeClient, "test@example.com", "production")
	g.Expect(err).NotTo(HaveOccurred())

	// Create ClusterIssuer second time (should be idempotent)
	err = ensureClusterIssuer(ctx, fakeClient, "test@example.com", "production")
	g.Expect(err).NotTo(HaveOccurred())

	// Verify only one ClusterIssuer exists
	issuer := &unstructured.Unstructured{}
	issuer.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "ClusterIssuer"})
	err = fakeClient.Get(ctx, client.ObjectKey{Name: "certmanager-acme-issuer"}, issuer)
	g.Expect(err).NotTo(HaveOccurred())
}
