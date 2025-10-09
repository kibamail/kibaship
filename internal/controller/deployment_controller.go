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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/config"
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
	// eventEmittedValue is the value used to mark events as emitted in annotations
	eventEmittedValue = "true"
)

// annotationTracker tracks annotation changes to batch updates and prevent unnecessary reconciliations
type annotationTracker struct {
	deployment *platformv1alpha1.Deployment
	changes    map[string]string
	hasChanges bool
}

// newAnnotationTracker creates a new annotation tracker for the deployment
func newAnnotationTracker(deployment *platformv1alpha1.Deployment) *annotationTracker {
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}
	return &annotationTracker{
		deployment: deployment,
		changes:    make(map[string]string),
		hasChanges: false,
	}
}

// setAnnotation sets an annotation if it's different from the current value
func (at *annotationTracker) setAnnotation(key, value string) {
	currentValue := at.deployment.Annotations[key]
	if currentValue != value {
		at.changes[key] = value
		at.hasChanges = true
	}
}

// applyChanges applies all tracked changes to the deployment annotations
func (at *annotationTracker) applyChanges() {
	if !at.hasChanges {
		return
	}
	for key, value := range at.changes {
		at.deployment.Annotations[key] = value
	}
}

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

	// Check if Application is of type MySQL
	if app.Spec.Type == platformv1alpha1.ApplicationTypeMySQL {
		if err := r.handleMySQLDeployment(ctx, &deployment, &app); err != nil {
			log.Error(err, "Failed to handle MySQL deployment")
			return ctrl.Result{}, err
		}
	}

	// Track previous phase before updating status
	prevPhase := deployment.Status.Phase

	// Update deployment status and check PipelineRun status
	if err := r.updateDeploymentStatus(ctx, &deployment); err != nil {
		log.Error(err, "Failed to update Deployment status")
		return ctrl.Result{}, err
	}

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

	// Check if other deployments exist for this application
	hasExistingDeployments := checkForExistingMySQLDeployments(deploymentList.Items, deployment, app)
	if hasExistingDeployments {
		log.Info("Existing MySQL deployments found for this application - treating as config change")
		// TODO: Handle configuration updates for existing MySQL deployments
		// For now, we'll just log and return successfully
		return nil
	}

	log.Info("No existing MySQL deployments found - creating new InnoDBCluster")

	// Generate resource names
	secretName, clusterName := generateMySQLResourceNames(deployment, projectSlug, appSlug)

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
			gitBranch = "main" // Final fallback
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

// updateDeploymentStatus updates the Deployment status based on PipelineRun status
func (r *DeploymentReconciler) updateDeploymentStatus(ctx context.Context, deployment *platformv1alpha1.Deployment) error {
	log := logf.FromContext(ctx)

	// Create annotation tracker to batch updates
	annotationTracker := newAnnotationTracker(deployment)

	deployment.Status.ObservedGeneration = deployment.Generation

	// Get the PipelineRun for this deployment to determine phase
	deploymentUUID := deployment.GetUUID()
	pipelineRunName := fmt.Sprintf("pipeline-run-%s-%d", deploymentUUID, deployment.Generation)
	pipelineRun := &tektonv1.PipelineRun{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: deployment.Namespace,
		Name:      pipelineRunName,
	}, pipelineRun)

	// Store current phase to check for changes
	previousPhase := deployment.Status.Phase
	var newPhase platformv1alpha1.DeploymentPhase

	if err != nil {
		if errors.IsNotFound(err) {
			// PipelineRun doesn't exist yet - stay in Initializing
			newPhase = platformv1alpha1.DeploymentPhaseInitializing
		} else {
			return fmt.Errorf("failed to get PipelineRun: %w", err)
		}
	} else {
		// PipelineRun exists - determine phase based on its status
		newPhase = r.determineDeploymentPhase(ctx, pipelineRun)

		// Emit events for completed TaskRuns using annotation tracker
		r.emitTaskRunEventsWithTracker(ctx, deployment, pipelineRun, annotationTracker)
	}

	// Early exit if phase hasn't changed and no annotation changes
	if previousPhase == newPhase && !annotationTracker.hasChanges {
		return nil
	}

	// Update phase if it changed
	deployment.Status.Phase = newPhase

	// Update conditions based on phase
	var condition metav1.Condition
	switch deployment.Status.Phase {
	case platformv1alpha1.DeploymentPhaseFailed:
		condition = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "PipelineRunFailed",
			Message:            "Pipeline run failed",
		}
	case platformv1alpha1.DeploymentPhaseSucceeded:
		condition = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "PipelineRunSucceeded",
			Message:            "Pipeline run succeeded",
		}
	case platformv1alpha1.DeploymentPhaseDeploying:
		condition = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: metav1.Now(),
			Reason:             "Deploying",
			Message:            "Deploying application",
		}
	case platformv1alpha1.DeploymentPhaseBuilding:
		condition = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: metav1.Now(),
			Reason:             "Building",
			Message:            "Building container image",
		}
	case platformv1alpha1.DeploymentPhasePreparing:
		condition = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: metav1.Now(),
			Reason:             "Preparing",
			Message:            "Preparing deployment",
		}
	default:
		condition = metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: metav1.Now(),
			Reason:             "Initializing",
			Message:            "Deployment is initializing",
		}
	}

	updated := false
	for i, existingCondition := range deployment.Status.Conditions {
		if existingCondition.Type == condition.Type {
			deployment.Status.Conditions[i] = condition
			updated = true
			break
		}
	}
	if !updated {
		deployment.Status.Conditions = append(deployment.Status.Conditions, condition)
	}

	if err := r.Status().Update(ctx, deployment); err != nil {
		return fmt.Errorf("failed to update Deployment status: %w", err)
	}

	log.Info("Updated deployment status", "phase", deployment.Status.Phase)

	// Emit events for phase changes using annotation tracker
	r.emitPhaseEventWithTracker(deployment, deployment.Status.Phase, annotationTracker)

	// Apply all annotation changes in a single update if there are any changes
	if annotationTracker.hasChanges {
		annotationTracker.applyChanges()
		if err := r.Update(ctx, deployment); err != nil {
			log.Error(err, "Failed to update deployment annotations")
			// Don't fail the reconcile if annotation update fails
		}
	}

	return nil
}

