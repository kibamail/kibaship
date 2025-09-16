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

package v1alpha1

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kibamail/kibaship-operator/pkg/validation"
)

// DeploymentPhase represents the current phase of the deployment
type DeploymentPhase string

const (
	// DeploymentPhaseInitializing indicates the deployment is being set up
	DeploymentPhaseInitializing DeploymentPhase = "Initializing"
	// DeploymentPhaseRunning indicates a pipeline is currently running
	DeploymentPhaseRunning DeploymentPhase = "Running"
	// DeploymentPhaseSucceeded indicates the last pipeline run succeeded
	DeploymentPhaseSucceeded DeploymentPhase = "Succeeded"
	// DeploymentPhaseFailed indicates the last pipeline run failed
	DeploymentPhaseFailed DeploymentPhase = "Failed"
	// DeploymentPhaseWaiting indicates the deployment is waiting for trigger
	DeploymentPhaseWaiting DeploymentPhase = "Waiting"
)

// PipelineRunStatus tracks pipeline run information
type PipelineRunStatus struct {
	// Name of the pipeline run
	Name string `json:"name"`

	// Namespace of the pipeline run
	Namespace string `json:"namespace"`

	// Phase of the pipeline run
	Phase string `json:"phase"`

	// Start time
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Completion time
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Git commit SHA that triggered this run
	// +optional
	CommitSha string `json:"commitSha,omitempty"`

	// Built image from this run
	// +optional
	Image string `json:"image,omitempty"`

	// Message provides additional information about the pipeline run
	// +optional
	Message string `json:"message,omitempty"`
}

// ImageInfo contains information about built images
type ImageInfo struct {
	// Full image reference with digest
	Reference string `json:"reference"`

	// Image digest
	Digest string `json:"digest"`

	// Image tag
	Tag string `json:"tag"`

	// Build timestamp
	BuiltAt metav1.Time `json:"builtAt"`

	// Git commit SHA used for this image
	CommitSha string `json:"commitSha"`
}

// GitRepositoryDeploymentConfig defines the configuration for GitRepository deployments
type GitRepositoryDeploymentConfig struct {
	// CommitSHA is the specific commit hash to deploy
	// +kubebuilder:validation:Required
	CommitSHA string `json:"commitSHA"`

	// Branch is the git branch to use (optional, defaults to application branch)
	// +optional
	Branch string `json:"branch,omitempty"`
}

// DeploymentSpec defines the desired state of Deployment.
type DeploymentSpec struct {
	// ApplicationRef references the Application this deployment belongs to
	// +kubebuilder:validation:Required
	ApplicationRef corev1.LocalObjectReference `json:"applicationRef"`

	// GitRepository contains configuration for GitRepository deployments
	// Required when ApplicationRef points to a GitRepository application
	// +optional
	GitRepository *GitRepositoryDeploymentConfig `json:"gitRepository,omitempty"`
}

// DeploymentStatus defines the observed state of Deployment.
type DeploymentStatus struct {
	// Current phase of the deployment
	// +optional
	Phase DeploymentPhase `json:"phase,omitempty"`

	// Current pipeline run information
	// +optional
	CurrentPipelineRun *PipelineRunStatus `json:"currentPipelineRun,omitempty"`

	// Last successful pipeline run
	// +optional
	LastSuccessfulRun *PipelineRunStatus `json:"lastSuccessfulRun,omitempty"`

	// Built image information from the last successful run
	// +optional
	BuiltImage *ImageInfo `json:"builtImage,omitempty"`

	// Conditions represent the latest available observations of the deployment's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed Deployment
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Pipeline run history (last 5 runs)
	// +optional
	RunHistory []PipelineRunStatus `json:"runHistory,omitempty"`

	// Message provides additional information about the current status
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Application",type="string",JSONPath=".spec.applicationRef.name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Current Run",type="string",JSONPath=".status.currentPipelineRun.name"
// +kubebuilder:printcolumn:name="Last Success",type="string",JSONPath=".status.lastSuccessfulRun.name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:webhook:path=/validate-platform-operator-kibaship-com-v1alpha1-deployment,mutating=false,failurePolicy=fail,sideEffects=None,groups=platform.operator.kibaship.com,resources=deployments,verbs=create;update,versions=v1alpha1,name=vdeployment.kb.io,admissionReviewVersions=v1

// Deployment is the Schema for the deployments API.
type Deployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentSpec   `json:"spec,omitempty"`
	Status DeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DeploymentList contains a list of Deployment.
type DeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Deployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Deployment{}, &DeploymentList{})
}

