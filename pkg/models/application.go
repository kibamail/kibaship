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

package models

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/validation"
)

// ApplicationType represents the type of application
type ApplicationType string

const (
	ApplicationTypeMySQL           ApplicationType = "MySQL"
	ApplicationTypeMySQLCluster    ApplicationType = "MySQLCluster"
	ApplicationTypePostgres        ApplicationType = "Postgres"
	ApplicationTypePostgresCluster ApplicationType = "PostgresCluster"
	ApplicationTypeDockerImage     ApplicationType = "DockerImage"
	ApplicationTypeGitRepository   ApplicationType = "GitRepository"
)

// GitProvider represents the Git provider
type GitProvider string

const (
	GitProviderGitHub    GitProvider = "github.com"
	GitProviderGitLab    GitProvider = "gitlab.com"
	GitProviderBitbucket GitProvider = "bitbucket.com"
)

// HealthCheckConfig defines the health check configuration for an application
type HealthCheckConfig struct {
	Path                string `json:"path,omitempty" example:"/health"`
	Port                int32  `json:"port,omitempty" example:"3000"`
	InitialDelaySeconds int32  `json:"initialDelaySeconds,omitempty" example:"30"`
	PeriodSeconds       int32  `json:"periodSeconds,omitempty" example:"10"`
	TimeoutSeconds      int32  `json:"timeoutSeconds,omitempty" example:"5"`
	SuccessThreshold    int32  `json:"successThreshold,omitempty" example:"1"`
	FailureThreshold    int32  `json:"failureThreshold,omitempty" example:"3"`
}

// GitRepositoryConfig defines configuration for GitRepository applications
type GitRepositoryConfig struct {
	Provider           GitProvider        `json:"provider" example:"github.com"`
	Repository         string             `json:"repository" example:"myorg/myapp"`
	PublicAccess       bool               `json:"publicAccess,omitempty" example:"false"`
	SecretRef          *string            `json:"secretRef,omitempty" example:"git-credentials"`
	Branch             string             `json:"branch,omitempty" example:"main"`
	Path               string             `json:"path,omitempty" example:""`
	RootDirectory      string             `json:"rootDirectory,omitempty" example:"./"`
	BuildCommand       string             `json:"buildCommand,omitempty" example:"npm run build"`
	StartCommand       string             `json:"startCommand,omitempty" example:"npm start"`
	SpaOutputDirectory string             `json:"spaOutputDirectory,omitempty" example:"dist"`
	HealthCheck        *HealthCheckConfig `json:"healthCheck,omitempty"`
}

// DockerImageConfig defines configuration for DockerImage applications
type DockerImageConfig struct {
	Image              string             `json:"image" example:"nginx:latest"`
	ImagePullSecretRef *string            `json:"imagePullSecretRef,omitempty" example:"docker-registry-secret"`
	Tag                string             `json:"tag,omitempty" example:"v1.0.0"`
	HealthCheck        *HealthCheckConfig `json:"healthCheck,omitempty"`
}

// MySQLConfig defines configuration for MySQL applications
type MySQLConfig struct {
	Version   string  `json:"version,omitempty" example:"8.0"`
	Database  string  `json:"database,omitempty" example:"myapp"`
	SecretRef *string `json:"secretRef,omitempty" example:"mysql-credentials"`
}

// MySQLClusterConfig defines configuration for MySQL cluster applications
type MySQLClusterConfig struct {
	Version   string  `json:"version,omitempty" example:"8.0"`
	Replicas  int32   `json:"replicas,omitempty" example:"3"`
	Database  string  `json:"database,omitempty" example:"myapp"`
	SecretRef *string `json:"secretRef,omitempty" example:"mysql-cluster-credentials"`
}

// PostgresConfig defines configuration for Postgres applications
type PostgresConfig struct {
	Version   string  `json:"version,omitempty" example:"15"`
	Database  string  `json:"database,omitempty" example:"myapp"`
	SecretRef *string `json:"secretRef,omitempty" example:"postgres-credentials"`
}

