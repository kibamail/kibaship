package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kibamail/kibaship/pkg/config"
)

const (
	AcmeDNSNamespace         = "kibaship"
	AcmeDNSName              = "acme-dns"
	AcmeDNSVersion           = "v1.0"
	AcmeDNSAccountSecretName = "acme-dns-account"
)

// AcmeDNSAccount represents the ACME-DNS account registration response
type AcmeDNSAccount struct {
	AllowFrom  []string `json:"allowfrom,omitempty"`
	FullDomain string   `json:"fulldomain"`
	Password   string   `json:"password"`
	Subdomain  string   `json:"subdomain"`
	Username   string   `json:"username"`
}

// AcmeDNSRegistrationRequest represents the ACME-DNS account registration request
type AcmeDNSRegistrationRequest struct {
	AllowFrom []string `json:"allowfrom,omitempty"`
}

// ProvisionAcmeDNS ensures ACME-DNS resources are present and ready.
// It is idempotent and safe to call on every manager start.
//
// Prerequisites:
//   - kibaship namespace must exist (operator namespace)
//   - Storage classes must be available
//
// Resources created (in order):
//  1. ConfigMap for ACME-DNS configuration
//  2. PersistentVolumeClaim for ACME-DNS data
//  3. Service for ACME-DNS (DNS and HTTP API)
//  4. Deployment for ACME-DNS
//  5. Wait for all resources to be ready
func ProvisionAcmeDNS(ctx context.Context, c client.Client, baseDomain string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns")

	if baseDomain == "" {
		log.Info("No base domain configured, skipping ACME-DNS provisioning")
		return nil
	}

	log.Info("Starting ACME-DNS provisioning", "domain", baseDomain)

	// 1. Ensure kibaship namespace exists (should already exist, but check anyway)
	log.Info("Step 1: Ensuring kibaship namespace exists")
	if err := ensureNamespace(ctx, c, AcmeDNSNamespace); err != nil {
		return fmt.Errorf("ensure kibaship namespace: %w", err)
	}

	// 2. ConfigMap for ACME-DNS configuration
	log.Info("Step 2: Ensuring ACME-DNS ConfigMap")
	if err := ensureAcmeDNSConfigMap(ctx, c, baseDomain); err != nil {
		return fmt.Errorf("ensure ACME-DNS ConfigMap: %w", err)
	}

	// 3. PersistentVolumeClaim for ACME-DNS data
	log.Info("Step 3: Ensuring ACME-DNS PersistentVolumeClaim")
	if err := ensureAcmeDNSPVC(ctx, c); err != nil {
		return fmt.Errorf("ensure ACME-DNS PVC: %w", err)
	}

	// 4. Services for ACME-DNS (DNS LoadBalancer and HTTP ClusterIP)
	log.Info("Step 4: Ensuring ACME-DNS Services")
	if err := ensureAcmeDNSServices(ctx, c); err != nil {
		return fmt.Errorf("ensure ACME-DNS Services: %w", err)
	}

	// 5. Deployment for ACME-DNS
	log.Info("Step 5: Ensuring ACME-DNS Deployment")
	if err := ensureAcmeDNSDeployment(ctx, c); err != nil {
		return fmt.Errorf("ensure ACME-DNS Deployment: %w", err)
	}

	// 6. Wait for ACME-DNS to be ready
	log.Info("Step 6: Waiting for ACME-DNS to be ready")
	if err := waitForAcmeDNSReady(ctx, c); err != nil {
		return fmt.Errorf("wait for ACME-DNS ready: %w", err)
	}

	// 7. Register ACME-DNS account and save credentials
	log.Info("Step 7: Ensuring ACME-DNS account registration")
	if err := ensureAcmeDNSAccount(ctx, c, baseDomain); err != nil {
		return fmt.Errorf("ensure ACME-DNS account: %w", err)
	}

	log.Info("ACME-DNS provisioning completed successfully")
	return nil
}

