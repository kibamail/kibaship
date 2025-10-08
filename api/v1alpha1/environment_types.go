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

	"github.com/kibamail/kibaship-operator/pkg/validation"
)

// EnvironmentSpec defines the desired state of Environment
type EnvironmentSpec struct {
	// ProjectRef references the Project this environment belongs to
	// +kubebuilder:validation:Required
	ProjectRef corev1.LocalObjectReference `json:"projectRef"`

	// Description of the environment (optional)
	// +optional
	Description string `json:"description,omitempty"`

	// Variables contains environment-specific configuration variables
	// +optional
	Variables map[string]string `json:"variables,omitempty"`
}

// EnvironmentStatus defines the observed state of Environment
type EnvironmentStatus struct {
	// Phase represents the current phase of the environment
	// +kubebuilder:validation:Enum=Pending;Ready;Failed
	// +optional
	Phase string `json:"phase,omitempty"`

	// ApplicationCount tracks number of applications in this environment
	// +optional
	ApplicationCount int32 `json:"applicationCount,omitempty"`

	// SecretReady indicates if environment secret exists
	// +optional
	SecretReady bool `json:"secretReady,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Message provides additional information about the current status
	// +optional
	Message string `json:"message,omitempty"`

	// LastReconcileTime is the timestamp of the last successful reconciliation
	// +optional
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.projectRef.name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Applications",type="integer",JSONPath=".status.applicationCount"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:webhook:path=/validate-platform-operator-kibaship-com-v1alpha1-environment,mutating=false,failurePolicy=fail,sideEffects=None,groups=platform.operator.kibaship.com,resources=environments,verbs=create;update,versions=v1alpha1,name=venvironment.kb.io,admissionReviewVersions=v1

// Environment is the Schema for the environments API
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentSpec   `json:"spec,omitempty"`
	Status EnvironmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvironmentList contains a list of Environment
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Environment{}, &EnvironmentList{})
}

var _ webhook.CustomValidator = &Environment{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Environment) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	environmentlog := logf.Log.WithName("environment-resource")

	env, ok := obj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected an Environment object, but got %T", obj)
	}

	environmentlog.Info("validate create", "name", env.Name)

	return nil, env.validateEnvironment(ctx)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Environment) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	environmentlog := logf.Log.WithName("environment-resource")

	env, ok := newObj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected an Environment object, but got %T", newObj)
	}

	environmentlog.Info("validate update", "name", env.Name)

	return nil, env.validateEnvironment(ctx)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Environment) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	environmentlog := logf.Log.WithName("environment-resource")

	env, ok := obj.(*Environment)
	if !ok {
		return nil, fmt.Errorf("expected an Environment object, but got %T", obj)
	}

	environmentlog.Info("validate delete", "name", env.Name)

	return nil, nil
}

// validateEnvironment validates the Environment resource
func (r *Environment) validateEnvironment(ctx context.Context) error {
	_ = ctx // context is not used in current validation but required for webhook interface
	var errors []string

	// Use the centralized labeling validation
	labels := r.GetLabels()
	if labels == nil {
		errors = append(errors, "environment must have labels")
	} else {
		// Validate UUID
		if resourceUUID, exists := labels[validation.LabelResourceUUID]; !exists {
			errors = append(errors, fmt.Sprintf("environment must have label %s", validation.LabelResourceUUID))
		} else if !validation.ValidateUUID(resourceUUID) {
			errors = append(errors, fmt.Sprintf("environment UUID must be valid: %s", resourceUUID))
		}

		// Validate Slug
		if resourceSlug, exists := labels[validation.LabelResourceSlug]; !exists {
			errors = append(errors, fmt.Sprintf("environment must have label %s", validation.LabelResourceSlug))
		} else if !validation.ValidateSlug(resourceSlug) {
			errors = append(errors, fmt.Sprintf("environment slug must be valid: %s", resourceSlug))
		}

		// Validate Project UUID
		if projectUUID, exists := labels[validation.LabelProjectUUID]; !exists {
			errors = append(errors, fmt.Sprintf("environment must have label %s", validation.LabelProjectUUID))
		} else if !validation.ValidateUUID(projectUUID) {
			errors = append(errors, fmt.Sprintf("project UUID must be valid: %s", projectUUID))
		}
	}

	// Validate environment name format: environment-<uuid>
	if !r.isValidEnvironmentName() {
		errors = append(errors, fmt.Sprintf("environment name '%s' must follow format 'environment-<uuid>'", r.Name))
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// isValidEnvironmentName validates if the environment name follows the required format
func (r *Environment) isValidEnvironmentName() bool {
	// Pattern: environment-<uuid>
	// UUID should be valid DNS label (lowercase alphanumeric with hyphens)
	pattern := regexp.MustCompile(`^environment-[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	return pattern.MatchString(r.Name)
}

// GetSlug returns the environment slug from labels
func (r *Environment) GetSlug() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelResourceSlug]
}

// GetProjectUUID returns the project UUID from labels
func (r *Environment) GetProjectUUID() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelProjectUUID]
}

// GetUUID returns the environment UUID from labels
func (r *Environment) GetUUID() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelResourceUUID]
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Environment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(r).
		Complete()
}
