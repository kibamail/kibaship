/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/utils"
)

const (
	// ResourceProfileStandard represents the standard resource profile
	ResourceProfileStandard = "standard"
)

// DeploymentProgressController manages phase transitions and K8s resource creation
type DeploymentProgressController struct {
	client.Client
	Scheme           *runtime.Scheme
	NamespaceManager *NamespaceManager
}

// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applications,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applicationdomains,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *DeploymentProgressController) SetupWithManager(mgr ctrl.Manager) error {
	// Watch for condition changes (which come from PipelineRunWatcherReconciler and DeploymentStatusWatcherReconciler)
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Deployment{}).
		WithEventFilter(predicate.Or(
			predicate.GenerationChangedPredicate{},
			conditionChangedPredicate{}, // Custom predicate for condition changes
		)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 20,
		}).
		Named("deployment-progress").
		Complete(r)
}

func (r *DeploymentProgressController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var deployment platformv1alpha1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &deployment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Capture current phase immediately after Get() to avoid cache inconsistencies in logs
	currentPhase := deployment.Status.Phase

	// Get the Application to determine type
	var app platformv1alpha1.Application
	if err := r.Get(ctx, client.ObjectKey{
		Name:      deployment.Spec.ApplicationRef.Name,
		Namespace: deployment.Namespace,
	}, &app); err != nil {
		log.Error(err, "Failed to get Application")
		return ctrl.Result{}, err
	}

	// State machine: Determine target phase based on application type and conditions
	targetPhase := r.computeTargetPhase(&deployment, &app)

	if currentPhase == targetPhase {
		// Already in correct phase
		return ctrl.Result{}, nil
	}

	log.Info("Phase transition",
		"from", currentPhase,
		"to", targetPhase)

	// Perform phase-specific actions
	switch targetPhase {
	case platformv1alpha1.DeploymentPhaseDeploying:
		// PipelineRun succeeded - create K8s resources
		if err := r.createKubernetesResources(ctx, &deployment); err != nil {
			return ctrl.Result{}, err
		}

	case platformv1alpha1.DeploymentPhaseSucceeded:
		// K8s resources created and ready
		// Check if this deployment should be promoted
		// Promote if: deployment.Spec.Promote is true OR no current deployment exists
		if err := r.checkAndPromoteDeployment(ctx, &deployment); err != nil {
			log.Error(err, "Failed to check and promote deployment")
			return ctrl.Result{}, err
		}

	case platformv1alpha1.DeploymentPhaseFailed:
		// PipelineRun failed
		// No action needed (could emit events here)
	}

	// Update phase
	deployment.Status.Phase = targetPhase
	if err := r.Status().Update(ctx, &deployment); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// computeTargetPhase - State machine logic based on application type
func (r *DeploymentProgressController) computeTargetPhase(
	deployment *platformv1alpha1.Deployment,
	app *platformv1alpha1.Application,
) platformv1alpha1.DeploymentPhase {
	// Handle different application types
	switch app.Spec.Type {
	case platformv1alpha1.ApplicationTypeGitRepository:
		return r.computeTargetPhaseForGitRepository(deployment)
	case platformv1alpha1.ApplicationTypeImageFromRegistry:
		return r.computeTargetPhaseForImageFromRegistry(deployment)
	case platformv1alpha1.ApplicationTypeMySQL,
		platformv1alpha1.ApplicationTypeMySQLCluster,
		platformv1alpha1.ApplicationTypeValkey,
		platformv1alpha1.ApplicationTypeValkeyCluster,
		platformv1alpha1.ApplicationTypePostgres,
		platformv1alpha1.ApplicationTypePostgresCluster:
		// TODO: Database application type progress tracking will be reimplemented
		// Current implementation removed
		return platformv1alpha1.DeploymentPhaseDeploying
	default:
		// Unknown application type - stay in initializing
		return platformv1alpha1.DeploymentPhaseInitializing
	}
}

// computeTargetPhaseForGitRepository handles GitRepository applications (existing logic)
func (r *DeploymentProgressController) computeTargetPhaseForGitRepository(
	deployment *platformv1alpha1.Deployment,
) platformv1alpha1.DeploymentPhase {
	// Check PipelineRun condition
	prCondition := meta.FindStatusCondition(deployment.Status.Conditions, "PipelineRunReady")

	if prCondition == nil {
		return platformv1alpha1.DeploymentPhaseInitializing
	}

	switch prCondition.Status {
	case metav1.ConditionTrue:
		// PipelineRun succeeded - check K8s Deployment readiness
		k8sCondition := meta.FindStatusCondition(deployment.Status.Conditions, "K8sDeploymentReady")

		if k8sCondition != nil {
			// Check for crash loop - if detected, mark as Failed
			if k8sCondition.Reason == "CrashLoopBackOff" {
				return platformv1alpha1.DeploymentPhaseFailed
			}

			// Check if pods are ready
			if k8sCondition.Status == metav1.ConditionTrue {
				return platformv1alpha1.DeploymentPhaseSucceeded
			}
		}

		// Resources created but pods not ready yet (or condition not set)
		return platformv1alpha1.DeploymentPhaseDeploying

	case metav1.ConditionFalse:
		return platformv1alpha1.DeploymentPhaseFailed

	default:
		// PipelineRun running
		return platformv1alpha1.DeploymentPhaseBuilding
	}
}

// computeTargetPhaseForImageFromRegistry handles ImageFromRegistry applications
func (r *DeploymentProgressController) computeTargetPhaseForImageFromRegistry(
	deployment *platformv1alpha1.Deployment,
) platformv1alpha1.DeploymentPhase {
	// For ImageFromRegistry, we only need to check K8s Deployment readiness
	// No PipelineRun is involved
	k8sCondition := meta.FindStatusCondition(deployment.Status.Conditions, "K8sDeploymentReady")

	if k8sCondition == nil {
		// No condition set yet - still initializing
		return platformv1alpha1.DeploymentPhaseInitializing
	}

	// Check for crash loop - if detected, mark as Failed
	if k8sCondition.Reason == "CrashLoopBackOff" {
		return platformv1alpha1.DeploymentPhaseFailed
	}

	// Determine phase based on K8s Deployment condition
	switch k8sCondition.Status {
	case metav1.ConditionTrue:
		// Pods are ready - deployment succeeded
		return platformv1alpha1.DeploymentPhaseSucceeded

	case metav1.ConditionFalse:
		// Check if this is a permanent failure or just deploying
		if k8sCondition.Reason == "DeploymentNotReady" || k8sCondition.Reason == "PodsNotReady" {
			// Still deploying - pods not ready yet
			return platformv1alpha1.DeploymentPhaseDeploying
		}
		// Other false conditions might indicate failure
		return platformv1alpha1.DeploymentPhaseFailed

	default:
		// Unknown status - still deploying
		return platformv1alpha1.DeploymentPhaseDeploying
	}
}

// TODO: computeTargetPhaseForMySQL - MySQL progress tracking will be reimplemented
func (r *DeploymentProgressController) computeTargetPhaseForMySQL(
	deployment *platformv1alpha1.Deployment,
) platformv1alpha1.DeploymentPhase {
	// TODO: Implement new MySQL progress tracking logic here
	return platformv1alpha1.DeploymentPhaseDeploying
}

// TODO: computeTargetPhaseForMySQLCluster - MySQL cluster progress tracking will be reimplemented
func (r *DeploymentProgressController) computeTargetPhaseForMySQLCluster(
	deployment *platformv1alpha1.Deployment,
) platformv1alpha1.DeploymentPhase {
	// TODO: Implement new MySQL cluster progress tracking logic here
	return platformv1alpha1.DeploymentPhaseDeploying
}

// TODO: computeTargetPhaseForValkey - Valkey progress tracking will be reimplemented
func (r *DeploymentProgressController) computeTargetPhaseForValkey(
	deployment *platformv1alpha1.Deployment,
) platformv1alpha1.DeploymentPhase {
	// TODO: Implement new Valkey progress tracking logic here
	return platformv1alpha1.DeploymentPhaseDeploying
}

// TODO: computeTargetPhaseForValkeyCluster - Valkey cluster progress tracking will be reimplemented
func (r *DeploymentProgressController) computeTargetPhaseForValkeyCluster(
	deployment *platformv1alpha1.Deployment,
) platformv1alpha1.DeploymentPhase {
	// TODO: Implement new Valkey cluster progress tracking logic here
	return platformv1alpha1.DeploymentPhaseDeploying
}

// checkAndPromoteDeployment checks if a deployment should be promoted and promotes it if needed
// Promotion happens if:
// 1. deployment.Spec.Promote is true, OR
// 2. application has no CurrentDeploymentRef (first successful deployment)
func (r *DeploymentProgressController) checkAndPromoteDeployment(
	ctx context.Context,
	deployment *platformv1alpha1.Deployment,
) error {
	log := ctrl.LoggerFrom(ctx)

	// Get the Application
	var app platformv1alpha1.Application
	if err := r.Get(ctx, client.ObjectKey{
		Name:      deployment.Spec.ApplicationRef.Name,
		Namespace: deployment.Namespace,
	}, &app); err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}

	// Check if already promoted
	if app.Spec.CurrentDeploymentRef != nil && app.Spec.CurrentDeploymentRef.Name == deployment.Name {
		log.V(1).Info("Deployment already promoted")
		return nil
	}

	// Determine if promotion should happen
	shouldPromote := false
	reason := ""

	if deployment.Spec.Promote {
		shouldPromote = true
		reason = "deployment has promote=true"
	} else if app.Spec.CurrentDeploymentRef == nil {
		shouldPromote = true
		reason = "no current deployment exists (first successful deployment)"
	}

	if !shouldPromote {
		log.V(1).Info("Deployment not promoted",
			"promote", deployment.Spec.Promote,
			"hasCurrentDeployment", app.Spec.CurrentDeploymentRef != nil)
		return nil
	}

	// Update CurrentDeploymentRef
	app.Spec.CurrentDeploymentRef = &corev1.LocalObjectReference{
		Name: deployment.Name,
	}

	if err := r.Update(ctx, &app); err != nil {
		return fmt.Errorf("failed to update application currentDeploymentRef: %w", err)
	}

	log.Info("Successfully promoted deployment",
		"deployment", deployment.Name,
		"application", app.Name,
		"reason", reason)
	return nil
}