// ensureAcmeDNSConfigMap creates the ConfigMap for ACME-DNS configuration
func ensureAcmeDNSConfigMap(ctx context.Context, c client.Client, baseDomain string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns-configmap")

	configMap := &corev1.ConfigMap{}
	configMapKey := client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName + "-config",
	}

	if err := c.Get(ctx, configMapKey, configMap); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating ACME-DNS ConfigMap", "name", configMapKey.Name)

		// Create ACME-DNS configuration
		acmeDomain := fmt.Sprintf("acme.%s", baseDomain)
		nsName := fmt.Sprintf("ns1.acme.%s", baseDomain)
		adminEmail := fmt.Sprintf("admin.%s", baseDomain)

		configData := fmt.Sprintf(`[general]
listen = ":53"
protocol = "both"
domain = "%s"
nsname = "%s"
nsadmin = "%s"
records = [
    "%s. A 127.0.0.1",
    "%s. NS %s.",
]
debug = false

[database]
engine = "sqlite3"
connection = "/var/lib/acme-dns/acme-dns.db"

[api]
ip = "0.0.0.0"
port = "80"
tls = "none"
disable_registration = false
corsorigins = [
    "*"
]
use_header = false
header_name = "X-Forwarded-For"

[logconfig]
loglevel = "debug"
logtype = "stdout"
logformat = "text"`, acmeDomain, nsName, adminEmail, acmeDomain, acmeDomain, nsName)

		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapKey.Name,
				Namespace: configMapKey.Namespace,
				Labels: map[string]string{
					"app":                          AcmeDNSName,
					"app.kubernetes.io/name":       AcmeDNSName,
					"app.kubernetes.io/component":  "dns",
					"app.kubernetes.io/managed-by": "kibaship",
				},
			},
			Data: map[string]string{
				"config.cfg": configData,
			},
		}

		if err := c.Create(ctx, configMap); err != nil {
			log.Error(err, "Failed to create ACME-DNS ConfigMap")
			return err
		}

		log.Info("ACME-DNS ConfigMap created successfully", "name", configMapKey.Name)
	} else {
		log.Info("ACME-DNS ConfigMap already exists", "name", configMapKey.Name)
	}

	return nil
}

// ensureAcmeDNSPVC creates the PersistentVolumeClaim for ACME-DNS data
func ensureAcmeDNSPVC(ctx context.Context, c client.Client) error {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns-pvc")

	pvc := &corev1.PersistentVolumeClaim{}
	pvcKey := client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName + "-data",
	}

	if err := c.Get(ctx, pvcKey, pvc); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating ACME-DNS PersistentVolumeClaim", "name", pvcKey.Name)

		pvc = &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcKey.Name,
				Namespace: pvcKey.Namespace,
				Labels: map[string]string{
					"app":                          AcmeDNSName,
					"app.kubernetes.io/name":       AcmeDNSName,
					"app.kubernetes.io/component":  "dns",
					"app.kubernetes.io/managed-by": "kibaship",
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				StorageClassName: func() *string { s := config.StorageClassReplica1; return &s }(),
			},
		}

		if err := c.Create(ctx, pvc); err != nil {
			log.Error(err, "Failed to create ACME-DNS PersistentVolumeClaim")
			return err
		}

		log.Info("ACME-DNS PersistentVolumeClaim created successfully", "name", pvcKey.Name)
	} else {
		log.Info("ACME-DNS PersistentVolumeClaim already exists", "name", pvcKey.Name)
	}

	return nil
}

// ensureAcmeDNSServices creates two services for ACME-DNS:
// 1. LoadBalancer service for DNS (port 53) - exposed externally
// 2. ClusterIP service for HTTP API (port 80) - internal only
func ensureAcmeDNSServices(ctx context.Context, c client.Client) error {
	// Create DNS LoadBalancer service
	if err := ensureAcmeDNSDNSService(ctx, c); err != nil {
		return fmt.Errorf("ensure ACME-DNS DNS service: %w", err)
	}

	// Create HTTP ClusterIP service
	if err := ensureAcmeDNSHTTPService(ctx, c); err != nil {
		return fmt.Errorf("ensure ACME-DNS HTTP service: %w", err)
	}

	return nil
}

