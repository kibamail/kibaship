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

// @title Kibaship Operator API
// @version 1.0
// @description REST API server for managing Kibaship Operator resources
// @termsOfService https://github.com/kibamail/kibaship
// @contact.name API Support
// @contact.email support@kibamail.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8080
// @BasePath /v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kibamail/kibaship/api/v1alpha1"
	_ "github.com/kibamail/kibaship/docs"
	"github.com/kibamail/kibaship/pkg/auth"
	"github.com/kibamail/kibaship/pkg/handlers"
	"github.com/kibamail/kibaship/pkg/services"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	// Set Gin to release mode if not in development
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Get namespace from environment or use default
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// Initialize secret manager and create/retrieve API key
	log.Println("Initializing API key authentication...")
	secretManager, err := auth.NewSecretManager(namespace)
	if err != nil {
		log.Fatalf("Failed to create secret manager: %v", err)
	}

	apiKey, err := secretManager.CreateOrGetAPIKey(context.Background())
	if err != nil {
		log.Fatalf("Failed to create or retrieve API key: %v", err)
	}

	log.Println("API key ready for authentication")

	// Initialize Kubernetes client and scheme
	log.Println("Initializing Kubernetes client...")
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client config: %v", err)
	}

	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	log.Println("Kubernetes client initialized successfully")

	// Create services
	projectService := services.NewProjectService(k8sClient, scheme)
	environmentService := services.NewEnvironmentService(k8sClient, scheme, projectService)

	// Create authenticator
	authenticator := auth.NewAPIKeyAuthenticator(apiKey)

	// Create Gin router
	router := gin.New()

	// Add basic middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Swagger documentation endpoints
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/openapi.yaml", serveSwaggerYAML)

	// Health check endpoints (public)
	router.GET("/healthz", healthzHandler)
	router.GET("/readyz", readyzHandler)

	// Protected routes - v1 API
	v1 := router.Group("/v1")
	v1.Use(authenticator.Middleware())
	{
		// Initialize services with dependency injection
		projectHandler := handlers.NewProjectHandler(projectService)
		environmentHandler := handlers.NewEnvironmentHandler(environmentService)
		applicationService := services.NewApplicationService(k8sClient, scheme, projectService, environmentService)
		deploymentService := services.NewDeploymentService(k8sClient, scheme, applicationService)
		applicationDomainService := services.NewApplicationDomainService(k8sClient, scheme, applicationService)

		// Set circular dependencies for auto-loading
		applicationService.SetDomainService(applicationDomainService)
		applicationService.SetDeploymentService(deploymentService)

		// Initialize handlers
		applicationHandler := handlers.NewApplicationHandler(applicationService)
		deploymentHandler := handlers.NewDeploymentHandler(deploymentService)
		applicationDomainHandler := handlers.NewApplicationDomainHandler(applicationDomainService)

		// Project endpoints
		v1.POST("/projects", projectHandler.CreateProject)
		v1.GET("/projects/:uuid", projectHandler.GetProject)
		v1.PATCH("/projects/:uuid", projectHandler.UpdateProject)
		v1.DELETE("/projects/:uuid", projectHandler.DeleteProject)

		// Environment endpoints
		v1.POST("/projects/:uuid/environments", environmentHandler.CreateEnvironment)
		v1.GET("/projects/:uuid/environments", environmentHandler.GetEnvironmentsByProject)
		v1.GET("/environments/:uuid", environmentHandler.GetEnvironment)
		v1.PATCH("/environments/:uuid", environmentHandler.UpdateEnvironment)
		v1.DELETE("/environments/:uuid", environmentHandler.DeleteEnvironment)

		// Application endpoints
		v1.POST("/environments/:uuid/applications", applicationHandler.CreateApplication)
		v1.GET("/environments/:uuid/applications", applicationHandler.GetApplicationsByEnvironment)
		v1.GET("/projects/:uuid/applications", applicationHandler.GetApplicationsByProject)
		v1.GET("/applications/:uuid", applicationHandler.GetApplication)
		v1.PATCH("/applications/:uuid", applicationHandler.UpdateApplication)
		v1.PATCH("/applications/:uuid/env", applicationHandler.UpdateApplicationEnv)
		v1.DELETE("/applications/:uuid", applicationHandler.DeleteApplication)

		// Deployment endpoints
		v1.POST("/applications/:uuid/deployments", deploymentHandler.CreateDeployment)
		v1.GET("/applications/:uuid/deployments", deploymentHandler.GetDeploymentsByApplication)
		v1.GET("/deployments/:uuid", deploymentHandler.GetDeployment)
		v1.POST("/deployments/:uuid/promote", deploymentHandler.PromoteDeployment)

		// Application Domain endpoints
		v1.POST("/applications/:uuid/domains", applicationDomainHandler.CreateApplicationDomain)
		v1.GET("/domains/:uuid", applicationDomainHandler.GetApplicationDomain)
		v1.DELETE("/domains/:uuid", applicationDomainHandler.DeleteApplicationDomain)
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting API server on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// healthzHandler handles the health check endpoint
// @Summary Health check
// @Description Check if the API server is healthy
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /healthz [get]
func healthzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// ReadyResponse represents the readiness check response
type ReadyResponse struct {
	Status string `json:"status" example:"ready"`
}

// readyzHandler handles the readiness check endpoint
// @Summary Readiness check
// @Description Check if the API server is ready to serve requests
// @Tags health
// @Produce json
// @Success 200 {object} ReadyResponse
// @Router /readyz [get]
func readyzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// serveSwaggerYAML serves the OpenAPI YAML file
// @Summary Get OpenAPI specification
// @Description Get the OpenAPI specification in YAML format
// @Tags documentation
// @Produce text/plain
// @Success 200 {string} string "OpenAPI YAML specification"
// @Router /openapi.yaml [get]
func serveSwaggerYAML(c *gin.Context) {
	c.File("docs/swagger.yaml")
}
