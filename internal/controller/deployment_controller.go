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
	"crypto/sha256"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/config"
	"github.com/kibamail/kibaship-operator/pkg/utils"
	"github.com/kibamail/kibaship-operator/pkg/webhooks"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	// DeploymentFinalizerName is the finalizer name for Deployment resources
	DeploymentFinalizerName = "platform.operator.kibaship.com/deployment-finalizer"
	// GitRepositoryPipelineName is the suffix for git repository pipeline names
	GitRepositoryPipelineSuffix = "git-repository-pipeline"
	// GitCloneTaskName is the name of the git clone task in tekton-pipelines namespace
	GitCloneTaskName = "tekton-task-git-clone-kibaship-com"
	// RailpackPrepareTaskName is the name of the railpack prepare task in tekton-pipelines namespace
	RailpackPrepareTaskName = "tekton-task-railpack-prepare-kibaship-com"
	// RailpackBuildTaskName is the name of the railpack build task in tekton-pipelines namespace
	RailpackBuildTaskName = "tekton-task-railpack-build-kibaship-com"
	// DefaultGitBranch is the default git branch when none is specified
	DefaultGitBranch = "main"
)

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	NamespaceManager *NamespaceManager
	Notifier         webhooks.Notifier
	Recorder         record.EventRecorder
}

// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applications,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applicationdomains,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects,verbs=get;list;watch
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tekton.dev,resources=taskruns,verbs=get;list;watch
// +kubebuilder:rbac:groups=mysql.oracle.com,resources=innodbclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hyperspike.io,resources=valkeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Deployment instance
	var deployment platformv1alpha1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &deployment); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Deployment not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if deployment.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, &deployment)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&deployment, DeploymentFinalizerName) {
		controllerutil.AddFinalizer(&deployment, DeploymentFinalizerName)
		if err := r.Update(ctx, &deployment); err != nil {
			log.Error(err, "Failed to add finalizer to Deployment")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Early exit if already reconciled this generation and in terminal state
	if deployment.Status.ObservedGeneration == deployment.Generation {
		if deployment.Status.Phase == platformv1alpha1.DeploymentPhaseSucceeded ||
			deployment.Status.Phase == platformv1alpha1.DeploymentPhaseFailed {
			log.V(1).Info("Deployment already reconciled to terminal state",
				"generation", deployment.Generation,
				"phase", deployment.Status.Phase)
			return ctrl.Result{}, nil
		}
	}

	// Fetch the referenced Application
	var app platformv1alpha1.Application
	if err := r.Get(ctx, types.NamespacedName{
		Name:      deployment.Spec.ApplicationRef.Name,
		Namespace: deployment.Namespace,
	}, &app); err != nil {
		log.Error(err, "Failed to get referenced Application")
		return ctrl.Result{}, err
	}

	// Check if Application is of type GitRepository
	if app.Spec.Type == platformv1alpha1.ApplicationTypeGitRepository {
		if err := r.handleGitRepositoryDeployment(ctx, &deployment, &app); err != nil {
			log.Error(err, "Failed to handle GitRepository deployment")
			return ctrl.Result{}, err
		}
	}

	// Check if Application is of type ImageFromRegistry
	if app.Spec.Type == platformv1alpha1.ApplicationTypeImageFromRegistry {
		if err := r.handleImageFromRegistryDeployment(ctx, &deployment, &app); err != nil {
			log.Error(err, "Failed to handle ImageFromRegistry deployment")
			return ctrl.Result{}, err
		}
	}

	// Check if Application is of type MySQL
	if app.Spec.Type == platformv1alpha1.ApplicationTypeMySQL {
		if err := r.handleMySQLDeployment(ctx, &deployment, &app); err != nil {
			log.Error(err, "Failed to handle MySQL deployment")
			return ctrl.Result{}, err
		}
	}

	// Check if Application is of type MySQLCluster
	if app.Spec.Type == platformv1alpha1.ApplicationTypeMySQLCluster {
		if err := r.handleMySQLClusterDeployment(ctx, &deployment, &app); err != nil {
			log.Error(err, "Failed to handle MySQLCluster deployment")
			return ctrl.Result{}, err
		}
	}

	// Check if Application is of type Valkey
	if app.Spec.Type == platformv1alpha1.ApplicationTypeValkey {
		if err := r.handleValkeyDeployment(ctx, &deployment, &app); err != nil {
			log.Error(err, "Failed to handle Valkey deployment")
			return ctrl.Result{}, err
		}
	}

	// Check if Application is of type ValkeyCluster
	if app.Spec.Type == platformv1alpha1.ApplicationTypeValkeyCluster {
		if err := r.handleValkeyClusterDeployment(ctx, &deployment, &app); err != nil {
			log.Error(err, "Failed to handle ValkeyCluster deployment")
			return ctrl.Result{}, err
		}
	}

	// Track previous phase before updating status
	prevPhase := deployment.Status.Phase

	// Note: Status updates are handled by DeploymentProgressController
	// This controller only handles resource creation

	// Emit webhook on phase transition
	r.emitDeploymentPhaseChange(ctx, &deployment, string(prevPhase), string(deployment.Status.Phase))

	// Track annotation changes before checking PipelineRun status
	annotationsBefore := make(map[string]string)
	if deployment.Annotations != nil {
		for k, v := range deployment.Annotations {
			annotationsBefore[k] = v
		}
	}

	// Check PipelineRun status and emit webhook if status changed
	if err := r.checkPipelineRunStatusAndEmitWebhook(ctx, &deployment); err != nil {
		log.Error(err, "Failed to check PipelineRun status")
		// Don't return error - this is non-critical
	}

	// Check if annotations changed and update if needed
	annotationsChanged := false
	if deployment.Annotations != nil {
		for k, v := range deployment.Annotations {
			if annotationsBefore[k] != v {
				annotationsChanged = true
				break
			}
		}
	}

	if annotationsChanged {
		if err := r.Update(ctx, &deployment); err != nil {
			log.Error(err, "Failed to update deployment annotations")
			// Don't fail the reconcile if annotation update fails
		}
	}

	// NOTE: K8s resource creation (Deployment, Service) is now handled by DeploymentProgressController
	// after the PipelineRun succeeds. This controller focuses on creating the Deployment CR and PipelineRun.
	//
	// DeploymentProgressController watches for PipelineRun completion via conditions and creates
	// K8s resources when transitioning to Deploying phase.

	log.Info("Successfully reconciled Deployment")
	return ctrl.Result{}, nil
}

// handleDeletion handles the deletion of a Deployment
func (r *DeploymentReconciler) handleDeletion(ctx context.Context, deployment *platformv1alpha1.Deployment) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("deployment", deployment.Name, "namespace", deployment.Namespace)

	if !controllerutil.ContainsFinalizer(deployment, DeploymentFinalizerName) {
		return ctrl.Result{}, nil
	}

	log.Info("Handling Deployment deletion")

	// TODO: Clean up any deployment-specific resources (e.g., PipelineRuns)
	// For now, we just remove the finalizer

	controllerutil.RemoveFinalizer(deployment, DeploymentFinalizerName)
	if err := r.Update(ctx, deployment); err != nil {
		log.Error(err, "Failed to remove finalizer from Deployment")
		return ctrl.Result{}, err
	}

	log.Info("Successfully handled Deployment deletion")
	return ctrl.Result{}, nil
}

