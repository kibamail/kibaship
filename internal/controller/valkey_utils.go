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
	"crypto/sha256"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/utils"
)

// generateValkeyCredentialsSecret creates a secret with Valkey credentials
func generateValkeyCredentialsSecret(deployment *platformv1alpha1.Deployment, projectName, projectSlug, appSlug string, namespace string) (*corev1.Secret, error) {
	password, err := generateSecurePassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Valkey password: %w", err)
	}

	secretName := utils.GetValkeyResourceName(deployment.GetUUID())

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":        projectName,
				"app.kubernetes.io/managed-by":  "kibaship",
				"app.kubernetes.io/component":   "valkey-credentials",
				"project.kibaship.com/slug":     projectSlug,
				"application.kibaship.com/name": deployment.Spec.ApplicationRef.Name,
			},
			Annotations: map[string]string{
				"description":                fmt.Sprintf("Valkey credentials for project %s application %s", projectSlug, appSlug),
				"project.kibaship.com/usage": "Contains Valkey authentication credentials for database access",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"password": password,
		},
	}

	return secret, nil
}

// generateValkeyInstance creates a Valkey resource for single-instance deployment
func generateValkeyInstance(deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application, projectName, projectSlug, appSlug string, secretName, namespace string) *unstructured.Unstructured {
	deploymentUUID := deployment.GetUUID()

	// Valkey operator has naming constraints, use simple naming
	instanceName := utils.GetValkeyResourceName(deploymentUUID)

	// If name is still too long, truncate it
	if len(instanceName) > 63 {
		// Use hash for uniqueness if too long
		hash := sha256.Sum256([]byte(deploymentUUID))
		instanceName = fmt.Sprintf("valkey-%x", hash)[:63]
	}

	instance := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "hyperspike.io/v1",
			"kind":       "Valkey",
			"metadata": map[string]interface{}{
				"name":      instanceName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/name":        projectName,
					"app.kubernetes.io/managed-by":  "kibaship",
					"app.kubernetes.io/component":   "valkey-database",
					"project.kibaship.com/slug":     projectSlug,
					"application.kibaship.com/name": deployment.Spec.ApplicationRef.Name,
				},
				"annotations": map[string]interface{}{
					"description":                fmt.Sprintf("Valkey database instance for project %s application %s", projectSlug, appSlug),
					"project.kibaship.com/usage": "Provides Valkey database services for the application",
				},
			},
			"spec": map[string]interface{}{
				"nodes":    1,
				"replicas": 0,
				"servicePassword": map[string]interface{}{
					"name": secretName,
					"key":  "password",
				},
				"anonymousAuth":     false,
				"clusterDomain":     "cluster.local",
				"prometheus":        false,
				"volumePermissions": false,
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"memory": "256Mi",
						"cpu":    "100m",
					},
					"limits": map[string]interface{}{
						"memory": "512Mi",
						"cpu":    "500m",
					},
				},
			},
		},
	}

	// Set version if specified in application config
	if app.Spec.Valkey != nil && app.Spec.Valkey.Version != "" {
		spec := instance.Object["spec"].(map[string]interface{})
		spec["image"] = fmt.Sprintf("valkey/valkey:%s", app.Spec.Valkey.Version)
	}

	return instance
}

