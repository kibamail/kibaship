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

package controller

import (
	"context"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ValkeyProvisioner handles provisioning of the system Valkey cluster
type ValkeyProvisioner struct {
	client.Client
}

// NewValkeyProvisioner creates a new ValkeyProvisioner
func NewValkeyProvisioner(k8sClient client.Client) *ValkeyProvisioner {
	return &ValkeyProvisioner{
		Client: k8sClient,
	}
}

// ProvisionSystemValkeyCluster provisions the system Valkey cluster if it doesn't exist
func (p *ValkeyProvisioner) ProvisionSystemValkeyCluster(ctx context.Context) error {
	log := logf.FromContext(ctx).WithName("valkey-provisioner")

	// Get the operator namespace
	namespace := getOperatorNamespace()

	// Generate cluster name following naming conventions
	clusterName := generateSystemValkeyClusterName()

	log.Info("Checking for existing system Valkey cluster", "name", clusterName, "namespace", namespace)

	// Check if Valkey cluster already exists
	exists, err := p.checkValkeyClusterExists(ctx, clusterName, namespace)
	if err != nil {
		return fmt.Errorf("failed to check for existing Valkey cluster: %w", err)
	}

	if exists {
		log.Info("System Valkey cluster already exists, skipping provisioning", "name", clusterName, "namespace", namespace)
		return nil
	}

	log.Info("System Valkey cluster does not exist, creating new cluster", "name", clusterName, "namespace", namespace)

	// Create the Valkey cluster
	if err := p.createValkeyCluster(ctx, clusterName, namespace); err != nil {
		return fmt.Errorf("failed to create Valkey cluster: %w", err)
	}

	log.Info("Successfully provisioned system Valkey cluster", "name", clusterName, "namespace", namespace)
	return nil
}

// checkValkeyClusterExists checks if a Valkey cluster exists in the given namespace
func (p *ValkeyProvisioner) checkValkeyClusterExists(ctx context.Context, name, namespace string) (bool, error) {
	valkey := &unstructured.Unstructured{}
	valkey.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "hyperspike.io",
		Version: "v1",
		Kind:    "Valkey",
	})

	err := p.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, valkey)

	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// createValkeyCluster creates a new Valkey cluster with the system configuration
func (p *ValkeyProvisioner) createValkeyCluster(ctx context.Context, name, namespace string) error {
	valkey := generateSystemValkeyCluster(name, namespace)

	if err := p.Create(ctx, valkey); err != nil {
		return fmt.Errorf("failed to create Valkey cluster: %w", err)
	}

	return nil
}

// generateSystemValkeyClusterName generates the name for the system Valkey cluster
// following the established naming convention
func generateSystemValkeyClusterName() string {
	return "kibaship-valkey-cluster-kibaship-com"
}

// generateSystemValkeyCluster creates the Valkey cluster resource with system configuration
func generateSystemValkeyCluster(name, namespace string) *unstructured.Unstructured {
	valkey := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "hyperspike.io/v1",
			"kind":       "Valkey",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/name":       "kibaship-valkey-cluster",
					"app.kubernetes.io/managed-by": "kibaship",
					"app.kubernetes.io/component":  "system-cache",
					"app.kubernetes.io/part-of":    "kibaship-platform",
				},
				"annotations": map[string]interface{}{
					"description":                "System Valkey cluster for KibaShip platform caching and session management",
					"platform.kibaship.com/role": "system-cache",
				},
			},
			"spec": map[string]interface{}{
				"nodes":    int64(3),
				"replicas": int64(2),
				"resources": map[string]interface{}{
					"limits": map[string]interface{}{
						"cpu":    "100m",
						"memory": "128Mi",
					},
					"requests": map[string]interface{}{
						"cpu":    "100m",
						"memory": "128Mi",
					},
				},
				"tls": false,
				"externalAccess": map[string]interface{}{
					"enabled": false,
				},
				"volumePermissions": true,
				"prometheus":        true,
				"storage": map[string]interface{}{
					"spec": map[string]interface{}{
						"accessModes": []interface{}{"ReadWriteOnce"},
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"storage": "1Gi",
							},
						},
						"storageClassName": "storage-replica-1",
					},
				},
			},
		},
	}

	return valkey
}

// getOperatorNamespace returns the namespace where the operator is running
func getOperatorNamespace() string {
	// First try to get from environment variable (standard for operators)
	if ns := os.Getenv("OPERATOR_NAMESPACE"); ns != "" {
		return ns
	}

	// Fallback to POD_NAMESPACE (common alternative)
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}

	// Try to read from the service account token file (when running in-cluster)
	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err == nil {
		return string(namespace)
	}

	// Final fallback - assume default operator namespace
	return "kibaship-system"
}