// handleGitRepositoryDeployment handles deployments for GitRepository applications
func (r *DeploymentReconciler) handleGitRepositoryDeployment(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx).WithValues("deployment", deployment.Name, "application", app.Name)

	// Validate GitRepository configuration is present in deployment spec
	if deployment.Spec.GitRepository == nil {
		return fmt.Errorf("GitRepository configuration is required for GitRepository application deployments")
	}

	// Detect and log BuildType for debugging
	buildType := app.Spec.GitRepository.BuildType
	if buildType == "" {
		buildType = platformv1alpha1.BuildTypeRailpack // Default
	}
	log.Info("Handling GitRepository deployment", "buildType", buildType)

	// Log Dockerfile configuration if BuildType is Dockerfile
	if buildType == platformv1alpha1.BuildTypeDockerfile && app.Spec.GitRepository.DockerfileBuild != nil {
		log.Info("Dockerfile build configuration",
			"dockerfilePath", app.Spec.GitRepository.DockerfileBuild.DockerfilePath,
			"buildContext", app.Spec.GitRepository.DockerfileBuild.BuildContext)
	}

	// Generate the pipeline name
	pipelineName := r.generateGitRepositoryPipelineName(ctx, deployment, app)

	// Check if pipeline already exists
	existingPipeline := &tektonv1.Pipeline{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      pipelineName,
		Namespace: deployment.Namespace,
	}, existingPipeline)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing pipeline: %w", err)
	}

	// If pipeline doesn't exist, create it
	if errors.IsNotFound(err) {
		log.Info("Creating GitRepository pipeline", "pipelineName", pipelineName)
		if err := r.createGitRepositoryPipeline(ctx, deployment, app, pipelineName); err != nil {
			return fmt.Errorf("failed to create GitRepository pipeline: %w", err)
		}
		log.Info("Successfully created GitRepository pipeline", "pipelineName", pipelineName)
	} else {
		log.Info("GitRepository pipeline already exists", "pipelineName", pipelineName)
	}

	// Create PipelineRun for the deployment
	if err := r.createPipelineRun(ctx, deployment, app, pipelineName); err != nil {
		return fmt.Errorf("failed to create PipelineRun: %w", err)
	}

	return nil
}

// handleImageFromRegistryDeployment handles deployments for ImageFromRegistry applications
func (r *DeploymentReconciler) handleImageFromRegistryDeployment(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx).WithValues("deployment", deployment.Name, "application", app.Name)

	// Validate ImageFromRegistry configuration is present in deployment spec
	if deployment.Spec.ImageFromRegistry == nil {
		return fmt.Errorf("ImageFromRegistry configuration is required for ImageFromRegistry application deployments")
	}

	log.Info("Handling ImageFromRegistry deployment", "registry", app.Spec.ImageFromRegistry.Registry, "repository", app.Spec.ImageFromRegistry.Repository, "tag", deployment.Spec.ImageFromRegistry.Tag)

	// Create Kubernetes Deployment
	if err := r.createKubernetesDeployment(ctx, deployment, app); err != nil {
		return fmt.Errorf("failed to create Kubernetes Deployment: %w", err)
	}

	// Create Kubernetes Service
	if err := r.createKubernetesService(ctx, deployment, app); err != nil {
		return fmt.Errorf("failed to create Kubernetes Service: %w", err)
	}

	// Create ApplicationDomain for routing
	if err := r.ensureApplicationDomain(ctx, deployment, app); err != nil {
		return fmt.Errorf("failed to ensure ApplicationDomain: %w", err)
	}

	log.Info("Successfully handled ImageFromRegistry deployment")
	return nil
}

// createKubernetesDeployment creates a Kubernetes Deployment for ImageFromRegistry applications
func (r *DeploymentReconciler) createKubernetesDeployment(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx)

	// Only handle ImageFromRegistry applications in this method
	if app.Spec.Type != platformv1alpha1.ApplicationTypeImageFromRegistry {
		return fmt.Errorf("createKubernetesDeployment called for non-ImageFromRegistry application")
	}

	k8sDepName := fmt.Sprintf("deployment-%s", deployment.GetUUID())

	// Check if deployment already exists
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

	// Build image name
	imageName := r.buildImageName(app.Spec.ImageFromRegistry, deployment.Spec.ImageFromRegistry)

	// Determine port
	port := app.Spec.ImageFromRegistry.Port
	if port == 0 {
		port = 3000 // Default port
	}

	// Merge environment variables
	envVars := r.mergeEnvVars(app.Spec.ImageFromRegistry.Env, deployment.Spec.ImageFromRegistry.Env)

	// Merge resource requirements
	resources := r.mergeResources(app.Spec.ImageFromRegistry.Resources, deployment.Spec.ImageFromRegistry.Resources)

	// Create Kubernetes Deployment
	replicas := int32(1)
	appUUID := app.GetUUID()

	k8sDep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sDepName,
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                 fmt.Sprintf("app-%s", appUUID),
				"app.kubernetes.io/managed-by":           "kibaship-operator",
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
						"app.kubernetes.io/managed-by":           "kibaship-operator",
						"app.kubernetes.io/component":            "application",
						"platform.kibaship.com/deployment-uuid":  deployment.GetUUID(),
						"platform.kibaship.com/application-uuid": app.GetUUID(),
						"platform.kibaship.com/project-uuid":     deployment.GetProjectUUID(),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: imageName,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: port,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env:       envVars,
							Resources: *resources,
						},
					},
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

