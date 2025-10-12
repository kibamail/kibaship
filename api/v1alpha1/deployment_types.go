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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kibamail/kibaship/pkg/validation"
)

// DeploymentPhase represents the current phase of the deployment
type DeploymentPhase string

const (
	// DeploymentPhaseInitializing indicates the deployment is being set up
	DeploymentPhaseInitializing DeploymentPhase = "Initializing"
	// DeploymentPhasePreparing indicates the prepare task is running
	DeploymentPhasePreparing DeploymentPhase = "Preparing"
	// DeploymentPhaseBuilding indicates the build task is running
	DeploymentPhaseBuilding DeploymentPhase = "Building"
	// DeploymentPhaseDeploying indicates the deployment is being deployed
	DeploymentPhaseDeploying DeploymentPhase = "Deploying"
	// DeploymentPhaseRunning indicates a pipeline is currently running
	DeploymentPhaseRunning DeploymentPhase = "Running"
	// DeploymentPhaseSucceeded indicates the last pipeline run succeeded
	DeploymentPhaseSucceeded DeploymentPhase = "Succeeded"
	// DeploymentPhaseFailed indicates the last pipeline run failed
	DeploymentPhaseFailed DeploymentPhase = "Failed"
	// DeploymentPhaseWaiting indicates the deployment is waiting for trigger
	DeploymentPhaseWaiting DeploymentPhase = "Waiting"
)

// GitRepositoryDeploymentConfig defines the configuration for GitRepository deployments
type GitRepositoryDeploymentConfig struct {
	// CommitSHA is the specific commit hash to deploy
	// +kubebuilder:validation:Required
	CommitSHA string `json:"commitSHA"`

	// Branch is the git branch to use (optional, defaults to application branch)
	// +optional
	Branch string `json:"branch,omitempty"`
}

// ImageFromRegistryDeploymentConfig defines deployment-specific config for registry images
type ImageFromRegistryDeploymentConfig struct {
	// Tag specifies the specific image tag to deploy (overrides application default)
	// +kubebuilder:validation:Required
	Tag string `json:"tag"`

	// Env defines environment variable overrides for this deployment
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Resources defines resource requirement overrides for this deployment
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// DeploymentSpec defines the desired state of Deployment.
type DeploymentSpec struct {
	// ApplicationRef references the Application this deployment belongs to
	// +kubebuilder:validation:Required
	ApplicationRef corev1.LocalObjectReference `json:"applicationRef"`

	// Promote indicates whether to promote this deployment as the current deployment on the application
	// When true and deployment succeeds, Application.spec.currentDeploymentRef will be updated to reference this deployment
	// +kubebuilder:default=false
	// +optional
	Promote bool `json:"promote,omitempty"`

	// GitRepository contains configuration for GitRepository deployments
	// Required when ApplicationRef points to a GitRepository application
	// +optional
	GitRepository *GitRepositoryDeploymentConfig `json:"gitRepository,omitempty"`

	// ImageFromRegistry contains configuration for ImageFromRegistry deployments
	// Required when ApplicationRef points to an ImageFromRegistry application
	// +optional
	ImageFromRegistry *ImageFromRegistryDeploymentConfig `json:"imageFromRegistry,omitempty"`
}

// DeploymentStatus defines the observed state of Deployment.
type DeploymentStatus struct {
	// Current phase of the deployment
	// +optional
	Phase DeploymentPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the deployment's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed Deployment
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Application",type="string",JSONPath=".spec.applicationRef.name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
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

		// Validate Environment UUID
		if environmentUUID, exists := labels[validation.LabelEnvironmentUUID]; !exists {
			errors = append(errors, fmt.Sprintf("deployment must have label %s", validation.LabelEnvironmentUUID))
		} else if !validation.ValidateUUID(environmentUUID) {
			errors = append(errors, fmt.Sprintf("environment UUID must be valid: %s", environmentUUID))
		}
	}

	// Validate deployment name format: deployment-<uuid>
	if !r.isValidDeploymentName() {
		errors = append(errors, fmt.Sprintf("deployment name '%s' must follow format 'deployment-<uuid>'", r.Name))
	}

	// TODO: Add validation for GitRepository config when application type is GitRepository
	// This would require fetching the application, which isn't available in webhook validation
	// The validation should be done in the controller reconcile loop

	// Basic validation for ImageFromRegistry configuration
	if r.Spec.ImageFromRegistry != nil {
		if err := r.validateImageFromRegistryDeployment(); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// validateImageFromRegistryDeployment validates ImageFromRegistry deployment configuration
func (r *Deployment) validateImageFromRegistryDeployment() error {
	config := r.Spec.ImageFromRegistry

	// Validate tag is required
	if config.Tag == "" {
		return fmt.Errorf("tag is required for ImageFromRegistry deployments")
	}

	// Validate tag format
	if !r.isValidImageTag(config.Tag) {
		return fmt.Errorf("invalid image tag format: %s", config.Tag)
	}

	return nil
}

// isValidImageTag validates image tag format
func (r *Deployment) isValidImageTag(tag string) bool {
	// Basic tag validation - alphanumeric, dots, hyphens, underscores
	pattern := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	return pattern.MatchString(tag)
}

// isValidDeploymentName validates if the deployment name follows the required format
func (r *Deployment) isValidDeploymentName() bool {
	// Pattern: deployment-<uuid>
	// UUID should be valid DNS label (lowercase alphanumeric with hyphens)
	pattern := regexp.MustCompile(`^deployment-[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	return pattern.MatchString(r.Name)
}

// GetSlug returns the deployment slug from labels
func (r *Deployment) GetSlug() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelResourceSlug]
}

// GetUUID returns the deployment UUID from labels
func (r *Deployment) GetUUID() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelResourceUUID]
}

// GetProjectUUID returns the project UUID from labels
func (r *Deployment) GetProjectUUID() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelProjectUUID]
}

// GetApplicationUUID returns the application UUID from labels
func (r *Deployment) GetApplicationUUID() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelApplicationUUID]
}

// GetEnvironmentUUID returns the environment UUID from labels
func (r *Deployment) GetEnvironmentUUID() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelEnvironmentUUID]
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Deployment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(r).
		Complete()
}
