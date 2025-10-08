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
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ResourceProfile represents the available resource configuration profiles
type ResourceProfile string

const (
	ResourceProfileDevelopment ResourceProfile = "development"
	ResourceProfileProduction  ResourceProfile = "production"
	ResourceProfileCustom      ResourceProfile = "custom"
)

// ApplicationTypeSettings represents the enablement settings for application types
type ApplicationTypeSettings struct {
	MySQL           *bool `json:"mysql,omitempty" example:"true"`
	MySQLCluster    *bool `json:"mysqlCluster,omitempty" example:"false"`
	Postgres        *bool `json:"postgres,omitempty" example:"true"`
	PostgresCluster *bool `json:"postgresCluster,omitempty" example:"false"`
	DockerImage     *bool `json:"dockerImage,omitempty" example:"true"`
	GitRepository   *bool `json:"gitRepository,omitempty" example:"true"`
}

// ResourceLimitsSpec represents resource limit configuration
type ResourceLimitsSpec struct {
	CPU     string `json:"cpu,omitempty" example:"1"`
	Memory  string `json:"memory,omitempty" example:"2Gi"`
	Storage string `json:"storage,omitempty" example:"10Gi"`
}

// ResourceBoundsSpec represents min/max resource constraints
type ResourceBoundsSpec struct {
	MinLimits ResourceLimitsSpec `json:"minLimits,omitempty"`
	MaxLimits ResourceLimitsSpec `json:"maxLimits,omitempty"`
}

// ApplicationTypeResourceConfig represents resource configuration for an application type
type ApplicationTypeResourceConfig struct {
	DefaultLimits  ResourceLimitsSpec `json:"defaultLimits,omitempty"`
	ResourceBounds ResourceBoundsSpec `json:"resourceBounds,omitempty"`
}

// CustomResourceLimits represents custom resource limits for all application types
type CustomResourceLimits struct {
	MySQL         *ApplicationTypeResourceConfig `json:"mysql,omitempty"`
	Postgres      *ApplicationTypeResourceConfig `json:"postgres,omitempty"`
	DockerImage   *ApplicationTypeResourceConfig `json:"dockerImage,omitempty"`
	GitRepository *ApplicationTypeResourceConfig `json:"gitRepository,omitempty"`
}

// VolumeSettings represents volume-related settings
type VolumeSettings struct {
	MaxStorageSize string `json:"maxStorageSize,omitempty" example:"100Gi"`
}

// ProjectCreateRequest represents the request payload for creating a project
type ProjectCreateRequest struct {
	Name                    string                   `json:"name" example:"my-awesome-project"`
	Description             string                   `json:"description,omitempty" example:"A project for my awesome application"`
	WorkspaceUUID           string                   `json:"workspaceUuid" example:"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`
	EnabledApplicationTypes *ApplicationTypeSettings `json:"enabledApplicationTypes,omitempty"`
	ResourceProfile         *ResourceProfile         `json:"resourceProfile,omitempty" example:"development"`
	CustomResourceLimits    *CustomResourceLimits    `json:"customResourceLimits,omitempty"`
	VolumeSettings          *VolumeSettings          `json:"volumeSettings,omitempty"`
}