var _ webhook.CustomValidator = &Deployment{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Deployment) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	deploymentlog := logf.Log.WithName("deployment-resource")

	dep, ok := obj.(*Deployment)
	if !ok {
		return nil, fmt.Errorf("expected a Deployment object, but got %T", obj)
	}

	deploymentlog.Info("validate create", "name", dep.Name)

	return nil, dep.validateDeployment(ctx)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Deployment) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	deploymentlog := logf.Log.WithName("deployment-resource")

	dep, ok := newObj.(*Deployment)
	if !ok {
		return nil, fmt.Errorf("expected a Deployment object, but got %T", newObj)
	}

	deploymentlog.Info("validate update", "name", dep.Name)

	return nil, dep.validateDeployment(ctx)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Deployment) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	deploymentlog := logf.Log.WithName("deployment-resource")

	dep, ok := obj.(*Deployment)
	if !ok {
		return nil, fmt.Errorf("expected a Deployment object, but got %T", obj)
	}

	deploymentlog.Info("validate delete", "name", dep.Name)

	return nil, nil
}

// validateDeployment validates the Deployment resource
func (r *Deployment) validateDeployment(ctx context.Context) error {
	_ = ctx // context is not used in current validation but required for webhook interface
	var errors []string

	// Use the centralized labeling validation
	// Note: In webhook context, we don't have access to the client for uniqueness checks
	// Uniqueness will be validated in the controller reconcile loop
	labels := r.GetLabels()
	if labels == nil {
		errors = append(errors, "deployment must have labels")
	} else {
		// Validate UUID
		if resourceUUID, exists := labels[validation.LabelResourceUUID]; !exists {
			errors = append(errors, fmt.Sprintf("deployment must have label %s", validation.LabelResourceUUID))
		} else if !validation.ValidateUUID(resourceUUID) {
			errors = append(errors, fmt.Sprintf("deployment UUID must be valid: %s", resourceUUID))
		}

		// Validate Slug
		if resourceSlug, exists := labels[validation.LabelResourceSlug]; !exists {
			errors = append(errors, fmt.Sprintf("deployment must have label %s", validation.LabelResourceSlug))
		} else if !validation.ValidateSlug(resourceSlug) {
			errors = append(errors, fmt.Sprintf("deployment slug must be valid: %s", resourceSlug))
		}

		// Validate Project UUID
		if projectUUID, exists := labels[validation.LabelProjectUUID]; !exists {
			errors = append(errors, fmt.Sprintf("deployment must have label %s", validation.LabelProjectUUID))
		} else if !validation.ValidateUUID(projectUUID) {
			errors = append(errors, fmt.Sprintf("project UUID must be valid: %s", projectUUID))
		}

		// Validate Application UUID
		if applicationUUID, exists := labels[validation.LabelApplicationUUID]; !exists {
			errors = append(errors, fmt.Sprintf("deployment must have label %s", validation.LabelApplicationUUID))
		} else if !validation.ValidateUUID(applicationUUID) {
			errors = append(errors, fmt.Sprintf("application UUID must be valid: %s", applicationUUID))
		}
	}

	// Validate deployment name format: project-<project-slug>-app-<app-slug>-deployment-<deployment-slug>-kibaship-com
	if !r.isValidDeploymentName() {
		errors = append(errors, fmt.Sprintf("deployment name '%s' must follow format 'project-<project-slug>-app-<app-slug>-deployment-<deployment-slug>-kibaship-com'", r.Name))
	}

	// TODO: Add validation for GitRepository config when application type is GitRepository
	// This would require fetching the application, which isn't available in webhook validation
	// The validation should be done in the controller reconcile loop

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// isValidDeploymentName validates if the deployment name follows the required format
func (r *Deployment) isValidDeploymentName() bool {
	// Pattern: project-<project-slug>-app-<app-slug>-deployment-<deployment-slug>-kibaship-com
	// All slugs should be valid DNS labels (lowercase alphanumeric with hyphens)
	pattern := regexp.MustCompile(`^project-[a-z0-9]([a-z0-9-]*[a-z0-9])?-app-[a-z0-9]([a-z0-9-]*[a-z0-9])?-deployment-[a-z0-9]([a-z0-9-]*[a-z0-9])?-kibaship-com$`)
	return pattern.MatchString(r.Name)
}

// GetProjectSlugFromName extracts the project slug from deployment name
func (r *Deployment) GetProjectSlugFromName() string {
	// Extract project slug from name format
	if !r.isValidDeploymentName() {
		return ""
	}

	parts := strings.Split(r.Name, "-")
	if len(parts) < 7 {
		return ""
	}

	// Find the "app" delimiter and extract everything between "project" and "app"
	const appDelimiter = "app"
	projectStart := 1 // after "project-"
	appIndex := -1
	for i, part := range parts {
		if part == appDelimiter {
			appIndex = i
			break
		}
	}

	if appIndex == -1 || appIndex <= projectStart {
		return ""
	}

	return strings.Join(parts[projectStart:appIndex], "-")
}

// GetAppSlugFromName extracts the app slug from deployment name
func (r *Deployment) GetAppSlugFromName() string {
	// Extract app slug from name format
	if !r.isValidDeploymentName() {
		return ""
	}

	parts := strings.Split(r.Name, "-")
	if len(parts) < 7 {
		return ""
	}

	// Find "app" and "deployment" delimiters
	appIndex := -1
	deploymentIndex := -1
	for i, part := range parts {
		if part == "app" {
			appIndex = i
		} else if part == "deployment" && appIndex != -1 {
			deploymentIndex = i
			break
		}
	}

	if appIndex == -1 || deploymentIndex == -1 || deploymentIndex <= appIndex+1 {
		return ""
	}

	return strings.Join(parts[appIndex+1:deploymentIndex], "-")
}

// GetDeploymentSlugFromName extracts the deployment slug from deployment name
func (r *Deployment) GetDeploymentSlugFromName() string {
	// Extract deployment slug from name format
	if !r.isValidDeploymentName() {
		return ""
	}

	parts := strings.Split(r.Name, "-")
	if len(parts) < 7 {
		return ""
	}

	// Find "deployment" and "kibaship" delimiters
	deploymentIndex := -1
	kibashipIndex := -1
	for i, part := range parts {
		if part == "deployment" {
			deploymentIndex = i
		} else if part == "kibaship" {
			kibashipIndex = i
			break
		}
	}

	if deploymentIndex == -1 || kibashipIndex == -1 || kibashipIndex <= deploymentIndex+1 {
		return ""
	}

	return strings.Join(parts[deploymentIndex+1:kibashipIndex], "-")
}

// GenerateExpectedApplicationName generates the expected application name for this deployment
func (r *Deployment) GenerateExpectedApplicationName() string {
	projectSlug := r.GetProjectSlugFromName()
	appSlug := r.GetAppSlugFromName()

	if projectSlug == "" || appSlug == "" {
		return ""
	}

	return fmt.Sprintf("project-%s-app-%s-kibaship-com", projectSlug, appSlug)
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Deployment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(r).
		Complete()
}
