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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/streaming"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	// DeploymentFinalizerName is the finalizer name for Deployment resources
	DeploymentFinalizerName = "platform.operator.kibaship.com/deployment-finalizer"
	// GitRepositoryPipelineName is the suffix for git repository pipeline names
	GitRepositoryPipelineSuffix = "git-repository-pipeline-kibaship-com"
	// GitCloneTaskName is the name of the git clone task in tekton-pipelines namespace
	GitCloneTaskName = "tekton-task-git-clone-kibaship-com"
)

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	NamespaceManager *NamespaceManager
	StreamPublisher  streaming.ProjectStreamPublisher
}

// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applications,verbs=get;list;watch
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mysql.oracle.com,resources=innodbclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

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

	// Update deployment status
	if err := r.updateDeploymentStatus(ctx, &deployment); err != nil {
		log.Error(err, "Failed to update Deployment status")
		return ctrl.Result{}, err
	}

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

	// Publish Delete event before actual deletion
	r.publishDeploymentEvent(ctx, deployment, streaming.OperationDelete)

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

	// Generate the pipeline name
	pipelineName, err := r.generateGitRepositoryPipelineName(ctx, deployment, app)
	if err != nil {
		return fmt.Errorf("failed to generate pipeline name: %w", err)
	}

	// Check if pipeline already exists
	existingPipeline := &tektonv1.Pipeline{}
	err = r.Get(ctx, types.NamespacedName{
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
		secret, err := generateMySQLCredentialsSecret(deployment, projectSlug, appSlug, deployment.Namespace)
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
		cluster := generateInnoDBCluster(deployment, app, projectSlug, appSlug, secretName, deployment.Namespace)

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
func (r *DeploymentReconciler) generateGitRepositoryPipelineName(ctx context.Context, deployment *platformv1alpha1.Deployment, _ *platformv1alpha1.Application) (string, error) {
	deploymentSlug := deployment.GetSlug()
	return fmt.Sprintf("pipeline-%s-kibaship-com", deploymentSlug), nil
}

// createGitRepositoryPipeline creates a Tekton Pipeline for GitRepository applications
func (r *DeploymentReconciler) createGitRepositoryPipeline(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application, pipelineName string) error {
	log := logf.FromContext(ctx)

	deploymentSlug := deployment.GetSlug()

	// Get project slug for labels
	projectSlug, err := r.getProjectSlug(ctx, deployment.GetProjectUUID())
	if err != nil {
		return fmt.Errorf("failed to get project slug: %w", err)
	}

	// Get git configuration from application
	gitConfig := app.Spec.GitRepository
	if gitConfig == nil {
		return fmt.Errorf("GitRepository configuration is nil")
	}

	// Construct git URL from provider and repository
	gitURL := fmt.Sprintf("https://%s/%s", gitConfig.Provider, gitConfig.Repository)

	// Get branch (use default if empty)
	gitBranch := gitConfig.Branch
	if gitBranch == "" {
		gitBranch = "main" // Default branch
	}

	// Get secret name (only if not public access)
	var tokenSecret string
	if !gitConfig.PublicAccess && gitConfig.SecretRef != nil {
		tokenSecret = gitConfig.SecretRef.Name
	}

	// Generate workspace name based on deployment
	workspaceName := fmt.Sprintf("workspace-%s-kibaship-com", deploymentSlug)

	pipeline := &tektonv1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineName,
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       fmt.Sprintf("project-%s", projectSlug),
				"app.kubernetes.io/managed-by": "kibaship",
				"app.kubernetes.io/component":  "ci-cd-pipeline",
				"tekton.dev/pipeline":          "git-repository-clone",
				"project.kibaship.com/slug":    projectSlug,
			},
			Annotations: map[string]string{
				"description":                fmt.Sprintf("CI/CD pipeline for deployment %s that clones source code from git repository", deploymentSlug),
				"project.kibaship.com/usage": "Clones repository code for build and deployment processes",
				"tekton.dev/displayName":     fmt.Sprintf("Deployment %s GitRepository Pipeline", deploymentSlug),
			},
		},
		Spec: tektonv1.PipelineSpec{
			Description: "Pipeline that clones source code from a Git repository using an access token. This is the foundation pipeline that can be extended with build, test, and deploy tasks.",
			Params: []tektonv1.ParamSpec{
				{
					Name:        "git-commit",
					Description: "Specific commit hash to checkout",
					Type:        tektonv1.ParamTypeString,
				},
				{
					Name:        "git-branch",
					Description: "Git branch to checkout (optional, defaults to configured branch)",
					Type:        tektonv1.ParamTypeString,
					Default:     &tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: gitBranch},
				},
			},
			Workspaces: []tektonv1.PipelineWorkspaceDeclaration{
				{
					Name:        workspaceName,
					Description: "Workspace where the cloned source code will be stored",
				},
			},
			Tasks: []tektonv1.PipelineTask{
				{
					Name: "clone-repository",
					TaskRef: &tektonv1.TaskRef{
						ResolverRef: tektonv1.ResolverRef{
							Resolver: "cluster",
							Params: []tektonv1.Param{
								{
									Name:  "kind",
									Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "task"},
								},
								{
									Name:  "name",
									Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: GitCloneTaskName},
								},
								{
									Name:  "namespace",
									Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "tekton-pipelines"},
								},
							},
						},
					},
					Params: []tektonv1.Param{
						{
							Name:  "url",
							Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: gitURL},
						},
						{
							Name:  "branch",
							Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.git-branch)"},
						},
						{
							Name:  "commit",
							Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(params.git-commit)"},
						},
						{
							Name:  "token-secret",
							Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: tokenSecret},
						},
					},
					Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
						{
							Name:      "output",
							Workspace: workspaceName,
						},
					},
				},
			},
			Results: []tektonv1.PipelineResult{
				{
					Name:        "commit-sha",
					Description: "The actual commit SHA that was checked out",
					Value:       tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(tasks.clone-repository.results.commit)"},
				},
				{
					Name:        "repository-url",
					Description: "The repository URL that was cloned",
					Value:       tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(tasks.clone-repository.results.url)"},
				},
			},
		},
	}

	// Set owner reference to the deployment
	if err := controllerutil.SetControllerReference(deployment, pipeline, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	if err := r.Create(ctx, pipeline); err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}

	log.Info("Created GitRepository pipeline", "pipeline", pipelineName, "namespace", deployment.Namespace)
	return nil
}