// PostgresClusterConfig defines configuration for Postgres cluster applications
type PostgresClusterConfig struct {
	Version   string  `json:"version,omitempty" example:"15"`
	Replicas  int32   `json:"replicas,omitempty" example:"3"`
	Database  string  `json:"database,omitempty" example:"myapp"`
	SecretRef *string `json:"secretRef,omitempty" example:"postgres-cluster-credentials"`
}

// ApplicationCreateRequest represents a request to create an application
type ApplicationCreateRequest struct {
	Name            string                 `json:"name" example:"my-web-app"`
	EnvironmentUUID string                 `json:"environmentUuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Type            ApplicationType        `json:"type" example:"DockerImage"`
	GitRepository   *GitRepositoryConfig   `json:"gitRepository,omitempty"`
	DockerImage     *DockerImageConfig     `json:"dockerImage,omitempty"`
	MySQL           *MySQLConfig           `json:"mysql,omitempty"`
	MySQLCluster    *MySQLClusterConfig    `json:"mysqlCluster,omitempty"`
	Postgres        *PostgresConfig        `json:"postgres,omitempty"`
	PostgresCluster *PostgresClusterConfig `json:"postgresCluster,omitempty"`
}

// ApplicationUpdateRequest represents a request to update an application
type ApplicationUpdateRequest struct {
	Name            *string                `json:"name,omitempty" example:"updated-web-app"`
	GitRepository   *GitRepositoryConfig   `json:"gitRepository,omitempty"`
	DockerImage     *DockerImageConfig     `json:"dockerImage,omitempty"`
	MySQL           *MySQLConfig           `json:"mysql,omitempty"`
	MySQLCluster    *MySQLClusterConfig    `json:"mysqlCluster,omitempty"`
	Postgres        *PostgresConfig        `json:"postgres,omitempty"`
	PostgresCluster *PostgresClusterConfig `json:"postgresCluster,omitempty"`
}

// ApplicationEnvUpdateRequest represents a request to update environment variables
type ApplicationEnvUpdateRequest struct {
	Variables map[string]string `json:"variables" example:"{\"API_KEY\":\"secret123\",\"DB_HOST\":\"localhost\"}"`
}

// Application represents an application in the system
type Application struct {
	UUID             string                 `json:"uuid"`
	Name             string                 `json:"name"`
	Slug             string                 `json:"slug"`
	ProjectUUID      string                 `json:"projectUuid"`
	ProjectSlug      string                 `json:"projectSlug"`
	EnvironmentUUID  string                 `json:"environmentUuid"`
	Type             ApplicationType        `json:"type"`
	GitRepository    *GitRepositoryConfig   `json:"gitRepository,omitempty"`
	DockerImage      *DockerImageConfig     `json:"dockerImage,omitempty"`
	MySQL            *MySQLConfig           `json:"mysql,omitempty"`
	MySQLCluster     *MySQLClusterConfig    `json:"mysqlCluster,omitempty"`
	Postgres         *PostgresConfig        `json:"postgres,omitempty"`
	PostgresCluster  *PostgresClusterConfig `json:"postgresCluster,omitempty"`
	Status           string                 `json:"status"`
	Domains          []*ApplicationDomain   `json:"domains,omitempty"`
	LatestDeployment *Deployment            `json:"latestDeployment,omitempty"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
}

