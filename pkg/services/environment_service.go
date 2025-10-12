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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/models"
	"github.com/kibamail/kibaship/pkg/utils"
	"github.com/kibamail/kibaship/pkg/validation"
)

// EnvironmentService handles CRUD operations for environments
type EnvironmentService struct {
	client         client.Client
	scheme         *runtime.Scheme
	projectService *ProjectService
}

// NewEnvironmentService creates a new EnvironmentService
func NewEnvironmentService(k8sClient client.Client, scheme *runtime.Scheme, projectService *ProjectService) *EnvironmentService {
	return &EnvironmentService{
		client:         k8sClient,
		scheme:         scheme,
		projectService: projectService,
	}
}

// CreateEnvironment creates a new environment
func (s *EnvironmentService) CreateEnvironment(ctx context.Context, req *models.EnvironmentCreateRequest) (*models.Environment, error) {
	// First, verify the project exists and get its details
	project, err := s.projectService.GetProject(ctx, req.ProjectUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Generate random slug
	slug, err := utils.GenerateRandomSlug()
	if err != nil {
		return nil, fmt.Errorf("failed to generate environment slug: %w", err)
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
			return nil, fmt.Errorf("failed to generate environment slug: %w", err)
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

	// Create internal environment model
	environment := models.NewEnvironment(
		req.Name,
		project.UUID,
		project.Slug,
		slug,
	)

	// Set optional fields
	if req.Description != "" {
		environment.Description = req.Description
	}
	if req.Variables != nil {
		environment.Variables = req.Variables
	}

	// Create Kubernetes Environment CRD
	crd := s.convertToEnvironmentCRD(environment)

	err = s.client.Create(ctx, crd)
	if err != nil {
		return nil, fmt.Errorf("failed to create Environment CRD: %w", err)
	}

	return environment, nil
}

// GetEnvironment retrieves an environment by UUID
func (s *EnvironmentService) GetEnvironment(ctx context.Context, uuid string) (*models.Environment, error) {
	// List all environments and find by UUID label
	var environmentList v1alpha1.EnvironmentList
	err := s.client.List(ctx, &environmentList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	if len(environmentList.Items) == 0 {
		return nil, fmt.Errorf("environment with UUID %s not found", uuid)
	}

	if len(environmentList.Items) > 1 {
		return nil, fmt.Errorf("multiple environments found with UUID %s", uuid)
	}

	environment := s.convertFromEnvironmentCRD(&environmentList.Items[0])

	// Count applications in this environment
	var applicationList v1alpha1.ApplicationList
	err = s.client.List(ctx, &applicationList, client.MatchingLabels{
		validation.LabelEnvironmentUUID: environment.UUID,
	})
	if err == nil {
		environment.ApplicationCount = int32(len(applicationList.Items))
	}

	return environment, nil
}

// GetEnvironmentsByProject retrieves all environments for a project
func (s *EnvironmentService) GetEnvironmentsByProject(ctx context.Context, projectUUID string) ([]*models.Environment, error) {
	// List all environments for this project UUID
	var environmentList v1alpha1.EnvironmentList
	err := s.client.List(ctx, &environmentList, client.MatchingLabels{
		validation.LabelProjectUUID: projectUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	environments := make([]*models.Environment, 0, len(environmentList.Items))
	for _, item := range environmentList.Items {
		env := s.convertFromEnvironmentCRD(&item)

		// Count applications for each environment
		var applicationList v1alpha1.ApplicationList
		err = s.client.List(ctx, &applicationList, client.MatchingLabels{
			validation.LabelEnvironmentUUID: env.UUID,
		})
		if err == nil {
			env.ApplicationCount = int32(len(applicationList.Items))
		}

		environments = append(environments, env)
	}

	return environments, nil
}

// UpdateEnvironment updates an environment by UUID with partial updates (PATCH)
func (s *EnvironmentService) UpdateEnvironment(ctx context.Context, uuid string, req *models.EnvironmentUpdateRequest) (*models.Environment, error) {
	// First get the existing environment
	var environmentList v1alpha1.EnvironmentList
	err := s.client.List(ctx, &environmentList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	if len(environmentList.Items) == 0 {
		return nil, fmt.Errorf("environment with UUID %s not found", uuid)
	}

	if len(environmentList.Items) > 1 {
		return nil, fmt.Errorf("multiple environments found with UUID %s", uuid)
	}

	// Get the existing CRD
	existingCRD := &environmentList.Items[0]

	// Apply updates to annotations and spec
	s.applyEnvironmentUpdates(existingCRD, req)

	// Update the CRD in Kubernetes with a simple conflict retry loop
	var lastErr error
	for i := 0; i < 3; i++ {
		if err = s.client.Update(ctx, existingCRD); err == nil {
			break
		}
		if apierrors.IsConflict(err) {
			// Fetch latest, re-apply updates, and retry
			var latest v1alpha1.Environment
			if getErr := s.client.Get(ctx, client.ObjectKey{Namespace: existingCRD.Namespace, Name: existingCRD.Name}, &latest); getErr != nil {
				lastErr = fmt.Errorf("failed to refetch Environment for conflict resolution: %w", getErr)
				break
			}
			existingCRD = latest.DeepCopy()
			s.applyEnvironmentUpdates(existingCRD, req)
			lastErr = err
			continue
		}
		lastErr = err
		break
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update Environment CRD: %w", lastErr)
	}

	// Convert back to internal model and return
	updatedEnvironment := s.convertFromEnvironmentCRD(existingCRD)

	// Count applications
	var applicationList v1alpha1.ApplicationList
	err = s.client.List(ctx, &applicationList, client.MatchingLabels{
		validation.LabelEnvironmentUUID: updatedEnvironment.UUID,
	})
	if err == nil {
		updatedEnvironment.ApplicationCount = int32(len(applicationList.Items))
	}

	return updatedEnvironment, nil
}

// DeleteEnvironment deletes an environment by UUID
func (s *EnvironmentService) DeleteEnvironment(ctx context.Context, uuid string) error {
	// First check if environment exists
	var environmentList v1alpha1.EnvironmentList
	err := s.client.List(ctx, &environmentList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	if len(environmentList.Items) == 0 {
		return fmt.Errorf("environment with UUID %s not found", uuid)
	}

	if len(environmentList.Items) > 1 {
		return fmt.Errorf("multiple environments found with UUID %s", uuid)
	}

	// Delete the environment CRD
	environment := &environmentList.Items[0]
	err = s.client.Delete(ctx, environment)
	if err != nil {
		return fmt.Errorf("failed to delete Environment CRD: %w", err)
	}

	return nil
}

// Helper methods

// slugExists checks if an environment with the given slug already exists
func (s *EnvironmentService) slugExists(ctx context.Context, slug string) (bool, error) {
	var environmentList v1alpha1.EnvironmentList
	err := s.client.List(ctx, &environmentList, client.MatchingLabels{
		validation.LabelResourceSlug: slug,
	})
	if err != nil {
		return false, err
	}
	return len(environmentList.Items) > 0, nil
}

// convertToEnvironmentCRD converts internal environment model to Kubernetes Environment CRD
func (s *EnvironmentService) convertToEnvironmentCRD(env *models.Environment) *v1alpha1.Environment {
	crd := &v1alpha1.Environment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.operator.kibaship.com/v1alpha1",
			Kind:       "Environment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("environment-%s", env.UUID),
			Namespace: "default",
			Labels: map[string]string{
				validation.LabelResourceUUID: env.UUID,
				validation.LabelResourceSlug: env.Slug,
				validation.LabelProjectUUID:  env.ProjectUUID,
			},
			Annotations: map[string]string{
				validation.AnnotationResourceName: env.Name,
			},
		},
		Spec: v1alpha1.EnvironmentSpec{
			ProjectRef: corev1.LocalObjectReference{
				Name: fmt.Sprintf("project-%s", env.ProjectUUID),
			},
		},
	}

	// Add optional description
	if env.Description != "" {
		crd.Annotations[validation.AnnotationResourceDescription] = env.Description
	}

	// Note: Variables are no longer stored on Environment CRD
	// They should be managed at the Application level via secrets

	return crd
}

// convertFromEnvironmentCRD converts Kubernetes Environment CRD to internal environment model
func (s *EnvironmentService) convertFromEnvironmentCRD(crd *v1alpha1.Environment) *models.Environment {
	labels := crd.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	annotations := crd.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	return &models.Environment{
		UUID:        labels[validation.LabelResourceUUID],
		Name:        annotations[validation.AnnotationResourceName],
		Slug:        labels[validation.LabelResourceSlug],
		Description: annotations[validation.AnnotationResourceDescription],
		ProjectUUID: labels[validation.LabelProjectUUID],
		ProjectSlug: s.extractProjectSlugFromRef(crd.Spec.ProjectRef.Name),
		CreatedAt:   crd.CreationTimestamp.Time,
		UpdatedAt:   crd.CreationTimestamp.Time, // Would need to track updates
	}
}

// applyEnvironmentUpdates applies patch updates to the existing CRD
func (s *EnvironmentService) applyEnvironmentUpdates(crd *v1alpha1.Environment, req *models.EnvironmentUpdateRequest) {
	annotations := crd.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Update description in annotations
	if req.Description != nil {
		annotations[validation.AnnotationResourceDescription] = *req.Description
		crd.SetAnnotations(annotations)
	}

	// Note: Variables are no longer stored on Environment CRD
	// They should be managed at the Application level via secrets
}

// extractProjectSlugFromRef extracts the project slug from the ProjectRef name
// Name format: "project-{slug}-kibaship-com"
func (s *EnvironmentService) extractProjectSlugFromRef(refName string) string {
	// Remove "project-" prefix and "-kibaship-com" suffix
	if len(refName) > len("project-") && len(refName) > len("-kibaship-com") {
		slug := refName[len("project-"):]
		if len(slug) > len("-kibaship-com") {
			slug = slug[:len(slug)-len("-kibaship-com")]
			return slug
		}
	}
	return ""
}
