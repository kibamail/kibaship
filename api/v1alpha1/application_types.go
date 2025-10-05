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

// ApplicationType defines the type of application
// +kubebuilder:validation:Enum=MySQL;MySQLCluster;Postgres;PostgresCluster;DockerImage;GitRepository
type ApplicationType string

const (
	// ApplicationTypeMySQL represents a MySQL database application
	ApplicationTypeMySQL ApplicationType = "MySQL"
	// ApplicationTypeMySQLCluster represents a MySQL cluster application
	ApplicationTypeMySQLCluster ApplicationType = "MySQLCluster"
	// ApplicationTypePostgres represents a PostgreSQL database application
	ApplicationTypePostgres ApplicationType = "Postgres"
	// ApplicationTypePostgresCluster represents a PostgreSQL cluster application
	ApplicationTypePostgresCluster ApplicationType = "PostgresCluster"
	// ApplicationTypeDockerImage represents a Docker image application
	ApplicationTypeDockerImage ApplicationType = "DockerImage"
	// ApplicationTypeGitRepository represents a Git repository application
	ApplicationTypeGitRepository ApplicationType = "GitRepository"
)

// GitProvider defines the Git provider
// +kubebuilder:validation:Enum=github.com;gitlab.com;bitbucket.com
type GitProvider string

const (
	// GitProviderGitHub represents GitHub provider
	GitProviderGitHub GitProvider = "github.com"
	// GitProviderGitLab represents GitLab provider
	GitProviderGitLab GitProvider = "gitlab.com"
	// GitProviderBitbucket represents Bitbucket provider
	GitProviderBitbucket GitProvider = "bitbucket.com"
)

// GitRepositoryConfig defines the configuration for GitRepository applications
type GitRepositoryConfig struct {
	// Provider is the Git provider (github.com, gitlab.com, bitbucket.com)
	// +kubebuilder:validation:Required
	Provider GitProvider `json:"provider"`

	// Repository is the repository name in the format <org-name>/<repo-name>
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`
	Repository string `json:"repository"`

	// PublicAccess indicates if the repository is publicly accessible
	// If true, SecretRef is optional. If false, SecretRef is required and must exist in project namespace
	// +kubebuilder:default=false
	// +optional
	PublicAccess bool `json:"publicAccess,omitempty"`

	// SecretRef references the secret containing the git access token
	// Required when PublicAccess is false, optional when PublicAccess is true
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`

	// Branch is the git branch to use (optional, defaults to main/master)
	// +optional
	Branch string `json:"branch,omitempty"`

	// Path is the path within the repository (optional, defaults to root)
	// +optional
	Path string `json:"path,omitempty"`

	// RootDirectory is the root directory for the application (optional, defaults to ./)
	// +kubebuilder:default="./"
	// +optional
	RootDirectory string `json:"rootDirectory,omitempty"`

	// BuildCommand is the command to build the application (optional)
	// +optional
	BuildCommand string `json:"buildCommand,omitempty"`

	// StartCommand is the command to start the application (optional)
	// +optional
	StartCommand string `json:"startCommand,omitempty"`

	// Env is a reference to a secret containing environment variables for this application (optional)
	// +optional
	Env *corev1.LocalObjectReference `json:"env,omitempty"`

	// SpaOutputDirectory is the output directory for SPA builds (optional)
	// +optional
	SpaOutputDirectory string `json:"spaOutputDirectory,omitempty"`
}

// DockerImageConfig defines the configuration for DockerImage applications
type DockerImageConfig struct {
	// Image is the Docker image reference (e.g., nginx:latest, registry.com/org/image:tag)
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// ImagePullSecretRef references the secret containing image pull credentials
	// +optional
	ImagePullSecretRef *corev1.LocalObjectReference `json:"imagePullSecretRef,omitempty"`

	// Tag is the image tag (optional if already specified in Image)
	// +optional
	Tag string `json:"tag,omitempty"`
}

// MySQLConfig defines the configuration for MySQL applications
type MySQLConfig struct {
	// Version is the MySQL version to deploy
	// +optional
	Version string `json:"version,omitempty"`

	// Database is the initial database name to create
	// +optional
	Database string `json:"database,omitempty"`

	// SecretRef references the secret containing MySQL credentials
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
}

