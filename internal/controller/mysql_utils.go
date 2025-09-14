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
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
)

const (
	// passwordLength is the length of generated MySQL passwords
	passwordLength = 32
	// alphanumericChars are the characters used for password generation
	alphanumericChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// generateMySQLCredentialsSecret creates a secret with MySQL root credentials
func generateMySQLCredentialsSecret(deployment *platformv1alpha1.Deployment, projectSlug, appSlug string, namespace string) (*corev1.Secret, error) {
	password, err := generateSecurePassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate MySQL password: %w", err)
	}

	secretName := fmt.Sprintf("project-%s-app-%s-mysql-credentials-kibaship-com", projectSlug, appSlug)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":        fmt.Sprintf("project-%s", projectSlug),
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
func generateInnoDBCluster(deployment *platformv1alpha1.Deployment, app *platformv1alpha1.Application, projectSlug, appSlug string, secretName, namespace string) *unstructured.Unstructured {
	// MySQL operator has a 40-character limit, so we need shorter names
	clusterName := fmt.Sprintf("%s-%s-mysql", projectSlug, appSlug)

	// If name is still too long, truncate it
	if len(clusterName) > 40 {
		// Use first 8 characters of hash for uniqueness
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%s", projectSlug, appSlug)))
		hashSuffix := fmt.Sprintf("-%x", hash)[:9] // 8 chars + dash
		maxPrefix := 40 - len("-mysql") - len(hashSuffix)
		if maxPrefix > 0 {
			clusterName = fmt.Sprintf("%s%s-mysql", (projectSlug + "-" + appSlug)[:maxPrefix], hashSuffix)
		} else {
			// Fallback to just hash-mysql if still too long
			clusterName = fmt.Sprintf("%x-mysql", hash)[:40]
		}
	}

	cluster := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "mysql.oracle.com/v2",
			"kind":       "InnoDBCluster",
			"metadata": map[string]interface{}{
				"name":      clusterName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app.kubernetes.io/name":        fmt.Sprintf("project-%s", projectSlug),
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
							"app.kubernetes.io/name":       fmt.Sprintf("project-%s", projectSlug),
							"app.kubernetes.io/managed-by": "kibaship",
							"project.kibaship.com/slug":    projectSlug,
						},
					},
					"spec": map[string]interface{}{
						"accessModes":      []string{"ReadWriteOnce"},
						"storageClassName": "storage-replica-2",
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
func generateMySQLResourceNames(_ *platformv1alpha1.Deployment, projectSlug, appSlug string) (secretName, clusterName string) {
	// For secrets (no length limit in practice)
	secretName = fmt.Sprintf("project-%s-app-%s-mysql-credentials-kibaship-com", projectSlug, appSlug)

	// For InnoDBCluster (40 character limit)
	clusterName = fmt.Sprintf("%s-%s-mysql", projectSlug, appSlug)
	if len(clusterName) > 40 {
		// Use hash for uniqueness if too long
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%s", projectSlug, appSlug)))
		hashSuffix := fmt.Sprintf("-%x", hash)[:9] // 8 chars + dash
		maxPrefix := 40 - len("-mysql") - len(hashSuffix)
		if maxPrefix > 0 {
			clusterName = fmt.Sprintf("%s%s-mysql", (projectSlug + "-" + appSlug)[:maxPrefix], hashSuffix)
		} else {
			// Fallback to just hash-mysql if still too long
			clusterName = fmt.Sprintf("%x-mysql", hash)[:40]
		}
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

// extractProjectAndAppSlugs extracts project and application slugs from deployment name
func extractProjectAndAppSlugs(deploymentName string) (projectSlug, appSlug string, err error) {
	// Expected format: project-<project-slug>-app-<app-slug>-deployment-<deployment-slug>-kibaship-com
	parts := strings.Split(deploymentName, "-")
	if len(parts) < 7 {
		return "", "", fmt.Errorf("invalid deployment name format: %s", deploymentName)
	}

	// Find indices of key parts - need to find the correct delimiters
	projectIndex := -1
	appIndex := -1
	deploymentIndex := -1

	// Find "project" - should be first
	for i, part := range parts {
		if part == "project" {
			projectIndex = i
			break
		}
	}

	// Find "deployment" - should be towards the end
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "deployment" {
			deploymentIndex = i
			break
		}
	}

	// Find "app" - should be between project and deployment
	for i := projectIndex + 1; i < deploymentIndex; i++ {
		if parts[i] == "app" {
			// Check if this could be a valid delimiter by ensuring there's content after it
			if i+1 < deploymentIndex {
				appIndex = i
				break
			}
		}
	}

	if projectIndex == -1 || appIndex == -1 || deploymentIndex == -1 {
		return "", "", fmt.Errorf("invalid deployment name format: missing required parts in %s", deploymentName)
	}

	if appIndex <= projectIndex+1 || deploymentIndex <= appIndex+1 {
		return "", "", fmt.Errorf("invalid deployment name format: incorrect part ordering in %s", deploymentName)
	}

	// Extract project slug (between "project" and "app")
	projectSlug = strings.Join(parts[projectIndex+1:appIndex], "-")

	// Extract app slug (between "app" and "deployment")
	appSlug = strings.Join(parts[appIndex+1:deploymentIndex], "-")

	if projectSlug == "" || appSlug == "" {
		return "", "", fmt.Errorf("invalid deployment name format: empty slugs in %s", deploymentName)
	}

	return projectSlug, appSlug, nil
}