// ensureAcmeDNSDNSService creates the LoadBalancer Service for ACME-DNS DNS (port 53)
func ensureAcmeDNSDNSService(ctx context.Context, c client.Client) error {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns-dns-service")

	service := &corev1.Service{}
	serviceKey := client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName + "-dns",
	}

	if err := c.Get(ctx, serviceKey, service); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating ACME-DNS DNS LoadBalancer Service", "name", serviceKey.Name)

		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceKey.Name,
				Namespace: serviceKey.Namespace,
				Labels: map[string]string{
					"app":                          AcmeDNSName,
					"app.kubernetes.io/name":       AcmeDNSName,
					"app.kubernetes.io/component":  "dns",
					"app.kubernetes.io/managed-by": "kibaship",
					"service-type":                 "dns",
				},
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeLoadBalancer,
				Selector: map[string]string{
					"app": AcmeDNSName,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "dns-udp",
						Port:       53,
						TargetPort: intstr.FromString("dns-udp"),
						Protocol:   corev1.ProtocolUDP,
					},
					{
						Name:       "dns-tcp",
						Port:       53,
						TargetPort: intstr.FromString("dns-tcp"),
						Protocol:   corev1.ProtocolTCP,
					},
				},
				SessionAffinity: corev1.ServiceAffinityClientIP,
				SessionAffinityConfig: &corev1.SessionAffinityConfig{
					ClientIP: &corev1.ClientIPConfig{
						TimeoutSeconds: &[]int32{10800}[0],
					},
				},
			},
		}

		if err := c.Create(ctx, service); err != nil {
			log.Error(err, "Failed to create ACME-DNS DNS LoadBalancer Service")
			return err
		}

		log.Info("ACME-DNS DNS LoadBalancer Service created successfully", "name", serviceKey.Name)
	} else {
		log.Info("ACME-DNS DNS LoadBalancer Service already exists", "name", serviceKey.Name)
	}

	return nil
}

// ensureAcmeDNSHTTPService creates the ClusterIP Service for ACME-DNS HTTP API (port 80)
func ensureAcmeDNSHTTPService(ctx context.Context, c client.Client) error {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns-http-service")

	service := &corev1.Service{}
	serviceKey := client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName,
	}

	if err := c.Get(ctx, serviceKey, service); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating ACME-DNS HTTP ClusterIP Service", "name", serviceKey.Name)

		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceKey.Name,
				Namespace: serviceKey.Namespace,
				Labels: map[string]string{
					"app":                          AcmeDNSName,
					"app.kubernetes.io/name":       AcmeDNSName,
					"app.kubernetes.io/component":  "api",
					"app.kubernetes.io/managed-by": "kibaship",
					"service-type":                 "http",
				},
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Selector: map[string]string{
					"app": AcmeDNSName,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Port:       80,
						TargetPort: intstr.FromString("http"),
						Protocol:   corev1.ProtocolTCP,
					},
				},
			},
		}

		if err := c.Create(ctx, service); err != nil {
			log.Error(err, "Failed to create ACME-DNS HTTP ClusterIP Service")
			return err
		}

		log.Info("ACME-DNS HTTP ClusterIP Service created successfully", "name", serviceKey.Name)
	} else {
		log.Info("ACME-DNS HTTP ClusterIP Service already exists", "name", serviceKey.Name)
	}

	return nil
}

