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

package templates

import (
	"github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/models"
)

// GetResourceProfileTemplate returns the CRD configuration for a given resource profile
func GetResourceProfileTemplate(profile models.ResourceProfile, customLimits *models.CustomResourceLimits) v1alpha1.ApplicationTypesConfig {
	switch profile {
	case models.ResourceProfileProduction:
		return getProductionTemplate()
	case models.ResourceProfileCustom:
		if customLimits != nil {
			return getCustomTemplate(customLimits)
		}
		return getDevelopmentTemplate() // fallback
	default: // development
		return getDevelopmentTemplate()
	}
}

// getDevelopmentTemplate returns resource configuration optimized for development
func getDevelopmentTemplate() v1alpha1.ApplicationTypesConfig {
	return v1alpha1.ApplicationTypesConfig{
		MySQL: v1alpha1.ApplicationTypeConfig{
			Enabled: true,
			DefaultLimits: v1alpha1.ResourceLimits{
				CPU:     "0.5",
				Memory:  "1Gi",
				Storage: "5Gi",
			},
			ResourceBounds: v1alpha1.ResourceBounds{
				Min: v1alpha1.ResourceLimits{
					CPU:     "0.1",
					Memory:  "256Mi",
					Storage: "1Gi",
				},
				Max: v1alpha1.ResourceLimits{
					CPU:     "2",
					Memory:  "4Gi",
					Storage: "20Gi",
				},
			},
		},
		MySQLCluster: v1alpha1.ClusterApplicationTypeConfig{
			Enabled: false,
		},
		Postgres: v1alpha1.ApplicationTypeConfig{
			Enabled: true,
			DefaultLimits: v1alpha1.ResourceLimits{
				CPU:     "0.5",
				Memory:  "1Gi",
				Storage: "5Gi",
			},
			ResourceBounds: v1alpha1.ResourceBounds{
				Min: v1alpha1.ResourceLimits{
					CPU:     "0.1",
					Memory:  "256Mi",
					Storage: "1Gi",
				},
				Max: v1alpha1.ResourceLimits{
					CPU:     "2",
					Memory:  "4Gi",
					Storage: "20Gi",
				},
			},
		},
		PostgresCluster: v1alpha1.ClusterApplicationTypeConfig{
			Enabled: false,
		},
		DockerImage: v1alpha1.ApplicationTypeConfig{
			Enabled: true,
			DefaultLimits: v1alpha1.ResourceLimits{
				CPU:     "0.25",
				Memory:  "512Mi",
				Storage: "2Gi",
			},
			ResourceBounds: v1alpha1.ResourceBounds{
				Min: v1alpha1.ResourceLimits{
					CPU:     "0.1",
					Memory:  "128Mi",
					Storage: "1Gi",
				},
				Max: v1alpha1.ResourceLimits{
					CPU:     "4",
					Memory:  "8Gi",
					Storage: "10Gi",
				},
			},
		},
		GitRepository: v1alpha1.ApplicationTypeConfig{
			Enabled: true,
			DefaultLimits: v1alpha1.ResourceLimits{
				CPU:     "0.25",
				Memory:  "512Mi",
				Storage: "2Gi",
			},
			ResourceBounds: v1alpha1.ResourceBounds{
				Min: v1alpha1.ResourceLimits{
					CPU:     "0.1",
					Memory:  "128Mi",
					Storage: "1Gi",
				},
				Max: v1alpha1.ResourceLimits{
					CPU:     "4",
					Memory:  "8Gi",
					Storage: "10Gi",
				},
			},
		},
		ImageFromRegistry: v1alpha1.ApplicationTypeConfig{
			Enabled: true,
			DefaultLimits: v1alpha1.ResourceLimits{
				CPU:     "0.25",
				Memory:  "512Mi",
				Storage: "2Gi",
			},
			ResourceBounds: v1alpha1.ResourceBounds{
				Min: v1alpha1.ResourceLimits{
					CPU:     "0.1",
					Memory:  "128Mi",
					Storage: "1Gi",
				},
				Max: v1alpha1.ResourceLimits{
					CPU:     "4",
					Memory:  "8Gi",
					Storage: "10Gi",
				},
			},
		},
	}
}