// buildImageName constructs the full container image name from registry, repository, and tag
func (r *DeploymentReconciler) buildImageName(appConfig *platformv1alpha1.ImageFromRegistryConfig, deployConfig *platformv1alpha1.ImageFromRegistryDeploymentConfig) string {
	var registryURL string
	switch appConfig.Registry {
	case platformv1alpha1.RegistryTypeDockerHub:
		registryURL = "docker.io"
	case platformv1alpha1.RegistryTypeGHCR:
		registryURL = "ghcr.io"
	default:
		registryURL = "docker.io" // fallback
	}

	// Use deployment tag or fall back to app default tag or "latest"
	tag := deployConfig.Tag
	if tag == "" {
		tag = appConfig.DefaultTag
		if tag == "" {
			tag = "latest"
		}
	}

	return fmt.Sprintf("%s/%s:%s", registryURL, appConfig.Repository, tag)
}

// mergeEnvVars merges application and deployment environment variables
// Deployment env vars override application env vars by name
func (r *DeploymentReconciler) mergeEnvVars(appEnv []corev1.EnvVar, deployEnv []corev1.EnvVar) []corev1.EnvVar {
	envMap := make(map[string]corev1.EnvVar)

	// Add application env vars first
	for _, env := range appEnv {
		envMap[env.Name] = env
	}

	// Override with deployment env vars
	for _, env := range deployEnv {
		envMap[env.Name] = env
	}

	// Convert back to slice
	result := make([]corev1.EnvVar, 0, len(envMap))
	for _, env := range envMap {
		result = append(result, env)
	}

	return result
}

// mergeResources merges application and deployment resource requirements
// Deployment resources override application resources
func (r *DeploymentReconciler) mergeResources(appResources *corev1.ResourceRequirements, deployResources *corev1.ResourceRequirements) *corev1.ResourceRequirements {
	// Start with application resources or empty if nil
	result := &corev1.ResourceRequirements{}
	if appResources != nil {
		result = appResources.DeepCopy()
	}

	// Override with deployment resources if provided
	if deployResources != nil {
		if deployResources.Limits != nil {
			if result.Limits == nil {
				result.Limits = make(corev1.ResourceList)
			}
			for k, v := range deployResources.Limits {
				result.Limits[k] = v
			}
		}
		if deployResources.Requests != nil {
			if result.Requests == nil {
				result.Requests = make(corev1.ResourceList)
			}
			for k, v := range deployResources.Requests {
				result.Requests[k] = v
			}
		}
	}

	return result
}

// createKubernetesService creates a Kubernetes Service for ImageFromRegistry applications
func (r *DeploymentReconciler) createKubernetesService(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx)

	// Only handle ImageFromRegistry applications in this method
	if app.Spec.Type != platformv1alpha1.ApplicationTypeImageFromRegistry {
		return fmt.Errorf("createKubernetesService called for non-ImageFromRegistry application")
	}

	serviceName := fmt.Sprintf("service-%s", deployment.GetUUID())

	// Check if service already exists
	var existing corev1.Service
	err := r.Get(ctx, client.ObjectKey{
		Name:      serviceName,
		Namespace: deployment.Namespace,
	}, &existing)

	if err == nil {
		log.V(1).Info("K8s Service already exists", "name", serviceName)
		return nil // Already exists
	}

	if !errors.IsNotFound(err) {
		return err
	}

	// Determine port
	port := app.Spec.ImageFromRegistry.Port
	if port == 0 {
		port = 3000 // Default port
	}

	appUUID := app.GetUUID()

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                 fmt.Sprintf("app-%s", appUUID),
				"app.kubernetes.io/managed-by":           "kibaship-operator",
				"app.kubernetes.io/component":            "application",
				"platform.kibaship.com/deployment-uuid":  deployment.GetUUID(),
				"platform.kibaship.com/application-uuid": app.GetUUID(),
				"platform.kibaship.com/project-uuid":     deployment.GetProjectUUID(),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app.kubernetes.io/name":                fmt.Sprintf("app-%s", appUUID),
				"platform.kibaship.com/deployment-uuid": deployment.GetUUID(),
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       port,
					TargetPort: intstr.FromInt32(port),
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

	log.Info("Created K8s Service", "name", serviceName, "port", port)
	return nil
}

// ensureApplicationDomain creates an ApplicationDomain for ImageFromRegistry applications
func (r *DeploymentReconciler) ensureApplicationDomain(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx)

	// Only handle ImageFromRegistry applications in this method
	if app.Spec.Type != platformv1alpha1.ApplicationTypeImageFromRegistry {
		return fmt.Errorf("ensureApplicationDomain called for non-ImageFromRegistry application")
	}

	deploymentUUID := deployment.GetUUID()
	appUUID := app.GetUUID()

	// Check if this deployment's domain already exists
	domainName := fmt.Sprintf("domain-%s", deploymentUUID)
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

	// ImageFromRegistry applications use <deployment-uuid>.apps.<baseDomain>
	domain := fmt.Sprintf("%s.apps.%s", deploymentUUID, opConfig.Domain)

	// Determine port
	port := app.Spec.ImageFromRegistry.Port
	if port == 0 {
		port = 3000 // Default port
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

	log.Info("Created ApplicationDomain", "domain", domain, "port", port)
	return nil
}

