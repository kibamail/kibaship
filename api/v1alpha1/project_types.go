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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ProjectSpec defines the desired state of Project.
type ProjectSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// ProjectStatus defines the observed state of Project.
type ProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:webhook:path=/validate-platform-operator-kibaship-com-v1alpha1-project,mutating=false,failurePolicy=fail,sideEffects=None,groups=platform.operator.kibaship.com,resources=projects,verbs=create;update,versions=v1alpha1,name=vproject.kb.io,admissionReviewVersions=v1

// Project is the Schema for the projects API.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}

var _ webhook.CustomValidator = &Project{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Project) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	projectlog := logf.Log.WithName("project-resource")

	projectlog.Info("validate create", "name", r.Name)

	return nil, r.validateProject()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Project) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	projectlog := logf.Log.WithName("project-resource")

	projectlog.Info("validate update", "name", r.Name)

	return nil, r.validateProject()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Project) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	projectlog := logf.Log.WithName("project-resource")

	projectlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

// validateProject validates the Project resource
func (r *Project) validateProject() error {
	var errors []string

	// Validate required UUID label
	if uuid, exists := r.Labels["platform.kibaship.com/uuid"]; !exists {
		errors = append(errors, "required label 'platform.kibaship.com/uuid' is missing")
	} else if !isValidUUID(uuid) {
		errors = append(errors, fmt.Sprintf("label 'platform.kibaship.com/uuid' must be a valid UUID, got: %s", uuid))
	}

	// Validate workspace UUID label if present
	if workspaceUUID, exists := r.Labels["platform.kibaship.com/workspace-uuid"]; exists {
		if !isValidUUID(workspaceUUID) {
			errors = append(errors, fmt.Sprintf("label 'platform.kibaship.com/workspace-uuid' must be a valid UUID, got: %s", workspaceUUID))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// isValidUUID validates if a string is a valid UUID format
func isValidUUID(uuid string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return uuidRegex.MatchString(uuid)
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Project) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(r).
		Complete()
}
