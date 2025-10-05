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

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kibamail/kibaship-operator/pkg/models"
	"github.com/kibamail/kibaship-operator/pkg/services"
)

// EnvironmentHandler handles environment-related HTTP requests
type EnvironmentHandler struct {
	environmentService *services.EnvironmentService
}

// NewEnvironmentHandler creates a new environment handler
func NewEnvironmentHandler(environmentService *services.EnvironmentService) *EnvironmentHandler {
	return &EnvironmentHandler{
		environmentService: environmentService,
	}
}

// CreateEnvironment handles POST /projects/:projectSlug/environments
// @Summary Create a new environment
// @Description Create a new environment within a project. The environment will be assigned a random 8-character slug.
// @Tags environments
// @Accept json
// @Produce json
// @Param projectSlug path string true "Project slug (8-character identifier)"
// @Param environment body models.EnvironmentCreateRequest true "Environment creation data"
// @Success 201 {object} models.EnvironmentResponse "Environment created successfully"
// @Failure 400 {object} models.ValidationErrors "Validation errors in request data"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Project not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /projects/{projectSlug}/environments [post]
func (h *EnvironmentHandler) CreateEnvironment(c *gin.Context) {
	projectSlug := c.Param("projectSlug")

	if projectSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Project slug is required",
		})
		return
	}

	var req models.EnvironmentCreateRequest

	// Parse JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ValidationErrors{
			Errors: []models.ValidationError{
				{
					Field:   "request",
					Message: "Invalid JSON format: " + err.Error(),
				},
			},
		})
		return
	}

	// Set the project slug from URL param (overrides any value in body)
	req.ProjectSlug = projectSlug

	// Validate request
	if validationErrors := req.Validate(); validationErrors != nil {
		c.JSON(http.StatusBadRequest, validationErrors)
		return
	}

	// Create environment using service
	environment, err := h.environmentService.CreateEnvironment(c.Request.Context(), &req)
	if err != nil {
		// Check if it's a "project not found" error
		if err.Error() == "failed to get project: project with slug "+projectSlug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Project with slug '" + projectSlug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create environment: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, environment.ToResponse())
}

// GetEnvironment handles GET /environments/:slug
// @Summary Get environment by slug
// @Description Retrieve an environment by its unique slug identifier
// @Tags environments
// @Produce json
// @Param slug path string true "Environment slug (8-character identifier)"
// @Success 200 {object} models.EnvironmentResponse "Environment details"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Environment not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /environments/{slug} [get]
func (h *EnvironmentHandler) GetEnvironment(c *gin.Context) {
	slug := c.Param("slug")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Environment slug is required",
		})
		return
	}

	environment, err := h.environmentService.GetEnvironment(c.Request.Context(), slug)
	if err != nil {
		if err.Error() == "environment with slug "+slug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Environment with slug '" + slug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve environment: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, environment.ToResponse())
}

// GetEnvironmentsByProject handles GET /projects/:projectSlug/environments
// @Summary List environments for a project
// @Description Retrieve all environments for a specific project
// @Tags environments
// @Produce json
// @Param projectSlug path string true "Project slug (8-character identifier)"
// @Success 200 {array} models.EnvironmentResponse "List of environments"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Project not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /projects/{projectSlug}/environments [get]
func (h *EnvironmentHandler) GetEnvironmentsByProject(c *gin.Context) {
	projectSlug := c.Param("projectSlug")

	if projectSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Project slug is required",
		})
		return
	}

	environments, err := h.environmentService.GetEnvironmentsByProject(c.Request.Context(), projectSlug)
	if err != nil {
		// Check if it's a "project not found" error
		if err.Error() == "failed to get project: project with slug "+projectSlug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Project with slug '" + projectSlug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve environments: " + err.Error(),
		})
		return
	}

	// Convert to response models
	responses := make([]*models.EnvironmentResponse, len(environments))
	for i, env := range environments {
		responses[i] = env.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
}

// UpdateEnvironment handles PATCH /environments/:slug
// @Summary Update environment by slug
// @Description Update an environment by its unique slug identifier with partial updates
// @Tags environments
// @Accept json
// @Produce json
// @Param slug path string true "Environment slug (8-character identifier)"
// @Param environment body models.EnvironmentUpdateRequest true "Environment update data"
// @Success 200 {object} models.EnvironmentResponse "Updated environment details"
// @Failure 400 {object} models.ValidationErrors "Validation errors in request data"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Environment not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /environments/{slug} [patch]
func (h *EnvironmentHandler) UpdateEnvironment(c *gin.Context) {
	slug := c.Param("slug")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Environment slug is required",
		})
		return
	}

	var req models.EnvironmentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid JSON format: " + err.Error(),
		})
		return
	}

	// Validate the update request
	if validationErr := req.Validate(); validationErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": validationErr.Error(),
		})
		return
	}

	environment, err := h.environmentService.UpdateEnvironment(c.Request.Context(), slug, &req)
	if err != nil {
		if err.Error() == "environment with slug "+slug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Environment with slug '" + slug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to update environment: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, environment.ToResponse())
}

// DeleteEnvironment handles DELETE /environments/:slug
// @Summary Delete environment by slug
// @Description Delete an environment by its unique slug identifier
// @Tags environments
// @Param slug path string true "Environment slug (8-character identifier)"
// @Success 204 "Environment deleted successfully"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Environment not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /environments/{slug} [delete]
func (h *EnvironmentHandler) DeleteEnvironment(c *gin.Context) {
	slug := c.Param("slug")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Environment slug is required",
		})
		return
	}

	err := h.environmentService.DeleteEnvironment(c.Request.Context(), slug)
	if err != nil {
		if err.Error() == "environment with slug "+slug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Environment with slug '" + slug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to delete environment: " + err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}