// handleMySQLDeployment handles deployments for MySQL applications
func (r *DeploymentReconciler) handleMySQLDeployment(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx).WithValues("deployment", deployment.Name, "application", app.Name)

	// Validate MySQL configuration
	if err := validateMySQLConfiguration(app); err != nil {
		return fmt.Errorf("invalid MySQL configuration: %w", err)
	}

	// Get project for resource metadata
	var project platformv1alpha1.Project
	if err := r.Get(ctx, types.NamespacedName{
		Name: fmt.Sprintf("project-%s", deployment.GetProjectUUID()),
	}, &project); err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Get project and app slugs from labels
	projectSlug, err := r.getProjectSlug(ctx, deployment.GetProjectUUID())
	if err != nil {
		return fmt.Errorf("failed to get project slug: %w", err)
	}
	appSlug, err := r.getApplicationSlug(ctx, deployment.GetApplicationUUID(), deployment.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get application slug: %w", err)
	}

	// Check for existing deployments of this application
	var deploymentList platformv1alpha1.DeploymentList
	if err := r.List(ctx, &deploymentList, client.InNamespace(deployment.Namespace)); err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	// Generate resource names
	secretName, clusterName := generateMySQLResourceNames(deployment, app, projectSlug, appSlug)

	// Check if secret already exists
	existingSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: deployment.Namespace,
	}, existingSecret)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing secret: %w", err)
	}

	// Create secret if it doesn't exist
	if errors.IsNotFound(err) {
		log.Info("Creating MySQL credentials secret", "secretName", secretName)
		secret, err := generateMySQLCredentialsSecret(deployment, project.Name, projectSlug, appSlug, deployment.Namespace)
		if err != nil {
			return fmt.Errorf("failed to generate MySQL credentials secret: %w", err)
		}

		// Set owner reference to the deployment
		if err := controllerutil.SetControllerReference(deployment, secret, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference on secret: %w", err)
		}

		if err := r.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create MySQL credentials secret: %w", err)
		}
		log.Info("Successfully created MySQL credentials secret", "secretName", secretName)
	} else {
		log.Info("MySQL credentials secret already exists", "secretName", secretName)
	}

	// Check if InnoDBCluster already exists
	existingCluster := &unstructured.Unstructured{}
	existingCluster.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "mysql.oracle.com",
		Version: "v2",
		Kind:    "InnoDBCluster",
	})
	err = r.Get(ctx, types.NamespacedName{
		Name:      clusterName,
		Namespace: deployment.Namespace,
	}, existingCluster)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing InnoDBCluster: %w", err)
	}

	// Create InnoDBCluster if it doesn't exist
	if errors.IsNotFound(err) {
		log.Info("Creating InnoDBCluster", "clusterName", clusterName)
		cluster := generateInnoDBCluster(deployment, app, project.Name, projectSlug, appSlug, secretName, deployment.Namespace)

		// Set owner reference to the deployment
		if err := controllerutil.SetControllerReference(deployment, cluster, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference on InnoDBCluster: %w", err)
		}

		if err := r.Create(ctx, cluster); err != nil {
			return fmt.Errorf("failed to create InnoDBCluster: %w", err)
		}
		log.Info("Successfully created InnoDBCluster", "clusterName", clusterName)
	} else {
		log.Info("InnoDBCluster already exists", "clusterName", clusterName)
	}

	return nil
}

// handleMySQLClusterDeployment handles deployments for MySQLCluster applications
func (r *DeploymentReconciler) handleMySQLClusterDeployment(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx).WithValues("deployment", deployment.Name, "application", app.Name)

	// Validate MySQLCluster configuration
	if err := validateMySQLClusterConfiguration(app); err != nil {
		return fmt.Errorf("invalid MySQLCluster configuration: %w", err)
	}

	// Get project for resource metadata
	var project platformv1alpha1.Project
	if err := r.Get(ctx, types.NamespacedName{
		Name: fmt.Sprintf("project-%s", deployment.GetProjectUUID()),
	}, &project); err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Get project and app slugs from labels
	projectSlug, err := r.getProjectSlug(ctx, deployment.GetProjectUUID())
	if err != nil {
		return fmt.Errorf("failed to get project slug: %w", err)
	}
	appSlug, err := r.getApplicationSlug(ctx, deployment.GetApplicationUUID(), deployment.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get application slug: %w", err)
	}

	// Generate resource names
	secretName, clusterName := generateMySQLClusterResourceNames(deployment, app, projectSlug, appSlug)

	// Check if secret already exists
	existingSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: deployment.Namespace,
	}, existingSecret)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing secret: %w", err)
	}

	// Create secret if it doesn't exist
	if errors.IsNotFound(err) {
		log.Info("Creating MySQLCluster credentials secret", "secretName", secretName)
		secret, err := generateMySQLCredentialsSecret(deployment, project.Name, projectSlug, appSlug, deployment.Namespace)
		if err != nil {
			return fmt.Errorf("failed to generate MySQLCluster credentials secret: %w", err)
		}

		// Set owner reference to the deployment
		if err := controllerutil.SetControllerReference(deployment, secret, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference on secret: %w", err)
		}

		if err := r.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create MySQLCluster credentials secret: %w", err)
		}
		log.Info("Successfully created MySQLCluster credentials secret", "secretName", secretName)
	} else {
		log.Info("MySQLCluster credentials secret already exists", "secretName", secretName)
	}

	// Check if MySQLCluster already exists
	existingCluster := &unstructured.Unstructured{}
	existingCluster.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "mysql.oracle.com",
		Version: "v2",
		Kind:    "InnoDBCluster",
	})
	err = r.Get(ctx, types.NamespacedName{
		Name:      clusterName,
		Namespace: deployment.Namespace,
	}, existingCluster)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing MySQLCluster: %w", err)
	}

	if errors.IsNotFound(err) {
		// Create new MySQLCluster
		log.Info("Creating MySQLCluster", "clusterName", clusterName)
		cluster := generateMySQLCluster(deployment, app, project.Name, projectSlug, appSlug, secretName, deployment.Namespace)

		// Set owner reference to the deployment
		if err := controllerutil.SetControllerReference(deployment, cluster, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference on MySQLCluster: %w", err)
		}

		if err := r.Create(ctx, cluster); err != nil {
			return fmt.Errorf("failed to create MySQLCluster: %w", err)
		}
		log.Info("Successfully created MySQLCluster", "clusterName", clusterName)
	} else {
		// Update existing MySQLCluster with new configuration
		log.Info("Updating existing MySQLCluster", "clusterName", clusterName)
		updatedCluster := generateMySQLCluster(deployment, app, project.Name, projectSlug, appSlug, secretName, deployment.Namespace)

		// Preserve the existing metadata but update spec
		existingCluster.Object["spec"] = updatedCluster.Object["spec"]

		if err := r.Update(ctx, existingCluster); err != nil {
			return fmt.Errorf("failed to update MySQLCluster: %w", err)
		}
		log.Info("Successfully updated MySQLCluster", "clusterName", clusterName)
	}

	return nil
}