// ensureAcmeDNSDeployment creates the Deployment for ACME-DNS or triggers a rollout restart if it exists
func ensureAcmeDNSDeployment(ctx context.Context, c client.Client) error {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns-deployment")

	deployment := &appsv1.Deployment{}
	deploymentKey := client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName,
	}

	if err := c.Get(ctx, deploymentKey, deployment); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		log.Info("Creating ACME-DNS Deployment", "name", deploymentKey.Name)

		replicas := int32(2)

		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentKey.Name,
				Namespace: deploymentKey.Namespace,
				Labels: map[string]string{
					"app":                          AcmeDNSName,
					"app.kubernetes.io/name":       AcmeDNSName,
					"app.kubernetes.io/component":  "dns",
					"app.kubernetes.io/managed-by": "kibaship",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": AcmeDNSName,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app":                         AcmeDNSName,
							"app.kubernetes.io/name":      AcmeDNSName,
							"app.kubernetes.io/component": "dns",
						},
					},
					Spec: corev1.PodSpec{
						SecurityContext: &corev1.PodSecurityContext{
							RunAsUser:  &[]int64{0}[0],
							RunAsGroup: &[]int64{0}[0],
							FSGroup:    &[]int64{0}[0],
						},
						Containers: []corev1.Container{
							{
								Name:            AcmeDNSName,
								Image:           "joohoi/acme-dns:" + AcmeDNSVersion,
								ImagePullPolicy: corev1.PullIfNotPresent,
								Ports: []corev1.ContainerPort{
									{
										Name:          "dns-udp",
										ContainerPort: 53,
										Protocol:      corev1.ProtocolUDP,
									},
									{
										Name:          "dns-tcp",
										ContainerPort: 53,
										Protocol:      corev1.ProtocolTCP,
									},
									{
										Name:          "http",
										ContainerPort: 80,
										Protocol:      corev1.ProtocolTCP,
									},
								},
								Env: []corev1.EnvVar{
									{
										Name:  "ACMEDNS_CONFIG",
										Value: "/etc/acme-dns/config.cfg",
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "config",
										MountPath: "/etc/acme-dns",
										ReadOnly:  true,
									},
									{
										Name:      "data",
										MountPath: "/var/lib/acme-dns",
									},
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("100m"),
										corev1.ResourceMemory: resource.MustParse("128Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("500m"),
										corev1.ResourceMemory: resource.MustParse("256Mi"),
									},
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/health",
											Port: intstr.FromString("http"),
										},
									},
									InitialDelaySeconds: 30,
									PeriodSeconds:       10,
									TimeoutSeconds:      5,
									FailureThreshold:    3,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/health",
											Port: intstr.FromString("http"),
										},
									},
									InitialDelaySeconds: 10,
									PeriodSeconds:       5,
									TimeoutSeconds:      3,
									FailureThreshold:    3,
								},
								SecurityContext: &corev1.SecurityContext{
									AllowPrivilegeEscalation: &[]bool{true}[0],
									RunAsNonRoot:             &[]bool{false}[0],
									RunAsUser:                &[]int64{0}[0],
									Capabilities: &corev1.Capabilities{
										Add: []corev1.Capability{"NET_BIND_SERVICE"},
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "config",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: AcmeDNSName + "-config",
										},
									},
								},
							},
							{
								Name: "data",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: AcmeDNSName + "-data",
									},
								},
							},
						},
						Affinity: &corev1.Affinity{
							PodAntiAffinity: &corev1.PodAntiAffinity{
								PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
									{
										Weight: 100,
										PodAffinityTerm: corev1.PodAffinityTerm{
											LabelSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{
													"app": AcmeDNSName,
												},
											},
											TopologyKey: "kubernetes.io/hostname",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		if err := c.Create(ctx, deployment); err != nil {
			log.Error(err, "Failed to create ACME-DNS Deployment")
			return err
		}

		log.Info("ACME-DNS Deployment created successfully", "name", deploymentKey.Name)
	} else {
		log.Info("ACME-DNS Deployment already exists, triggering rollout restart", "name", deploymentKey.Name)

		// Trigger rollout restart by updating the deployment's restart annotation
		if err := triggerAcmeDNSRolloutRestart(ctx, c, deployment); err != nil {
			return fmt.Errorf("failed to trigger rollout restart: %w", err)
		}

		log.Info("ACME-DNS Deployment rollout restart triggered successfully", "name", deploymentKey.Name)
	}

	return nil
}

// triggerAcmeDNSRolloutRestart triggers a rollout restart of the ACME-DNS deployment
func triggerAcmeDNSRolloutRestart(ctx context.Context, c client.Client, deployment *appsv1.Deployment) error {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns-rollout-restart")

	// Add or update the restart annotation to trigger a rollout restart
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}

	// Use current timestamp to ensure the annotation value changes
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	log.Info("Updating ACME-DNS Deployment to trigger rollout restart",
		"deployment", deployment.Name,
		"restartedAt", deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"])

	if err := c.Update(ctx, deployment); err != nil {
		log.Error(err, "Failed to update ACME-DNS Deployment for rollout restart")
		return err
	}

	return nil
}

// waitForAcmeDNSReady waits for ACME-DNS deployment and LoadBalancer service to be ready
func waitForAcmeDNSReady(ctx context.Context, c client.Client) error {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns-wait")

	deploymentKey := client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName,
	}

	dnsServiceKey := client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSName + "-dns",
	}

	// Wait up to 5 minutes for both deployment and LoadBalancer service to be ready
	deadline := time.Now().Add(5 * time.Minute)
	deploymentReady := false
	loadBalancerReady := false

	for {
		// Check deployment readiness
		if !deploymentReady {
			deployment := &appsv1.Deployment{}
			err := c.Get(ctx, deploymentKey, deployment)
			if err != nil {
				if errors.IsNotFound(err) {
					log.Info("ACME-DNS deployment not found yet", "deployment", deploymentKey.Name)
				} else {
					log.Error(err, "Failed to check ACME-DNS deployment", "deployment", deploymentKey.Name)
					return fmt.Errorf("failed to check ACME-DNS deployment: %w", err)
				}
			} else {
				// Check if deployment is ready
				if deployment.Status.ReadyReplicas > 0 && deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
					log.Info("ACME-DNS deployment is ready", "deployment", deploymentKey.Name, "readyReplicas", deployment.Status.ReadyReplicas)
					deploymentReady = true
				} else {
					log.Info("ACME-DNS deployment not ready yet", "deployment", deploymentKey.Name,
						"readyReplicas", deployment.Status.ReadyReplicas, "desiredReplicas", *deployment.Spec.Replicas)
				}
			}
		}

		// Check LoadBalancer service readiness
		if !loadBalancerReady {
			service := &corev1.Service{}
			err := c.Get(ctx, dnsServiceKey, service)
			if err != nil {
				if errors.IsNotFound(err) {
					log.Info("ACME-DNS DNS LoadBalancer service not found yet", "service", dnsServiceKey.Name)
				} else {
					log.Error(err, "Failed to check ACME-DNS DNS LoadBalancer service", "service", dnsServiceKey.Name)
					return fmt.Errorf("failed to check ACME-DNS DNS LoadBalancer service: %w", err)
				}
			} else {
				// Check if LoadBalancer has external IP assigned
				if len(service.Status.LoadBalancer.Ingress) > 0 && service.Status.LoadBalancer.Ingress[0].IP != "" {
					externalIP := service.Status.LoadBalancer.Ingress[0].IP
					log.Info("ACME-DNS DNS LoadBalancer service has external IP", "service", dnsServiceKey.Name, "externalIP", externalIP)
					loadBalancerReady = true
				} else {
					log.Info("ACME-DNS DNS LoadBalancer service waiting for external IP", "service", dnsServiceKey.Name)
				}
			}
		}

		// If both are ready, we're done
		if deploymentReady && loadBalancerReady {
			log.Info("ACME-DNS is fully ready (deployment and LoadBalancer service)")
			break
		}

		// Check if we've exceeded deadline
		if time.Now().After(deadline) {
			if !deploymentReady {
				log.Error(nil, "ACME-DNS deployment did not become ready within timeout", "deployment", deploymentKey.Name, "timeout", "5m")
			}
			if !loadBalancerReady {
				log.Error(nil, "ACME-DNS DNS LoadBalancer service did not get external IP within timeout", "service", dnsServiceKey.Name, "timeout", "5m")
			}
			return fmt.Errorf("ACME-DNS did not become fully ready within 5 minutes")
		}

		// Wait before checking again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
			// Continue loop
		}
	}

	return nil
}

