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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// valkeyReadyGate implements ValkeyReadyGate with simple polling
type valkeyReadyGate struct {
	client        KubernetesClient
	config        *Config
	checkInterval time.Duration
	maxWait       time.Duration
}

// NewValkeyReadyGate creates a new Valkey ready gate with simple polling
func NewValkeyReadyGate(kubeClient KubernetesClient, config *Config) ValkeyReadyGate {
	return &valkeyReadyGate{
		client:        kubeClient,
		config:        config,
		checkInterval: 20 * time.Second, // Check every 20 seconds
		maxWait:       5 * time.Minute,  // Timeout after 5 minutes
	}
}

// WaitForReady blocks until Valkey cluster is ready or timeout
func (v *valkeyReadyGate) WaitForReady(ctx context.Context) error {
	log := logf.FromContext(ctx).WithName("valkey-ready-gate")
	log.Info("Starting Valkey readiness check",
		"checkInterval", v.checkInterval,
		"maxWait", v.maxWait,
		"resource", v.config.ValkeyServiceName)

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, v.maxWait)
	defer cancel()

	// Check immediately first
	log.V(1).Info("Performing initial Valkey readiness check")
	if v.isValkeyReady(timeoutCtx) {
		log.Info("✅ Valkey cluster is already ready")
		return nil
	}
	log.Info("Valkey cluster not ready yet, starting polling loop")

	// Start polling loop
	ticker := time.NewTicker(v.checkInterval)
	defer ticker.Stop()

	checkCount := 0
	startTime := time.Now()

	for {
		select {
		case <-timeoutCtx.Done():
			log.Error(timeoutCtx.Err(), "❌ Timeout waiting for Valkey cluster to become ready",
				"timeout", v.maxWait,
				"checks", checkCount,
				"resource", v.config.ValkeyServiceName)
			return fmt.Errorf("timeout: Valkey cluster '%s' not ready after %v (%d checks)",
				v.config.ValkeyServiceName, v.maxWait, checkCount)

		case <-ticker.C:
			checkCount++
			log.V(1).Info("Checking Valkey readiness", "check", checkCount)

			if v.isValkeyReady(timeoutCtx) {
				elapsed := time.Since(startTime)
				log.Info("✅ Valkey cluster became ready", "checks", checkCount, "elapsed", elapsed.Round(time.Second))
				return nil
			}

			elapsed := time.Since(startTime)
			remaining := v.maxWait - elapsed
			log.Info("Valkey cluster not ready yet, checking again",
				"check", checkCount,
				"nextCheck", v.checkInterval,
				"elapsed", elapsed.Round(time.Second),
				"timeRemaining", remaining.Round(time.Second))

		case <-ctx.Done():
			log.Info("Context cancelled while waiting for Valkey readiness")
			return ctx.Err()
		}
	}
}

// isValkeyReady checks if the Valkey cluster is currently ready
func (v *valkeyReadyGate) isValkeyReady(ctx context.Context) bool {
	log := logf.FromContext(ctx).WithName("valkey-ready-check")

	// Get the Valkey resource
	valkeyResource := &unstructured.Unstructured{}
	valkeyResource.SetAPIVersion("hyperspike.io/v1")
	valkeyResource.SetKind("Valkey")

	err := v.client.Get(ctx, client.ObjectKey{
		Name:      v.config.ValkeyServiceName,
		Namespace: v.config.Namespace,
	}, valkeyResource)

	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("Valkey resource not found", "name", v.config.ValkeyServiceName, "namespace", v.config.Namespace)
		} else {
			log.Error(err, "Error getting Valkey resource", "name", v.config.ValkeyServiceName, "namespace", v.config.Namespace)
		}
		return false
	}

	// Check the Ready field in status
	status, found, err := unstructured.NestedMap(valkeyResource.Object, "status")
	if err != nil || !found {
		log.V(1).Info("Valkey resource has no status field", "name", v.config.ValkeyServiceName)
		return false
	}

	ready, found, err := unstructured.NestedBool(status, "ready")
	if err != nil || !found {
		log.V(1).Info("Valkey resource status has no ready field", "name", v.config.ValkeyServiceName)
		return false
	}

	log.V(1).Info("Valkey resource status", "name", v.config.ValkeyServiceName, "ready", ready)
	return ready
}