// generateGitRepositoryPipelineName generates the pipeline name for GitRepository applications
func (r *DeploymentReconciler) generateGitRepositoryPipelineName(_ context.Context, deployment *platformv1alpha1.Deployment, _ *platformv1alpha1.Application) string {
	deploymentUUID := deployment.GetUUID()
	return fmt.Sprintf("pipeline-%s", deploymentUUID)
}

// createGitRepositoryPipeline creates a Tekton Pipeline for GitRepository applications
func (r *DeploymentReconciler) createGitRepositoryPipeline(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application, pipelineName string) error {
	log := logf.FromContext(ctx)

	projectUUID := deployment.GetProjectUUID()

	// Get project slug for labels
	projectSlug, err := r.getProjectSlug(ctx, projectUUID)
	if err != nil {
		return fmt.Errorf("failed to get project slug: %w", err)
	}

	// Generate pipeline using the new unified pipeline generator
	pipeline, err := r.generatePipeline(ctx, deployment, app, pipelineName, projectSlug)
	if err != nil {
		return fmt.Errorf("failed to generate pipeline: %w", err)
	}

	if err := r.Create(ctx, pipeline); err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}

	log.Info("Created GitRepository pipeline", "pipeline", pipelineName, "namespace", deployment.Namespace)
	return nil
}

// getEnvSecretName returns the environment secret name from the application based on its type
func (r *DeploymentReconciler) getEnvSecretName(app *platformv1alpha1.Application) string {
	switch app.Spec.Type {
	case platformv1alpha1.ApplicationTypeGitRepository:
		if app.Spec.GitRepository != nil && app.Spec.GitRepository.Env != nil {
			return app.Spec.GitRepository.Env.Name
		}
	case platformv1alpha1.ApplicationTypeDockerImage:
		if app.Spec.DockerImage != nil && app.Spec.DockerImage.Env != nil {
			return app.Spec.DockerImage.Env.Name
		}
	case platformv1alpha1.ApplicationTypeImageFromRegistry:
		// ImageFromRegistry applications don't have a specific env secret reference
		// Environment variables are defined directly in the application spec
		return ""
	case platformv1alpha1.ApplicationTypeMySQL:
		if app.Spec.MySQL != nil && app.Spec.MySQL.Env != nil {
			return app.Spec.MySQL.Env.Name
		}
	case platformv1alpha1.ApplicationTypeMySQLCluster:
		if app.Spec.MySQLCluster != nil && app.Spec.MySQLCluster.Env != nil {
			return app.Spec.MySQLCluster.Env.Name
		}
	case platformv1alpha1.ApplicationTypePostgres:
		if app.Spec.Postgres != nil && app.Spec.Postgres.Env != nil {
			return app.Spec.Postgres.Env.Name
		}
	case platformv1alpha1.ApplicationTypePostgresCluster:
		if app.Spec.PostgresCluster != nil && app.Spec.PostgresCluster.Env != nil {
			return app.Spec.PostgresCluster.Env.Name
		}
	}
	// Fallback: generate from app UUID
	if appUUID, exists := app.Labels["platform.kibaship.com/uuid"]; exists {
		return fmt.Sprintf("env-%s", appUUID)
	}
	return ""
}

// getEnvWorkspaceBinding returns a workspace binding for the application's env secret
func (r *DeploymentReconciler) getEnvWorkspaceBinding(app *platformv1alpha1.Application) *tektonv1.WorkspaceBinding {
	secretName := r.getEnvSecretName(app)
	if secretName == "" {
		return nil
	}

	return &tektonv1.WorkspaceBinding{
		Name: "app-env-vars",
		Secret: &corev1.SecretVolumeSource{
			SecretName: secretName,
		},
	}
}

