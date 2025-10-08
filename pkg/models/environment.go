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
	"time"

	"github.com/google/uuid"
	"github.com/kibamail/kibaship-operator/pkg/validation"
)

// EnvironmentCreateRequest represents the request to create an environment
type EnvironmentCreateRequest struct {
	Name        string            `json:"name" example:"production"`
	Description string            `json:"description,omitempty" example:"Production environment"`
	Variables   map[string]string `json:"variables,omitempty"`
	ProjectUUID string            `json:"projectUuid" example:"123e4567-e89b-12d3-a456-426614174001"`
}

// Validate validates the environment create request
func (r *EnvironmentCreateRequest) Validate() *ValidationErrors {
	errors := &ValidationErrors{
		Errors: []ValidationError{},
	}

	if r.Name == "" {
		errors.Errors = append(errors.Errors, ValidationError{
			Field:   "name",
			Message: "name is required",
		})
	}

	if r.ProjectUUID == "" {
		errors.Errors = append(errors.Errors, ValidationError{
			Field:   "projectUuid",
			Message: "projectUuid is required",
		})
	}

	if !validation.ValidateUUID(r.ProjectUUID) {
		errors.Errors = append(errors.Errors, ValidationError{
			Field:   "projectUuid",
			Message: "projectUuid must be a valid UUID",
		})
	}

	if len(errors.Errors) > 0 {
		return errors
	}

	return nil
}

// EnvironmentUpdateRequest represents the request to update an environment
type EnvironmentUpdateRequest struct {
	Description *string            `json:"description,omitempty" example:"Updated production environment"`
	Variables   *map[string]string `json:"variables,omitempty"`
}

// Validate validates the environment update request
func (r *EnvironmentUpdateRequest) Validate() error {
	// At least one field must be provided
	if r.Description == nil && r.Variables == nil {
		return fmt.Errorf("at least one field must be provided for update")
	}

	return nil
}

// Environment represents an environment in the system
type Environment struct {
	UUID             string            `json:"uuid"`
	Name             string            `json:"name"`
	Slug             string            `json:"slug"`
	Description      string            `json:"description,omitempty"`
	Variables        map[string]string `json:"variables,omitempty"`
	ProjectUUID      string            `json:"projectUuid"`
	ProjectSlug      string            `json:"projectSlug"`
	ApplicationCount int32             `json:"applicationCount"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

// NewEnvironment creates a new Environment
func NewEnvironment(name, projectUUID, projectSlug, slug string) *Environment {
	now := time.Now()
	return &Environment{
		UUID:        uuid.New().String(),
		Name:        name,
		Slug:        slug,
		ProjectUUID: projectUUID,
		ProjectSlug: projectSlug,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// EnvironmentResponse represents an environment response
type EnvironmentResponse struct {
	UUID             string            `json:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name             string            `json:"name" example:"production"`
	Slug             string            `json:"slug" example:"abc123de"`
	Description      string            `json:"description,omitempty" example:"Production environment"`
	Variables        map[string]string `json:"variables,omitempty"`
	ProjectUUID      string            `json:"projectUuid" example:"123e4567-e89b-12d3-a456-426614174001"`
	ProjectSlug      string            `json:"projectSlug" example:"xyz789ab"`
	ApplicationCount int32             `json:"applicationCount" example:"5"`
	CreatedAt        time.Time         `json:"createdAt" example:"2023-01-01T00:00:00Z"`
	UpdatedAt        time.Time         `json:"updatedAt" example:"2023-01-01T00:00:00Z"`
}

// ToResponse converts an Environment to EnvironmentResponse
func (e *Environment) ToResponse() *EnvironmentResponse {
	return &EnvironmentResponse{
		UUID:             e.UUID,
		Name:             e.Name,
		Slug:             e.Slug,
		Description:      e.Description,
		Variables:        e.Variables,
		ProjectUUID:      e.ProjectUUID,
		ProjectSlug:      e.ProjectSlug,
		ApplicationCount: e.ApplicationCount,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}