// generateValkeyCluster creates a Valkey resource for cluster deployment
func generateValkeyCluster(deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application, projectName, projectSlug, appSlug string, secretName, namespace string) *unstructured.Unstructured {
	deploymentUUID := deployment.GetUUID()

	// Valkey operator has naming constraints, use simple naming
	clusterName := utils.GetValkeyClusterResourceName(deploymentUUID)

	// If name is still too long, truncate it
	if len(clusterName) > 63 {
		// Use hash for uniqueness if too long
		hash := sha256.Sum256([]byte(deploymentUUID))
		clusterName = fmt.Sprintf("valkey-cluster-%x", hash)[:63]
	}

	// Default values for cluster
	nodes := int32(3)
	replicas := int32(1)

	// Use configuration from application spec if provided
	if app.Spec.ValkeyCluster != nil && app.Spec.ValkeyCluster.Replicas > 0 {
		// For Valkey cluster, replicas field represents total instances
		// We need to calculate nodes and replicas per node
		totalInstances := app.Spec.ValkeyCluster.Replicas
		if totalInstances >= 6 {
			nodes = 3
			replicas = (totalInstances / nodes) - 1 // -1 because master doesn't count as replica
		} else if totalInstances >= 3 {
			nodes = totalInstances
			replicas = 0
		}
	}

	cluster := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "hyperspike.io/v1",
			"kind":       "Valkey",
			"metadata": map[string]interface{}{
				"name":      clusterName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/name":        projectName,
					"app.kubernetes.io/managed-by":  "kibaship",
					"app.kubernetes.io/component":   "valkey-cluster",
					"project.kibaship.com/slug":     projectSlug,
					"application.kibaship.com/name": deployment.Spec.ApplicationRef.Name,
				},
				"annotations": map[string]interface{}{
					"description":                fmt.Sprintf("Valkey cluster for project %s application %s", projectSlug, appSlug),
					"project.kibaship.com/usage": "Provides Valkey cluster services for the application",
				},
			},
			"spec": map[string]interface{}{
				"nodes":    nodes,
				"replicas": replicas,
				"servicePassword": map[string]interface{}{
					"name": secretName,
					"key":  "password",
				},
				"anonymousAuth":     false,
				"clusterDomain":     "cluster.local",
				"prometheus":        false,
				"volumePermissions": false,
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"memory": "256Mi",
						"cpu":    "100m",
					},
					"limits": map[string]interface{}{
						"memory": "512Mi",
						"cpu":    "500m",
					},
				},
			},
		},
	}

	// Set version if specified in application config
	if app.Spec.ValkeyCluster != nil && app.Spec.ValkeyCluster.Version != "" {
		spec := cluster.Object["spec"].(map[string]interface{})
		spec["image"] = fmt.Sprintf("valkey/valkey:%s", app.Spec.ValkeyCluster.Version)
	}

	return cluster
}

// generateValkeyResourceNames generates resource names following naming conventions
func generateValkeyResourceNames(deployment *platformv1alpha1.Deployment, _, _ string) (secretName, instanceName string) {
	deploymentUUID := deployment.GetUUID()

	// Use unified resource naming helpers
	secretName = utils.GetValkeyResourceName(deploymentUUID)
	instanceName = utils.GetValkeyResourceName(deploymentUUID)

	// For Valkey instances (63 character limit)
	if len(instanceName) > 63 {
		// Use hash for uniqueness if too long
		hash := sha256.Sum256([]byte(deploymentUUID))
		instanceName = fmt.Sprintf("valkey-%x", hash)[:63]
	}
	return
}

// generateValkeyClusterResourceNames generates resource names for cluster deployments
func generateValkeyClusterResourceNames(deployment *platformv1alpha1.Deployment, _, _ string) (secretName, clusterName string) {
	deploymentUUID := deployment.GetUUID()

	// Use unified resource naming helpers (secret uses same base name as single valkey)
	secretName = utils.GetValkeyResourceName(deploymentUUID)
	clusterName = utils.GetValkeyClusterResourceName(deploymentUUID)

	// For Valkey clusters (63 character limit)
	if len(clusterName) > 63 {
		// Use hash for uniqueness if too long
		hash := sha256.Sum256([]byte(deploymentUUID))
		clusterName = fmt.Sprintf("valkey-cluster-%x", hash)[:63]
	}
	return
}

// validateValkeyConfiguration validates Valkey configuration from application spec
func validateValkeyConfiguration(app *platformv1alpha1.Application) error {
	if app.Spec.Type != platformv1alpha1.ApplicationTypeValkey {
		return fmt.Errorf("application type must be Valkey")
	}

	// Valkey configuration is optional, so no additional validation needed for now
	return nil
}

// validateValkeyClusterConfiguration validates Valkey cluster configuration from application spec
func validateValkeyClusterConfiguration(app *platformv1alpha1.Application) error {
	if app.Spec.Type != platformv1alpha1.ApplicationTypeValkeyCluster {
		return fmt.Errorf("application type must be ValkeyCluster")
	}

	// Validate cluster-specific requirements
	if app.Spec.ValkeyCluster != nil && app.Spec.ValkeyCluster.Replicas > 0 && app.Spec.ValkeyCluster.Replicas < 3 {
		return fmt.Errorf("valkey cluster requires at least 3 instances")
	}

	return nil
}