// createPipelineRun creates a PipelineRun for the deployment
func (r *DeploymentReconciler) createPipelineRun(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application, pipelineName string) error {
	log := logf.FromContext(ctx)

	deploymentSlug := deployment.GetSlug()
	deploymentUUID := deployment.GetUUID()

	// Generate PipelineRun name with generation for uniqueness
	pipelineRunName := fmt.Sprintf("pipeline-run-%s-%d", deploymentUUID, deployment.Generation)

	// Check if PipelineRun already exists for this generation
	existingPipelineRun := &tektonv1.PipelineRun{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      pipelineRunName,
		Namespace: deployment.Namespace,
	}, existingPipelineRun)

	if err == nil {
		log.Info("PipelineRun already exists for this deployment generation", "pipelineRunName", pipelineRunName)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing PipelineRun: %w", err)
	}

	// Get git configuration from application
	gitConfig := app.Spec.GitRepository
	if gitConfig == nil {
		return fmt.Errorf("GitRepository configuration is nil")
	}

	// Get branch from deployment spec or use application default
	gitBranch := deployment.Spec.GitRepository.Branch
	if gitBranch == "" {
		gitBranch = gitConfig.Branch
		if gitBranch == "" {
			gitBranch = DefaultGitBranch // Final fallback
		}
	}

	// Generate workspace name based on deployment UUID
	workspaceName := fmt.Sprintf("workspace-%s", deploymentUUID)

	// Get project UUID for service account name
	projectUUID := deployment.GetProjectUUID()
	// Generate service account name - must match project controller naming
	serviceAccountName := fmt.Sprintf("project-%s-sa", projectUUID)

	pipelineRun := &tektonv1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineRunName,
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       truncateLabel(fmt.Sprintf("pipeline-run-%s", deploymentSlug)),
				"app.kubernetes.io/managed-by": "kibaship",
				"app.kubernetes.io/component":  "ci-cd-pipeline-run",
				"tekton.dev/pipeline":          truncateLabel(pipelineName),
				"deployment.kibaship.com/name": truncateLabel(deployment.Name),
			},
			Annotations: map[string]string{
				"description":                fmt.Sprintf("CI/CD pipeline run for deployment %s", deploymentSlug),
				"project.kibaship.com/usage": fmt.Sprintf("Executes pipeline for commit %s", deployment.Spec.GitRepository.CommitSHA),
				"tekton.dev/displayName":     fmt.Sprintf("Deployment %s Pipeline Run", deploymentSlug),
			},
		},
		Spec: tektonv1.PipelineRunSpec{
			PipelineRef: &tektonv1.PipelineRef{
				Name: pipelineName,
			},
			Params: []tektonv1.Param{
				{
					Name:  "git-commit",
					Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: deployment.Spec.GitRepository.CommitSHA},
				},
				{
					Name:  "git-branch",
					Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: gitBranch},
				},
			},
			TaskRunTemplate: tektonv1.PipelineTaskRunTemplate{
				ServiceAccountName: serviceAccountName,
			},
			Workspaces: func() []tektonv1.WorkspaceBinding {
				workspaces := []tektonv1.WorkspaceBinding{
					{
						Name: workspaceName,
						VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/name":       fmt.Sprintf("workspace-%s", deploymentSlug),
									"app.kubernetes.io/managed-by": "kibaship",
								},
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{
									corev1.ReadWriteOnce,
								},
								StorageClassName: func() *string { s := config.StorageClassReplica1; return &s }(),
								Resources: corev1.VolumeResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: resource.MustParse("24Gi"),
									},
								},
							},
						},
					},
					{
						Name: "registry-docker-config",
						Secret: &corev1.SecretVolumeSource{
							SecretName: "registry-docker-config",
						},
					},
					{
						Name: "registry-ca-cert",
						Secret: &corev1.SecretVolumeSource{
							SecretName: "registry-ca-cert",
						},
					},
				}
				// Add env vars workspace if available
				if envWorkspace := r.getEnvWorkspaceBinding(app); envWorkspace != nil {
					workspaces = append(workspaces, *envWorkspace)
				}
				return workspaces
			}(),
		},
	}

	// Set owner reference to the deployment
	if err := controllerutil.SetControllerReference(deployment, pipelineRun, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	if err := r.Create(ctx, pipelineRun); err != nil {
		return fmt.Errorf("failed to create PipelineRun: %w", err)
	}

	log.Info("Created PipelineRun", "pipelineRun", pipelineRunName, "namespace", deployment.Namespace, "commit", deployment.Spec.GitRepository.CommitSHA)
	return nil
}

// truncateLabel truncates a label to 63 characters and adds a hash suffix if needed
func truncateLabel(label string) string {
	if len(label) <= 63 {
		return label
	}

	// Hash the full label to create a unique suffix
	hash := sha256.Sum256([]byte(label))
	hashSuffix := fmt.Sprintf("-%x", hash)[:9] // Use first 8 chars of hex hash + dash = 9 chars

	// Truncate to 54 chars (63 - 9 for hash suffix)
	truncated := label[:54]
	return truncated + hashSuffix
}

// getProjectSlug retrieves the project slug by UUID
func (r *DeploymentReconciler) getProjectSlug(ctx context.Context, projectUUID string) (string, error) {
	var projects platformv1alpha1.ProjectList
	if err := r.List(ctx, &projects, client.MatchingLabels{
		"platform.kibaship.com/uuid": projectUUID,
	}); err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}
	if len(projects.Items) == 0 {
		return "", fmt.Errorf("project with UUID %s not found", projectUUID)
	}
	return projects.Items[0].GetSlug(), nil
}

// getApplicationSlug retrieves the application slug by UUID within a namespace
func (r *DeploymentReconciler) getApplicationSlug(ctx context.Context, appUUID, namespace string) (string, error) {
	var apps platformv1alpha1.ApplicationList
	if err := r.List(ctx, &apps, client.InNamespace(namespace), client.MatchingLabels{
		"platform.kibaship.com/uuid": appUUID,
	}); err != nil {
		return "", fmt.Errorf("failed to list applications: %w", err)
	}
	if len(apps.Items) == 0 {
		return "", fmt.Errorf("application with UUID %s not found in namespace %s", appUUID, namespace)
	}
	return apps.Items[0].GetSlug(), nil
}

