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

package services

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/models"
	"github.com/kibamail/kibaship/pkg/templates"
	"github.com/kibamail/kibaship/pkg/utils"
	"github.com/kibamail/kibaship/pkg/validation"
)

// ProjectService handles Project CRD operations
type ProjectService struct {
	client client.Client
	scheme *runtime.Scheme
}

// NewProjectService creates a new project service
func NewProjectService(k8sClient client.Client, scheme *runtime.Scheme) *ProjectService {
	return &ProjectService{
		client: k8sClient,
		scheme: scheme,
	}
}

// CreateProject creates a new Project CRD in Kubernetes
func (s *ProjectService) CreateProject(ctx context.Context, req *models.ProjectCreateRequest) (*models.Project, error) {
	// Generate random slug
	slug, err := utils.GenerateRandomSlug()
	if err != nil {
		return nil, fmt.Errorf("failed to generate project slug: %w", err)
	}

	// Check if slug already exists (very unlikely but possible)
	exists, err := s.slugExists(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check slug uniqueness: %w", err)
	}

	// If slug exists, try generating a new one (up to 3 attempts)
	attempts := 0
	for exists && attempts < 3 {
		slug, err = utils.GenerateRandomSlug()
		if err != nil {
			return nil, fmt.Errorf("failed to generate project slug: %w", err)
		}
		exists, err = s.slugExists(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("failed to check slug uniqueness: %w", err)
		}
		attempts++
	}

	if exists {
		return nil, fmt.Errorf("failed to generate unique slug after 3 attempts")
	}

	// Create internal project model
	project := models.NewProject(
		req.Name,
		req.Description,
		req.WorkspaceUUID,
		slug,
		req.EnabledApplicationTypes,
		req.ResourceProfile,
		req.VolumeSettings,
	)

	// Create Kubernetes Project CRD
	crd := s.convertToProjectCRD(project, req)

	err = s.client.Create(ctx, crd)
	if err != nil {
		return nil, fmt.Errorf("failed to create Project CRD: %w", err)
	}

	// Update project with CRD information
	project.Status = "Pending" // Will be updated by the operator

	return project, nil
}

// GetProject retrieves a project by UUID
func (s *ProjectService) GetProject(ctx context.Context, uuid string) (*models.Project, error) {
	// List all projects and find by UUID label
	var projectList v1alpha1.ProjectList
	err := s.client.List(ctx, &projectList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projectList.Items) == 0 {
		return nil, fmt.Errorf("project with UUID %s not found", uuid)
	}

	if len(projectList.Items) > 1 {
		return nil, fmt.Errorf("multiple projects found with UUID %s", uuid)
	}

	project := s.convertFromProjectCRD(&projectList.Items[0])
	return project, nil
}

// DeleteProject deletes a project by UUID
func (s *ProjectService) DeleteProject(ctx context.Context, uuid string) error {
	// First check if project exists
	var projectList v1alpha1.ProjectList
	err := s.client.List(ctx, &projectList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projectList.Items) == 0 {
		return fmt.Errorf("project with UUID %s not found", uuid)
	}

	if len(projectList.Items) > 1 {
		return fmt.Errorf("multiple projects found with UUID %s", uuid)
	}

	// Delete the project CRD
	project := &projectList.Items[0]
	err = s.client.Delete(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to delete Project CRD: %w", err)
	}

	return nil
}

// UpdateProject updates a project by UUID with partial updates (PATCH)
func (s *ProjectService) UpdateProject(ctx context.Context, uuid string, req *models.ProjectUpdateRequest) (*models.Project, error) {
	// First get the existing project
	var projectList v1alpha1.ProjectList
	err := s.client.List(ctx, &projectList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projectList.Items) == 0 {
		return nil, fmt.Errorf("project with UUID %s not found", uuid)
	}

	if len(projectList.Items) > 1 {
		return nil, fmt.Errorf("multiple projects found with UUID %s", uuid)
	}

	// Get the existing CRD
	existingCRD := &projectList.Items[0]

	// Apply updates to annotations and spec
	s.applyProjectUpdates(existingCRD, req)

	// Update the CRD in Kubernetes with a simple conflict retry loop
	var lastErr error
	for i := 0; i < 3; i++ {
		if err = s.client.Update(ctx, existingCRD); err == nil {
			break
		}
		if apierrors.IsConflict(err) {
			// Fetch latest, re-apply updates, and retry
			var latest v1alpha1.Project
			if getErr := s.client.Get(ctx, client.ObjectKey{Name: existingCRD.Name}, &latest); getErr != nil {
				lastErr = fmt.Errorf("failed to refetch Project for conflict resolution: %w", getErr)
				break
			}
			existingCRD = latest.DeepCopy()
			s.applyProjectUpdates(existingCRD, req)
			lastErr = err
			continue
		}
		lastErr = err
		break
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update Project CRD: %w", lastErr)
	}

	// Convert back to internal model and return
	updatedProject := s.convertFromProjectCRD(existingCRD)
	return updatedProject, nil
}

