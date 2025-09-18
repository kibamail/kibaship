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

package streaming

import (
	"context"
	"fmt"
	"sync"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// startupSequenceController implements StartupSequenceController
type startupSequenceController struct {
	readinessMonitor  ValkeyReadinessMonitor
	secretManager     SecretManager
	connectionManager ConnectionManager
	config            *Config
	ready             bool
	mutex             sync.RWMutex
}

// NewStartupSequenceController creates a new startup sequence controller
func NewStartupSequenceController(
	readinessMonitor ValkeyReadinessMonitor,
	secretManager SecretManager,
	connectionManager ConnectionManager,
	config *Config,
) StartupSequenceController {
	return &startupSequenceController{
		readinessMonitor:  readinessMonitor,
		secretManager:     secretManager,
		connectionManager: connectionManager,
		config:            config,
	}
}

// Initialize runs the complete startup sequence
func (s *startupSequenceController) Initialize(ctx context.Context) error {
	log := logf.FromContext(ctx).WithName("startup-sequence")
	log.Info("Starting Valkey streaming initialization sequence")

	// Step 1: Wait for Valkey cluster to become ready
	log.Info("Step 1: Waiting for Valkey cluster to become ready", "timeout", s.config.StartupTimeout)
	err := s.readinessMonitor.WaitForReady(ctx, s.config.StartupTimeout)
	if err != nil {
		return fmt.Errorf("valkey cluster failed to become ready within %v: %w", s.config.StartupTimeout, err)
	}
	log.Info("✓ Valkey cluster is ready")

	// Step 2: Fetch Valkey authentication secret
	log.Info("Step 2: Fetching Valkey authentication secret")
	password, err := s.secretManager.GetValkeyPassword(ctx)
	if err != nil {
		return fmt.Errorf("valkey authentication secret not found after cluster ready: %w", err)
	}
	if password == "" {
		return fmt.Errorf("valkey authentication secret empty after cluster ready")
	}
	log.Info("✓ Valkey authentication secret retrieved")

	// Step 3: Initialize cluster connection with auto-discovery
	log.Info("Step 3: Initializing Valkey cluster connection with auto-discovery")

	// Build seed address for cluster discovery
	seedAddress := fmt.Sprintf("%s.%s.svc.cluster.local",
		s.config.ValkeyServiceName,
		s.config.Namespace)

	err = s.connectionManager.InitializeCluster(ctx, seedAddress, password)
	if err != nil {
		return fmt.Errorf("failed to initialize Valkey cluster connection: %w", err)
	}
	log.Info("✓ Valkey cluster connection initialized")

	// Mark as ready
	s.mutex.Lock()
	s.ready = true
	s.mutex.Unlock()

	log.Info("🎉 Valkey streaming initialization completed successfully")
	return nil
}

// IsReady returns true if streaming is ready
func (s *startupSequenceController) IsReady() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if !s.ready {
		return false
	}
	return s.connectionManager.IsConnected()
}

// Shutdown gracefully shuts down the streaming components
func (s *startupSequenceController) Shutdown(ctx context.Context) error {
	log := logf.FromContext(ctx).WithName("startup-sequence")
	log.Info("Shutting down Valkey streaming components")

	s.mutex.Lock()
	s.ready = false
	s.mutex.Unlock()

	// Close connection
	err := s.connectionManager.Close()
	if err != nil {
		log.Error(err, "Error closing Valkey connection")
		return err
	}

	log.Info("Valkey streaming components shut down successfully")
	return nil
}
