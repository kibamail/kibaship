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

package utils

import "fmt"

// Resource naming conventions for CRDs and associated Kubernetes resources.
// These functions ensure consistent naming across all resources.

// GetProjectResourceName returns the standard name for a Project resource
func GetProjectResourceName(uuid string) string {
	return fmt.Sprintf("project-%s", uuid)
}

// GetEnvironmentResourceName returns the standard name for an Environment resource
func GetEnvironmentResourceName(uuid string) string {
	return fmt.Sprintf("environment-%s", uuid)
}

// GetApplicationResourceName returns the standard name for an Application resource
// This name is used for both the Application CR and its associated secret
func GetApplicationResourceName(uuid string) string {
	return fmt.Sprintf("application-%s", uuid)
}

// GetDeploymentResourceName returns the standard name for a Deployment resource
// This name is used for the Deployment CR, its associated secret, and the Kubernetes Deployment
func GetDeploymentResourceName(uuid string) string {
	return fmt.Sprintf("deployment-%s", uuid)
}

// GetApplicationDomainResourceName returns the standard name for an ApplicationDomain resource
func GetApplicationDomainResourceName(uuid string) string {
	return fmt.Sprintf("domain-%s", uuid)
}

// GetValkeyResourceName returns the standard name for a Valkey resource
// This name is used for both the Valkey CR and its associated secret
func GetValkeyResourceName(uuid string) string {
	return fmt.Sprintf("valkey-%s", uuid)
}

// GetValkeyClusterResourceName returns the standard name for a ValkeyCluster resource
// This name is used for both the ValkeyCluster CR and its associated secret
func GetValkeyClusterResourceName(uuid string) string {
	return fmt.Sprintf("valkey-cluster-%s", uuid)
}

// MySQL uses a unique slug instead of UUID due to name length limits

// GetMySQLResourceName returns the standard name for a MySQL resource
// MySQL uses a shorter slug format due to MySQL Operator's 28-char limit
// This name is used for both the MySQL CR and its associated secret
func GetMySQLResourceName(slug string) string {
	return fmt.Sprintf("m-%s", slug)
}

// GetMySQLClusterResourceName returns the standard name for a MySQLCluster resource
// This name is used for both the MySQLCluster CR and its associated secret
func GetMySQLClusterResourceName(slug string) string {
	return fmt.Sprintf("mc-%s", slug)
}

// GetServiceName returns the standard name for a Kubernetes Service associated with an Application
func GetServiceName(applicationUUID string) string {
	return fmt.Sprintf("service-%s", applicationUUID)
}

// GetKubernetesDeploymentName returns the standard name for a Kubernetes Deployment resource
// This is the same as the Deployment CRD name
func GetKubernetesDeploymentName(deploymentUUID string) string {
	return GetDeploymentResourceName(deploymentUUID)
}