// applyProjectUpdates applies patch updates to the existing CRD
func (s *ProjectService) applyProjectUpdates(crd *v1alpha1.Project, req *models.ProjectUpdateRequest) {
	annotations := crd.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Update name in annotations
	if req.Name != nil {
		annotations[validation.AnnotationResourceName] = *req.Name
		crd.SetAnnotations(annotations)
	}

	// Update description in annotations
	if req.Description != nil {
		annotations[validation.AnnotationResourceDescription] = *req.Description
		crd.SetAnnotations(annotations)
	}

	// Update resource profile and regenerate spec if needed
	if req.ResourceProfile != nil || req.CustomResourceLimits != nil {
		var profile models.ResourceProfile
		if req.ResourceProfile != nil {
			profile = *req.ResourceProfile
		} else {
			// Determine current profile from existing spec (simplified)
			profile = s.determineCurrentResourceProfile(&crd.Spec.ApplicationTypes)
		}

		// Get new application types config
		applicationTypesConfig := templates.GetResourceProfileTemplate(profile, req.CustomResourceLimits)

		// Apply enablement settings if provided
		if req.EnabledApplicationTypes != nil {
			s.applyApplicationTypeEnablement(&applicationTypesConfig, req.EnabledApplicationTypes)
		} else {
			// Preserve existing enablement settings
			existingEnablement := s.extractApplicationTypeSettings(&crd.Spec.ApplicationTypes)
			s.applyApplicationTypeEnablement(&applicationTypesConfig, &existingEnablement)
		}

		crd.Spec.ApplicationTypes = applicationTypesConfig
	} else if req.EnabledApplicationTypes != nil {
		// Only update enablement settings
		s.applyApplicationTypeEnablement(&crd.Spec.ApplicationTypes, req.EnabledApplicationTypes)
	}

	// Update volume settings
	if req.VolumeSettings != nil && req.VolumeSettings.MaxStorageSize != "" {
		crd.Spec.Volumes.MaxStorageSize = req.VolumeSettings.MaxStorageSize
	}
}

// determineCurrentResourceProfile determines the resource profile from the current spec
func (s *ProjectService) determineCurrentResourceProfile(appTypes *v1alpha1.ApplicationTypesConfig) models.ResourceProfile {
	// Check MySQL default limits to determine profile
	mysqlCPU := appTypes.MySQL.DefaultLimits.CPU
	mysqlMemory := appTypes.MySQL.DefaultLimits.Memory

	// Production profile has higher limits: 2 CPU, 4Gi memory
	if mysqlCPU == "2" && mysqlMemory == "4Gi" {
		return models.ResourceProfileProduction
	}

	// Development profile has lower limits: 500m CPU, 1Gi memory
	if mysqlCPU == "500m" && mysqlMemory == "1Gi" {
		return models.ResourceProfileDevelopment
	}

	// If it doesn't match standard profiles, it's likely custom
	return models.ResourceProfileCustom
}

// slugExists checks if a project with the given slug already exists
func (s *ProjectService) slugExists(ctx context.Context, slug string) (bool, error) {
	var projectList v1alpha1.ProjectList
	err := s.client.List(ctx, &projectList, client.MatchingLabels{
		validation.LabelResourceSlug: slug,
	})
	if err != nil {
		return false, err
	}
	return len(projectList.Items) > 0, nil
}