// checkPipelineRunStatusAndEmitWebhook checks the PipelineRun status for this deployment and emits webhook on changes
// This function now uses generation-based tracking and returns annotation changes instead of updating directly
func (r *DeploymentReconciler) checkPipelineRunStatusAndEmitWebhook(ctx context.Context, deployment *platformv1alpha1.Deployment) error {
	if r.Notifier == nil {
		return nil
	}

	// Get the PipelineRun for this deployment
	deploymentUUID := deployment.GetUUID()
	pipelineRunName := fmt.Sprintf("pipeline-run-%s-%d", deploymentUUID, deployment.Generation)
	pipelineRun := &tektonv1.PipelineRun{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: deployment.Namespace,
		Name:      pipelineRunName,
	}, pipelineRun)

	if err != nil {
		if errors.IsNotFound(err) {
			// PipelineRun doesn't exist yet, nothing to report
			return nil
		}
		return fmt.Errorf("failed to get PipelineRun: %w", err)
	}

	// Get the status condition
	succeededCondition := pipelineRun.Status.GetCondition("Succeeded")
	if succeededCondition == nil {
		// No status yet
		return nil
	}

	// Use generation-based tracking to avoid infinite loops
	currentStatus := string(succeededCondition.Status)
	currentGeneration := fmt.Sprintf("%d", pipelineRun.GetGeneration())

	// Check both status and generation to determine if we should emit webhook
	lastProcessedStatus := deployment.Annotations["platform.kibaship.com/last-pipelinerun-status"]
	lastProcessedGeneration := deployment.Annotations["platform.kibaship.com/last-pipelinerun-generation"]

	// If both status and generation haven't changed, don't emit webhook
	if currentStatus == lastProcessedStatus && currentGeneration == lastProcessedGeneration {
		return nil
	}

	// Don't update annotations here - let the caller handle batched updates
	// Just emit the webhook

	// Emit optimized webhook about PipelineRun status change
	optimizedEvt := webhooks.OptimizedDeploymentStatusEvent{
		Type:          "deployment.pipelinerun.status.changed",
		PreviousPhase: lastProcessedStatus,
		NewPhase:      currentStatus,
		DeploymentRef: struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			UUID      string `json:"uuid"`
			Phase     string `json:"phase"`
			Slug      string `json:"slug"`
		}{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			UUID:      deployment.GetUUID(),
			Phase:     string(deployment.Status.Phase),
			Slug:      deployment.GetSlug(),
		},
		PipelineRunRef: &struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Reason string `json:"reason"`
		}{
			Name:   pipelineRun.Name,
			Status: currentStatus,
			Reason: succeededCondition.Reason,
		},
		Timestamp: time.Now().UTC(),
	}
	_ = r.Notifier.NotifyOptimizedDeploymentStatusChange(ctx, optimizedEvt)

	// Update annotations to track this status and generation (caller will handle the actual update)
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}
	deployment.Annotations["platform.kibaship.com/last-pipelinerun-status"] = currentStatus
	deployment.Annotations["platform.kibaship.com/last-pipelinerun-generation"] = currentGeneration

	return nil
}

// emitDeploymentPhaseChange sends a webhook if Notifier is configured and the phase actually changed.
func (r *DeploymentReconciler) emitDeploymentPhaseChange(ctx context.Context, deployment *platformv1alpha1.Deployment, prev, next string) {
	if r.Notifier == nil {
		return
	}
	if prev == next {
		return
	}
	// Use optimized webhook event to reduce memory usage
	optimizedEvt := r.createOptimizedWebhookEvent(deployment, prev, next, nil)
	_ = r.Notifier.NotifyOptimizedDeploymentStatusChange(ctx, optimizedEvt)
}

// createOptimizedWebhookEvent creates a memory-optimized webhook event with only essential fields
func (r *DeploymentReconciler) createOptimizedWebhookEvent(deployment *platformv1alpha1.Deployment, prev, next string, pipelineRun *tektonv1.PipelineRun) webhooks.OptimizedDeploymentStatusEvent {
	evt := webhooks.OptimizedDeploymentStatusEvent{
		Type:          "deployment.status.changed",
		PreviousPhase: prev,
		NewPhase:      next,
		DeploymentRef: struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			UUID      string `json:"uuid"`
			Phase     string `json:"phase"`
			Slug      string `json:"slug"`
		}{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			UUID:      deployment.GetUUID(),
			Phase:     string(deployment.Status.Phase),
			Slug:      deployment.GetSlug(),
		},
		Timestamp: time.Now().UTC(),
	}

	if pipelineRun != nil {
		condition := pipelineRun.Status.GetCondition("Succeeded")
		if condition != nil {
			evt.PipelineRunRef = &struct {
				Name   string `json:"name"`
				Status string `json:"status"`
				Reason string `json:"reason"`
			}{
				Name:   pipelineRun.Name,
				Status: string(condition.Status),
				Reason: condition.Reason,
			}
		}
	}

	return evt
}

