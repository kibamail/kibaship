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
)

// ProjectCreateRequest represents the request payload for creating a project
type ProjectCreateRequest struct {
	Name        string `json:"name" binding:"required" example:"my-awesome-project"`
	Description string `json:"description" example:"A description of my project"`
}

// ProjectResponse represents the response when returning project information
type ProjectResponse struct {
	UUID        string    `json:"uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string    `json:"name" example:"my-awesome-project"`
	Description string    `json:"description" example:"A description of my project"`
	CreatedAt   time.Time `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}

// Project represents the internal project model
type Project struct {
	UUID        string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewProject creates a new project with generated UUID and timestamps
func NewProject(name, description string) *Project {
	now := time.Now()
	return &Project{
		UUID:        uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ToResponse converts a Project to a ProjectResponse
func (p *Project) ToResponse() ProjectResponse {
	return ProjectResponse{
		UUID:        p.UUID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