// createPipelineRun creates a PipelineRun for the deployment
func (r *DeploymentReconciler) createPipelineRun(ctx context.Context, deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application, pipelineName string) error {
	log := logf.FromContext(ctx)

	projectSlug, err := r.getProjectSlug(ctx, deployment.GetProjectUUID())
	if err != nil {
		return fmt.Errorf("failed to get project slug: %w", err)
	}
	deploymentSlug := deployment.GetSlug()

	// Generate PipelineRun name with generation for uniqueness
	pipelineRunName := fmt.Sprintf("pipeline-run-%s-%d-kibaship-com", deploymentSlug, deployment.Generation)

	// Check if PipelineRun already exists for this generation
	existingPipelineRun := &tektonv1.PipelineRun{}
	err = r.Get(ctx, types.NamespacedName{
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

	// Generate workspace name based on deployment
	workspaceName := fmt.Sprintf("workspace-%s-kibaship-com", deploymentSlug)

	// Generate service account name
	serviceAccountName := fmt.Sprintf("service-account-%s-kibaship-com", projectSlug)

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
			Workspaces: []tektonv1.WorkspaceBinding{
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
							StorageClassName: func() *string { s := "storage-replica-1"; return &s }(),
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("24Gi"),
								},
							},
						},
					},
				},
			},
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

// updateDeploymentStatus updates the Deployment status
func (r *DeploymentReconciler) updateDeploymentStatus(ctx context.Context, deployment *platformv1alpha1.Deployment) error {
	// Check if this is a new deployment (no status set yet)
	isNewDeployment := deployment.Status.Phase == ""

	deployment.Status.ObservedGeneration = deployment.Generation
	deployment.Status.Phase = platformv1alpha1.DeploymentPhaseWaiting

	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "DeploymentReady",
		Message:            "Deployment is ready and pipeline created",
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

	// Publish appropriate event based on whether this is a new deployment
	if isNewDeployment {
		r.publishDeploymentEvent(ctx, deployment, streaming.OperationCreate)
	} else {
		r.publishDeploymentEvent(ctx, deployment, streaming.OperationReady)
	}

	return nil
}

// publishDeploymentEvent publishes a deployment event to the Redis stream
func (r *DeploymentReconciler) publishDeploymentEvent(ctx context.Context, deployment *platformv1alpha1.Deployment, operation streaming.OperationType) {
	log := logf.FromContext(ctx)

	// Skip publishing if StreamPublisher is not available
	if r.StreamPublisher == nil {
		return
	}

	// Extract required UUIDs from labels
	projectUUID := deployment.Labels["platform.operator.kibaship.com/project-uuid"]
	workspaceUUID := deployment.Labels["platform.operator.kibaship.com/workspace-uuid"]
	resourceUUID := deployment.Labels["platform.operator.kibaship.com/resource-uuid"]

	if projectUUID == "" || workspaceUUID == "" || resourceUUID == "" {
		log.Info("Skipping stream publish - missing required UUIDs",
			"projectUUID", projectUUID,
			"workspaceUUID", workspaceUUID,
			"resourceUUID", resourceUUID)
		return
	}

	// Create event with full Kubernetes resource
	event, err := streaming.NewResourceEventFromK8sResource(
		projectUUID,
		workspaceUUID,
		streaming.ResourceTypeDeployment,
		resourceUUID,
		deployment.Name,
		deployment.Namespace,
		operation,
		deployment,
	)
	if err != nil {
		log.Error(err, "Failed to create resource event for deployment")
		return
	}

	// Publish event
	if err := r.StreamPublisher.PublishEvent(ctx, event); err != nil {
		log.Error(err, "Failed to publish deployment event to stream")
	} else {
		log.Info("Successfully published deployment event",
			"operation", operation,
			"deploymentUUID", resourceUUID,
			"projectUUID", projectUUID)
	}
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

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Deployment{}).
		Named("deployment").
		Complete(r)
}