// ProjectResponse represents the response when returning project information
type ProjectResponse struct {
	UUID                    string                  `json:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name                    string                  `json:"name" example:"my-awesome-project"`
	Slug                    string                  `json:"slug" example:"abc123de"`
	Description             string                  `json:"description" example:"A project for my awesome application"`
	WorkspaceUUID           string                  `json:"workspaceUuid" example:"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`
	EnabledApplicationTypes ApplicationTypeSettings `json:"enabledApplicationTypes"`
	ResourceProfile         ResourceProfile         `json:"resourceProfile" example:"development"`
	VolumeSettings          VolumeSettings          `json:"volumeSettings"`
	Status                  string                  `json:"status" example:"Ready"`
	NamespaceName           string                  `json:"namespaceName,omitempty" example:"project-550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt               time.Time               `json:"createdAt" example:"2023-01-01T12:00:00Z"`
	UpdatedAt               time.Time               `json:"updatedAt" example:"2023-01-01T12:00:00Z"`
}

// Project represents the internal project model
type Project struct {
	UUID                    string
	Name                    string
	Slug                    string
	Description             string
	WorkspaceUUID           string
	EnabledApplicationTypes ApplicationTypeSettings
	ResourceProfile         ResourceProfile
	VolumeSettings          VolumeSettings
	Status                  string
	NamespaceName           string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// ValidationError represents a validation error with field and message
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// NewProject creates a new project with generated UUID, slug, and timestamps
func NewProject(name, description, workspaceUUID, slug string,
	enabledTypes *ApplicationTypeSettings, profile *ResourceProfile,
	volumeSettings *VolumeSettings) *Project {
	now := time.Now()

	// Set defaults for application types if not provided
	if enabledTypes == nil {
		enabledTypes = getDefaultApplicationTypes()
	}

	// Set default resource profile if not provided
	if profile == nil {
		defaultProfile := ResourceProfileDevelopment
		profile = &defaultProfile
	}

	// Set default volume settings if not provided
	if volumeSettings == nil {
		volumeSettings = getDefaultVolumeSettings(*profile)
	}

	projectUUID := uuid.New().String()

	return &Project{
		UUID:                    projectUUID,
		Name:                    name,
		Slug:                    slug,
		Description:             description,
		WorkspaceUUID:           workspaceUUID,
		EnabledApplicationTypes: *enabledTypes,
		ResourceProfile:         *profile,
		VolumeSettings:          *volumeSettings,
		Status:                  "Pending",
		NamespaceName:           fmt.Sprintf("project-%s", projectUUID),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}

// Validate validates the project creation request
func (req *ProjectCreateRequest) Validate() *ValidationErrors {
	var errors []ValidationError

	// Validate name
	if strings.TrimSpace(req.Name) == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Project name is required and cannot be empty",
		})
	} else if len(req.Name) < 2 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Project name must be at least 2 characters long",
		})
	} else if len(req.Name) > 100 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Project name cannot exceed 100 characters",
		})
	}

	// Validate workspace UUID
	if strings.TrimSpace(req.WorkspaceUUID) == "" {
		errors = append(errors, ValidationError{
			Field:   "workspaceUuid",
			Message: "Workspace UUID is required",
		})
	} else if !isValidUUID(req.WorkspaceUUID) {
		errors = append(errors, ValidationError{
			Field:   "workspaceUuid",
			Message: "Workspace UUID must be a valid UUID format (e.g., 6ba7b810-9dad-11d1-80b4-00c04fd430c8)",
		})
	}

	// Validate resource profile
	if req.ResourceProfile != nil {
		if !isValidResourceProfile(*req.ResourceProfile) {
			errors = append(errors, ValidationError{
				Field:   "resourceProfile",
				Message: "Resource profile must be one of: development, production, custom",
			})
		}

		// If custom profile, custom resource limits should be provided
		if *req.ResourceProfile == ResourceProfileCustom && req.CustomResourceLimits == nil {
			errors = append(errors, ValidationError{
				Field:   "customResourceLimits",
				Message: "Custom resource limits are required when using 'custom' resource profile",
			})
		}
	}

	// Validate custom resource limits if provided
	if req.CustomResourceLimits != nil {
		errors = append(errors, validateCustomResourceLimits(req.CustomResourceLimits)...)
	}

	// Validate volume settings
	if req.VolumeSettings != nil && req.VolumeSettings.MaxStorageSize != "" {
		if !isValidStorageSize(req.VolumeSettings.MaxStorageSize) {
			errors = append(errors, ValidationError{
				Field:   "volumeSettings.maxStorageSize",
				Message: "Max storage size must be in valid format (e.g., '100Gi', '500Mi', '1Ti')",
			})
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

// ToResponse converts a Project to a ProjectResponse
func (p *Project) ToResponse() ProjectResponse {
	return ProjectResponse{
		UUID:                    p.UUID,
		Name:                    p.Name,
		Slug:                    p.Slug,
		Description:             p.Description,
		WorkspaceUUID:           p.WorkspaceUUID,
		EnabledApplicationTypes: p.EnabledApplicationTypes,
		ResourceProfile:         p.ResourceProfile,
		VolumeSettings:          p.VolumeSettings,
		Status:                  p.Status,
		NamespaceName:           p.NamespaceName,
		CreatedAt:               p.CreatedAt,
		UpdatedAt:               p.UpdatedAt,
	}
}

// Helper functions

func getDefaultApplicationTypes() *ApplicationTypeSettings {
	return &ApplicationTypeSettings{
		MySQL:           boolPtr(true),
		MySQLCluster:    boolPtr(false),
		Postgres:        boolPtr(true),
		PostgresCluster: boolPtr(false),
		DockerImage:     boolPtr(true),
		GitRepository:   boolPtr(true),
	}
}

func getDefaultVolumeSettings(profile ResourceProfile) *VolumeSettings {
	switch profile {
	case ResourceProfileProduction:
		return &VolumeSettings{MaxStorageSize: "500Gi"}
	case ResourceProfileCustom:
		return &VolumeSettings{MaxStorageSize: "100Gi"}
	default: // development
		return &VolumeSettings{MaxStorageSize: "50Gi"}
	}
}

func isValidUUID(s string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return uuidRegex.MatchString(s)
}

func isValidResourceProfile(profile ResourceProfile) bool {
	return profile == ResourceProfileDevelopment ||
		profile == ResourceProfileProduction ||
		profile == ResourceProfileCustom
}

func isValidStorageSize(size string) bool {
	storageRegex := regexp.MustCompile(`^[0-9]+(\.[0-9]+)?(Mi|Gi|Ti)$`)
	return storageRegex.MatchString(size)
}

func isValidCPU(cpu string) bool {
	cpuRegex := regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)
	return cpuRegex.MatchString(cpu)
}

func isValidMemory(memory string) bool {
	return isValidStorageSize(memory) // Same format as storage
}

func validateCustomResourceLimits(limits *CustomResourceLimits) []ValidationError {
	var errors []ValidationError

	if limits.MySQL != nil {
		errors = append(errors, validateApplicationTypeResourceConfig("customResourceLimits.mysql", limits.MySQL)...)
	}
	if limits.Postgres != nil {
		errors = append(errors, validateApplicationTypeResourceConfig("customResourceLimits.postgres", limits.Postgres)...)
	}
	if limits.DockerImage != nil {
		errors = append(errors, validateApplicationTypeResourceConfig("customResourceLimits.dockerImage", limits.DockerImage)...)
	}
	if limits.GitRepository != nil {
		errors = append(errors, validateApplicationTypeResourceConfig("customResourceLimits.gitRepository", limits.GitRepository)...)
	}

	return errors
}

func validateApplicationTypeResourceConfig(prefix string, config *ApplicationTypeResourceConfig) []ValidationError {
	var errors []ValidationError

	// Validate default limits
	if config.DefaultLimits.CPU != "" && !isValidCPU(config.DefaultLimits.CPU) {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.defaultLimits.cpu", prefix),
			Message: "CPU must be a valid number (e.g., '1', '0.5', '2.5')",
		})
	}
	if config.DefaultLimits.Memory != "" && !isValidMemory(config.DefaultLimits.Memory) {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.defaultLimits.memory", prefix),
			Message: "Memory must be in valid format (e.g., '1Gi', '512Mi', '2Ti')",
		})
	}
	if config.DefaultLimits.Storage != "" && !isValidStorageSize(config.DefaultLimits.Storage) {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("%s.defaultLimits.storage", prefix),
			Message: "Storage must be in valid format (e.g., '10Gi', '500Mi', '1Ti')",
		})
	}

	// Similar validation for min/max limits would go here

	return errors
}

// ProjectUpdateRequest represents a request to update a project (PATCH operation)
type ProjectUpdateRequest struct {
	Name                    *string                  `json:"name,omitempty" example:"updated-project-name"`
	Description             *string                  `json:"description,omitempty" example:"Updated project description"`
	EnabledApplicationTypes *ApplicationTypeSettings `json:"enabledApplicationTypes,omitempty"`
	ResourceProfile         *ResourceProfile         `json:"resourceProfile,omitempty" example:"production"`
	CustomResourceLimits    *CustomResourceLimits    `json:"customResourceLimits,omitempty"`
	VolumeSettings          *VolumeSettings          `json:"volumeSettings,omitempty"`
}

// ValidateUpdate validates a project update request
func (req *ProjectUpdateRequest) ValidateUpdate() *ValidationErrors {
	var errors []ValidationError

	// Validate name if provided
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			errors = append(errors, ValidationError{
				Field:   "name",
				Message: "Project name cannot be empty",
			})
		}
		if len(*req.Name) > 100 {
			errors = append(errors, ValidationError{
				Field:   "name",
				Message: "Project name cannot exceed 100 characters",
			})
		}
	}

	// Validate description if provided
	if req.Description != nil && len(*req.Description) > 500 {
		errors = append(errors, ValidationError{
			Field:   "description",
			Message: "Project description cannot exceed 500 characters",
		})
	}

	// Validate resource profile if provided
	if req.ResourceProfile != nil {
		if !isValidResourceProfile(*req.ResourceProfile) {
			errors = append(errors, ValidationError{
				Field:   "resourceProfile",
				Message: "Resource profile must be one of: development, production, custom",
			})
		}

		// If custom profile, custom resource limits should be provided
		if *req.ResourceProfile == ResourceProfileCustom && req.CustomResourceLimits == nil {
			errors = append(errors, ValidationError{
				Field:   "customResourceLimits",
				Message: "Custom resource limits are required when using 'custom' resource profile",
			})
		}
	}

	// Validate custom resource limits if provided
	if req.CustomResourceLimits != nil {
		errors = append(errors, validateCustomResourceLimits(req.CustomResourceLimits)...)
	}

	// Validate volume settings if provided
	if req.VolumeSettings != nil && req.VolumeSettings.MaxStorageSize != "" {
		if !isValidStorageSize(req.VolumeSettings.MaxStorageSize) {
			errors = append(errors, ValidationError{
				Field:   "volumeSettings.maxStorageSize",
				Message: "Max storage size must be in valid format (e.g., '100Gi', '500Mi', '1Ti')",
			})
		}
	}

	if len(errors) > 0 {
		return &ValidationErrors{Errors: errors}
	}

	return nil
}

func boolPtr(b bool) *bool {
	return &b
}
