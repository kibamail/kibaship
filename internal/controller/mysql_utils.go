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
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/config"
)

const (
	// passwordLength is the length of generated MySQL passwords
	passwordLength = 32
	// alphanumericChars are the characters used for password generation
	alphanumericChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// generateMySQLCredentialsSecret creates a secret with MySQL root credentials
func generateMySQLCredentialsSecret(deployment *platformv1alpha1.Deployment, projectName, projectSlug, appSlug string, namespace string) (*corev1.Secret, error) {
	password, err := generateSecurePassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate MySQL password: %w", err)
	}

	secretName := fmt.Sprintf("mysql-secret-%s", deployment.GetUUID())

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":        projectName,
				"app.kubernetes.io/managed-by":  "kibaship",
				"app.kubernetes.io/component":   "mysql-credentials",
				"project.kibaship.com/slug":     projectSlug,
				"application.kibaship.com/name": deployment.Spec.ApplicationRef.Name,
			},
			Annotations: map[string]string{
				"description":                fmt.Sprintf("MySQL root credentials for project %s application %s", projectSlug, appSlug),
				"project.kibaship.com/usage": "Contains MySQL root user credentials for database authentication",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"rootUser":     "root",
			"rootHost":     "%",
			"rootPassword": password,
		},
	}

	return secret, nil
}

// generateInnoDBCluster creates an InnoDBCluster resource for MySQL deployment
func generateInnoDBCluster(deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application, projectName, projectSlug, appSlug string, secretName, namespace string) *unstructured.Unstructured {
	deploymentUUID := deployment.GetUUID()

	// MySQL operator has a 40-character limit, use simple naming
	clusterName := fmt.Sprintf("mysql-%s", deploymentUUID)

	// If name is still too long, truncate it
	if len(clusterName) > 40 {
		// Use hash for uniqueness if still too long
		hash := sha256.Sum256([]byte(deploymentUUID))
		clusterName = fmt.Sprintf("mysql-%x", hash)[:40]
	}

	cluster := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "mysql.oracle.com/v2",
			"kind":       "InnoDBCluster",
			"metadata": map[string]interface{}{
				"name":      clusterName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/name":        projectName,
					"app.kubernetes.io/managed-by":  "kibaship",
					"app.kubernetes.io/component":   "mysql-database",
					"project.kibaship.com/slug":     projectSlug,
					"application.kibaship.com/name": deployment.Spec.ApplicationRef.Name,
				},
				"annotations": map[string]interface{}{
					"description":                fmt.Sprintf("MySQL database cluster for project %s application %s", projectSlug, appSlug),
					"project.kibaship.com/usage": "Provides MySQL database services for the application",
				},
			},
			"spec": map[string]interface{}{
				"secretName":       secretName,
				"tlsUseSelfSigned": true,
				"instances":        1,
				"datadirVolumeClaimTemplate": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app.kubernetes.io/name":       projectName,
							"app.kubernetes.io/managed-by": "kibaship",
							"project.kibaship.com/slug":    projectSlug,
						},
					},
					"spec": map[string]interface{}{
						"accessModes":      []string{"ReadWriteOnce"},
						"storageClassName": config.StorageClassReplica2,
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"storage": "512Mi",
							},
						},
					},
				},
				"router": map[string]interface{}{
					"instances": 0,
				},
			},
		},
	}

	// Set version if specified in application config
	if app.Spec.MySQL != nil && app.Spec.MySQL.Version != "" {
		spec := cluster.Object["spec"].(map[string]interface{})
		spec["version"] = app.Spec.MySQL.Version
	}

	return cluster
}

// generateSecurePassword generates a cryptographically secure random password
func generateSecurePassword() (string, error) {
	password := make([]byte, passwordLength)
	charsLen := big.NewInt(int64(len(alphanumericChars)))

	for i := 0; i < passwordLength; i++ {
		charIndex, err := rand.Int(rand.Reader, charsLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random character: %w", err)
		}
		password[i] = alphanumericChars[charIndex.Int64()]
	}

	return string(password), nil
}

// generateMySQLResourceNames generates resource names following naming conventions
func generateMySQLResourceNames(deployment *platformv1alpha1.Deployment, _, _ string) (secretName, clusterName string) {
	deploymentUUID := deployment.GetUUID()

	// For secrets
	secretName = fmt.Sprintf("mysql-secret-%s", deploymentUUID)

	// For InnoDBCluster (40 character limit)
	clusterName = fmt.Sprintf("mysql-%s", deploymentUUID)
	if len(clusterName) > 40 {
		// Use hash for uniqueness if too long
		hash := sha256.Sum256([]byte(deploymentUUID))
		clusterName = fmt.Sprintf("mysql-%x", hash)[:40]
	}
	return
}

// validateMySQLConfiguration validates MySQL configuration from application spec
func validateMySQLConfiguration(app *platformv1alpha1.Application) error {
	if app.Spec.Type != platformv1alpha1.ApplicationTypeMySQL {
		return fmt.Errorf("application type must be MySQL")
	}

	// MySQL configuration is optional, so no additional validation needed for now
	return nil
}

// checkForExistingMySQLDeployments checks if any other deployments exist for this MySQL application
func checkForExistingMySQLDeployments(deployments []platformv1alpha1.Deployment, currentDeployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application) bool {
	for _, deployment := range deployments {
		// Skip the current deployment
		if deployment.Name == currentDeployment.Name {
			continue
		}

		// Check if this deployment references the same application
		if deployment.Spec.ApplicationRef.Name == app.Name &&
			deployment.Namespace == currentDeployment.Namespace {
			return true
		}
	}

	return false
}
