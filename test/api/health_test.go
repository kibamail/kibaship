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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRouter creates a test router with the health endpoints
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Health check endpoints (same as main.go)
	router.GET("/healthz", healthzHandler)
	router.GET("/readyz", readyzHandler)

	return router
}

// healthzHandler handles the health check endpoint
func healthzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// readyzHandler handles the readiness check endpoint
func readyzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

func TestHealthzEndpoint(t *testing.T) {
	router := setupTestRouter()

	// Create a test request
	req, err := http.NewRequest("GET", "/healthz", nil)
	require.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Assert the response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Parse and validate JSON response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
}

func TestReadyzEndpoint(t *testing.T) {
	router := setupTestRouter()

	// Create a test request
	req, err := http.NewRequest("GET", "/readyz", nil)
	require.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Assert the response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Parse and validate JSON response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ready", response["status"])
}

func TestHealthEndpointsWithInvalidMethods(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		method   string
		path     string
		expected int
	}{
		{"POST", "/healthz", http.StatusNotFound},
		{"PUT", "/healthz", http.StatusNotFound},
		{"DELETE", "/healthz", http.StatusNotFound},
		{"POST", "/readyz", http.StatusNotFound},
		{"PUT", "/readyz", http.StatusNotFound},
		{"DELETE", "/readyz", http.StatusNotFound},
	}

	for _, test := range tests {
		req, err := http.NewRequest(test.method, test.path, nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, test.expected, w.Code, "Method %s on %s should return %d", test.method, test.path, test.expected)
	}
}
