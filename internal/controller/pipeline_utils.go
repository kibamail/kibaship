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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	// DockerfileBuildTaskName is the name of the Dockerfile build task in tekton-pipelines namespace
	DockerfileBuildTaskName = "tekton-task-dockerfile-build-kibaship-com"
)

// generatePipeline generates a Tekton Pipeline based on the application's BuildType
// This is the main entry point for pipeline generation
func (r *DeploymentReconciler) generatePipeline(
	ctx context.Context,
	deployment *platformv1alpha1.Deployment,
	app *platformv1alpha1.Application,
	pipelineName string,
	projectSlug string,
) (*tektonv1.Pipeline, error) {
	log := logf.FromContext(ctx)

	// Get git configuration
	gitConfig := app.Spec.GitRepository
	if gitConfig == nil {
		return nil, fmt.Errorf("GitRepository configuration is nil")
	}

	// Determine BuildType (default to Railpack for backward compatibility)
	buildType := gitConfig.BuildType
	if buildType == "" {
		buildType = platformv1alpha1.BuildTypeRailpack
	}

	log.Info("Generating pipeline", "buildType", buildType, "deployment", deployment.Name)

	// Generate pipeline based on BuildType
	switch buildType {
	case platformv1alpha1.BuildTypeRailpack:
		return r.generateRailpackPipeline(ctx, deployment, pipelineName, projectSlug, gitConfig)
	case platformv1alpha1.BuildTypeDockerfile:
		return r.generateDockerfilePipeline(ctx, deployment, pipelineName, projectSlug, gitConfig)
	default:
		return nil, fmt.Errorf("unsupported BuildType: %s", buildType)
	}
}

// generateRailpackPipeline generates a Tekton Pipeline for Railpack builds
func (r *DeploymentReconciler) generateRailpackPipeline(
	ctx context.Context,
	deployment *platformv1alpha1.Deployment,
	pipelineName string,
	projectSlug string,
	gitConfig *platformv1alpha1.GitRepositoryConfig,
) (*tektonv1.Pipeline, error) {
	log := logf.FromContext(ctx)

	deploymentSlug := deployment.GetSlug()
	deploymentUUID := deployment.GetUUID()
	projectUUID := deployment.GetProjectUUID()

	// Construct git URL from provider and repository
	gitURL := fmt.Sprintf("https://%s/%s", gitConfig.Provider, gitConfig.Repository)

	// Get branch (use default if empty)
	gitBranch := gitConfig.Branch
	if gitBranch == "" {
		gitBranch = DefaultGitBranch
	}

	// Get secret name (only if not public access)
	var tokenSecret string
	if !gitConfig.PublicAccess && gitConfig.SecretRef != nil {
		tokenSecret = gitConfig.SecretRef.Name
	}

	// Generate workspace name based on deployment UUID
	workspaceName := fmt.Sprintf("workspace-%s", deploymentUUID)

	pipeline := &tektonv1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineName,
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                 fmt.Sprintf("project-%s", projectUUID),
				"app.kubernetes.io/managed-by":           "kibaship",
				"app.kubernetes.io/component":            "ci-cd-pipeline",
				"tekton.dev/pipeline":                    "git-repository-railpack",
				"project.kibaship.com/slug":              projectSlug,
				"platform.kibaship.com/deployment-uuid":  deployment.Labels["platform.kibaship.com/uuid"],
				"platform.kibaship.com/application-uuid": deployment.Labels["platform.kibaship.com/application-uuid"],
				"platform.kibaship.com/project-uuid":     deployment.Labels["platform.kibaship.com/project-uuid"],
				"platform.kibaship.com/build-type":       string(platformv1alpha1.BuildTypeRailpack),
			},
			Annotations: map[string]string{
				"description":                fmt.Sprintf("CI/CD pipeline for deployment %s using Railpack build", deploymentSlug),
				"project.kibaship.com/usage": "Clones repository, prepares with Railpack, builds and pushes image",
				"tekton.dev/displayName":     fmt.Sprintf("Deployment %s Railpack Pipeline", deploymentSlug),
			},
		},
		Spec: tektonv1.PipelineSpec{
			Description: "Pipeline that builds applications using Railpack. Clones source code from Git, runs railpack prepare, builds the image with BuildKit, and pushes to registry.",
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
				{
					Name:        "app-env-vars",
					Description: "Application environment variables from secret",
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
						{Name: "railpackFrontendSource", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "ghcr.io/railwayapp/railpack-frontend:v0.9.0"}},
						{Name: "imageTag", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: fmt.Sprintf("registry.registry.svc.cluster.local/%s/%s:%s", deployment.Namespace, deployment.Labels["platform.kibaship.com/application-uuid"], deployment.Labels["platform.kibaship.com/uuid"])}},
					},
					Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
						{Name: "output", Workspace: workspaceName},
						{Name: "docker-config", Workspace: "registry-docker-config"},
						{Name: "registry-ca", Workspace: "registry-ca-cert"},
						{Name: "app-env-vars", Workspace: "app-env-vars"},
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
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}

	log.Info("Generated Railpack pipeline", "pipeline", pipelineName, "namespace", deployment.Namespace)
	return pipeline, nil
}