// handleValkeyDeployment handles deployments for Valkey applications
func (r *DeploymentReconciler) handleValkeyDeployment(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx).WithValues("deployment", deployment.Name, "application", app.Name)

	// Validate Valkey configuration
	if err := validateValkeyConfiguration(app); err != nil {
		return fmt.Errorf("invalid Valkey configuration: %w", err)
	}

	// Get project for resource metadata
	var project platformv1alpha1.Project
	if err := r.Get(ctx, types.NamespacedName{
		Name: fmt.Sprintf("project-%s", deployment.GetProjectUUID()),
	}, &project); err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Get project and app slugs from labels
	projectSlug, err := r.getProjectSlug(ctx, deployment.GetProjectUUID())
	if err != nil {
		return fmt.Errorf("failed to get project slug: %w", err)
	}
	appSlug, err := r.getApplicationSlug(ctx, deployment.GetApplicationUUID(), deployment.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get application slug: %w", err)
	}

	// Generate resource names
	secretName, instanceName := generateValkeyResourceNames(deployment, projectSlug, appSlug)

	// Check if secret already exists
	existingSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: deployment.Namespace,
	}, existingSecret)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing secret: %w", err)
	}

	// Create secret if it doesn't exist
	if errors.IsNotFound(err) {
		log.Info("Creating Valkey credentials secret", "secretName", secretName)
		secret, err := generateValkeyCredentialsSecret(deployment, project.Name, projectSlug, appSlug, deployment.Namespace)
		if err != nil {
			return fmt.Errorf("failed to generate Valkey credentials secret: %w", err)
		}

		// Set owner reference to the deployment
		if err := controllerutil.SetControllerReference(deployment, secret, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference on secret: %w", err)
		}

		if err := r.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create Valkey credentials secret: %w", err)
		}
		log.Info("Successfully created Valkey credentials secret", "secretName", secretName)
	} else {
		log.Info("Valkey credentials secret already exists", "secretName", secretName)
	}

	// Check if Valkey instance already exists
	existingInstance := &unstructured.Unstructured{}
	existingInstance.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "hyperspike.io",
		Version: "v1",
		Kind:    "Valkey",
	})
	err = r.Get(ctx, types.NamespacedName{
		Name:      instanceName,
		Namespace: deployment.Namespace,
	}, existingInstance)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing Valkey instance: %w", err)
	}

	if errors.IsNotFound(err) {
		// Create new Valkey instance
		log.Info("Creating Valkey instance", "instanceName", instanceName)
		instance := generateValkeyInstance(deployment, app, project.Name, projectSlug, appSlug, secretName, deployment.Namespace)

		// Set owner reference to the deployment
		if err := controllerutil.SetControllerReference(deployment, instance, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference on Valkey instance: %w", err)
		}

		if err := r.Create(ctx, instance); err != nil {
			return fmt.Errorf("failed to create Valkey instance: %w", err)
		}
		log.Info("Successfully created Valkey instance", "instanceName", instanceName)
	} else {
		// Update existing Valkey instance with new configuration
		log.Info("Updating existing Valkey instance", "instanceName", instanceName)
		updatedInstance := generateValkeyInstance(deployment, app, project.Name, projectSlug, appSlug, secretName, deployment.Namespace)

		// Preserve the existing metadata but update spec
		existingInstance.Object["spec"] = updatedInstance.Object["spec"]

		if err := r.Update(ctx, existingInstance); err != nil {
			return fmt.Errorf("failed to update Valkey instance: %w", err)
		}
		log.Info("Successfully updated Valkey instance", "instanceName", instanceName)
	}

	return nil
}

// handleValkeyClusterDeployment handles deployments for ValkeyCluster applications
func (r *DeploymentReconciler) handleValkeyClusterDeployment(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx).WithValues("deployment", deployment.Name, "application", app.Name)

	// Validate ValkeyCluster configuration
	if err := validateValkeyClusterConfiguration(app); err != nil {
		return fmt.Errorf("invalid ValkeyCluster configuration: %w", err)
	}

	// Get project for resource metadata
	var project platformv1alpha1.Project
	if err := r.Get(ctx, types.NamespacedName{
		Name: fmt.Sprintf("project-%s", deployment.GetProjectUUID()),
	}, &project); err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Get project and app slugs from labels
	projectSlug, err := r.getProjectSlug(ctx, deployment.GetProjectUUID())
	if err != nil {
		return fmt.Errorf("failed to get project slug: %w", err)
	}
	appSlug, err := r.getApplicationSlug(ctx, deployment.GetApplicationUUID(), deployment.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get application slug: %w", err)
	}

	// Generate resource names
	secretName, clusterName := generateValkeyClusterResourceNames(deployment, projectSlug, appSlug)

	// Check if secret already exists
	existingSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: deployment.Namespace,
	}, existingSecret)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing secret: %w", err)
	}

	// Create secret if it doesn't exist
	if errors.IsNotFound(err) {
		log.Info("Creating ValkeyCluster credentials secret", "secretName", secretName)
		secret, err := generateValkeyCredentialsSecret(deployment, project.Name, projectSlug, appSlug, deployment.Namespace)
		if err != nil {
			return fmt.Errorf("failed to generate ValkeyCluster credentials secret: %w", err)
		}

		// Set owner reference to the deployment
		if err := controllerutil.SetControllerReference(deployment, secret, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference on secret: %w", err)
		}

		if err := r.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create ValkeyCluster credentials secret: %w", err)
		}
		log.Info("Successfully created ValkeyCluster credentials secret", "secretName", secretName)
	} else {
		log.Info("ValkeyCluster credentials secret already exists", "secretName", secretName)
	}

	// Check if ValkeyCluster already exists
	existingCluster := &unstructured.Unstructured{}
	existingCluster.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "hyperspike.io",
		Version: "v1",
		Kind:    "Valkey",
	})
	err = r.Get(ctx, types.NamespacedName{
		Name:      clusterName,
		Namespace: deployment.Namespace,
	}, existingCluster)

	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing ValkeyCluster: %w", err)
	}

	if errors.IsNotFound(err) {
		// Create new ValkeyCluster
		log.Info("Creating ValkeyCluster", "clusterName", clusterName)
		cluster := generateValkeyCluster(deployment, app, project.Name, projectSlug, appSlug, secretName, deployment.Namespace)

		// Set owner reference to the deployment
		if err := controllerutil.SetControllerReference(deployment, cluster, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference on ValkeyCluster: %w", err)
		}

		if err := r.Create(ctx, cluster); err != nil {
			return fmt.Errorf("failed to create ValkeyCluster: %w", err)
		}
		log.Info("Successfully created ValkeyCluster", "clusterName", clusterName)
	} else {
		// Update existing ValkeyCluster with new configuration
		log.Info("Updating existing ValkeyCluster", "clusterName", clusterName)
		updatedCluster := generateValkeyCluster(deployment, app, project.Name, projectSlug, appSlug, secretName, deployment.Namespace)

		// Preserve the existing metadata but update spec
		existingCluster.Object["spec"] = updatedCluster.Object["spec"]

		if err := r.Update(ctx, existingCluster); err != nil {
			return fmt.Errorf("failed to update ValkeyCluster: %w", err)
		}
		log.Info("Successfully updated ValkeyCluster", "clusterName", clusterName)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Deployment{}).
		Owns(&tektonv1.PipelineRun{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&platformv1alpha1.ApplicationDomain{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}). // Only watch spec changes
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 10, // Scale: handle multiple deployments concurrently
		}).
		Named("deployment").
		Complete(r)
}