// convertToProjectCRD converts internal project model to Kubernetes Project CRD
func (s *ProjectService) convertToProjectCRD(project *models.Project, req *models.ProjectCreateRequest) *v1alpha1.Project {
	// Get resource profile template
	profile := models.ResourceProfileDevelopment
	if req.ResourceProfile != nil {
		profile = *req.ResourceProfile
	}

	applicationTypesConfig := templates.GetResourceProfileTemplate(profile, req.CustomResourceLimits)

	// Apply enablement settings
	if req.EnabledApplicationTypes != nil {
		s.applyApplicationTypeEnablement(&applicationTypesConfig, req.EnabledApplicationTypes)
	}

	// Get volume settings
	volumeConfig := v1alpha1.VolumeConfig{}
	if req.VolumeSettings != nil && req.VolumeSettings.MaxStorageSize != "" {
		volumeConfig.MaxStorageSize = req.VolumeSettings.MaxStorageSize
	} else {
		// Use default from profile
		switch profile {
		case models.ResourceProfileProduction:
			volumeConfig.MaxStorageSize = "500Gi"
		case models.ResourceProfileCustom:
			volumeConfig.MaxStorageSize = "100Gi"
		default:
			volumeConfig.MaxStorageSize = "50Gi"
		}
	}

	return &v1alpha1.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.operator.kibaship.com/v1alpha1",
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: utils.GetProjectResourceName(project.UUID),
			Labels: map[string]string{
				validation.LabelResourceUUID:  project.UUID,
				validation.LabelResourceSlug:  project.Slug,
				validation.LabelWorkspaceUUID: project.WorkspaceUUID,
			},
			Annotations: map[string]string{
				validation.AnnotationResourceName:        project.Name,
				validation.AnnotationResourceDescription: project.Description,
			},
		},
		Spec: v1alpha1.ProjectSpec{
			ApplicationTypes: applicationTypesConfig,
			Volumes:          volumeConfig,
		},
	}
}

// convertFromProjectCRD converts Kubernetes Project CRD to internal project model
func (s *ProjectService) convertFromProjectCRD(crd *v1alpha1.Project) *models.Project {
	labels := crd.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	annotations := crd.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Extract application type settings
	appTypes := s.extractApplicationTypeSettings(&crd.Spec.ApplicationTypes)

	// Determine resource profile from spec
	resourceProfile := s.determineCurrentResourceProfile(&crd.Spec.ApplicationTypes)

	return &models.Project{
		UUID:                    labels[validation.LabelResourceUUID],
		Name:                    annotations[validation.AnnotationResourceName],
		Slug:                    labels[validation.LabelResourceSlug],
		Description:             annotations[validation.AnnotationResourceDescription],
		WorkspaceUUID:           labels[validation.LabelWorkspaceUUID],
		EnabledApplicationTypes: appTypes,
		ResourceProfile:         resourceProfile,
		VolumeSettings: models.VolumeSettings{
			MaxStorageSize: crd.Spec.Volumes.MaxStorageSize,
		},
		Status:        crd.Status.Phase,
		NamespaceName: crd.Status.NamespaceName,
		CreatedAt:     crd.CreationTimestamp.Time,
		UpdatedAt:     crd.CreationTimestamp.Time, // Would need to track updates
	}
}

// applyApplicationTypeEnablement applies user-specified enablement settings
func (s *ProjectService) applyApplicationTypeEnablement(config *v1alpha1.ApplicationTypesConfig, settings *models.ApplicationTypeSettings) {
	if settings.MySQL != nil {
		config.MySQL.Enabled = *settings.MySQL
	}
	if settings.MySQLCluster != nil {
		config.MySQLCluster.Enabled = *settings.MySQLCluster
	}
	if settings.Postgres != nil {
		config.Postgres.Enabled = *settings.Postgres
	}
	if settings.PostgresCluster != nil {
		config.PostgresCluster.Enabled = *settings.PostgresCluster
	}
	if settings.DockerImage != nil {
		config.DockerImage.Enabled = *settings.DockerImage
	}
	if settings.GitRepository != nil {
		config.GitRepository.Enabled = *settings.GitRepository
	}
	if settings.ImageFromRegistry != nil {
		config.ImageFromRegistry.Enabled = *settings.ImageFromRegistry
	}
}

// extractApplicationTypeSettings extracts enablement settings from CRD
func (s *ProjectService) extractApplicationTypeSettings(config *v1alpha1.ApplicationTypesConfig) models.ApplicationTypeSettings {
	return models.ApplicationTypeSettings{
		MySQL:             &config.MySQL.Enabled,
		MySQLCluster:      &config.MySQLCluster.Enabled,
		Postgres:          &config.Postgres.Enabled,
		PostgresCluster:   &config.PostgresCluster.Enabled,
		DockerImage:       &config.DockerImage.Enabled,
		GitRepository:     &config.GitRepository.Enabled,
		ImageFromRegistry: &config.ImageFromRegistry.Enabled,
	}
}
