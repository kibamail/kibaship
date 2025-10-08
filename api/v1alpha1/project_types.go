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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kibamail/kibaship-operator/pkg/validation"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ResourceLimits defines resource constraints for applications
type ResourceLimits struct {
	// CPU limit in cores (e.g., "2", "0.5")
	// +kubebuilder:validation:Pattern=^[0-9]+(\.[0-9]+)?$
	CPU string `json:"cpu,omitempty"`

	// Memory limit (e.g., "4Gi", "512Mi")
	// +kubebuilder:validation:Pattern=^[0-9]+(\.[0-9]+)?(Mi|Gi|Ti)$
	Memory string `json:"memory,omitempty"`

	// Storage limit (e.g., "20Gi", "100Mi")
	// +kubebuilder:validation:Pattern=^[0-9]+(\.[0-9]+)?(Mi|Gi|Ti)$
	Storage string `json:"storage,omitempty"`
}

// ResourceBounds defines minimum and maximum resource constraints
type ResourceBounds struct {
	// Minimum resource limits
	Min ResourceLimits `json:"min,omitempty"`

	// Maximum resource limits
	Max ResourceLimits `json:"max,omitempty"`
}

// ClusterResourceLimits defines resource constraints for cluster-type applications
type ClusterResourceLimits struct {
	// CPU limit per node in cores (e.g., "2", "0.5")
	// +kubebuilder:validation:Pattern=^[0-9]+(\.[0-9]+)?$
	CPU string `json:"cpu,omitempty"`

	// Memory limit per node (e.g., "4Gi", "512Mi")
	// +kubebuilder:validation:Pattern=^[0-9]+(\.[0-9]+)?(Mi|Gi|Ti)$
	Memory string `json:"memory,omitempty"`

	// Storage limit per node (e.g., "20Gi", "100Mi")
	// +kubebuilder:validation:Pattern=^[0-9]+(\.[0-9]+)?(Mi|Gi|Ti)$
	Storage string `json:"storage,omitempty"`

	// Default number of nodes in the cluster
	// +kubebuilder:validation:Minimum=1
	Nodes int32 `json:"nodes,omitempty"`
}

// ClusterResourceBounds defines minimum and maximum resource constraints for clusters
type ClusterResourceBounds struct {
	// Minimum resource limits per node
	Min ClusterResourceLimits `json:"min,omitempty"`

	// Maximum resource limits per node
	Max ClusterResourceLimits `json:"max,omitempty"`
}

// ApplicationTypeConfig defines configuration for a specific application type
type ApplicationTypeConfig struct {
	// Whether this application type is enabled in the project
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// Default resource limits for applications of this type
	DefaultLimits ResourceLimits `json:"defaultLimits,omitempty"`

	// Resource bounds (min/max) for applications of this type
	ResourceBounds ResourceBounds `json:"resourceBounds,omitempty"`
}

