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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("OpenAPI Specification", func() {
	var spec OpenAPISpec

	BeforeEach(func() {
		// Get the project root directory
		projectRoot := filepath.Join("..", "..")
		specPath := filepath.Join(projectRoot, "docs", "swagger.yaml")

		// Read the OpenAPI spec file
		specData, err := os.ReadFile(specPath)
		Expect(err).NotTo(HaveOccurred(), "Should be able to read swagger.yaml file")

		// Parse the YAML
		err = yaml.Unmarshal(specData, &spec)
		Expect(err).NotTo(HaveOccurred(), "Should be able to parse swagger.yaml as valid YAML")
	})

	Describe("Required Fields", func() {
		It("has version information", func() {
			// Check version (should be either swagger 2.0 or openapi 3.x)
			Expect(spec.Swagger == "2.0" || spec.OpenAPI != "").To(BeTrue(), "Should have either swagger or openapi version")
		})

		It("has info section", func() {
			Expect(spec.Info).NotTo(BeNil(), "Should have info section")
			Expect(spec.Info["title"]).NotTo(BeEmpty(), "Should have title in info")
			Expect(spec.Info["version"]).NotTo(BeEmpty(), "Should have version in info")
		})

		It("has paths defined", func() {
			Expect(spec.Paths).NotTo(BeNil(), "Should have paths section")
			Expect(spec.Paths).NotTo(BeEmpty(), "Should have at least one path defined")
		})
	})

	Describe("Health Endpoints", func() {
		It("documents health endpoints", func() {
			Expect(spec.Paths).To(HaveKey("/healthz"), "Should document /healthz endpoint")
			Expect(spec.Paths).To(HaveKey("/readyz"), "Should document /readyz endpoint")
		})

		It("documents GET methods for health endpoints", func() {
			healthzPath, ok := spec.Paths["/healthz"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "/healthz should be a valid path object")
			Expect(healthzPath).To(HaveKey("get"), "/healthz should have GET method")

			readyzPath, ok := spec.Paths["/readyz"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "/readyz should be a valid path object")
			Expect(readyzPath).To(HaveKey("get"), "/readyz should have GET method")
		})
	})

	Describe("Response Definitions", func() {
		It("defines response types", func() {
			Expect(spec.Definitions).NotTo(BeNil(), "Should have definitions section")
			Expect(spec.Definitions).To(HaveKey("main.HealthResponse"), "Should define HealthResponse")
			Expect(spec.Definitions).To(HaveKey("main.ReadyResponse"), "Should define ReadyResponse")
		})

		It("has valid HealthResponse structure", func() {
			healthResp, ok := spec.Definitions["main.HealthResponse"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "HealthResponse should be a valid definition")
			Expect(healthResp["type"]).To(Equal("object"), "HealthResponse should be an object")

			properties, ok := healthResp["properties"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "HealthResponse should have properties")
			Expect(properties).To(HaveKey("status"), "HealthResponse should have status field")
		})

		It("has valid ReadyResponse structure", func() {
			readyResp, ok := spec.Definitions["main.ReadyResponse"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "ReadyResponse should be a valid definition")
			Expect(readyResp["type"]).To(Equal("object"), "ReadyResponse should be an object")

			readyProperties, ok := readyResp["properties"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "ReadyResponse should have properties")
			Expect(readyProperties).To(HaveKey("status"), "ReadyResponse should have status field")
		})
	})

	Describe("API Metadata", func() {
		It("has correct metadata", func() {
			Expect(spec.Info["title"]).To(Equal("Kibaship Operator API"), "Should have correct API title")
			Expect(spec.Info["version"]).To(Equal("1.0"), "Should have correct API version")
			Expect(spec.BasePath).To(Equal("/"), "Should have correct base path")
			Expect(spec.Host).NotTo(BeEmpty(), "Should have host defined")
		})
	})
})
