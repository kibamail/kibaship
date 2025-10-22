package bootstrap

import (
	"context"
	"fmt"
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
	AcmeDNSNamespace = "kibaship"
	AcmeDNSName      = "acme-dns"
	AcmeDNSVersion   = "v1.0"
)

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
