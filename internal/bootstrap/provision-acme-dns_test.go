package bootstrap

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kibamail/kibaship/pkg/config"
)

func TestProvisionAcmeDNS(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())
	g.Expect(appsv1.AddToScheme(scheme)).To(Succeed())

	// Create fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Test domain
	baseDomain := "example.com"

	// Call ProvisionAcmeDNS (without waiting for readiness in test)
	err := provisionAcmeDNSWithoutWait(ctx, fakeClient, baseDomain)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify ConfigMap was created
	configMap := &corev1.ConfigMap{}
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName + "-config",
	}, configMap)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(configMap.Data).To(HaveKey("config.cfg"))
	g.Expect(configMap.Data["config.cfg"]).To(ContainSubstring("domain = \"acme.example.com\""))
	g.Expect(configMap.Data["config.cfg"]).To(ContainSubstring("nsname = \"ns1.acme.example.com\""))

	// Verify PVC was created
	pvc := &corev1.PersistentVolumeClaim{}
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName + "-data",
	}, pvc)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
	g.Expect(*pvc.Spec.StorageClassName).To(Equal(config.StorageClassReplica1))

	// Verify DNS LoadBalancer Service was created
	dnsService := &corev1.Service{}
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName + "-dns",
	}, dnsService)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(dnsService.Spec.Ports).To(HaveLen(2)) // DNS UDP, DNS TCP
	g.Expect(dnsService.Spec.Type).To(Equal(corev1.ServiceTypeLoadBalancer))
	g.Expect(dnsService.Labels["service-type"]).To(Equal("dns"))

	// Verify HTTP ClusterIP Service was created
	httpService := &corev1.Service{}
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName,
	}, httpService)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(httpService.Spec.Ports).To(HaveLen(1)) // HTTP only
	g.Expect(httpService.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
	g.Expect(httpService.Labels["service-type"]).To(Equal("http"))

	// Verify Deployment was created
	deployment := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName,
	}, deployment)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(*deployment.Spec.Replicas).To(Equal(int32(2)))
	g.Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
	g.Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("joohoi/acme-dns:v1.0"))
}

func TestProvisionAcmeDNSIdempotent(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())
	g.Expect(appsv1.AddToScheme(scheme)).To(Succeed())

	// Create fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Test domain
	baseDomain := "example.com"

	// Call ProvisionAcmeDNS twice
	err := provisionAcmeDNSWithoutWait(ctx, fakeClient, baseDomain)
	g.Expect(err).NotTo(HaveOccurred())

	err = provisionAcmeDNSWithoutWait(ctx, fakeClient, baseDomain)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify only one of each resource exists
	configMapList := &corev1.ConfigMapList{}
	err = fakeClient.List(ctx, configMapList, client.InNamespace(AcmeDNSNamespace))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(configMapList.Items).To(HaveLen(1))

	pvcList := &corev1.PersistentVolumeClaimList{}
	err = fakeClient.List(ctx, pvcList, client.InNamespace(AcmeDNSNamespace))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(pvcList.Items).To(HaveLen(1))

	serviceList := &corev1.ServiceList{}
	err = fakeClient.List(ctx, serviceList, client.InNamespace(AcmeDNSNamespace))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(serviceList.Items).To(HaveLen(2)) // DNS LoadBalancer and HTTP ClusterIP services

	deploymentList := &appsv1.DeploymentList{}
	err = fakeClient.List(ctx, deploymentList, client.InNamespace(AcmeDNSNamespace))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(deploymentList.Items).To(HaveLen(1))
}

func TestProvisionAcmeDNSSkipsWhenNoDomain(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	// Create fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Call with empty domain
	err := ProvisionAcmeDNS(ctx, fakeClient, "")
	g.Expect(err).NotTo(HaveOccurred())

	// Verify no resources were created
	configMapList := &corev1.ConfigMapList{}
	err = fakeClient.List(ctx, configMapList, client.InNamespace(AcmeDNSNamespace))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(configMapList.Items).To(HaveLen(0))
}

func TestProvisionAcmeDNSTriggersRolloutRestart(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())
	g.Expect(appsv1.AddToScheme(scheme)).To(Succeed())

	// Create fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// First provision - creates the deployment
	err := provisionAcmeDNSWithoutWait(ctx, fakeClient, "example.com")
	g.Expect(err).NotTo(HaveOccurred())

	// Get the deployment and verify it exists
	deployment := &appsv1.Deployment{}
	deploymentKey := client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName,
	}
	err = fakeClient.Get(ctx, deploymentKey, deployment)
	g.Expect(err).NotTo(HaveOccurred())

	// Store the original restart annotation (should be empty initially)
	originalRestartAnnotation := ""
	if deployment.Spec.Template.Annotations != nil {
		originalRestartAnnotation = deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]
	}

	// Second provision - should trigger rollout restart
	err = provisionAcmeDNSWithoutWait(ctx, fakeClient, "example.com")
	g.Expect(err).NotTo(HaveOccurred())

	// Get the deployment again and verify the restart annotation was added/updated
	err = fakeClient.Get(ctx, deploymentKey, deployment)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify the restart annotation exists and is different from before
	g.Expect(deployment.Spec.Template.Annotations).NotTo(BeNil())
	newRestartAnnotation := deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]
	g.Expect(newRestartAnnotation).NotTo(BeEmpty())
	g.Expect(newRestartAnnotation).NotTo(Equal(originalRestartAnnotation))

	// Verify the annotation is a valid RFC3339 timestamp
	_, err = time.Parse(time.RFC3339, newRestartAnnotation)
	g.Expect(err).NotTo(HaveOccurred())
}

// provisionAcmeDNSWithoutWait is a helper that provisions ACME-DNS without waiting for readiness
func provisionAcmeDNSWithoutWait(ctx context.Context, c client.Client, baseDomain string) error {
	if baseDomain == "" {
		return nil
	}

	// 1. Ensure kibaship namespace exists
	if err := ensureNamespace(ctx, c, AcmeDNSNamespace); err != nil {
		return err
	}

	// 2. ConfigMap for ACME-DNS configuration
	if err := ensureAcmeDNSConfigMap(ctx, c, baseDomain); err != nil {
		return err
	}

	// 3. PersistentVolumeClaim for ACME-DNS data
	if err := ensureAcmeDNSPVC(ctx, c); err != nil {
		return err
	}

	// 4. Services for ACME-DNS
	if err := ensureAcmeDNSServices(ctx, c); err != nil {
		return err
	}

	// 5. Deployment for ACME-DNS
	if err := ensureAcmeDNSDeployment(ctx, c); err != nil {
		return err
	}

	// Skip waiting for readiness in tests
	return nil
}