func (r *DeploymentProgressController) createKubernetesResources(
	ctx context.Context,
	deployment *platformv1alpha1.Deployment,
) error {
	log := ctrl.LoggerFrom(ctx)

	// Get Application for configuration
	var app platformv1alpha1.Application
	if err := r.Get(ctx, client.ObjectKey{
		Name:      deployment.Spec.ApplicationRef.Name,
		Namespace: deployment.Namespace,
	}, &app); err != nil {
		return err
	}

	// Support GitRepository, ImageFromRegistry, and Dockerfile applications
	supportedTypes := []platformv1alpha1.ApplicationType{
		platformv1alpha1.ApplicationTypeGitRepository,
		platformv1alpha1.ApplicationTypeImageFromRegistry,
		platformv1alpha1.ApplicationTypeDockerImage, // Dockerfile applications
	}

	isSupported := false
	for _, supportedType := range supportedTypes {
		if app.Spec.Type == supportedType {
			isSupported = true
			break
		}
	}

	if !isSupported {
		log.Info("Skipping K8s resource creation for unsupported application type", "type", app.Spec.Type)
		return nil
	}

	// Create K8s Deployment (idempotent)
	if err := r.ensureKubernetesDeployment(ctx, deployment, &app); err != nil {
		return fmt.Errorf("failed to create Kubernetes Deployment: %w", err)
	}

	// Create Service (idempotent)
	if err := r.ensureKubernetesService(ctx, deployment, &app); err != nil {
		return fmt.Errorf("failed to create Kubernetes Service: %w", err)
	}

	// Create ApplicationDomain (idempotent)
	if err := r.ensureApplicationDomain(ctx, deployment, &app); err != nil {
		return fmt.Errorf("failed to create ApplicationDomain: %w", err)
	}

	// Create HTTPRoute for this specific deployment (idempotent)
	if err := r.ensureDeploymentHTTPRoute(ctx, deployment, &app); err != nil {
		return fmt.Errorf("failed to create deployment HTTPRoute: %w", err)
	}

	// Create/Update HTTPRoute for the main application using currentDeploymentRef (idempotent)
	if err := r.ensureApplicationHTTPRoute(ctx, deployment, &app); err != nil {
		return fmt.Errorf("failed to create/update application HTTPRoute: %w", err)
	}

	log.Info("Kubernetes resources created")
	return nil
}

