package controller

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

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/validation"
)

func TestEnsureCertificateForDomain_ProvisionsCertificateAndSetsRef(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	// Scheme

	scheme := runtime.NewScheme()
	g.Expect(platformv1alpha1.AddToScheme(scheme)).To(Succeed())

	// Register cert-manager Certificate GVK for fake client
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"}, &unstructured.Unstructured{})

	// Seed data: namespaces, application, applicationdomain
	app := &platformv1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-for-domain",
			Namespace: "default",
		},
		Spec: platformv1alpha1.ApplicationSpec{Type: platformv1alpha1.ApplicationTypeGitRepository},
	}
	ad := &platformv1alpha1.ApplicationDomain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "domain-test",
			Namespace: "default",
			Labels: map[string]string{
				validation.LabelResourceUUID:    "11111111-1111-1111-1111-111111111111",
				validation.LabelResourceSlug:    "domain-test",
				validation.LabelProjectUUID:     "22222222-2222-2222-2222-222222222222",
				validation.LabelApplicationUUID: "33333333-3333-3333-3333-333333333333",
			},
		},
		Spec: platformv1alpha1.ApplicationDomainSpec{
			ApplicationRef: corev1.LocalObjectReference{Name: app.Name},
			Domain:         "custom-unit.example.com",
			Port:           3000,
			Type:           platformv1alpha1.ApplicationDomainTypeCustom,
		},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(app, ad).Build()
	r := &ApplicationDomainReconciler{Client: cl, Scheme: scheme}

	// Directly ensure certificate (unit-scoped)
	_, _, err := r.ensureCertificateForDomain(ctx, ad)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify Certificate exists in kibaship namespace with copied labels
	cert := &unstructured.Unstructured{}
	cert.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"})
	certName := "ad-" + ad.Name
	g.Expect(cl.Get(ctx, client.ObjectKey{Namespace: "kibaship", Name: certName}, cert)).To(Succeed())
	labels := cert.GetLabels()
	g.Expect(labels[validation.LabelProjectUUID]).To(Equal("22222222-2222-2222-2222-222222222222"))
	g.Expect(labels[validation.LabelResourceUUID]).To(Equal("11111111-1111-1111-1111-111111111111"))
	g.Expect(labels[validation.LabelApplicationUUID]).To(Equal("33333333-3333-3333-3333-333333333333"))
}
