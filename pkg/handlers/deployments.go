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

// DeploymentHandler handles deployment-related HTTP requests
type DeploymentHandler struct {
	deploymentService *services.DeploymentService
}

// NewDeploymentHandler creates a new DeploymentHandler
func NewDeploymentHandler(deploymentService *services.DeploymentService) *DeploymentHandler {
	return &DeploymentHandler{
		deploymentService: deploymentService,
	}
}

// CreateDeployment handles POST /applications/{applicationSlug}/deployments
// @Summary Create a new deployment
// @Description Create a new deployment for an application
// @Tags deployments
// @Accept json
// @Produce json
// @Param applicationSlug path string true "Application slug (8-character identifier)"
// @Param deployment body models.DeploymentCreateRequest true "Deployment creation data"
// @Success 201 {object} models.DeploymentResponse "Deployment created successfully"
// @Failure 400 {object} models.ValidationErrors "Validation errors in request data"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /applications/{applicationSlug}/deployments [post]
func (h *DeploymentHandler) CreateDeployment(c *gin.Context) {
	applicationSlug := c.Param("applicationSlug")

	if applicationSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application slug is required",
		})
		return
	}

	var req models.DeploymentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Invalid JSON format: " + err.Error(),
		})
		return
	}

	// Set the application slug from the URL
	req.ApplicationSlug = applicationSlug

	// Validate the request
	if validationErr := req.Validate(); validationErr != nil {
		c.JSON(http.StatusBadRequest, validationErr)
		return
	}

	deployment, err := h.deploymentService.CreateDeployment(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "failed to get application: application with slug "+applicationSlug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application with slug '" + applicationSlug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to create deployment: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, deployment.ToResponse())
}

// GetDeployment handles GET /deployments/{slug}
// @Summary Get deployment by slug
// @Description Retrieve a deployment by its unique slug identifier
// @Tags deployments
// @Produce json
// @Param slug path string true "Deployment slug (8-character identifier)"
// @Success 200 {object} models.DeploymentResponse "Deployment details"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Deployment not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /deployments/{slug} [get]
func (h *DeploymentHandler) GetDeployment(c *gin.Context) {
	slug := c.Param("slug")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Deployment slug is required",
		})
		return
	}

	deployment, err := h.deploymentService.GetDeployment(c.Request.Context(), slug)
	if err != nil {
		if err.Error() == "deployment with slug "+slug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Deployment with slug '" + slug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve deployment: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, deployment.ToResponse())
}

// GetDeploymentsByApplication handles GET /applications/{applicationSlug}/deployments
// @Summary Get deployments by application
// @Description Retrieve all deployments for a specific application
// @Tags deployments
// @Produce json
// @Param applicationSlug path string true "Application slug (8-character identifier)"
// @Success 200 {array} models.DeploymentResponse "List of deployments"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /applications/{applicationSlug}/deployments [get]
func (h *DeploymentHandler) GetDeploymentsByApplication(c *gin.Context) {
	applicationSlug := c.Param("applicationSlug")

	if applicationSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application slug is required",
		})
		return
	}

	deployments, err := h.deploymentService.GetDeploymentsByApplication(c.Request.Context(), applicationSlug)
	if err != nil {
		if err.Error() == "failed to get application: application with slug "+applicationSlug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application with slug '" + applicationSlug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve deployments: " + err.Error(),
		})
		return
	}

	// Convert to response format
	responses := make([]models.DeploymentResponse, 0, len(deployments))
	for _, deployment := range deployments {
		responses = append(responses, deployment.ToResponse())
	}

	c.JSON(http.StatusOK, responses)
}