// MySQLClusterConfig defines the configuration for MySQL cluster applications
type MySQLClusterConfig struct {
	// Version is the MySQL version to deploy
	// +optional
	Version string `json:"version,omitempty"`

	// Replicas is the number of MySQL instances in the cluster
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3
	Replicas int32 `json:"replicas,omitempty"`

	// Database is the initial database name to create
	// +optional
	Database string `json:"database,omitempty"`

	// SecretRef references the secret containing MySQL credentials
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
}

// PostgresConfig defines the configuration for PostgreSQL applications
type PostgresConfig struct {
	// Version is the PostgreSQL version to deploy
	// +optional
	Version string `json:"version,omitempty"`

	// Database is the initial database name to create
	// +optional
	Database string `json:"database,omitempty"`

	// SecretRef references the secret containing PostgreSQL credentials
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
}

// PostgresClusterConfig defines the configuration for PostgreSQL cluster applications
type PostgresClusterConfig struct {
	// Version is the PostgreSQL version to deploy
	// +optional
	Version string `json:"version,omitempty"`

	// Replicas is the number of PostgreSQL instances in the cluster
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3
	Replicas int32 `json:"replicas,omitempty"`

	// Database is the initial database name to create
	// +optional
	Database string `json:"database,omitempty"`

	// SecretRef references the secret containing PostgreSQL credentials
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
}

// ApplicationSpec defines the desired state of Application.
type ApplicationSpec struct {
	// EnvironmentRef references the Environment this application belongs to
	// +kubebuilder:validation:Required
	EnvironmentRef corev1.LocalObjectReference `json:"environmentRef"`

	// Type defines the type of application
	// +kubebuilder:validation:Required
	Type ApplicationType `json:"type"`

	// GitRepository contains configuration for GitRepository applications
	// +optional
	GitRepository *GitRepositoryConfig `json:"gitRepository,omitempty"`

	// DockerImage contains configuration for DockerImage applications
	// +optional
	DockerImage *DockerImageConfig `json:"dockerImage,omitempty"`

	// MySQL contains configuration for MySQL applications
	// +optional
	MySQL *MySQLConfig `json:"mysql,omitempty"`

	// MySQLCluster contains configuration for MySQLCluster applications
	// +optional
	MySQLCluster *MySQLClusterConfig `json:"mysqlCluster,omitempty"`

	// Postgres contains configuration for Postgres applications
	// +optional
	Postgres *PostgresConfig `json:"postgres,omitempty"`

	// PostgresCluster contains configuration for PostgresCluster applications
	// +optional
	PostgresCluster *PostgresClusterConfig `json:"postgresCluster,omitempty"`
}