// ApplicationResponse represents an application response
type ApplicationResponse struct {
	UUID             string                      `json:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name             string                      `json:"name" example:"my-web-app"`
	Slug             string                      `json:"slug" example:"abc123de"`
	ProjectUUID      string                      `json:"projectUuid" example:"123e4567-e89b-12d3-a456-426614174001"`
	ProjectSlug      string                      `json:"projectSlug" example:"xyz789ab"`
	Type             ApplicationType             `json:"type" example:"DockerImage"`
	GitRepository    *GitRepositoryConfig        `json:"gitRepository,omitempty"`
	DockerImage      *DockerImageConfig          `json:"dockerImage,omitempty"`
	MySQL            *MySQLConfig                `json:"mysql,omitempty"`
	MySQLCluster     *MySQLClusterConfig         `json:"mysqlCluster,omitempty"`
	Postgres         *PostgresConfig             `json:"postgres,omitempty"`
	PostgresCluster  *PostgresClusterConfig      `json:"postgresCluster,omitempty"`
	Status           string                      `json:"status" example:"Running"`
	Domains          []ApplicationDomainResponse `json:"domains,omitempty"`
	LatestDeployment *DeploymentResponse         `json:"latestDeployment,omitempty"`
	CreatedAt        time.Time                   `json:"createdAt" example:"2023-01-01T12:00:00Z"`
	UpdatedAt        time.Time                   `json:"updatedAt" example:"2023-01-01T12:00:00Z"`
}

// NewApplication creates a new Application with default values
func NewApplication(name, projectUUID, projectSlug string, appType ApplicationType, slug string) *Application {
	now := time.Now()
	return &Application{
		UUID:        uuid.New().String(),
		Name:        name,
		Slug:        slug,
		ProjectUUID: projectUUID,
		ProjectSlug: projectSlug,
		Type:        appType,
		Status:      "Pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Validate validates an application create request
func (req *ApplicationCreateRequest) Validate() *ValidationErrors {
	var errors []ValidationError

	// Validate name
	if strings.TrimSpace(req.Name) == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Application name is required",
		})
	}
	if len(req.Name) > 100 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Application name cannot exceed 100 characters",
		})
	}

	// Validate environment UUID
	if strings.TrimSpace(req.EnvironmentUUID) == "" {
		errors = append(errors, ValidationError{
			Field:   "environmentUuid",
			Message: "Environment UUID is required",
		})
	}
	if !validation.ValidateUUID(req.EnvironmentUUID) {
		errors = append(errors, ValidationError{
			Field:   "environmentUuid",
			Message: "Environment UUID must be a valid UUID",
		})
	}

	// Validate type
	if !isValidApplicationType(req.Type) {
		errors = append(errors, ValidationError{
			Field:   "type",
			Message: "Application type must be one of: MySQL, MySQLCluster, Postgres, PostgresCluster, DockerImage, GitRepository",
		})
	}

	// Validate type-specific configuration
	switch req.Type {
	case ApplicationTypeGitRepository:
		if req.GitRepository == nil {
			errors = append(errors, ValidationError{
				Field:   "gitRepository",
				Message: "GitRepository configuration is required for GitRepository applications",
			})
		} else {
			errors = append(errors, validateGitRepository(req.GitRepository)...)
		}
	case ApplicationTypeDockerImage:
		if req.DockerImage == nil {
			errors = append(errors, ValidationError{
				Field:   "dockerImage",
				Message: "DockerImage configuration is required for DockerImage applications",
			})
		} else {
			errors = append(errors, validateDockerImage(req.DockerImage)...)
		}
	case ApplicationTypeMySQL:
		if req.MySQL != nil {
			errors = append(errors, validateMySQL(req.MySQL)...)
		}
	case ApplicationTypeMySQLCluster:
		if req.MySQLCluster != nil {
			errors = append(errors, validateMySQLCluster(req.MySQLCluster)...)
		}
	case ApplicationTypePostgres:
		if req.Postgres != nil {
			errors = append(errors, validatePostgres(req.Postgres)...)
		}
	case ApplicationTypePostgresCluster:
		if req.PostgresCluster != nil {
			errors = append(errors, validatePostgresCluster(req.PostgresCluster)...)
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// ValidateUpdate validates an application update request
func (req *ApplicationUpdateRequest) ValidateUpdate() *ValidationErrors {
	var errors []ValidationError

	// Validate name if provided
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			errors = append(errors, ValidationError{
				Field:   "name",
				Message: "Application name cannot be empty",
			})
		}
		if len(*req.Name) > 100 {
			errors = append(errors, ValidationError{
				Field:   "name",
				Message: "Application name cannot exceed 100 characters",
			})
		}
	}

	// Validate configurations if provided
	if req.GitRepository != nil {
		errors = append(errors, validateGitRepository(req.GitRepository)...)
	}
	if req.DockerImage != nil {
		errors = append(errors, validateDockerImage(req.DockerImage)...)
	}
	if req.MySQL != nil {
		errors = append(errors, validateMySQL(req.MySQL)...)
	}
	if req.MySQLCluster != nil {
		errors = append(errors, validateMySQLCluster(req.MySQLCluster)...)
	}
	if req.Postgres != nil {
		errors = append(errors, validatePostgres(req.Postgres)...)
	}
	if req.PostgresCluster != nil {
		errors = append(errors, validatePostgresCluster(req.PostgresCluster)...)
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// ToResponse converts an Application to an ApplicationResponse
func (a *Application) ToResponse() ApplicationResponse {
	var domains []ApplicationDomainResponse
	if a.Domains != nil {
		domains = make([]ApplicationDomainResponse, len(a.Domains))
		for i, domain := range a.Domains {
			domains[i] = domain.ToResponse()
		}
	}

	var latestDeployment *DeploymentResponse
	if a.LatestDeployment != nil {
		response := a.LatestDeployment.ToResponse()
		latestDeployment = &response
	}

	return ApplicationResponse{
		UUID:             a.UUID,
		Name:             a.Name,
		Slug:             a.Slug,
		ProjectUUID:      a.ProjectUUID,
		ProjectSlug:      a.ProjectSlug,
		Type:             a.Type,
		GitRepository:    a.GitRepository,
		DockerImage:      a.DockerImage,
		MySQL:            a.MySQL,
		MySQLCluster:     a.MySQLCluster,
		Postgres:         a.Postgres,
		PostgresCluster:  a.PostgresCluster,
		Status:           a.Status,
		Domains:          domains,
		LatestDeployment: latestDeployment,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}

// Helper functions for validation

func isValidApplicationType(appType ApplicationType) bool {
	return appType == ApplicationTypeMySQL ||
		appType == ApplicationTypeMySQLCluster ||
		appType == ApplicationTypePostgres ||
		appType == ApplicationTypePostgresCluster ||
		appType == ApplicationTypeDockerImage ||
		appType == ApplicationTypeGitRepository
}

func isValidSlug(slug string) bool {
	// Validate 8-character alphanumeric slug
	slugRegex := regexp.MustCompile(`^[a-z0-9]{8}$`)
	return slugRegex.MatchString(slug)
}

func isValidGitProvider(provider GitProvider) bool {
	return provider == GitProviderGitHub ||
		provider == GitProviderGitLab ||
		provider == GitProviderBitbucket
}

func validateGitRepository(config *GitRepositoryConfig) []ValidationError {
	var errors []ValidationError

	// Validate provider
	if !isValidGitProvider(config.Provider) {
		errors = append(errors, ValidationError{
			Field:   "gitRepository.provider",
			Message: "Provider must be one of: github.com, gitlab.com, bitbucket.com",
		})
	}

	// Validate repository format
	repoRegex := regexp.MustCompile(`^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`)
	if !repoRegex.MatchString(config.Repository) {
		errors = append(errors, ValidationError{
			Field:   "gitRepository.repository",
			Message: "Repository must be in format 'org/repo'",
		})
	}

	// Validate SecretRef for private repositories
	if !config.PublicAccess && config.SecretRef == nil {
		errors = append(errors, ValidationError{
			Field:   "gitRepository.secretRef",
			Message: "SecretRef is required when PublicAccess is false",
		})
	}

	return errors
}

func validateDockerImage(config *DockerImageConfig) []ValidationError {
	var errors []ValidationError

	// Validate image
	if strings.TrimSpace(config.Image) == "" {
		errors = append(errors, ValidationError{
			Field:   "dockerImage.image",
			Message: "Docker image is required",
		})
	}

	return errors
}

func validateMySQL(config *MySQLConfig) []ValidationError {
	var errors []ValidationError
	// MySQL validation can be added here if needed
	return errors
}

func validateMySQLCluster(config *MySQLClusterConfig) []ValidationError {
	var errors []ValidationError

	// Validate replicas
	if config.Replicas < 1 {
		errors = append(errors, ValidationError{
			Field:   "mysqlCluster.replicas",
			Message: "Replicas must be at least 1",
		})
	}

	return errors
}

func validatePostgres(config *PostgresConfig) []ValidationError {
	var errors []ValidationError
	// Postgres validation can be added here if needed
	return errors
}

func validatePostgresCluster(config *PostgresClusterConfig) []ValidationError {
	var errors []ValidationError

	// Validate replicas
	if config.Replicas < 1 {
		errors = append(errors, ValidationError{
			Field:   "postgresCluster.replicas",
			Message: "Replicas must be at least 1",
		})
	}

	return errors
}

// ConvertFromCRD converts a Kubernetes Application CRD to internal model
func (a *Application) ConvertFromCRD(crd *v1alpha1.Application) {
	a.UUID = crd.GetLabels()[validation.LabelResourceUUID]
	a.Slug = crd.GetLabels()[validation.LabelResourceSlug]
	a.ProjectUUID = crd.GetLabels()[validation.LabelProjectUUID]
	a.EnvironmentUUID = crd.GetLabels()[validation.LabelEnvironmentUUID]
	a.Name = crd.GetAnnotations()[validation.AnnotationResourceName]
	a.Type = ApplicationType(crd.Spec.Type)
	a.CreatedAt = crd.CreationTimestamp.Time
	a.UpdatedAt = crd.CreationTimestamp.Time

	// Convert type-specific configurations
	switch crd.Spec.Type {
	case v1alpha1.ApplicationTypeGitRepository:
		if crd.Spec.GitRepository != nil {
			a.GitRepository = &GitRepositoryConfig{
				Repository: crd.Spec.GitRepository.Repository,
				Branch:     crd.Spec.GitRepository.Branch,
				Provider:   GitProvider(crd.Spec.GitRepository.Provider),
			}
		}
	case v1alpha1.ApplicationTypeDockerImage:
		if crd.Spec.DockerImage != nil {
			a.DockerImage = &DockerImageConfig{
				Image: crd.Spec.DockerImage.Image,
				Tag:   crd.Spec.DockerImage.Tag,
			}
		}
	case v1alpha1.ApplicationTypeMySQL:
		if crd.Spec.MySQL != nil {
			mysqlConfig := &MySQLConfig{
				Version:  crd.Spec.MySQL.Version,
				Database: crd.Spec.MySQL.Database,
			}
			if crd.Spec.MySQL.SecretRef != nil {
				mysqlConfig.SecretRef = &crd.Spec.MySQL.SecretRef.Name
			}
			a.MySQL = mysqlConfig
		}
	case v1alpha1.ApplicationTypeMySQLCluster:
		if crd.Spec.MySQLCluster != nil {
			mysqlClusterConfig := &MySQLClusterConfig{
				Version:  crd.Spec.MySQLCluster.Version,
				Database: crd.Spec.MySQLCluster.Database,
				Replicas: crd.Spec.MySQLCluster.Replicas,
			}
			if crd.Spec.MySQLCluster.SecretRef != nil {
				mysqlClusterConfig.SecretRef = &crd.Spec.MySQLCluster.SecretRef.Name
			}
			a.MySQLCluster = mysqlClusterConfig
		}
	case v1alpha1.ApplicationTypePostgres:
		if crd.Spec.Postgres != nil {
			postgresConfig := &PostgresConfig{
				Version:  crd.Spec.Postgres.Version,
				Database: crd.Spec.Postgres.Database,
			}
			if crd.Spec.Postgres.SecretRef != nil {
				postgresConfig.SecretRef = &crd.Spec.Postgres.SecretRef.Name
			}
			a.Postgres = postgresConfig
		}
	case v1alpha1.ApplicationTypePostgresCluster:
		if crd.Spec.PostgresCluster != nil {
			postgresClusterConfig := &PostgresClusterConfig{
				Version:  crd.Spec.PostgresCluster.Version,
				Database: crd.Spec.PostgresCluster.Database,
				Replicas: crd.Spec.PostgresCluster.Replicas,
			}
			if crd.Spec.PostgresCluster.SecretRef != nil {
				postgresClusterConfig.SecretRef = &crd.Spec.PostgresCluster.SecretRef.Name
			}
			a.PostgresCluster = postgresClusterConfig
		}
	}
}