// ensureAcmeDNSAccount ensures an ACME-DNS account is registered and credentials are stored in a secret.
// This creates the "acme-dns-account" secret with "acmedns.json" key that cert-manager's ClusterIssuer expects.
// The secret format follows cert-manager's ACME-DNS integration requirements:
//
//	{
//	  "example.com": {
//	    "username": "...",
//	    "password": "...",
//	    "fulldomain": "...",
//	    "subdomain": "..."
//	  }
//	}
func ensureAcmeDNSAccount(ctx context.Context, c client.Client, baseDomain string) error {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns-account")

	// Check if account secret already exists
	log.Info("Checking if ACME-DNS account secret already exists", "secret", AcmeDNSAccountSecretName, "namespace", AcmeDNSNamespace)
	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: AcmeDNSNamespace,
		Name:      AcmeDNSAccountSecretName,
	}

	if err := c.Get(ctx, secretKey, secret); err == nil {
		// Secret already exists, verify it has the required key
		if _, ok := secret.Data["acmedns.json"]; ok {
			log.Info("ACME-DNS account secret already exists with valid credentials", "secret", AcmeDNSAccountSecretName)
			return nil
		}
		log.Info("ACME-DNS account secret exists but missing acmedns.json key, will recreate", "secret", AcmeDNSAccountSecretName)
	} else if !errors.IsNotFound(err) {
		log.Error(err, "Failed to check ACME-DNS account secret", "secret", AcmeDNSAccountSecretName)
		return fmt.Errorf("failed to check ACME-DNS account secret: %w", err)
	}

	// Register new ACME-DNS account
	log.Info("Registering new ACME-DNS account", "domain", baseDomain)
	account, err := registerAcmeDNSAccount(ctx, baseDomain)
	if err != nil {
		log.Error(err, "Failed to register ACME-DNS account")
		return fmt.Errorf("failed to register ACME-DNS account: %w", err)
	}

	log.Info("ACME-DNS account registered successfully",
		"username", account.Username,
		"subdomain", account.Subdomain,
		"fulldomain", account.FullDomain)

	// Create the credentials JSON for cert-manager
	credentialsJSON, err := json.Marshal(map[string]AcmeDNSAccount{
		baseDomain: *account,
	})
	if err != nil {
		log.Error(err, "Failed to marshal ACME-DNS credentials")
		return fmt.Errorf("failed to marshal ACME-DNS credentials: %w", err)
	}

	// Create or update the secret
	log.Info("Creating ACME-DNS account secret", "secret", AcmeDNSAccountSecretName, "namespace", AcmeDNSNamespace)
	secret = &corev1.Secret{
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
			"acmedns.json": credentialsJSON,
		},
	}

	// Try to create the secret, if it exists, update it
	if err := c.Create(ctx, secret); err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info("ACME-DNS account secret already exists, updating", "secret", AcmeDNSAccountSecretName)
			if err := c.Update(ctx, secret); err != nil {
				log.Error(err, "Failed to update ACME-DNS account secret", "secret", AcmeDNSAccountSecretName)
				return fmt.Errorf("failed to update ACME-DNS account secret: %w", err)
			}
		} else {
			log.Error(err, "Failed to create ACME-DNS account secret", "secret", AcmeDNSAccountSecretName)
			return fmt.Errorf("failed to create ACME-DNS account secret: %w", err)
		}
	}

	log.Info("ACME-DNS account secret created/updated successfully", "secret", AcmeDNSAccountSecretName, "namespace", AcmeDNSNamespace)
	return nil
}