// ensureKubernetesDeployment creates K8s Deployment if not exists
func (r *DeploymentProgressController) ensureKubernetesDeployment(
	ctx context.Context,
	deployment *platformv1alpha1.Deployment,
	app *platformv1alpha1.Application,
) error {
	log := ctrl.LoggerFrom(ctx)

	k8sDepName := utils.GetKubernetesDeploymentName(deployment.GetUUID())

	var existing appsv1.Deployment
	err := r.Get(ctx, client.ObjectKey{
		Name:      k8sDepName,
		Namespace: deployment.Namespace,
	}, &existing)

	if err == nil {
		log.V(1).Info("K8s Deployment already exists", "name", k8sDepName)
		return nil // Already exists
	}

	if !errors.IsNotFound(err) {
		return err
	}

	// Get project for resource limits
	var project platformv1alpha1.Project
	projectName := utils.GetProjectResourceName(deployment.GetProjectUUID())
	if err := r.Get(ctx, client.ObjectKey{Name: projectName}, &project); err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Derive image name based on application type
	var imageName string
	switch app.Spec.Type {
	case platformv1alpha1.ApplicationTypeGitRepository, platformv1alpha1.ApplicationTypeDockerImage:
		// For GitRepository and Dockerfile apps, use built image from registry
		imageName = fmt.Sprintf("registry.registry.svc.cluster.local/%s/%s:%s",
			deployment.Namespace,
			deployment.GetApplicationUUID(),
			deployment.GetUUID())
	case platformv1alpha1.ApplicationTypeImageFromRegistry:
		// For ImageFromRegistry apps, use the specified image
		if deployment.Spec.ImageFromRegistry == nil {
			return fmt.Errorf("ImageFromRegistry deployment missing image configuration")
		}
		imageName = fmt.Sprintf("%s/%s:%s",
			app.Spec.ImageFromRegistry.Registry,
			app.Spec.ImageFromRegistry.Repository,
			deployment.Spec.ImageFromRegistry.Tag)
	default:
		return fmt.Errorf("unsupported application type for K8s Deployment creation: %s", app.Spec.Type)
	}

	// Determine container port from application spec (default 3000)
	containerPort := r.getApplicationPort(app)
	if containerPort == 0 {
		containerPort = 3000
	}

	// Resource profile (default to standard)
	resourceProfile := ResourceProfileStandard

	// Define resource requirements
	var resourceRequirements corev1.ResourceRequirements
	switch resourceProfile {
	case "minimal":
		resourceRequirements = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		}
	case "performance":
		resourceRequirements = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("2000m"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
		}
	default: // ResourceProfileStandard
		resourceRequirements = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("250m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		}
	}

	replicas := int32(1)
	appUUID := app.GetUUID()

	k8sDep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sDepName,
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                 fmt.Sprintf("app-%s", appUUID),
				"app.kubernetes.io/managed-by":           "kibaship",
				"app.kubernetes.io/component":            "application",
				"platform.kibaship.com/deployment-uuid":  deployment.GetUUID(),
				"platform.kibaship.com/application-uuid": app.GetUUID(),
				"platform.kibaship.com/project-uuid":     deployment.GetProjectUUID(),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":                fmt.Sprintf("app-%s", appUUID),
					"platform.kibaship.com/deployment-uuid": deployment.GetUUID(),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":                 fmt.Sprintf("app-%s", appUUID),
						"app.kubernetes.io/managed-by":           "kibaship",
						"app.kubernetes.io/component":            "application",
						"platform.kibaship.com/deployment-uuid":  deployment.GetUUID(),
						"platform.kibaship.com/application-uuid": app.GetUUID(),
						"platform.kibaship.com/project-uuid":     deployment.GetProjectUUID(),
					},
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "registry-image-pull-secret"},
					},
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: imageName,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: containerPort,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: utils.GetDeploymentResourceName(deployment.GetUUID()),
										},
									},
								},
							},
							Resources: resourceRequirements,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "registry-ca-cert",
									MountPath: "/etc/ssl/certs/registry-ca.crt",
									SubPath:   "ca.crt",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "registry-ca-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "registry-ca-cert",
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
				},
			},
		},
	}

	// Set owner reference to Deployment CR
	if err := ctrl.SetControllerReference(deployment, k8sDep, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	if err := r.Create(ctx, k8sDep); err != nil {
		return fmt.Errorf("failed to create K8s Deployment: %w", err)
	}

	log.Info("Created K8s Deployment", "name", k8sDepName, "image", imageName)
	return nil
}

