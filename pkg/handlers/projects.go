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

// ProjectHandler handles project-related HTTP requests
type ProjectHandler struct {
	projectService *services.ProjectService
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(projectService *services.ProjectService) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
	}
}

// CreateProject handles POST /projects
// @Summary Create a new project
// @Description Create a new project with comprehensive resource management and application type configuration. The project will be assigned a random 8-character slug and configured with the specified resource profile.
// @Tags projects
// @Accept json
// @Produce json
// @Param project body models.ProjectCreateRequest true "Project creation data"
// @Success 201 {object} models.ProjectResponse "Project created successfully"
// @Failure 400 {object} models.ValidationErrors "Validation errors in request data"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /projects [post]
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req models.ProjectCreateRequest

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

	// Validate request
	if validationErrors := req.Validate(); validationErrors != nil {
		c.JSON(http.StatusBadRequest, validationErrors)
		return
	}

	// Create project using service
	project, err := h.projectService.CreateProject(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create project: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, project.ToResponse())
}

// GetProject handles GET /projects/:slug
// @Summary Get project by slug
// @Description Retrieve a project by its unique slug identifier
// @Tags projects
// @Produce json
// @Param slug path string true "Project slug (8-character identifier)"
// @Success 200 {object} models.ProjectResponse "Project details"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Project not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /projects/{slug} [get]
func (h *ProjectHandler) GetProject(c *gin.Context) {
	slug := c.Param("slug")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Project slug is required",
		})
		return
	}

	project, err := h.projectService.GetProject(c.Request.Context(), slug)
	if err != nil {
		if err.Error() == "project with slug "+slug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Project with slug '" + slug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve project: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, project.ToResponse())
}