// ClusterApplicationTypeConfig defines configuration for cluster-type applications
type ClusterApplicationTypeConfig struct {
	// Whether this application type is enabled in the project
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`

	// Default resource limits for cluster applications of this type
	DefaultLimits ClusterResourceLimits `json:"defaultLimits,omitempty"`

	// Resource bounds (min/max) for cluster applications of this type
	ResourceBounds ClusterResourceBounds `json:"resourceBounds,omitempty"`
}

// VolumeConfig defines volume-related configuration
type VolumeConfig struct {
	// Maximum storage size for volumes (e.g., "100Gi", "1Ti")
	// +kubebuilder:validation:Pattern=^[0-9]+(\.[0-9]+)?(Mi|Gi|Ti)$
	MaxStorageSize string `json:"maxStorageSize,omitempty"`
}

// ProjectSpec defines the desired state of Project.
type ProjectSpec struct {
	// Application type configurations defining resource limits and policies
	// for different types of applications that can be deployed in this project
	ApplicationTypes ApplicationTypesConfig `json:"applicationTypes,omitempty"`

	// Volume configuration for the project
	Volumes VolumeConfig `json:"volumes,omitempty"`
}

// ApplicationTypesConfig defines configurations for all supported application types
type ApplicationTypesConfig struct {
	// MySQL single-instance database configuration
	MySQL ApplicationTypeConfig `json:"mysql,omitempty"`

	// MySQL cluster configuration (disabled by default)
	MySQLCluster ClusterApplicationTypeConfig `json:"mysqlCluster,omitempty"`

	// PostgreSQL single-instance database configuration
	Postgres ApplicationTypeConfig `json:"postgres,omitempty"`

	// PostgreSQL cluster configuration (disabled by default)
	PostgresCluster ClusterApplicationTypeConfig `json:"postgresCluster,omitempty"`

	// Docker image application configuration
	DockerImage ApplicationTypeConfig `json:"dockerImage,omitempty"`

	// Git repository application configuration
	GitRepository ApplicationTypeConfig `json:"gitRepository,omitempty"`
}

// ProjectStatus defines the observed state of Project.
type ProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase represents the current phase of the project lifecycle
	// +kubebuilder:validation:Enum=Pending;Ready;Failed
	Phase string `json:"phase,omitempty"`

	// NamespaceName is the name of the namespace created for this project
	NamespaceName string `json:"namespaceName,omitempty"`

	// Message provides additional information about the current status
	Message string `json:"message,omitempty"`

	// LastReconcileTime is the timestamp of the last successful reconciliation
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
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

	project, ok := obj.(*Project)
	if !ok {
		return nil, fmt.Errorf("expected a Project object, but got %T", obj)
	}

	projectlog.Info("validate create", "name", project.Name)

	return nil, project.validateProject(ctx)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Project) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	projectlog := logf.Log.WithName("project-resource")

	project, ok := newObj.(*Project)
	if !ok {
		return nil, fmt.Errorf("expected a Project object, but got %T", newObj)
	}

	projectlog.Info("validate update", "name", project.Name)

	return nil, project.validateProject(ctx)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Project) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	projectlog := logf.Log.WithName("project-resource")

	project, ok := obj.(*Project)
	if !ok {
		return nil, fmt.Errorf("expected a Project object, but got %T", obj)
	}

	projectlog.Info("validate delete", "name", project.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

// validateProject validates the Project resource
func (r *Project) validateProject(ctx context.Context) error {
	_ = ctx // context is not used in current validation but required for webhook interface
	// Use the centralized labeling validation
	// Note: In webhook context, we don't have access to the client for uniqueness checks
	// Uniqueness will be validated in the controller reconcile loop
	labels := r.GetLabels()
	if labels == nil {
		return fmt.Errorf("project must have labels")
	}

	// Validate UUID
	resourceUUID, exists := labels[validation.LabelResourceUUID]
	if !exists {
		return fmt.Errorf("project must have label %s", validation.LabelResourceUUID)
	}
	if !validation.ValidateUUID(resourceUUID) {
		return fmt.Errorf("project UUID must be valid: %s", resourceUUID)
	}

	// Validate Slug
	resourceSlug, exists := labels[validation.LabelResourceSlug]
	if !exists {
		return fmt.Errorf("project must have label %s", validation.LabelResourceSlug)
	}
	if !validation.ValidateSlug(resourceSlug) {
		return fmt.Errorf("project slug must be valid: %s", resourceSlug)
	}

	// Validate Workspace UUID if present
	if workspaceUUID, exists := labels[validation.LabelWorkspaceUUID]; exists {
		if !validation.ValidateUUID(workspaceUUID) {
			return fmt.Errorf("workspace UUID must be valid: %s", workspaceUUID)
		}
	}

	return nil
}

// GetSlug returns the project slug from labels
func (r *Project) GetSlug() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelResourceSlug]
}

// GetUUID returns the project UUID from labels
func (r *Project) GetUUID() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelResourceUUID]
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Project) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(r).
		Complete()
}
