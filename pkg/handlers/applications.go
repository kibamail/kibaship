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

// CreateApplication handles POST /v1/environments/:uuid/applications
// @Summary Create a new application
// @Description Create a new application within an environment with type-specific configuration
// @Tags applications
// @Accept json
// @Produce json
// @Param uuid path string true "Environment UUID or slug"
// @Param application body models.ApplicationCreateRequest true "Application creation data"
// @Success 201 {object} models.ApplicationResponse "Application created successfully"
// @Failure 400 {object} models.ValidationErrors "Validation errors in request data"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Environment not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/environments/{uuid}/applications [post]
func (h *ApplicationHandler) CreateApplication(c *gin.Context) {
	environmentUUID := c.Param("uuid")

	if environmentUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Environment UUID is required",
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

	// Set the environment UUID from the URL
	req.EnvironmentUUID = environmentUUID

	// Validate the request
	if validationErr := req.Validate(); validationErr != nil {
		c.JSON(http.StatusBadRequest, validationErr)
		return
	}

	application, err := h.applicationService.CreateApplication(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "failed to get environment: environment with UUID "+environmentUUID+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Environment with UUID '" + environmentUUID + "' was not found",
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

// GetApplication handles GET /v1/applications/:uuid
// @Summary Get application by UUID
// @Description Retrieve an application by its unique UUID or slug identifier
// @Tags applications
// @Produce json
// @Param uuid path string true "Application UUID or slug"
// @Success 200 {object} models.ApplicationResponse "Application details"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/applications/{uuid} [get]
func (h *ApplicationHandler) GetApplication(c *gin.Context) {
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application UUID is required",
		})
		return
	}

	application, err := h.applicationService.GetApplication(c.Request.Context(), uuid)
	if err != nil {
		if err.Error() == "application with UUID "+uuid+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application with UUID '" + uuid + "' was not found",
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

// UpdateApplication handles PATCH /v1/applications/:uuid
// @Summary Update application by UUID
// @Description Update an application by its unique UUID or slug identifier with partial updates
// @Tags applications
// @Accept json
// @Produce json
// @Param uuid path string true "Application UUID or slug"
// @Param application body models.ApplicationUpdateRequest true "Application update data"
// @Success 200 {object} models.ApplicationResponse "Updated application details"
// @Failure 400 {object} models.ValidationErrors "Validation errors in request data"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/applications/{uuid} [patch]
func (h *ApplicationHandler) UpdateApplication(c *gin.Context) {
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application UUID is required",
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

	application, err := h.applicationService.UpdateApplication(c.Request.Context(), uuid, &req)
	if err != nil {
		if err.Error() == "application with UUID "+uuid+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application with UUID '" + uuid + "' was not found",
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

// DeleteApplication handles DELETE /v1/applications/:uuid
// @Summary Delete application by UUID
// @Description Delete an application by its unique UUID or slug identifier
// @Tags applications
// @Param uuid path string true "Application UUID or slug"
// @Success 204 "Application deleted successfully"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/applications/{uuid} [delete]
func (h *ApplicationHandler) DeleteApplication(c *gin.Context) {
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application UUID is required",
		})
		return
	}

	err := h.applicationService.DeleteApplication(c.Request.Context(), uuid)
	if err != nil {
		if err.Error() == "application with UUID "+uuid+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application with UUID '" + uuid + "' was not found",
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

// GetApplicationsByProject handles GET /v1/projects/:uuid/applications
// @Summary Get applications by project
// @Description Retrieve all applications for a specific project
// @Tags applications
// @Produce json
// @Param uuid path string true "Project UUID or slug"
// @Success 200 {array} models.ApplicationResponse "List of applications"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Project not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/projects/{uuid}/applications [get]
func (h *ApplicationHandler) GetApplicationsByProject(c *gin.Context) {
	projectUUID := c.Param("uuid")

	if projectUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Project UUID is required",
		})
		return
	}

	applications, err := h.applicationService.GetApplicationsByProject(c.Request.Context(), projectUUID)
	if err != nil {
		if err.Error() == "failed to get project: project with UUID "+projectUUID+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Project with UUID '" + projectUUID + "' was not found",
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

// GetApplicationsByEnvironment handles GET /v1/environments/:uuid/applications
// @Summary Get applications by environment
// @Description Retrieve all applications for a specific environment
// @Tags applications
// @Produce json
// @Param uuid path string true "Environment UUID or slug"
// @Success 200 {array} models.ApplicationResponse "List of applications"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Environment not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/environments/{uuid}/applications [get]
func (h *ApplicationHandler) GetApplicationsByEnvironment(c *gin.Context) {
	environmentUUID := c.Param("uuid")

	if environmentUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Environment UUID is required",
		})
		return
	}

	applications, err := h.applicationService.GetApplicationsByEnvironment(c.Request.Context(), environmentUUID)
	if err != nil {
		if err.Error() == "failed to get environment: environment with UUID "+environmentUUID+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Environment with UUID '" + environmentUUID + "' was not found",
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

// UpdateApplicationEnv handles PATCH /v1/applications/:uuid/env
// @Summary Update environment variables for an application
// @Description Update environment variables for a GitRepository application by merging new variables with existing ones
// @Tags applications
// @Accept json
// @Produce json
// @Param uuid path string true "Application UUID or slug"
// @Param variables body models.ApplicationEnvUpdateRequest true "Environment variables to set/update"
// @Success 200 {string} string "Environment variables updated successfully"
// @Failure 400 {object} auth.ErrorResponse "Invalid request data"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/applications/{uuid}/env [patch]
func (h *ApplicationHandler) UpdateApplicationEnv(c *gin.Context) {
	uuid := c.Param("uuid")

	if uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application UUID is required",
		})
		return
	}

	var req models.ApplicationEnvUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid JSON format: " + err.Error(),
		})
		return
	}

	if len(req.Variables) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Variables map cannot be empty",
		})
		return
	}

	err := h.applicationService.UpdateApplicationEnv(c.Request.Context(), uuid, &req)
	if err != nil {
		if err.Error() == "application with UUID "+uuid+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application with UUID '" + uuid + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to update environment variables: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Environment variables updated successfully",
	})
}
