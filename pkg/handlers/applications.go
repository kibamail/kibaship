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

// ApplicationHandler handles application-related HTTP requests
type ApplicationHandler struct {
	applicationService *services.ApplicationService
}

// NewApplicationHandler creates a new ApplicationHandler
func NewApplicationHandler(applicationService *services.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{
		applicationService: applicationService,
	}
}

// CreateApplication handles POST /environments/{environmentSlug}/applications
// @Summary Create a new application
// @Description Create a new application within an environment with type-specific configuration
// @Tags applications
// @Accept json
// @Produce json
// @Param environmentSlug path string true "Environment slug (8-character identifier)"
// @Param application body models.ApplicationCreateRequest true "Application creation data"
// @Success 201 {object} models.ApplicationResponse "Application created successfully"
// @Failure 400 {object} models.ValidationErrors "Validation errors in request data"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Environment not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /environments/{environmentSlug}/applications [post]
func (h *ApplicationHandler) CreateApplication(c *gin.Context) {
	environmentSlug := c.Param("slug")

	if environmentSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Environment slug is required",
		})
		return
	}

	var req models.ApplicationCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid JSON format: " + err.Error(),
		})
		return
	}

	// Set the environment slug from the URL
	req.EnvironmentSlug = environmentSlug

	// Validate the request
	if validationErr := req.Validate(); validationErr != nil {
		c.JSON(http.StatusBadRequest, validationErr)
		return
	}

	application, err := h.applicationService.CreateApplication(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "failed to get environment: environment with slug "+environmentSlug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Environment with slug '" + environmentSlug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create application: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, application.ToResponse())
}

// GetApplication handles GET /applications/{slug}
// @Summary Get application by slug
// @Description Retrieve an application by its unique slug identifier
// @Tags applications
// @Produce json
// @Param slug path string true "Application slug (8-character identifier)"
// @Success 200 {object} models.ApplicationResponse "Application details"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /applications/{slug} [get]
func (h *ApplicationHandler) GetApplication(c *gin.Context) {
	slug := c.Param("slug")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application slug is required",
		})
		return
	}

	application, err := h.applicationService.GetApplication(c.Request.Context(), slug)
	if err != nil {
		if err.Error() == "application with slug "+slug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application with slug '" + slug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve application: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, application.ToResponse())
}

// UpdateApplication handles PATCH /applications/{slug}
// @Summary Update application by slug
// @Description Update an application by its unique slug identifier with partial updates
// @Tags applications
// @Accept json
// @Produce json
// @Param slug path string true "Application slug (8-character identifier)"
// @Param application body models.ApplicationUpdateRequest true "Application update data"
// @Success 200 {object} models.ApplicationResponse "Updated application details"
// @Failure 400 {object} models.ValidationErrors "Validation errors in request data"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /applications/{slug} [patch]
func (h *ApplicationHandler) UpdateApplication(c *gin.Context) {
	slug := c.Param("slug")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application slug is required",
		})
		return
	}

	var req models.ApplicationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid JSON format: " + err.Error(),
		})
		return
	}

	// Validate the update request
	if validationErr := req.ValidateUpdate(); validationErr != nil {
		c.JSON(http.StatusBadRequest, validationErr)
		return
	}

	application, err := h.applicationService.UpdateApplication(c.Request.Context(), slug, &req)
	if err != nil {
		if err.Error() == "application with slug "+slug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application with slug '" + slug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to update application: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, application.ToResponse())
}

// DeleteApplication handles DELETE /applications/{slug}
// @Summary Delete application by slug
// @Description Delete an application by its unique slug identifier
// @Tags applications
// @Param slug path string true "Application slug (8-character identifier)"
// @Success 204 "Application deleted successfully"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /applications/{slug} [delete]
func (h *ApplicationHandler) DeleteApplication(c *gin.Context) {
	slug := c.Param("slug")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application slug is required",
		})
		return
	}

	err := h.applicationService.DeleteApplication(c.Request.Context(), slug)
	if err != nil {
		if err.Error() == "application with slug "+slug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application with slug '" + slug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to delete application: " + err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetApplicationsByProject handles GET /projects/{projectSlug}/applications
// @Summary Get applications by project
// @Description Retrieve all applications for a specific project
// @Tags applications
// @Produce json
// @Param projectSlug path string true "Project slug (8-character identifier)"
// @Success 200 {array} models.ApplicationResponse "List of applications"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Project not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /projects/{projectSlug}/applications [get]
func (h *ApplicationHandler) GetApplicationsByProject(c *gin.Context) {
	projectSlug := c.Param("projectSlug")

	if projectSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Project slug is required",
		})
		return
	}

	applications, err := h.applicationService.GetApplicationsByProject(c.Request.Context(), projectSlug)
	if err != nil {
		if err.Error() == "failed to get project: project with slug "+projectSlug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Project with slug '" + projectSlug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve applications: " + err.Error(),
		})
		return
	}

	// Convert to response format
	responses := make([]models.ApplicationResponse, 0, len(applications))
	for _, app := range applications {
		responses = append(responses, app.ToResponse())
	}

	c.JSON(http.StatusOK, responses)
}

// GetApplicationsByEnvironment handles GET /environments/{environmentSlug}/applications
// @Summary Get applications by environment
// @Description Retrieve all applications for a specific environment
// @Tags applications
// @Produce json
// @Param environmentSlug path string true "Environment slug (8-character identifier)"
// @Success 200 {array} models.ApplicationResponse "List of applications"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Environment not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /environments/{environmentSlug}/applications [get]
func (h *ApplicationHandler) GetApplicationsByEnvironment(c *gin.Context) {
	environmentSlug := c.Param("slug")

	if environmentSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Environment slug is required",
		})
		return
	}

	applications, err := h.applicationService.GetApplicationsByEnvironment(c.Request.Context(), environmentSlug)
	if err != nil {
		if err.Error() == "failed to get environment: environment with slug "+environmentSlug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Environment with slug '" + environmentSlug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve applications: " + err.Error(),
		})
		return
	}

	// Convert to response format
	responses := make([]models.ApplicationResponse, 0, len(applications))
	for _, app := range applications {
		responses = append(responses, app.ToResponse())
	}

	c.JSON(http.StatusOK, responses)
}
