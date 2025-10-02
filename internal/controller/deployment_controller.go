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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/config"
	"github.com/kibamail/kibaship-operator/pkg/webhooks"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	// DeploymentFinalizerName is the finalizer name for Deployment resources
	DeploymentFinalizerName = "platform.operator.kibaship.com/deployment-finalizer"
	// GitRepositoryPipelineName is the suffix for git repository pipeline names
	GitRepositoryPipelineSuffix = "git-repository-pipeline-kibaship-com"
	// GitCloneTaskName is the name of the git clone task in tekton-pipelines namespace
	GitCloneTaskName = "tekton-task-git-clone-kibaship-com"
	// RailpackPrepareTaskName is the name of the railpack prepare task in tekton-pipelines namespace
	RailpackPrepareTaskName = "tekton-task-railpack-prepare-kibaship-com"
	// RailpackBuildTaskName is the name of the railpack build task in tekton-pipelines namespace
	RailpackBuildTaskName = "tekton-task-railpack-build-kibaship-com"
	// eventEmittedValue is the value used to mark events as emitted in annotations
	eventEmittedValue = "true"
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
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tekton.dev,resources=taskruns,verbs=get;list;watch
// +kubebuilder:rbac:groups=mysql.oracle.com,resources=innodbclusters,verbs=get;list;watch;create;update;patch;delete
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

	// Check PipelineRun status and emit webhook if status changed
	if err := r.checkPipelineRunStatusAndEmitWebhook(ctx, &deployment); err != nil {
		log.Error(err, "Failed to check PipelineRun status")
		// Don't return error - this is non-critical
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
func (r *DeploymentReconciler) generateGitRepositoryPipelineName(_ context.Context, deployment *platformv1alpha1.Deployment, _ *platformv1alpha1.Application) string {
	deploymentSlug := deployment.GetSlug()
	return fmt.Sprintf("pipeline-%s-kibaship-com", deploymentSlug)
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
				"app.kubernetes.io/name":                 fmt.Sprintf("project-%s", projectSlug),
				"app.kubernetes.io/managed-by":           "kibaship-operator",
				"app.kubernetes.io/component":            "ci-cd-pipeline",
				"tekton.dev/pipeline":                    "git-repository-clone",
				"project.kibaship.com/slug":              projectSlug,
				"platform.kibaship.com/deployment-uuid":  deployment.Labels["platform.kibaship.com/uuid"],
				"platform.kibaship.com/application-uuid": deployment.Labels["platform.kibaship.com/application-uuid"],
				"platform.kibaship.com/project-uuid":     deployment.Labels["platform.kibaship.com/project-uuid"],
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
				{
					Name:        "registry-docker-config",
					Description: "Docker config for registry authentication",
					Optional:    true,
				},
				{
					Name:        "registry-ca-cert",
					Description: "Registry CA certificate for TLS trust",
					Optional:    true,
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
						{
							Name:  "public-access",
							Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: fmt.Sprintf("%t", gitConfig.PublicAccess)},
						},
					},
					Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
						{
							Name:      "output",
							Workspace: workspaceName,
						},
					},
				},
				{
					Name:     "prepare",
					RunAfter: []string{"clone-repository"},
					TaskRef: &tektonv1.TaskRef{
						ResolverRef: tektonv1.ResolverRef{
							Resolver: "cluster",
							Params: []tektonv1.Param{
								{Name: "kind", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "task"}},
								{Name: "name", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: RailpackPrepareTaskName}},
								{Name: "namespace", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "tekton-pipelines"}},
							},
						},
					},
					Params: []tektonv1.Param{
						{Name: "contextPath", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: func() string {
							if gitConfig.RootDirectory == "" {
								return "."
							}
							return gitConfig.RootDirectory
						}()}},
						{Name: "railpackVersion", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "0.1.2"}},
					},
					Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
						{Name: "output", Workspace: workspaceName},
					},
				},
				{
					Name:     "build",
					RunAfter: []string{"prepare"},
					TaskRef: &tektonv1.TaskRef{
						ResolverRef: tektonv1.ResolverRef{
							Resolver: "cluster",
							Params: []tektonv1.Param{
								{Name: "kind", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "task"}},
								{Name: "name", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: RailpackBuildTaskName}},
								{Name: "namespace", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "tekton-pipelines"}},
							},
						},
					},
					Params: []tektonv1.Param{
						{Name: "contextPath", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: func() string {
							if gitConfig.RootDirectory == "" {
								return "."
							}
							return gitConfig.RootDirectory
						}()}},
						{Name: "railpackFrontendSource", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "ghcr.io/railwayapp/railpack-frontend:v0.7.2"}},
						{Name: "imageTag", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: fmt.Sprintf("registry.registry.svc.cluster.local/%s/%s:%s", deployment.Namespace, deployment.Labels["platform.kibaship.com/application-uuid"], deployment.Labels["platform.kibaship.com/uuid"])}},
					},
					Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
						{Name: "output", Workspace: workspaceName},
						{Name: "docker-config", Workspace: "registry-docker-config"},
						{Name: "registry-ca", Workspace: "registry-ca-cert"},
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

	// Generate service account name - must match project controller naming
	serviceAccountName := fmt.Sprintf("project-%s-sa-kibaship-com", projectSlug)

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

