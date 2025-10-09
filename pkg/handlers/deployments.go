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

// CreateDeployment handles POST /v1/applications/:uuid/deployments
// @Summary Create a new deployment
// @Description Create a new deployment for an application
// @Tags deployments
// @Accept json
// @Produce json
// @Param uuid path string true "Application UUID or slug"
// @Param deployment body models.DeploymentCreateRequest true "Deployment creation data"
// @Success 201 {object} models.DeploymentResponse "Deployment created successfully"
// @Failure 400 {object} models.ValidationErrors "Validation errors in request data"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/applications/{uuid}/deployments [post]
func (h *DeploymentHandler) CreateDeployment(c *gin.Context) {
	applicationUUID := c.Param("uuid")

	if applicationUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application UUID is required",
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

	// Set the application UUID from the URL
	req.ApplicationUUID = applicationUUID

	// Validate the request
	if validationErr := req.Validate(); validationErr != nil {
		c.JSON(http.StatusBadRequest, validationErr)
		return
	}

	deployment, err := h.deploymentService.CreateDeployment(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "failed to get application: application with UUID "+applicationUUID+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application with UUID '" + applicationUUID + "' was not found",
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

// GetDeployment handles GET /v1/deployments/:uuid
// @Summary Get deployment by UUID
// @Description Retrieve a deployment by its unique UUID or slug identifier
// @Tags deployments
// @Produce json
// @Param uuid path string true "Deployment UUID or slug (8-character identifier)"
// @Success 200 {object} models.DeploymentResponse "Deployment details"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Deployment not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/deployments/{uuid} [get]
func (h *DeploymentHandler) GetDeployment(c *gin.Context) {
	slug := c.Param("uuid")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Deployment slug is required",
		})
		return
	}

	deployment, err := h.deploymentService.GetDeployment(c.Request.Context(), slug)
	if err != nil {
		if err.Error() == "deployment with UUID "+slug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Deployment with UUID '" + slug + "' was not found",
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

// GetDeploymentsByApplication handles GET /v1/applications/:uuid/deployments
// @Summary Get deployments by application
// @Description Retrieve all deployments for a specific application
// @Tags deployments
// @Produce json
// @Param uuid path string true "Application UUID or slug (8-character identifier)"
// @Success 200 {array} models.DeploymentResponse "List of deployments"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/applications/{uuid}/deployments [get]
func (h *DeploymentHandler) GetDeploymentsByApplication(c *gin.Context) {
	applicationSlug := c.Param("uuid")

	if applicationSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application slug is required",
		})
		return
	}

	deployments, err := h.deploymentService.GetDeploymentsByApplication(c.Request.Context(), applicationSlug)
	if err != nil {
		if err.Error() == "failed to get application: application with UUID "+applicationSlug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application with UUID '" + applicationSlug + "' was not found",
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

// PromoteDeployment handles POST /v1/deployments/:uuid/promote
// @Summary Promote a deployment
// @Description Promote a deployment by updating the application's currentDeploymentRef to point to this deployment
// @Tags deployments
// @Produce json
// @Param uuid path string true "Deployment UUID or slug"
// @Success 200 {object} map[string]string "Deployment promoted successfully"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Deployment not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/deployments/{uuid}/promote [post]
func (h *DeploymentHandler) PromoteDeployment(c *gin.Context) {
	deploymentUUID := c.Param("uuid")

	if deploymentUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Deployment UUID is required",
		})
		return
	}

	err := h.deploymentService.PromoteDeployment(c.Request.Context(), deploymentUUID)
	if err != nil {
		// Check if deployment not found (checking for substring to handle wrapped errors)
		errMsg := err.Error()
		if errMsg == "failed to get deployment: deployment with UUID "+deploymentUUID+" not found" ||
			errMsg == "deployment with UUID "+deploymentUUID+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Deployment with UUID '" + deploymentUUID + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to promote deployment: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Deployment promoted successfully",
	})
}
