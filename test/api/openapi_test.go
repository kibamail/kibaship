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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// OpenAPISpec represents the structure of an OpenAPI specification
type OpenAPISpec struct {
	Swagger     string                 `yaml:"swagger"`
	OpenAPI     string                 `yaml:"openapi"`
	Info        map[string]interface{} `yaml:"info"`
	Host        string                 `yaml:"host"`
	BasePath    string                 `yaml:"basePath"`
	Paths       map[string]interface{} `yaml:"paths"`
	Definitions map[string]interface{} `yaml:"definitions"`
}

func TestOpenAPISpecValidation(t *testing.T) {
	// Get the project root directory
	projectRoot := filepath.Join("..", "..")
	specPath := filepath.Join(projectRoot, "docs", "swagger.yaml")

	// Read the OpenAPI spec file
	specData, err := os.ReadFile(specPath)
	require.NoError(t, err, "Should be able to read swagger.yaml file")

	// Parse the YAML
	var spec OpenAPISpec
	err = yaml.Unmarshal(specData, &spec)
	require.NoError(t, err, "Should be able to parse swagger.yaml as valid YAML")

	// Validate required fields
	t.Run("Required Fields", func(t *testing.T) {
		// Check version (should be either swagger 2.0 or openapi 3.x)
		assert.True(t, spec.Swagger == "2.0" || spec.OpenAPI != "", "Should have either swagger or openapi version")

		// Check info section
		assert.NotNil(t, spec.Info, "Should have info section")
		assert.NotEmpty(t, spec.Info["title"], "Should have title in info")
		assert.NotEmpty(t, spec.Info["version"], "Should have version in info")

		// Check paths
		assert.NotNil(t, spec.Paths, "Should have paths section")
		assert.NotEmpty(t, spec.Paths, "Should have at least one path defined")
	})

	t.Run("Health Endpoints", func(t *testing.T) {
		// Check that health endpoints are documented
		assert.Contains(t, spec.Paths, "/healthz", "Should document /healthz endpoint")
		assert.Contains(t, spec.Paths, "/readyz", "Should document /readyz endpoint")

		// Check that health endpoints have GET methods
		healthzPath, ok := spec.Paths["/healthz"].(map[string]interface{})
		require.True(t, ok, "/healthz should be a valid path object")
		assert.Contains(t, healthzPath, "get", "/healthz should have GET method")

		readyzPath, ok := spec.Paths["/readyz"].(map[string]interface{})
		require.True(t, ok, "/readyz should be a valid path object")
		assert.Contains(t, readyzPath, "get", "/readyz should have GET method")
	})

	t.Run("Response Definitions", func(t *testing.T) {
		// Check that response types are defined
		assert.NotNil(t, spec.Definitions, "Should have definitions section")
		assert.Contains(t, spec.Definitions, "main.HealthResponse", "Should define HealthResponse")
		assert.Contains(t, spec.Definitions, "main.ReadyResponse", "Should define ReadyResponse")

		// Validate HealthResponse structure
		healthResp, ok := spec.Definitions["main.HealthResponse"].(map[string]interface{})
		require.True(t, ok, "HealthResponse should be a valid definition")
		assert.Equal(t, "object", healthResp["type"], "HealthResponse should be an object")

		properties, ok := healthResp["properties"].(map[string]interface{})
		require.True(t, ok, "HealthResponse should have properties")
		assert.Contains(t, properties, "status", "HealthResponse should have status field")

		// Validate ReadyResponse structure
		readyResp, ok := spec.Definitions["main.ReadyResponse"].(map[string]interface{})
		require.True(t, ok, "ReadyResponse should be a valid definition")
		assert.Equal(t, "object", readyResp["type"], "ReadyResponse should be an object")

		readyProperties, ok := readyResp["properties"].(map[string]interface{})
		require.True(t, ok, "ReadyResponse should have properties")
		assert.Contains(t, readyProperties, "status", "ReadyResponse should have status field")
	})

	t.Run("API Metadata", func(t *testing.T) {
		// Validate API metadata
		assert.Equal(t, "Kibaship Operator API", spec.Info["title"], "Should have correct API title")
		assert.Equal(t, "1.0", spec.Info["version"], "Should have correct API version")
		assert.Equal(t, "/", spec.BasePath, "Should have correct base path")
		assert.NotEmpty(t, spec.Host, "Should have host defined")
	})
}