// ensureKubernetesService creates Service if not exists
func (r *DeploymentProgressController) ensureKubernetesService(
	ctx context.Context,
	deployment *platformv1alpha1.Deployment,
	app *platformv1alpha1.Application,
) error {
	log := ctrl.LoggerFrom(ctx)

	appUUID := app.GetUUID()
	serviceName := utils.GetServiceName(appUUID)

	var existing corev1.Service
	err := r.Get(ctx, client.ObjectKey{
		Name:      serviceName,
		Namespace: deployment.Namespace,
	}, &existing)

	if err == nil {
		log.V(1).Info("Service already exists", "name", serviceName)
		return nil // Already exists
	}

	if !errors.IsNotFound(err) {
		return err
	}

	// Determine container port (default 3000)
	containerPort := int32(3000)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                 fmt.Sprintf("project-%s", app.GetProjectUUID()),
				"app.kubernetes.io/managed-by":           "kibaship",
				"app.kubernetes.io/component":            "application-service",
				"platform.kibaship.com/application-uuid": app.GetUUID(),
				"platform.kibaship.com/project-uuid":     app.GetProjectUUID(),
				"platform.kibaship.com/deployment-uuid":  deployment.GetUUID(),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app.kubernetes.io/name":                 fmt.Sprintf("app-%s", appUUID),
				"platform.kibaship.com/application-uuid": app.GetUUID(),
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       containerPort,
					TargetPort: intstr.FromInt32(containerPort),
				},
			},
		},
	}

	// Set owner reference to Deployment CR
	if err := ctrl.SetControllerReference(deployment, service, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	if err := r.Create(ctx, service); err != nil {
		return fmt.Errorf("failed to create Service: %w", err)
	}

	log.Info("Created Service", "name", serviceName, "port", containerPort)
	return nil
}

