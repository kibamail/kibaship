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

	// SecretRef references the secret containing the git access token
	// +kubebuilder:validation:Required
	SecretRef corev1.LocalObjectReference `json:"secretRef"`

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
	// ProjectRef references the Project this application belongs to
	// +kubebuilder:validation:Required
	ProjectRef corev1.LocalObjectReference `json:"projectRef"`

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
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.projectRef.name"
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
	applicationlog.Info("validate create", "name", r.Name)

	return nil, r.validateApplication()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Application) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	applicationlog := logf.Log.WithName("application-resource")
	applicationlog.Info("validate update", "name", r.Name)

	return nil, r.validateApplication()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Application) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	applicationlog := logf.Log.WithName("application-resource")
	applicationlog.Info("validate delete", "name", r.Name)

	return nil, nil
}

// validateApplication validates the Application resource
func (r *Application) validateApplication() error {
	var errors []string

	// Validate application name format: project-<project-slug>-app-<app-slug>-kibaship-com
	if !r.isValidApplicationName() {
		errors = append(errors, fmt.Sprintf("application name '%s' must follow format 'project-<project-slug>-app-<app-slug>-kibaship-com'", r.Name))
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// isValidApplicationName validates if the application name follows the required format
func (r *Application) isValidApplicationName() bool {
	// Pattern: project-<project-slug>-app-<app-slug>-kibaship-com
	// project-slug and app-slug should be valid DNS labels (lowercase alphanumeric with hyphens)
	pattern := regexp.MustCompile(`^project-[a-z0-9]([a-z0-9-]*[a-z0-9])?-app-[a-z0-9]([a-z0-9-]*[a-z0-9])?-kibaship-com$`)
	return pattern.MatchString(r.Name)
}

// GetProjectSlugFromName extracts the project slug from application name
func (r *Application) GetProjectSlugFromName() string {
	// Extract project slug from name format: project-<project-slug>-app-<app-slug>-kibaship-com
	if !r.isValidApplicationName() {
		return ""
	}

	parts := strings.Split(r.Name, "-")
	if len(parts) < 5 {
		return ""
	}

	// Find the "app" delimiter and extract everything between "project" and "app"
	projectStart := 1 // after "project-"
	appIndex := -1
	for i, part := range parts {
		if part == "app" {
			appIndex = i
			break
		}
	}

	if appIndex == -1 || appIndex <= projectStart {
		return ""
	}

	return strings.Join(parts[projectStart:appIndex], "-")
}

// GetAppSlugFromName extracts the app slug from application name
func (r *Application) GetAppSlugFromName() string {
	// Extract app slug from name format: project-<project-slug>-app-<app-slug>-kibaship-com
	if !r.isValidApplicationName() {
		return ""
	}

	parts := strings.Split(r.Name, "-")
	if len(parts) < 5 {
		return ""
	}

	// Find the "app" delimiter and extract everything between "app" and "kibaship"
	appIndex := -1
	kibashipIndex := -1
	for i, part := range parts {
		if part == "app" {
			appIndex = i
		} else if part == "kibaship" {
			kibashipIndex = i
			break
		}
	}

	if appIndex == -1 || kibashipIndex == -1 || kibashipIndex <= appIndex+1 {
		return ""
	}

	return strings.Join(parts[appIndex+1:kibashipIndex], "-")
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Application) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(r).
		Complete()
}