// determineDeploymentPhase determines the deployment phase based on PipelineRun status
func (r *DeploymentReconciler) determineDeploymentPhase(ctx context.Context, pipelineRun *tektonv1.PipelineRun) platformv1alpha1.DeploymentPhase {
	// Check if PipelineRun has completed
	succeededCondition := pipelineRun.Status.GetCondition("Succeeded")
	if succeededCondition == nil {
		// No status yet - initializing
		return platformv1alpha1.DeploymentPhaseInitializing
	}

	// Check if failed
	if succeededCondition.Status == corev1.ConditionFalse {
		return platformv1alpha1.DeploymentPhaseFailed
	}

	// Check if succeeded
	if succeededCondition.Status == corev1.ConditionTrue {
		// Pipeline completed successfully - for now return Deploying
		// Will transition to Succeeded after actual deployment phase is implemented
		return platformv1alpha1.DeploymentPhaseDeploying
	}

	// Pipeline is running - check which task is active using childReferences
	if len(pipelineRun.Status.ChildReferences) > 0 {
		prepareCompleted := false
		prepareRunning := false
		buildRunning := false

		// Check each child TaskRun
		for _, childRef := range pipelineRun.Status.ChildReferences {
			if childRef.Kind != "TaskRun" {
				continue
			}

			taskName := childRef.PipelineTaskName

			// Get the actual TaskRun to check its status
			taskRun := &tektonv1.TaskRun{}
			err := r.Get(ctx, types.NamespacedName{
				Namespace: pipelineRun.Namespace,
				Name:      childRef.Name,
			}, taskRun)

			if err != nil {
				continue // Skip if we can't fetch the TaskRun
			}

			taskCondition := taskRun.Status.GetCondition("Succeeded")

			// Check for Railpack prepare task
			if taskName == "prepare" {
				if taskCondition != nil && taskCondition.Status == corev1.ConditionTrue {
					prepareCompleted = true
				} else if taskCondition != nil && taskCondition.Status == corev1.ConditionUnknown {
					prepareRunning = true
				}
			}

			// Check for Railpack build task or Dockerfile build task
			if taskName == "build" || taskName == "build-dockerfile" {
				if taskCondition != nil && taskCondition.Status == corev1.ConditionUnknown {
					buildRunning = true
				}
			}
		}

		if buildRunning {
			return platformv1alpha1.DeploymentPhaseBuilding
		}
		if prepareCompleted {
			// Prepare completed, waiting for build to start
			return platformv1alpha1.DeploymentPhaseBuilding
		}
		if prepareRunning {
			return platformv1alpha1.DeploymentPhasePreparing
		}
		// Pipeline started but no task status yet
		return platformv1alpha1.DeploymentPhasePreparing
	}

	// Default to initializing
	return platformv1alpha1.DeploymentPhaseInitializing
}

