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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/models"
	"github.com/kibamail/kibaship/pkg/utils"
	"github.com/kibamail/kibaship/pkg/validation"
)

// DeploymentService handles CRUD operations for deployments
type DeploymentService struct {
	client             client.Client
	scheme             *runtime.Scheme
	applicationService *ApplicationService
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(k8sClient client.Client, scheme *runtime.Scheme, applicationService *ApplicationService) *DeploymentService {
	return &DeploymentService{
		client:             k8sClient,
		scheme:             scheme,
		applicationService: applicationService,
	}
}

// CreateDeployment creates a new deployment
func (s *DeploymentService) CreateDeployment(ctx context.Context, req *models.DeploymentCreateRequest) (*models.Deployment, error) {
	// First, verify the application exists and get its details
	application, err := s.getApplicationByUUID(ctx, req.ApplicationUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	// Generate random slug for deployment
	slug, err := utils.GenerateRandomSlug()
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployment slug: %w", err)
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
			return nil, fmt.Errorf("failed to generate deployment slug: %w", err)
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

	// Create internal deployment model
	deployment := models.NewDeployment(
		application.UUID,
		application.Slug,
		application.ProjectUUID,
		slug,
		req.GitRepository,
	)

	// Set ImageFromRegistry config if present
	if req.ImageFromRegistry != nil {
		deployment.ImageFromRegistry = req.ImageFromRegistry
	}

	// Create Kubernetes Deployment CRD
	crd := s.convertToDeploymentCRD(deployment, application, req.Promote)

	err = s.client.Create(ctx, crd)
	if err != nil {
		return nil, fmt.Errorf("failed to create Deployment CRD: %w", err)
	}

	// Update deployment with CRD information if status is set
	if crd.Status.Phase != "" {
		deployment.Phase = models.DeploymentPhase(crd.Status.Phase)
	}
	// Otherwise keep the initial phase set in NewDeployment

	return deployment, nil
}

// GetDeployment retrieves a deployment by UUID
func (s *DeploymentService) GetDeployment(ctx context.Context, uuid string) (*models.Deployment, error) {
	// List all deployments and find by UUID label
	var deploymentList v1alpha1.DeploymentList
	err := s.client.List(ctx, &deploymentList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	if len(deploymentList.Items) == 0 {
		return nil, fmt.Errorf("deployment with UUID %s not found", uuid)
	}

	if len(deploymentList.Items) > 1 {
		return nil, fmt.Errorf("multiple deployments found with UUID %s", uuid)
	}

	crd := deploymentList.Items[0]

	// Get application to retrieve application slug
	applicationUUID := crd.GetLabels()[validation.LabelApplicationUUID]
	application, err := s.getApplicationByUUID(ctx, applicationUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	// Convert CRD to internal model
	deployment := &models.Deployment{}
	deployment.ConvertFromCRD(&crd, application.Slug)

	return deployment, nil
}

// GetDeploymentsByApplication retrieves all deployments for a specific application
func (s *DeploymentService) GetDeploymentsByApplication(ctx context.Context, applicationUUID string) ([]*models.Deployment, error) {
	// First, verify the application exists and get its details
	application, err := s.getApplicationByUUID(ctx, applicationUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	// List all deployments for this application
	var deploymentList v1alpha1.DeploymentList
	err = s.client.List(ctx, &deploymentList, client.MatchingLabels{
		validation.LabelApplicationUUID: application.UUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	deployments := make([]*models.Deployment, 0, len(deploymentList.Items))
	for _, crd := range deploymentList.Items {
		deployment := &models.Deployment{}
		deployment.ConvertFromCRD(&crd, application.Slug)
		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

// PromoteDeployment promotes a deployment by updating the application's currentDeploymentRef
func (s *DeploymentService) PromoteDeployment(ctx context.Context, deploymentUUID string) error {
	// Get the deployment
	deployment, err := s.GetDeployment(ctx, deploymentUUID)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get the application CRD
	var applicationList v1alpha1.ApplicationList
	err = s.client.List(ctx, &applicationList, client.MatchingLabels{
		validation.LabelResourceUUID: deployment.ApplicationUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	if len(applicationList.Items) == 0 {
		return fmt.Errorf("application with UUID %s not found", deployment.ApplicationUUID)
	}

	if len(applicationList.Items) > 1 {
		return fmt.Errorf("multiple applications found with UUID %s", deployment.ApplicationUUID)
	}

	application := &applicationList.Items[0]

	// Check if already promoted
	if application.Spec.CurrentDeploymentRef != nil &&
		application.Spec.CurrentDeploymentRef.Name == fmt.Sprintf("deployment-%s", deploymentUUID) {
		return nil // Already promoted
	}

	// Update the currentDeploymentRef
	application.Spec.CurrentDeploymentRef = &corev1.LocalObjectReference{
		Name: fmt.Sprintf("deployment-%s", deploymentUUID),
	}

	if err := s.client.Update(ctx, application); err != nil {
		return fmt.Errorf("failed to update application currentDeploymentRef: %w", err)
	}

	return nil
}

// GetLatestDeploymentByApplicationUUID retrieves the most recent deployment for an application by UUID
func (s *DeploymentService) GetLatestDeploymentByApplicationUUID(ctx context.Context, applicationUUID string) (*models.Deployment, error) {
	// List all deployments for this application UUID
	var deploymentList v1alpha1.DeploymentList
	err := s.client.List(ctx, &deploymentList, client.MatchingLabels{
		validation.LabelApplicationUUID: applicationUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	// If no deployments found, return nil (not an error - applications may not have deployments yet)
	if len(deploymentList.Items) == 0 {
		return nil, nil
	}

	// Find the most recent deployment by CreationTimestamp
	var latestCRD *v1alpha1.Deployment
	for i := range deploymentList.Items {
		crd := &deploymentList.Items[i]
		if latestCRD == nil || crd.CreationTimestamp.After(latestCRD.CreationTimestamp.Time) {
			latestCRD = crd
		}
	}

	// Get application details for conversion
	application, err := s.getApplicationByUUID(ctx, applicationUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	// Convert CRD to internal model
	deployment := &models.Deployment{}
	deployment.ConvertFromCRD(latestCRD, application.Slug)

	return deployment, nil
}

// slugExists checks if a deployment with the given slug already exists
func (s *DeploymentService) slugExists(ctx context.Context, slug string) (bool, error) {
	var deploymentList v1alpha1.DeploymentList
	err := s.client.List(ctx, &deploymentList, client.MatchingLabels{
		validation.LabelResourceSlug: slug,
	})
	if err != nil {
		return false, err
	}
	return len(deploymentList.Items) > 0, nil
}

// getApplicationByUUID retrieves an application by its UUID
func (s *DeploymentService) getApplicationByUUID(ctx context.Context, uuid string) (*models.Application, error) {
	var applicationList v1alpha1.ApplicationList
	err := s.client.List(ctx, &applicationList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	if len(applicationList.Items) == 0 {
		return nil, fmt.Errorf("application with UUID %s not found", uuid)
	}

	if len(applicationList.Items) > 1 {
		return nil, fmt.Errorf("multiple applications found with UUID %s", uuid)
	}

	crd := applicationList.Items[0]

	// Convert CRD to internal model
	application := &models.Application{}
	application.ConvertFromCRD(&crd)

	return application, nil
}

// convertToDeploymentCRD converts internal deployment model to Kubernetes Deployment CRD
func (s *DeploymentService) convertToDeploymentCRD(deployment *models.Deployment, application *models.Application, promote bool) *v1alpha1.Deployment {
	crd := &v1alpha1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.operator.kibaship.com/v1alpha1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("deployment-%s", deployment.UUID),
			Namespace: "default",
			Labels: map[string]string{
				validation.LabelResourceUUID:    deployment.UUID,
				validation.LabelResourceSlug:    deployment.Slug,
				validation.LabelProjectUUID:     deployment.ProjectUUID,
				validation.LabelApplicationUUID: deployment.ApplicationUUID,
				validation.LabelEnvironmentUUID: application.EnvironmentUUID,
			},
			Annotations: map[string]string{
				validation.AnnotationResourceName: fmt.Sprintf("Deployment for %s", application.Name),
			},
		},
		Spec: v1alpha1.DeploymentSpec{
			ApplicationRef: corev1.LocalObjectReference{
				Name: fmt.Sprintf("application-%s", deployment.ApplicationUUID),
			},
			Promote: promote,
		},
	}

	// Add GitRepository config if present
	if deployment.GitRepository != nil {
		crd.Spec.GitRepository = &v1alpha1.GitRepositoryDeploymentConfig{
			CommitSHA: deployment.GitRepository.CommitSHA,
			Branch:    deployment.GitRepository.Branch,
		}
	}

	// Add ImageFromRegistry config if present
	if deployment.ImageFromRegistry != nil {
		crd.Spec.ImageFromRegistry = &v1alpha1.ImageFromRegistryDeploymentConfig{
			Tag: deployment.ImageFromRegistry.Tag,
		}

		// Note: Environment variables are now stored in deployment secrets (deployment-{uuid})
		// and are not part of the deployment spec anymore

		// Convert resource requirements
		if deployment.ImageFromRegistry.Resources != nil {
			crd.Spec.ImageFromRegistry.Resources = deployment.ImageFromRegistry.Resources.DeepCopy()
		}
	}

	return crd
}
