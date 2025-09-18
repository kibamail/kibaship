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

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("Health API", func() {
	var router *gin.Engine

	BeforeEach(func() {
		router = setupTestRouter()
	})

	Describe("GET /healthz", func() {
		It("returns health status", func() {
			req, err := http.NewRequest("GET", "/healthz", nil)
			Expect(err).NotTo(HaveOccurred())

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json; charset=utf-8"))

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())

			Expect(response["status"]).To(Equal("ok"))
		})
	})

	Describe("GET /readyz", func() {
		It("returns readiness status", func() {
			req, err := http.NewRequest("GET", "/readyz", nil)
			Expect(err).NotTo(HaveOccurred())

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json; charset=utf-8"))

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())

			Expect(response["status"]).To(Equal("ready"))
		})
	})

	Describe("Invalid HTTP methods", func() {
		DescribeTable("health endpoints with invalid methods",
			func(method, path string, expectedStatus int) {
				req, err := http.NewRequest(method, path, nil)
				Expect(err).NotTo(HaveOccurred())

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(expectedStatus))
			},
			Entry("POST /healthz", "POST", "/healthz", http.StatusNotFound),
			Entry("PUT /healthz", "PUT", "/healthz", http.StatusNotFound),
			Entry("DELETE /healthz", "DELETE", "/healthz", http.StatusNotFound),
			Entry("POST /readyz", "POST", "/readyz", http.StatusNotFound),
			Entry("PUT /readyz", "PUT", "/readyz", http.StatusNotFound),
			Entry("DELETE /readyz", "DELETE", "/readyz", http.StatusNotFound),
		)
	})
})
