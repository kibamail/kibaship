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

package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kibamail/kibaship-operator/pkg/auth"
	"github.com/kibamail/kibaship-operator/pkg/models"
)

// generateAPIKey generates a random 48-character API key
func generateAPIKey() string {
	keyBytes := make([]byte, 24) // 24 bytes = 48 hex characters
	if _, err := rand.Read(keyBytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(keyBytes)
}

// setupAuthenticatedRouter creates a router with authentication for testing
func setupAuthenticatedRouter(apiKey string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create authenticator
	authenticator := auth.NewAPIKeyAuthenticator(apiKey)

	// Add health endpoints (public)
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Protected routes
	protected := router.Group("/")
	protected.Use(authenticator.Middleware())
	{
		// Use mock project handler for testing
		mockHandler := &mockAuthTestProjectHandler{}
		protected.POST("/projects", mockHandler.CreateProject)
	}

	return router
}

// mockAuthTestProjectHandler provides a mock project handler for auth testing
type mockAuthTestProjectHandler struct{}

func (h *mockAuthTestProjectHandler) CreateProject(c *gin.Context) {
	var req models.ProjectCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	// Create mock project response
	project := models.NewProject(
		req.Name,
		req.Description,
		req.WorkspaceUUID,
		"abc123de", // Mock slug
		req.EnabledApplicationTypes,
		req.ResourceProfile,
		req.VolumeSettings,
	)

	c.JSON(http.StatusCreated, project.ToResponse())
}

var _ = Describe("API Authentication", func() {
	var apiKey string
	var router *gin.Engine

	BeforeEach(func() {
		apiKey = generateAPIKey()
		router = setupAuthenticatedRouter(apiKey)
	})

	Describe("API Key Authentication", func() {
		DescribeTable("authentication scenarios",
			func(authorizationTemplate string, expectedStatus int, expectedError string) {
				// Replace {{API_KEY}} placeholder with actual API key
				authorization := authorizationTemplate
				if authorizationTemplate == "Bearer {{API_KEY}}" {
					authorization = "Bearer " + apiKey
				}
				// Create request
				projectData := models.ProjectCreateRequest{
					Name:        "test-project",
					Description: "A test project",
				}
				jsonData, err := json.Marshal(projectData)
				Expect(err).NotTo(HaveOccurred())

				req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
				Expect(err).NotTo(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				if authorization != "" {
					req.Header.Set("Authorization", authorization)
				}

				// Perform request
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				// Assert response
				Expect(w.Code).To(Equal(expectedStatus))

				if expectedError != "" {
					var response map[string]interface{}
					err = json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())

					Expect(response["message"]).To(ContainSubstring(expectedError))
				} else {
					// For successful requests, verify project was created
					var response models.ProjectResponse
					err = json.Unmarshal(w.Body.Bytes(), &response)
					Expect(err).NotTo(HaveOccurred())

					Expect(response.UUID).NotTo(BeEmpty())
					Expect(response.Name).To(Equal("test-project"))
					Expect(response.Description).To(Equal("A test project"))
					Expect(response.CreatedAt).NotTo(BeZero())
					Expect(response.UpdatedAt).NotTo(BeZero())
				}
			},
			Entry("Valid API key", "Bearer {{API_KEY}}", http.StatusCreated, ""),
			Entry("Missing authorization header", "", http.StatusUnauthorized, "Missing authorization header"),
			Entry("Invalid bearer format", "Token invalid-key", http.StatusUnauthorized, "Authorization header must use Bearer scheme"),
			Entry("Invalid API key", "Bearer invalid-key", http.StatusUnauthorized, "Invalid API key"),
			Entry("Empty bearer token", "Bearer ", http.StatusUnauthorized, "Invalid API key"),
		)

		It("accepts valid API key", func() {
			// Create request
			projectData := models.ProjectCreateRequest{
				Name:        "test-project",
				Description: "A test project",
			}
			jsonData, err := json.Marshal(projectData)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert response
			Expect(w.Code).To(Equal(http.StatusCreated))

			var response models.ProjectResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())

			Expect(response.UUID).NotTo(BeEmpty())
			Expect(response.Name).To(Equal("test-project"))
			Expect(response.Description).To(Equal("A test project"))
		})
	})

})