// ensureApplicationDomain creates a deployment-specific ApplicationDomain using deployment UUID
func (r *DeploymentProgressController) ensureApplicationDomain(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application) error {
	log := ctrl.LoggerFrom(ctx)

	deploymentUUID := deployment.GetUUID()
	appUUID := app.GetUUID()

	// Check if this deployment's domain already exists
	domainName := utils.GetApplicationDomainResourceName(deploymentUUID)
	var existing platformv1alpha1.ApplicationDomain
	err := r.Get(ctx, client.ObjectKey{
		Name:      domainName,
		Namespace: deployment.Namespace,
	}, &existing)

	if err == nil {
		log.V(1).Info("ApplicationDomain already exists for this deployment", "domain", existing.Spec.Domain)
		return nil // Already exists
	}

	if !errors.IsNotFound(err) {
		return err
	}

	// Get operator configuration for base domain
	opConfig, err := GetOperatorConfig()
	if err != nil {
		return fmt.Errorf("failed to get operator configuration: %w", err)
	}

	// Determine domain pattern based on Application type
	// Deployment domains use the deployment UUID: <deployment-uuid>.apps.<baseDomain>
	var domain string
	var port int32 = 3000 // Default port

	switch app.Spec.Type {
	case platformv1alpha1.ApplicationTypeGitRepository, platformv1alpha1.ApplicationTypeDockerImage:
		// Web applications use <deployment-uuid>.apps.<baseDomain>
		domain = fmt.Sprintf("%s.apps.%s", deploymentUUID, opConfig.Domain)
	default:
		return fmt.Errorf("unsupported application type for per-deployment domain creation: %s", app.Spec.Type)
	}

	// Generate slug for ApplicationDomain
	domainSlug, err := utils.GenerateRandomSlug()
	if err != nil {
		return fmt.Errorf("failed to generate domain slug: %w", err)
	}

	// Create ApplicationDomain CR
	applicationDomain := &platformv1alpha1.ApplicationDomain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      domainName,
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"platform.kibaship.com/uuid":             deploymentUUID,
				"platform.kibaship.com/slug":             domainSlug,
				"platform.kibaship.com/project-uuid":     app.GetProjectUUID(),
				"platform.kibaship.com/application-uuid": appUUID,
				"platform.kibaship.com/deployment-uuid":  deploymentUUID,
			},
			Annotations: map[string]string{
				"platform.kibaship.com/resource-name": fmt.Sprintf("Deployment domain %s", domain),
			},
		},
		Spec: platformv1alpha1.ApplicationDomainSpec{
			ApplicationRef: corev1.LocalObjectReference{
				Name: app.Name,
			},
			Domain:     domain,
			Port:       port,
			Type:       platformv1alpha1.ApplicationDomainTypeDefault,
			Default:    false, // Deployment domains are not default
			TLSEnabled: true,
		},
	}

	// Set owner reference to the Deployment CR for cascading deletion
	if err := ctrl.SetControllerReference(deployment, applicationDomain, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference on ApplicationDomain: %w", err)
	}

	if err := r.Create(ctx, applicationDomain); err != nil {
		return fmt.Errorf("failed to create ApplicationDomain: %w", err)
	}

	log.Info("Successfully created deployment-specific ApplicationDomain", "domain", domain, "port", port)
	return nil
}