// registerAcmeDNSAccount registers a new account with the ACME-DNS service
func registerAcmeDNSAccount(ctx context.Context, baseDomain string) (*AcmeDNSAccount, error) {
	log := ctrl.Log.WithName("bootstrap").WithName("acme-dns-register")

	// ACME-DNS service URL (internal cluster service)
	acmeDNSURL := "http://acme-dns.kibaship.svc.cluster.local/register"

	// Create registration request (no allowfrom restrictions since it's internal)
	reqBody := AcmeDNSRegistrationRequest{}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal registration request: %w", err)
	}

	log.Info("Sending registration request to ACME-DNS", "url", acmeDNSURL)

	// Create HTTP request with timeout
	req, err := http.NewRequestWithContext(ctx, "POST", acmeDNSURL, bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send the request with retries
	var resp *http.Response
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		resp, err = client.Do(req)
		if err == nil {
			break
		}

		log.Info("ACME-DNS registration request failed, retrying", "attempt", i+1, "maxRetries", maxRetries, "error", err.Error())
		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(i+1) * 2 * time.Second):
				// Exponential backoff: 2s, 4s, 6s, 8s
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to send registration request after %d retries: %w", maxRetries, err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusCreated {
		log.Error(nil, "ACME-DNS registration failed", "statusCode", resp.StatusCode, "response", string(respBody))
		return nil, fmt.Errorf("ACME-DNS registration failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var account AcmeDNSAccount
	if err := json.Unmarshal(respBody, &account); err != nil {
		return nil, fmt.Errorf("failed to parse registration response: %w", err)
	}

	log.Info("ACME-DNS account registration successful",
		"username", account.Username,
		"subdomain", account.Subdomain,
		"fulldomain", account.FullDomain)

	return &account, nil
}
