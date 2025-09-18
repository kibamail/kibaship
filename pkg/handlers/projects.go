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
)

// ProjectHandler handles project-related HTTP requests
type ProjectHandler struct {
	// In a real implementation, this would have database connections, etc.
	// For now, we'll just create projects in memory as proof of concept
}

// NewProjectHandler creates a new project handler
func NewProjectHandler() *ProjectHandler {
	return &ProjectHandler{}
}

// CreateProject handles POST /projects
// @Summary Create a new project
// @Description Create a new project with the provided name and description
// @Tags projects
// @Accept json
// @Produce json
// @Param project body models.ProjectCreateRequest true "Project data"
// @Success 201 {object} models.ProjectResponse
// @Failure 400 {object} auth.ErrorResponse
// @Failure 401 {object} auth.ErrorResponse
// @Security BearerAuth
// @Router /projects [post]
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req models.ProjectCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	// Create new project
	project := models.NewProject(req.Name, req.Description)

	// In a real implementation, we would save this to a database here
	// For now, we just return the created project

	c.JSON(http.StatusCreated, project.ToResponse())
}
