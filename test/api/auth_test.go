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
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kibamail/kibaship-operator/pkg/auth"
	"github.com/kibamail/kibaship-operator/pkg/models"
)

// generateAPIKey generates a random 48-character API key
func generateAPIKey() string {
	bytes := make([]byte, 24) // 24 bytes = 48 hex characters
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

// createTestSecret creates a test secret with the API key
func createTestSecret(apiKey string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      auth.SecretName,
			Namespace: "default",
		},
		Data: map[string][]byte{
			auth.SecretKey: []byte(apiKey),
		},
	}
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

func TestAPIKeyAuthentication(t *testing.T) {
	apiKey := generateAPIKey()
	router := setupAuthenticatedRouter(apiKey)

	tests := []struct {
		name           string
		authorization  string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Valid API key",
			authorization:  "Bearer " + apiKey,
			expectedStatus: http.StatusCreated, // Will create a project
		},
		{
			name:           "Missing authorization header",
			authorization:  "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Missing authorization header",
		},
		{
			name:           "Invalid bearer format",
			authorization:  "Token " + apiKey,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authorization header must use Bearer scheme",
		},
		{
			name:           "Invalid API key",
			authorization:  "Bearer invalid-key",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid API key",
		},
		{
			name:           "Empty bearer token",
			authorization:  "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			projectData := models.ProjectCreateRequest{
				Name:        "test-project",
				Description: "A test project",
			}
			jsonData, err := json.Marshal(projectData)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/projects", bytes.NewBuffer(jsonData))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}

			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Contains(t, response["message"], tt.expectedError)
			} else {
				// For successful requests, verify project was created
				var response models.ProjectResponse
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.NotEmpty(t, response.UUID)
				assert.Equal(t, "test-project", response.Name)
				assert.Equal(t, "A test project", response.Description)
				assert.NotZero(t, response.CreatedAt)
				assert.NotZero(t, response.UpdatedAt)
			}
		})
	}
}

func TestSecretManager(t *testing.T) {
	apiKey := generateAPIKey()
	secret := createTestSecret(apiKey)

	// Create fake Kubernetes client
	fakeClient := fake.NewSimpleClientset(secret)

	// Create secret manager with fake client
	secretManager := auth.NewSecretManagerWithClient(fakeClient, "default")

	// Test successful API key retrieval
	retrievedKey, err := secretManager.GetAPIKeyWithRetry(context.Background())
	require.NoError(t, err)
	assert.Equal(t, apiKey, retrievedKey)
}

func TestSecretManagerMissingSecret(t *testing.T) {
	// Create fake client without the secret
	fakeClient := fake.NewSimpleClientset()

	secretManager := auth.NewSecretManagerWithClient(fakeClient, "default")

	// Test that it fails when secret is missing
	ctx, cancel := context.WithTimeout(context.Background(), 100) // Very short timeout for test
	defer cancel()

	_, err := secretManager.GetAPIKeyWithRetry(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout waiting for secret")
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
