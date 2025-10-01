package registryauth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server wraps the HTTP server for the auth service
type Server struct {
	handler *Handler
	config  Config
	server  *http.Server
}

// NewServer creates a new auth server
func NewServer(config Config) (*Server, error) {
	// Initialize Kubernetes client
	k8sClient, err := NewK8sClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Initialize credential cache
	cache := NewCredentialCache(config.Cache.TTLSeconds)
	cache.StartCleanupRoutine()

	// Initialize validator
	validator := NewValidator(k8sClient, cache)

	// Initialize token generator
	tokenGenerator, err := NewTokenGenerator(
		config.JWT.PrivateKeyPath,
		config.JWT.Issuer,
		config.JWT.ExpirationSec,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create token generator: %w", err)
	}

	// Initialize handler
	handler := NewHandler(validator, tokenGenerator, config.Registry.ServiceName)

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", handler.ServeAuth)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              config.Server.Listen,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{
		handler: handler,
		config:  config,
		server:  server,
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("starting registry auth service on %s", s.config.Server.Listen)
	log.Printf("jwt issuer: %s, expiration: %ds", s.config.JWT.Issuer, s.config.JWT.ExpirationSec)
	log.Printf("registry service: %s", s.config.Registry.ServiceName)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Printf("shutting down registry auth service...")
	return s.server.Shutdown(ctx)
}
