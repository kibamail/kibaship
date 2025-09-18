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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// valkeyReadinessMonitor implements ValkeyReadinessMonitor (DEPRECATED: use ValkeyReadyGate)
type valkeyReadinessMonitor struct {
	client       KubernetesClient
	timeProvider TimeProvider
	config       *Config
}

// NewValkeyReadinessMonitor creates a new Valkey readiness monitor
func NewValkeyReadinessMonitor(
	kubeClient KubernetesClient, timeProvider TimeProvider, config *Config,
) ValkeyReadinessMonitor {
	return &valkeyReadinessMonitor{
		client:       kubeClient,
		timeProvider: timeProvider,
		config:       config,
	}
}

// WaitForReady waits for Valkey cluster to become ready within timeout
func (v *valkeyReadinessMonitor) WaitForReady(ctx context.Context, timeout time.Duration) error {
	log := logf.FromContext(ctx).WithName("valkey-readiness")
	log.Info("Starting Valkey cluster readiness check", "timeout", timeout, "resource", v.config.ValkeyServiceName)

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check if already ready
	if v.IsReady(timeoutCtx) {
		log.Info("Valkey cluster is already ready")
		return nil
	}

	// DEPRECATED: Using simple polling since watch functionality was removed
	log.Info("DEPRECATED: Using old ValkeyReadinessMonitor, consider switching to ValkeyReadyGate")

	// Simple polling every 10 seconds (fallback implementation)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for Valkey cluster to become ready after %v", timeout)
		case <-ticker.C:
			if v.IsReady(timeoutCtx) {
				log.Info("Valkey cluster became ready")
				return nil
			}
			log.V(1).Info("Valkey cluster not ready yet, checking again in 10s")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// IsReady checks if Valkey cluster is currently ready
func (v *valkeyReadinessMonitor) IsReady(ctx context.Context) bool {
	valkeyResource, err := v.getValkeyResource(ctx)
	if err != nil {
		return false
	}

	return v.isValkeyResourceReady(valkeyResource)
}

// DEPRECATED: Use ValkeyReadyGate instead of watch-based monitoring

// getValkeyResource retrieves the Valkey resource
func (v *valkeyReadinessMonitor) getValkeyResource(ctx context.Context) (*unstructured.Unstructured, error) {
	valkeyResource := &unstructured.Unstructured{}
	valkeyResource.SetAPIVersion("hyperspike.io/v1")
	valkeyResource.SetKind("Valkey")

	err := v.client.Get(ctx, client.ObjectKey{
		Name:      v.config.ValkeyServiceName,
		Namespace: v.config.Namespace,
	}, valkeyResource)

	return valkeyResource, err
}

// isValkeyResourceReady checks if the Valkey resource is ready
func (v *valkeyReadinessMonitor) isValkeyResourceReady(valkeyResource *unstructured.Unstructured) bool {
	// Based on the kubectl output, the Valkey resource has a Status.Ready field
	status, found, err := unstructured.NestedMap(valkeyResource.Object, "status")
	if err != nil || !found {
		return false
	}

	// Check the Ready field in status
	ready, found, err := unstructured.NestedBool(status, "ready")
	if err != nil || !found {
		return false
	}

	return ready
}

// Close stops the Valkey watcher (DEPRECATED: ValkeyReadyGate doesn't need explicit close)
func (v *valkeyReadinessMonitor) Close() {
	// No-op: watch functionality removed
}