// generateDockerfilePipeline generates a Tekton Pipeline for Dockerfile builds
func (r *DeploymentReconciler) generateDockerfilePipeline(
	ctx context.Context,
	deployment *platformv1alpha1.Deployment,
	pipelineName string,
	projectSlug string,
	gitConfig *platformv1alpha1.GitRepositoryConfig,
) (*tektonv1.Pipeline, error) {
	log := logf.FromContext(ctx)

	deploymentSlug := deployment.GetSlug()
	deploymentUUID := deployment.GetUUID()
	projectUUID := deployment.GetProjectUUID()

	// Validate DockerfileBuild configuration
	if gitConfig.DockerfileBuild == nil {
		return nil, fmt.Errorf("DockerfileBuild configuration is required for Dockerfile BuildType")
	}

	// Get Dockerfile configuration
	dockerfilePath := gitConfig.DockerfileBuild.DockerfilePath
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile" // Default
	}

	buildContext := gitConfig.DockerfileBuild.BuildContext
	if buildContext == "" {
		buildContext = "." // Default to root
	}

	// Construct git URL from provider and repository
	gitURL := fmt.Sprintf("https://%s/%s", gitConfig.Provider, gitConfig.Repository)

	// Get branch (use default if empty)
	gitBranch := gitConfig.Branch
	if gitBranch == "" {
		gitBranch = DefaultGitBranch
	}

	// Get secret name (only if not public access)
	var tokenSecret string
	if !gitConfig.PublicAccess && gitConfig.SecretRef != nil {
		tokenSecret = gitConfig.SecretRef.Name
	}

	// Generate workspace name based on deployment UUID
	workspaceName := fmt.Sprintf("workspace-%s", deploymentUUID)

	pipeline := &tektonv1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineName,
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                 fmt.Sprintf("project-%s", projectUUID),
				"app.kubernetes.io/managed-by":           "kibaship",
				"app.kubernetes.io/component":            "ci-cd-pipeline",
				"tekton.dev/pipeline":                    "git-repository-dockerfile",
				"project.kibaship.com/slug":              projectSlug,
				"platform.kibaship.com/deployment-uuid":  deployment.Labels["platform.kibaship.com/uuid"],
				"platform.kibaship.com/application-uuid": deployment.Labels["platform.kibaship.com/application-uuid"],
				"platform.kibaship.com/project-uuid":     deployment.Labels["platform.kibaship.com/project-uuid"],
				"platform.kibaship.com/build-type":       string(platformv1alpha1.BuildTypeDockerfile),
			},
			Annotations: map[string]string{
				"description":                fmt.Sprintf("CI/CD pipeline for deployment %s using Dockerfile build", deploymentSlug),
				"project.kibaship.com/usage": fmt.Sprintf("Clones repository, builds image from %s, and pushes to registry", dockerfilePath),
				"tekton.dev/displayName":     fmt.Sprintf("Deployment %s Dockerfile Pipeline", deploymentSlug),
			},
		},
		Spec: tektonv1.PipelineSpec{
			Description: fmt.Sprintf("Pipeline that builds applications using Dockerfile. Clones source code from Git, builds the image from %s using BuildKit, and pushes to registry.", dockerfilePath),
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
					Optional:    false, // Required for Dockerfile builds
				},
				{
					Name:        "registry-ca-cert",
					Description: "Registry CA certificate for TLS trust",
					Optional:    false, // Required for Dockerfile builds
				},
				{
					Name:        "app-env-vars",
					Description: "Application environment variables from secret",
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
					Name:     "build-dockerfile",
					RunAfter: []string{"clone-repository"},
					TaskRef: &tektonv1.TaskRef{
						ResolverRef: tektonv1.ResolverRef{
							Resolver: "cluster",
							Params: []tektonv1.Param{
								{Name: "kind", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "task"}},
								{Name: "name", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: DockerfileBuildTaskName}},
								{Name: "namespace", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "tekton-pipelines"}},
							},
						},
					},
					Params: []tektonv1.Param{
						{Name: "dockerfilePath", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: dockerfilePath}},
						{Name: "contextPath", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: buildContext}},
						{Name: "imageTag", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: fmt.Sprintf("registry.registry.svc.cluster.local/%s/%s:%s", deployment.Namespace, deployment.Labels["platform.kibaship.com/application-uuid"], deployment.Labels["platform.kibaship.com/uuid"])}},
					},
					Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
						{Name: "output", Workspace: workspaceName},
						{Name: "docker-config", Workspace: "registry-docker-config"},
						{Name: "registry-ca", Workspace: "registry-ca-cert"},
						{Name: "app-env-vars", Workspace: "app-env-vars"},
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
				{
					Name:        "build-output",
					Description: "The image tag that was built and pushed",
					Value:       tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "$(tasks.build-dockerfile.results.buildOutput)"},
				},
			},
		},
	}

	// Set owner reference to the deployment
	if err := controllerutil.SetControllerReference(deployment, pipeline, r.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}

	log.Info("Generated Dockerfile pipeline", "pipeline", pipelineName, "namespace", deployment.Namespace, "dockerfilePath", dockerfilePath)
	return pipeline, nil
}