// getProductionTemplate returns resource configuration optimized for production
func getProductionTemplate() v1alpha1.ApplicationTypesConfig {
	return v1alpha1.ApplicationTypesConfig{
		MySQL: v1alpha1.ApplicationTypeConfig{
			Enabled: true,
			DefaultLimits: v1alpha1.ResourceLimits{
				CPU:     "2",
				Memory:  "4Gi",
				Storage: "20Gi",
			},
			ResourceBounds: v1alpha1.ResourceBounds{
				Min: v1alpha1.ResourceLimits{
					CPU:     "0.5",
					Memory:  "1Gi",
					Storage: "10Gi",
				},
				Max: v1alpha1.ResourceLimits{
					CPU:     "8",
					Memory:  "16Gi",
					Storage: "100Gi",
				},
			},
		},
		MySQLCluster: v1alpha1.ClusterApplicationTypeConfig{
			Enabled: false, // Can be enabled if needed
		},
		Postgres: v1alpha1.ApplicationTypeConfig{
			Enabled: true,
			DefaultLimits: v1alpha1.ResourceLimits{
				CPU:     "2",
				Memory:  "4Gi",
				Storage: "20Gi",
			},
			ResourceBounds: v1alpha1.ResourceBounds{
				Min: v1alpha1.ResourceLimits{
					CPU:     "0.5",
					Memory:  "1Gi",
					Storage: "10Gi",
				},
				Max: v1alpha1.ResourceLimits{
					CPU:     "8",
					Memory:  "16Gi",
					Storage: "100Gi",
				},
			},
		},
		PostgresCluster: v1alpha1.ClusterApplicationTypeConfig{
			Enabled: false, // Can be enabled if needed
		},
		DockerImage: v1alpha1.ApplicationTypeConfig{
			Enabled: true,
			DefaultLimits: v1alpha1.ResourceLimits{
				CPU:     "1",
				Memory:  "2Gi",
				Storage: "10Gi",
			},
			ResourceBounds: v1alpha1.ResourceBounds{
				Min: v1alpha1.ResourceLimits{
					CPU:     "0.1",
					Memory:  "128Mi",
					Storage: "1Gi",
				},
				Max: v1alpha1.ResourceLimits{
					CPU:     "10",
					Memory:  "32Gi",
					Storage: "50Gi",
				},
			},
		},
		GitRepository: v1alpha1.ApplicationTypeConfig{
			Enabled: true,
			DefaultLimits: v1alpha1.ResourceLimits{
				CPU:     "0.5",
				Memory:  "1Gi",
				Storage: "5Gi",
			},
			ResourceBounds: v1alpha1.ResourceBounds{
				Min: v1alpha1.ResourceLimits{
					CPU:     "0.1",
					Memory:  "256Mi",
					Storage: "1Gi",
				},
				Max: v1alpha1.ResourceLimits{
					CPU:     "2",
					Memory:  "4Gi",
					Storage: "20Gi",
				},
			},
		},
		ImageFromRegistry: v1alpha1.ApplicationTypeConfig{
			Enabled: true,
			DefaultLimits: v1alpha1.ResourceLimits{
				CPU:     "1",
				Memory:  "2Gi",
				Storage: "10Gi",
			},
			ResourceBounds: v1alpha1.ResourceBounds{
				Min: v1alpha1.ResourceLimits{
					CPU:     "0.1",
					Memory:  "128Mi",
					Storage: "1Gi",
				},
				Max: v1alpha1.ResourceLimits{
					CPU:     "10",
					Memory:  "32Gi",
					Storage: "50Gi",
				},
			},
		},
	}
}

// getCustomTemplate converts custom resource limits to CRD format
func getCustomTemplate(customLimits *models.CustomResourceLimits) v1alpha1.ApplicationTypesConfig {
	config := getDevelopmentTemplate() // Start with development defaults

	// Override with custom limits if provided
	if customLimits.MySQL != nil {
		config.MySQL = convertToApplicationTypeConfig(customLimits.MySQL, config.MySQL)
	}
	if customLimits.Postgres != nil {
		config.Postgres = convertToApplicationTypeConfig(customLimits.Postgres, config.Postgres)
	}
	if customLimits.DockerImage != nil {
		config.DockerImage = convertToApplicationTypeConfig(customLimits.DockerImage, config.DockerImage)
	}
	if customLimits.GitRepository != nil {
		config.GitRepository = convertToApplicationTypeConfig(customLimits.GitRepository, config.GitRepository)
	}
	if customLimits.ImageFromRegistry != nil {
		config.ImageFromRegistry = convertToApplicationTypeConfig(customLimits.ImageFromRegistry, config.ImageFromRegistry)
	}

	return config
}

// convertToApplicationTypeConfig converts API model to CRD format
func convertToApplicationTypeConfig(
	custom *models.ApplicationTypeResourceConfig,
	defaultConfig v1alpha1.ApplicationTypeConfig) v1alpha1.ApplicationTypeConfig {

	result := defaultConfig

	// Override default limits if provided
	if custom.DefaultLimits.CPU != "" {
		result.DefaultLimits.CPU = custom.DefaultLimits.CPU
	}
	if custom.DefaultLimits.Memory != "" {
		result.DefaultLimits.Memory = custom.DefaultLimits.Memory
	}
	if custom.DefaultLimits.Storage != "" {
		result.DefaultLimits.Storage = custom.DefaultLimits.Storage
	}

	// Override resource bounds if provided
	if custom.ResourceBounds.MinLimits.CPU != "" {
		result.ResourceBounds.Min.CPU = custom.ResourceBounds.MinLimits.CPU
	}
	if custom.ResourceBounds.MinLimits.Memory != "" {
		result.ResourceBounds.Min.Memory = custom.ResourceBounds.MinLimits.Memory
	}
	if custom.ResourceBounds.MinLimits.Storage != "" {
		result.ResourceBounds.Min.Storage = custom.ResourceBounds.MinLimits.Storage
	}

	if custom.ResourceBounds.MaxLimits.CPU != "" {
		result.ResourceBounds.Max.CPU = custom.ResourceBounds.MaxLimits.CPU
	}
	if custom.ResourceBounds.MaxLimits.Memory != "" {
		result.ResourceBounds.Max.Memory = custom.ResourceBounds.MaxLimits.Memory
	}
	if custom.ResourceBounds.MaxLimits.Storage != "" {
		result.ResourceBounds.Max.Storage = custom.ResourceBounds.MaxLimits.Storage
	}

	return result
}
