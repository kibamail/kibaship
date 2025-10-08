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

// ApplicationDomainHandler handles application domain-related HTTP requests
type ApplicationDomainHandler struct {
	applicationDomainService *services.ApplicationDomainService
}

// NewApplicationDomainHandler creates a new ApplicationDomainHandler
func NewApplicationDomainHandler(applicationDomainService *services.ApplicationDomainService) *ApplicationDomainHandler {
	return &ApplicationDomainHandler{
		applicationDomainService: applicationDomainService,
	}
}

// CreateApplicationDomain handles POST /v1/applications/:uuid/domains
// @Summary Create a new application domain
// @Description Create a new domain for an application
// @Tags application-domains
// @Accept json
// @Produce json
// @Param uuid path string true "Application UUID or slug (8-character identifier)"
// @Param domain body models.ApplicationDomainCreateRequest true "Application domain creation data"
// @Success 201 {object} models.ApplicationDomainResponse "Application domain created successfully"
// @Failure 400 {object} models.ValidationErrors "Validation errors in request data"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/applications/{uuid}/domains [post]
func (h *ApplicationDomainHandler) CreateApplicationDomain(c *gin.Context) {
	applicationSlug := c.Param("uuid")

	if applicationSlug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application slug is required",
		})
		return
	}

	var req models.ApplicationDomainCreateRequest
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

	applicationDomain, err := h.applicationDomainService.CreateApplicationDomain(c.Request.Context(), &req)
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
			"message": "Failed to create application domain: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, applicationDomain.ToResponse())
}

// GetApplicationDomain handles GET /v1/domains/:uuid
// @Summary Get application domain by UUID
// @Description Retrieve an application domain by its unique UUID or slug identifier
// @Tags application-domains
// @Produce json
// @Param uuid path string true "Application domain UUID or slug (8-character identifier)"
// @Success 200 {object} models.ApplicationDomainResponse "Application domain details"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application domain not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/domains/{uuid} [get]
func (h *ApplicationDomainHandler) GetApplicationDomain(c *gin.Context) {
	slug := c.Param("uuid")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application domain slug is required",
		})
		return
	}

	applicationDomain, err := h.applicationDomainService.GetApplicationDomain(c.Request.Context(), slug)
	if err != nil {
		if err.Error() == "application domain with slug "+slug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application domain with slug '" + slug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve application domain: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, applicationDomain.ToResponse())
}

// DeleteApplicationDomain handles DELETE /v1/domains/:uuid
// @Summary Delete application domain by UUID
// @Description Delete an application domain by its unique UUID or slug identifier
// @Tags application-domains
// @Param uuid path string true "Application domain UUID or slug (8-character identifier)"
// @Success 204 "Application domain deleted successfully"
// @Failure 401 {object} auth.ErrorResponse "Authentication required"
// @Failure 404 {object} auth.ErrorResponse "Application domain not found"
// @Failure 500 {object} auth.ErrorResponse "Internal server error"
// @Security BearerAuth
// @Router /v1/domains/{uuid} [delete]
func (h *ApplicationDomainHandler) DeleteApplicationDomain(c *gin.Context) {
	slug := c.Param("uuid")

	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Application domain slug is required",
		})
		return
	}

	err := h.applicationDomainService.DeleteApplicationDomain(c.Request.Context(), slug)
	if err != nil {
		if err.Error() == "application domain with slug "+slug+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Application domain with slug '" + slug + "' was not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to delete application domain: " + err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}