// ApplicationStatus defines the observed state of Application.
type ApplicationStatus struct {
	// Phase represents the current phase of the application lifecycle
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the application's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Message provides additional information about the current status
	// +optional
	Message string `json:"message,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed Application
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Environment",type="string",JSONPath=".spec.environmentRef.name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:webhook:path=/validate-platform-operator-kibaship-com-v1alpha1-application,mutating=false,failurePolicy=fail,sideEffects=None,groups=platform.operator.kibaship.com,resources=applications,verbs=create;update,versions=v1alpha1,name=vapplication.kb.io,admissionReviewVersions=v1

// Application is the Schema for the applications API.
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application.
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}

var _ webhook.CustomValidator = &Application{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Application) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	applicationlog := logf.Log.WithName("application-resource")

	app, ok := obj.(*Application)
	if !ok {
		return nil, fmt.Errorf("expected an Application object, but got %T", obj)
	}

	applicationlog.Info("validate create", "name", app.Name)

	return nil, app.validateApplication(ctx)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Application) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	applicationlog := logf.Log.WithName("application-resource")

	app, ok := newObj.(*Application)
	if !ok {
		return nil, fmt.Errorf("expected an Application object, but got %T", newObj)
	}

	applicationlog.Info("validate update", "name", app.Name)

	return nil, app.validateApplication(ctx)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Application) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	applicationlog := logf.Log.WithName("application-resource")

	app, ok := obj.(*Application)
	if !ok {
		return nil, fmt.Errorf("expected an Application object, but got %T", obj)
	}

	applicationlog.Info("validate delete", "name", app.Name)

	return nil, nil
}

// validateApplication validates the Application resource
func (r *Application) validateApplication(ctx context.Context) error {
	_ = ctx // context is not used in current validation but required for webhook interface
	var errors []string

	// Use the centralized labeling validation
	// Note: In webhook context, we don't have access to the client for uniqueness checks
	// Uniqueness will be validated in the controller reconcile loop
	labels := r.GetLabels()
	if labels == nil {
		errors = append(errors, "application must have labels")
	} else {
		// Validate UUID
		if resourceUUID, exists := labels[validation.LabelResourceUUID]; !exists {
			errors = append(errors, fmt.Sprintf("application must have label %s", validation.LabelResourceUUID))
		} else if !validation.ValidateUUID(resourceUUID) {
			errors = append(errors, fmt.Sprintf("application UUID must be valid: %s", resourceUUID))
		}

		// Validate Slug
		if resourceSlug, exists := labels[validation.LabelResourceSlug]; !exists {
			errors = append(errors, fmt.Sprintf("application must have label %s", validation.LabelResourceSlug))
		} else if !validation.ValidateSlug(resourceSlug) {
			errors = append(errors, fmt.Sprintf("application slug must be valid: %s", resourceSlug))
		}

		// Validate Project UUID
		if projectUUID, exists := labels[validation.LabelProjectUUID]; !exists {
			errors = append(errors, fmt.Sprintf("application must have label %s", validation.LabelProjectUUID))
		} else if !validation.ValidateUUID(projectUUID) {
			errors = append(errors, fmt.Sprintf("project UUID must be valid: %s", projectUUID))
		}

		// Validate Environment UUID
		if environmentUUID, exists := labels[validation.LabelEnvironmentUUID]; !exists {
			errors = append(errors, fmt.Sprintf("application must have label %s", validation.LabelEnvironmentUUID))
		} else if !validation.ValidateUUID(environmentUUID) {
			errors = append(errors, fmt.Sprintf("environment UUID must be valid: %s", environmentUUID))
		}
	}

	// Validate application name format: application-<slug>-kibaship-com
	if !r.isValidApplicationName() {
		errors = append(errors, fmt.Sprintf("application name '%s' must follow format 'application-<slug>-kibaship-com'", r.Name))
	}

	// Validate GitRepository configuration
	if r.Spec.Type == ApplicationTypeGitRepository && r.Spec.GitRepository != nil {
		if err := r.validateGitRepository(); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// validateGitRepository validates GitRepository configuration
func (r *Application) validateGitRepository() error {
	gitRepo := r.Spec.GitRepository

	// Validate SecretRef based on PublicAccess setting
	if !gitRepo.PublicAccess {
		// For private repositories, SecretRef is required
		if gitRepo.SecretRef == nil {
			return fmt.Errorf("SecretRef is required when PublicAccess is false")
		}

		// TODO: In a real implementation, we would validate that the secret exists in the project namespace
		// This would require access to the Kubernetes client, which isn't available in the webhook validation
		// The actual secret existence validation should be done in the controller reconcile loop
	}

	return nil
}

// isValidApplicationName validates if the application name follows the required format
func (r *Application) isValidApplicationName() bool {
	// Pattern: application-<slug>-kibaship-com
	// slug should be valid DNS label (lowercase alphanumeric with hyphens)
	pattern := regexp.MustCompile(`^application-[a-z0-9]([a-z0-9-]*[a-z0-9])?-kibaship-com$`)
	return pattern.MatchString(r.Name)
}

// GetSlug returns the application slug from labels
func (r *Application) GetSlug() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelResourceSlug]
}

// GetProjectUUID returns the project UUID from labels
func (r *Application) GetProjectUUID() string {
	if r.Labels == nil {
		return ""
	}
	return r.Labels[validation.LabelProjectUUID]
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Application) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(r).
		Complete()
}