// ensureDeploymentHTTPRoute creates an HTTPRoute for this specific deployment
func (r *DeploymentProgressController) ensureDeploymentHTTPRoute(
	ctx context.Context,
	deployment *platformv1alpha1.Deployment,
	app *platformv1alpha1.Application,
) error {
	log := ctrl.LoggerFrom(ctx)

	// Get operator configuration for domain
	opConfig, err := GetOperatorConfig()
	if err != nil {
		return fmt.Errorf("failed to get operator config: %w", err)
	}

	if opConfig.Domain == "" {
		log.Info("No base domain configured, skipping HTTPRoute creation")
		return nil
	}

	deploymentUUID := deployment.GetUUID()
	appUUID := app.GetUUID()

	// Generate deployment-specific domain: <deployment-uuid>.apps.<baseDomain>
	deploymentDomain := fmt.Sprintf("%s.apps.%s", deploymentUUID, opConfig.Domain)

	// Determine service name and port
	serviceName := utils.GetServiceName(appUUID)
	servicePort := r.getApplicationPort(app)
	if servicePort == 0 {
		servicePort = 3000 // Default port
	}

	// Create HTTPRoute for HTTPS traffic
	httpsRouteName := fmt.Sprintf("httproute-%s", deploymentUUID)
	if err := r.createHTTPRoute(ctx, deployment.Namespace, httpsRouteName, deploymentDomain, serviceName, servicePort, "https", deployment); err != nil {
		return fmt.Errorf("failed to create HTTPS HTTPRoute: %w", err)
	}

	// Create HTTPRoute for HTTP->HTTPS redirect
	httpRouteName := fmt.Sprintf("httproute-%s-redirect", deploymentUUID)
	if err := r.createHTTPRedirectRoute(ctx, deployment.Namespace, httpRouteName, deploymentDomain, deployment); err != nil {
		return fmt.Errorf("failed to create HTTP redirect HTTPRoute: %w", err)
	}

	log.Info("Created HTTPRoutes for deployment", "domain", deploymentDomain, "service", serviceName, "port", servicePort)
	return nil
}