// emitPhaseEventWithTracker emits an event for the current deployment phase using annotation tracker
func (r *DeploymentReconciler) emitPhaseEventWithTracker(deployment *platformv1alpha1.Deployment, phase platformv1alpha1.DeploymentPhase, tracker *annotationTracker) {
	if r.Recorder == nil {
		return
	}

	// Track last emitted phase to avoid duplicate events using annotation tracker
	lastPhaseKey := "platform.kibaship.com/last-phase-event"
	lastPhase := deployment.Annotations[lastPhaseKey]
	currentPhase := string(phase)

	if lastPhase == currentPhase {
		return // Already emitted event for this phase
	}

	switch phase {
	case platformv1alpha1.DeploymentPhaseInitializing:
		r.Recorder.Event(deployment, corev1.EventTypeNormal, "deployment.initializing", "Deployment is being initialized")
	case platformv1alpha1.DeploymentPhasePreparing:
		r.Recorder.Event(deployment, corev1.EventTypeNormal, "deployment.preparing", "Preparing deployment")
	case platformv1alpha1.DeploymentPhaseBuilding:
		r.Recorder.Event(deployment, corev1.EventTypeNormal, "deployment.building", "Building container image with BuildKit")
	case platformv1alpha1.DeploymentPhaseDeploying:
		r.Recorder.Event(deployment, corev1.EventTypeNormal, "deployment.deploying", "Pipeline completed successfully, ready to deploy")
	case platformv1alpha1.DeploymentPhaseFailed:
		r.Recorder.Event(deployment, corev1.EventTypeWarning, "deployment.failed", "Deployment pipeline failed")
	case platformv1alpha1.DeploymentPhaseSucceeded:
		r.Recorder.Event(deployment, corev1.EventTypeNormal, "deployment.succeeded", "Deployment completed successfully")
	}

	tracker.setAnnotation(lastPhaseKey, currentPhase)
}

// emitTaskRunEventsWithTracker checks TaskRun statuses and emits detailed events using annotation tracker
func (r *DeploymentReconciler) emitTaskRunEventsWithTracker(ctx context.Context, deployment *platformv1alpha1.Deployment, pipelineRun *tektonv1.PipelineRun, tracker *annotationTracker) {
	if r.Recorder == nil {
		return
	}

	// Use annotation tracker to track which tasks we've already emitted events for

	// Check each child TaskRun
	for _, childRef := range pipelineRun.Status.ChildReferences {
		if childRef.Kind != "TaskRun" {
			continue
		}

		taskName := childRef.PipelineTaskName
		startedKey := fmt.Sprintf("platform.kibaship.com/taskrun.%s.started", taskName)
		completedKey := fmt.Sprintf("platform.kibaship.com/taskrun.%s.completed", taskName)
		failedKey := fmt.Sprintf("platform.kibaship.com/taskrun.%s.failed", taskName)

		// Get the actual TaskRun to check its status
		taskRun := &tektonv1.TaskRun{}
		err := r.Get(ctx, types.NamespacedName{
			Namespace: pipelineRun.Namespace,
			Name:      childRef.Name,
		}, taskRun)

		if err != nil {
			continue
		}

		// Emit started event if not already emitted
		if taskRun.Status.StartTime != nil && deployment.Annotations[startedKey] == "" {
			eventKey := fmt.Sprintf("taskrun.%s.started", taskName)
			message := fmt.Sprintf("Task '%s' started at %s", taskName, taskRun.Status.StartTime.Format(time.RFC3339))
			r.Recorder.Event(deployment, corev1.EventTypeNormal, eventKey, message)
			tracker.setAnnotation(startedKey, eventEmittedValue)
		}

		taskCondition := taskRun.Status.GetCondition("Succeeded")
		if taskCondition == nil {
			continue
		}

		// Emit completion or failure events
		if taskCondition.Status == corev1.ConditionTrue && deployment.Annotations[completedKey] == "" {
			eventKey := fmt.Sprintf("taskrun.%s.completed", taskName)
			message := fmt.Sprintf("Task '%s' completed successfully", taskName)
			if taskRun.Status.CompletionTime != nil && taskRun.Status.StartTime != nil {
				duration := taskRun.Status.CompletionTime.Sub(taskRun.Status.StartTime.Time)
				message = fmt.Sprintf("Task '%s' completed in %s", taskName, duration.Round(time.Second))
			}
			r.Recorder.Event(deployment, corev1.EventTypeNormal, eventKey, message)
			tracker.setAnnotation(completedKey, eventEmittedValue)
		} else if taskCondition.Status == corev1.ConditionFalse && deployment.Annotations[failedKey] == "" {
			eventKey := fmt.Sprintf("taskrun.%s.failed", taskName)
			message := fmt.Sprintf("Task '%s' failed: %s", taskName, taskCondition.Reason)
			if taskCondition.Message != "" {
				message = fmt.Sprintf("Task '%s' failed: %s - %s", taskName, taskCondition.Reason, taskCondition.Message)
			}
			r.Recorder.Event(deployment, corev1.EventTypeWarning, eventKey, message)
			tracker.setAnnotation(failedKey, eventEmittedValue)
		}
	}

	// Annotations will be updated by the caller
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