// updateDeploymentStatus updates the Deployment status based on PipelineRun status
func (r *DeploymentReconciler) updateDeploymentStatus(ctx context.Context, deployment *platformv1alpha1.Deployment) error {
	log := logf.FromContext(ctx)

	// Ensure annotations map exists
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}

	deployment.Status.ObservedGeneration = deployment.Generation

	// Get the PipelineRun for this deployment to determine phase
	deploymentSlug := deployment.GetSlug()
	pipelineRunName := fmt.Sprintf("pipeline-run-%s-%d-kibaship-com", deploymentSlug, deployment.Generation)
	pipelineRun := &tektonv1.PipelineRun{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: deployment.Namespace,
		Name:      pipelineRunName,
	}, pipelineRun)

	if err != nil {
		if errors.IsNotFound(err) {
			// PipelineRun doesn't exist yet - stay in Initializing
			deployment.Status.Phase = platformv1alpha1.DeploymentPhaseInitializing
		} else {
			return fmt.Errorf("failed to get PipelineRun: %w", err)
		}
	} else {
		// PipelineRun exists - determine phase based on its status
		deployment.Status.Phase = r.determineDeploymentPhase(ctx, pipelineRun)

		// Emit events for completed TaskRuns (before status update to check current annotations)
		r.emitTaskRunEvents(ctx, deployment, pipelineRun)
	}

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

	// Emit events for phase changes (this modifies annotations)
	r.emitPhaseEvent(deployment, deployment.Status.Phase)

	// Update annotations separately after status update
	if err := r.Update(ctx, deployment); err != nil {
		log.Error(err, "Failed to update deployment annotations after event emission")
		// Don't fail the reconcile if annotation update fails
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

			if taskName == "prepare" {
				if taskCondition != nil && taskCondition.Status == corev1.ConditionTrue {
					prepareCompleted = true
				} else if taskCondition != nil && taskCondition.Status == corev1.ConditionUnknown {
					prepareRunning = true
				}
			}

			if taskName == "build" {
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

// emitPhaseEvent emits an event for the current deployment phase
func (r *DeploymentReconciler) emitPhaseEvent(deployment *platformv1alpha1.Deployment, phase platformv1alpha1.DeploymentPhase) {
	if r.Recorder == nil {
		return
	}

	// Track last emitted phase to avoid duplicate events
	lastPhaseKey := "platform.kibaship.com/last-phase-event"
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}

	lastPhase := deployment.Annotations[lastPhaseKey]
	currentPhase := string(phase)

	if lastPhase == currentPhase {
		return // Already emitted event for this phase
	}

	switch phase {
	case platformv1alpha1.DeploymentPhaseInitializing:
		r.Recorder.Event(deployment, corev1.EventTypeNormal, "deployment.initializing", "Deployment is being initialized")
	case platformv1alpha1.DeploymentPhasePreparing:
		r.Recorder.Event(deployment, corev1.EventTypeNormal, "deployment.preparing", "Preparing deployment with railpack")
	case platformv1alpha1.DeploymentPhaseBuilding:
		r.Recorder.Event(deployment, corev1.EventTypeNormal, "deployment.building", "Building container image with BuildKit")
	case platformv1alpha1.DeploymentPhaseDeploying:
		r.Recorder.Event(deployment, corev1.EventTypeNormal, "deployment.deploying", "Pipeline completed successfully, ready to deploy")
	case platformv1alpha1.DeploymentPhaseFailed:
		r.Recorder.Event(deployment, corev1.EventTypeWarning, "deployment.failed", "Deployment pipeline failed")
	case platformv1alpha1.DeploymentPhaseSucceeded:
		r.Recorder.Event(deployment, corev1.EventTypeNormal, "deployment.succeeded", "Deployment completed successfully")
	}

	deployment.Annotations[lastPhaseKey] = currentPhase
}

// emitTaskRunEvents checks TaskRun statuses and emits detailed events
func (r *DeploymentReconciler) emitTaskRunEvents(ctx context.Context, deployment *platformv1alpha1.Deployment, pipelineRun *tektonv1.PipelineRun) {
	if r.Recorder == nil {
		return
	}

	// Track which tasks we've already emitted events for using annotations
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}

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
			deployment.Annotations[startedKey] = eventEmittedValue
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
			deployment.Annotations[completedKey] = eventEmittedValue
		} else if taskCondition.Status == corev1.ConditionFalse && deployment.Annotations[failedKey] == "" {
			eventKey := fmt.Sprintf("taskrun.%s.failed", taskName)
			message := fmt.Sprintf("Task '%s' failed: %s", taskName, taskCondition.Reason)
			if taskCondition.Message != "" {
				message = fmt.Sprintf("Task '%s' failed: %s - %s", taskName, taskCondition.Reason, taskCondition.Message)
			}
			r.Recorder.Event(deployment, corev1.EventTypeWarning, eventKey, message)
			deployment.Annotations[failedKey] = eventEmittedValue
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
func (r *DeploymentReconciler) checkPipelineRunStatusAndEmitWebhook(ctx context.Context, deployment *platformv1alpha1.Deployment) error {
	if r.Notifier == nil {
		return nil
	}

	// Get the PipelineRun for this deployment
	deploymentSlug := deployment.GetSlug()
	pipelineRunName := fmt.Sprintf("pipeline-run-%s-%d-kibaship-com", deploymentSlug, deployment.Generation)
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

	// Check if status stored in deployment annotation matches current status
	currentStatus := string(succeededCondition.Status)
	previousStatus := deployment.Annotations["platform.kibaship.com/last-pipelinerun-status"]

	// If status hasn't changed, don't emit webhook
	if currentStatus == previousStatus {
		return nil
	}

	// Update the annotation to track this status
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}
	deployment.Annotations["platform.kibaship.com/last-pipelinerun-status"] = currentStatus
	if err := r.Update(ctx, deployment); err != nil {
		return fmt.Errorf("failed to update deployment annotation: %w", err)
	}

	// Emit webhook about PipelineRun status change
	evt := webhooks.DeploymentStatusEvent{
		Type:          "deployment.pipelinerun.status.changed",
		PreviousPhase: previousStatus,
		NewPhase:      currentStatus,
		Deployment:    *deployment,
		Timestamp:     time.Now().UTC(),
	}
	_ = r.Notifier.NotifyDeploymentStatusChange(ctx, evt)

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
	evt := webhooks.DeploymentStatusEvent{
		Type:          "deployment.status.changed",
		PreviousPhase: prev,
		NewPhase:      next,
		Deployment:    *deployment,
		Timestamp:     time.Now().UTC(),
	}
	_ = r.Notifier.NotifyDeploymentStatusChange(ctx, evt)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Deployment{}).
		Owns(&tektonv1.PipelineRun{}).
		Named("deployment").
		Complete(r)
}