// ensureApplicationHTTPRoute creates/updates HTTPRoute for the main application using currentDeploymentRef
func (r *DeploymentProgressController) ensureApplicationHTTPRoute(
	ctx context.Context,
	deployment *platformv1alpha1.Deployment,
	app *platformv1alpha1.Application,
) error {
	log := ctrl.LoggerFrom(ctx)

	// Check if this deployment is the current one
	if app.Spec.CurrentDeploymentRef == nil {
		log.Error(nil, "Application has no currentDeploymentRef set, cannot create application HTTPRoute",
			"application", app.Name, "deployment", deployment.Name)
		return nil // Gracefully degrade - don't fail the deployment
	}

	if app.Spec.CurrentDeploymentRef.Name != deployment.Name {
		log.Info("This deployment is not the current deployment, skipping application HTTPRoute creation",
			"deployment", deployment.Name, "currentDeployment", app.Spec.CurrentDeploymentRef.Name)
		return nil // This is not the current deployment
	}

	// Get operator configuration for domain
	opConfig, err := GetOperatorConfig()
	if err != nil {
		return fmt.Errorf("failed to get operator config: %w", err)
	}

	if opConfig.Domain == "" {
		log.Info("No base domain configured, skipping application HTTPRoute creation")
		return nil
	}

	// Find the default ApplicationDomain for this application
	var domains platformv1alpha1.ApplicationDomainList
	if err := r.List(ctx, &domains,
		client.InNamespace(app.Namespace),
		client.MatchingLabels{"platform.kibaship.com/application-uuid": app.GetUUID()},
	); err != nil {
		return fmt.Errorf("failed to list ApplicationDomains: %w", err)
	}

	var defaultDomain *platformv1alpha1.ApplicationDomain
	for _, domain := range domains.Items {
		if domain.Spec.Default {
			defaultDomain = &domain
			break
		}
	}

	if defaultDomain == nil {
		log.Info("No default ApplicationDomain found for application, skipping application HTTPRoute creation",
			"application", app.Name)
		return nil // Gracefully degrade
	}

	// Determine service name and port for the current deployment
	serviceName := utils.GetServiceName(app.GetUUID())
	servicePort := defaultDomain.Spec.Port

	// Create HTTPRoute for HTTPS traffic
	httpsRouteName := fmt.Sprintf("httproute-app-%s", app.GetUUID())
	if err := r.createHTTPRoute(ctx, app.Namespace, httpsRouteName, defaultDomain.Spec.Domain, serviceName, servicePort, "https", deployment); err != nil {
		return fmt.Errorf("failed to create application HTTPS HTTPRoute: %w", err)
	}

	// Create HTTPRoute for HTTP->HTTPS redirect
	httpRouteName := fmt.Sprintf("httproute-app-%s-redirect", app.GetUUID())
	if err := r.createHTTPRedirectRoute(ctx, app.Namespace, httpRouteName, defaultDomain.Spec.Domain, deployment); err != nil {
		return fmt.Errorf("failed to create application HTTP redirect HTTPRoute: %w", err)
	}

	log.Info("Created/updated application HTTPRoutes", "domain", defaultDomain.Spec.Domain, "service", serviceName, "port", servicePort)
	return nil
}

// Custom predicate to detect condition changes
type conditionChangedPredicate struct{}

func (conditionChangedPredicate) Create(e event.CreateEvent) bool {
	return true
}

func (conditionChangedPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (conditionChangedPredicate) Update(e event.UpdateEvent) bool {
	oldDep, ok := e.ObjectOld.(*platformv1alpha1.Deployment)
	if !ok {
		return false
	}
	newDep, ok := e.ObjectNew.(*platformv1alpha1.Deployment)
	if !ok {
		return false
	}

	// Trigger if conditions changed
	return !conditionsEqual(oldDep.Status.Conditions, newDep.Status.Conditions)
}

func (conditionChangedPredicate) Generic(e event.GenericEvent) bool {
	return false
}

func conditionsEqual(a, b []metav1.Condition) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Type != b[i].Type ||
			a[i].Status != b[i].Status ||
			a[i].Reason != b[i].Reason {
			return false
		}
	}
	return true
}

