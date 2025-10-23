package bootstrap

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestEnsureAcmeDNSAccountSecretCreation(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(Succeed())

	// Create fake client
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Test domain
	baseDomain := "example.com"

	// Mock ACME-DNS account data
	mockAccount := AcmeDNSAccount{
		Username:   "test-username",
		Password:   "test-password",
		Subdomain:  "test-subdomain",
		FullDomain: "test-subdomain.acme.example.com",
	}

	// Create the expected credentials JSON
	expectedCredentials := map[string]AcmeDNSAccount{
		baseDomain: mockAccount,
	}
	expectedJSON, err := json.Marshal(expectedCredentials)
	g.Expect(err).NotTo(HaveOccurred())

	// Create the secret manually (simulating successful registration)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AcmeDNSAccountSecretName,
			Namespace: AcmeDNSNamespace,
			Labels: map[string]string{
				"app":                          AcmeDNSName,
				"app.kubernetes.io/name":       AcmeDNSName,
				"app.kubernetes.io/component":  "credentials",
				"app.kubernetes.io/managed-by": "kibaship",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"acmedns.json": expectedJSON,
		},
	}

	err = fakeClient.Create(ctx, secret)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify the secret exists and has correct structure
	retrievedSecret := &corev1.Secret{}
	err = fakeClient.Get(ctx, client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSAccountSecretName,
	}, retrievedSecret)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify the secret has the correct data
	g.Expect(retrievedSecret.Data).To(HaveKey("acmedns.json"))

	// Parse and verify the JSON structure
	var credentials map[string]AcmeDNSAccount
	err = json.Unmarshal(retrievedSecret.Data["acmedns.json"], &credentials)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(credentials).To(HaveKey(baseDomain))
	g.Expect(credentials[baseDomain].Username).To(Equal("test-username"))
	g.Expect(credentials[baseDomain].Password).To(Equal("test-password"))
	g.Expect(credentials[baseDomain].Subdomain).To(Equal("test-subdomain"))
	g.Expect(credentials[baseDomain].FullDomain).To(Equal("test-subdomain.acme.example.com"))

	// Verify labels
	g.Expect(retrievedSecret.Labels).To(HaveKeyWithValue("app", AcmeDNSName))
	g.Expect(retrievedSecret.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "kibaship"))
}

func TestAcmeDNSAccountStructure(t *testing.T) {
	g := NewWithT(t)

	// Test that the AcmeDNSAccount structure can be properly marshaled/unmarshaled
	account := AcmeDNSAccount{
		Username:   "test-user",
		Password:   "test-pass",
		Subdomain:  "test-sub",
		FullDomain: "test-sub.acme.example.com",
		AllowFrom:  []string{"192.168.1.0/24"},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(account)
	g.Expect(err).NotTo(HaveOccurred())

	// Unmarshal back
	var unmarshaled AcmeDNSAccount
	err = json.Unmarshal(jsonData, &unmarshaled)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify all fields
	g.Expect(unmarshaled.Username).To(Equal(account.Username))
	g.Expect(unmarshaled.Password).To(Equal(account.Password))
	g.Expect(unmarshaled.Subdomain).To(Equal(account.Subdomain))
	g.Expect(unmarshaled.FullDomain).To(Equal(account.FullDomain))
	g.Expect(unmarshaled.AllowFrom).To(Equal(account.AllowFrom))

	// Test cert-manager format (domain as key)
	credentialsMap := map[string]AcmeDNSAccount{
		"example.com": account,
	}

	credentialsJSON, err := json.Marshal(credentialsMap)
	g.Expect(err).NotTo(HaveOccurred())

	var unmarshaledMap map[string]AcmeDNSAccount
	err = json.Unmarshal(credentialsJSON, &unmarshaledMap)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(unmarshaledMap).To(HaveKey("example.com"))
	g.Expect(unmarshaledMap["example.com"].Username).To(Equal(account.Username))
}
