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
	"time"

	"github.com/google/uuid"
	"github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/validation"
)

type DeploymentPhase string

const (
	DeploymentPhaseInitializing DeploymentPhase = "Initializing"
	DeploymentPhaseRunning      DeploymentPhase = "Running"
	DeploymentPhaseSucceeded    DeploymentPhase = "Succeeded"
	DeploymentPhaseFailed       DeploymentPhase = "Failed"
	DeploymentPhaseWaiting      DeploymentPhase = "Waiting"
)

// GitRepositoryDeploymentConfig defines the configuration for GitRepository deployments
type GitRepositoryDeploymentConfig struct {
	CommitSHA string `json:"commitSHA" example:"abc123def456" validate:"required"`
	Branch    string `json:"branch,omitempty" example:"main"`
}

// DeploymentCreateRequest represents the request to create a new deployment
type DeploymentCreateRequest struct {
	ApplicationUUID string                         `json:"applicationUuid" example:"550e8400-e29b-41d4-a716-446655440001" validate:"required"`
	Promote         bool                           `json:"promote,omitempty" example:"false"`
	GitRepository   *GitRepositoryDeploymentConfig `json:"gitRepository,omitempty"`
}

// DeploymentResponse represents the deployment data returned to clients
type DeploymentResponse struct {
	UUID            string                         `json:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Slug            string                         `json:"slug" example:"def456gh"`
	ApplicationUUID string                         `json:"applicationUuid" example:"550e8400-e29b-41d4-a716-446655440001"`
	ApplicationSlug string                         `json:"applicationSlug" example:"abc123de"`
	ProjectUUID     string                         `json:"projectUuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	Phase           DeploymentPhase                `json:"phase" example:"Initializing"`
	GitRepository   *GitRepositoryDeploymentConfig `json:"gitRepository,omitempty"`
	CreatedAt       time.Time                      `json:"createdAt" example:"2023-01-01T12:00:00Z"`
	UpdatedAt       time.Time                      `json:"updatedAt" example:"2023-01-01T12:00:00Z"`
}

// Deployment represents the internal deployment model
type Deployment struct {
	UUID            string
	Slug            string
	ApplicationUUID string
	ApplicationSlug string
	ProjectUUID     string
	Phase           DeploymentPhase
	GitRepository   *GitRepositoryDeploymentConfig
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewDeployment creates a new deployment with the given parameters
func NewDeployment(applicationUUID, applicationSlug, projectUUID, slug string, gitRepo *GitRepositoryDeploymentConfig) *Deployment {
	now := time.Now()
	return &Deployment{
		UUID:            uuid.New().String(),
		Slug:            slug,
		ApplicationUUID: applicationUUID,
		ApplicationSlug: applicationSlug,
		ProjectUUID:     projectUUID,
		Phase:           DeploymentPhaseInitializing,
		GitRepository:   gitRepo,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// ToResponse converts the internal deployment to a response model
func (d *Deployment) ToResponse() DeploymentResponse {
	return DeploymentResponse{
		UUID:            d.UUID,
		Slug:            d.Slug,
		ApplicationUUID: d.ApplicationUUID,
		ApplicationSlug: d.ApplicationSlug,
		ProjectUUID:     d.ProjectUUID,
		Phase:           d.Phase,
		GitRepository:   d.GitRepository,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

// Validate validates the deployment create request
func (req *DeploymentCreateRequest) Validate() *ValidationErrors {
	var validationErrors []ValidationError

	if req.ApplicationUUID == "" {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "applicationUuid",
			Message: "Application UUID is required",
		})
	} else if !validation.ValidateUUID(req.ApplicationUUID) {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "applicationUuid",
			Message: "Application UUID must be a valid UUID",
		})
	}

	// Validate GitRepository config if provided
	if req.GitRepository != nil {
		if req.GitRepository.CommitSHA == "" {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "gitRepository.commitSHA",
				Message: "Commit SHA is required for GitRepository deployments",
			})
		}
	}

	if len(validationErrors) > 0 {
		return &ValidationErrors{
			Errors: validationErrors,
		}
	}

	return nil
}

// ConvertFromCRD converts a Kubernetes Deployment CRD to internal model
func (d *Deployment) ConvertFromCRD(crd *v1alpha1.Deployment, applicationSlug string) {
	d.UUID = crd.GetLabels()[validation.LabelResourceUUID]
	d.Slug = crd.GetLabels()[validation.LabelResourceSlug]
	d.ApplicationUUID = crd.GetLabels()[validation.LabelApplicationUUID]
	d.ApplicationSlug = applicationSlug
	d.ProjectUUID = crd.GetLabels()[validation.LabelProjectUUID]
	d.Phase = DeploymentPhase(crd.Status.Phase)
	d.CreatedAt = crd.CreationTimestamp.Time
	d.UpdatedAt = crd.CreationTimestamp.Time

	// Convert GitRepository config if present
	if crd.Spec.GitRepository != nil {
		d.GitRepository = &GitRepositoryDeploymentConfig{
			CommitSHA: crd.Spec.GitRepository.CommitSHA,
			Branch:    crd.Spec.GitRepository.Branch,
		}
	}
}