// createHTTPRoute creates an HTTPRoute for HTTPS traffic
func (r *DeploymentProgressController) createHTTPRoute(
	ctx context.Context,
	namespace, routeName, hostname, serviceName string,
	servicePort int32,
	listenerName string,
	owner metav1.Object,
) error {
	log := ctrl.LoggerFrom(ctx)

	// Check if HTTPRoute already exists
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	})

	if err := r.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      routeName,
	}, obj); err == nil {
		log.V(1).Info("HTTPRoute already exists", "name", routeName)
		return nil // Already exists
	} else if !errors.IsNotFound(err) {
		return err
	}

	// Create new HTTPRoute
	obj.SetNamespace(namespace)
	obj.SetName(routeName)

	// Set labels
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "kibaship",
		"platform.kibaship.com/type":   "httproute",
	}
	obj.SetLabels(labels)

	// Set owner reference for cleanup
	if err := ctrl.SetControllerReference(owner, obj, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	obj.Object["spec"] = map[string]any{
		"parentRefs": []any{
			map[string]any{
				"name":        "kibaship-gateway",
				"namespace":   "kibaship",
				"sectionName": listenerName,
			},
		},
		"hostnames": []any{hostname},
		"rules": []any{
			map[string]any{
				"matches": []any{
					map[string]any{
						"path": map[string]any{
							"type":  "PathPrefix",
							"value": "/",
						},
					},
				},
				"backendRefs": []any{
					map[string]any{
						"name": serviceName,
						"port": int64(servicePort),
					},
				},
			},
		},
	}

	if err := r.Create(ctx, obj); err != nil {
		return fmt.Errorf("failed to create HTTPRoute: %w", err)
	}

	log.Info("Created HTTPRoute", "name", routeName, "hostname", hostname, "service", serviceName, "port", servicePort)
	return nil
}

// createHTTPRedirectRoute creates an HTTPRoute for HTTP->HTTPS redirect
func (r *DeploymentProgressController) createHTTPRedirectRoute(
	ctx context.Context,
	namespace, routeName, hostname string,
	owner metav1.Object,
) error {
	log := ctrl.LoggerFrom(ctx)

	// Check if HTTPRoute already exists
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	})

	if err := r.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      routeName,
	}, obj); err == nil {
		log.V(1).Info("HTTP redirect HTTPRoute already exists", "name", routeName)
		return nil // Already exists
	} else if !errors.IsNotFound(err) {
		return err
	}

	// Create new HTTPRoute for redirect
	obj.SetNamespace(namespace)
	obj.SetName(routeName)

	// Set labels
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "kibaship",
		"platform.kibaship.com/type":   "httproute-redirect",
	}
	obj.SetLabels(labels)

	// Set owner reference for cleanup
	if err := ctrl.SetControllerReference(owner, obj, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	obj.Object["spec"] = map[string]any{
		"parentRefs": []any{
			map[string]any{
				"name":        "kibaship-gateway",
				"namespace":   "kibaship",
				"sectionName": "http",
			},
		},
		"hostnames": []any{hostname},
		"rules": []any{
			map[string]any{
				"matches": []any{
					map[string]any{
						"path": map[string]any{
							"type":  "PathPrefix",
							"value": "/",
						},
					},
				},
				"filters": []any{
					map[string]any{
						"type": "RequestRedirect",
						"requestRedirect": map[string]any{
							"scheme": "https",
						},
					},
				},
			},
		},
	}

	if err := r.Create(ctx, obj); err != nil {
		return fmt.Errorf("failed to create HTTP redirect HTTPRoute: %w", err)
	}

	log.Info("Created HTTP redirect HTTPRoute", "name", routeName, "hostname", hostname)
	return nil
}

// getApplicationPort returns the port for the application from app.Spec.Port
func (r *DeploymentProgressController) getApplicationPort(app *platformv1alpha1.Application) int32 {
	return app.Spec.Port
}
