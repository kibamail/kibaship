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

// ApplicationDomainService handles CRUD operations for application domains
type ApplicationDomainService struct {
	client             client.Client
	scheme             *runtime.Scheme
	applicationService *ApplicationService
}

// NewApplicationDomainService creates a new application domain service
func NewApplicationDomainService(k8sClient client.Client, scheme *runtime.Scheme, applicationService *ApplicationService) *ApplicationDomainService {
	return &ApplicationDomainService{
		client:             k8sClient,
		scheme:             scheme,
		applicationService: applicationService,
	}
}

// CreateApplicationDomain creates a new application domain
func (s *ApplicationDomainService) CreateApplicationDomain(ctx context.Context, req *models.ApplicationDomainCreateRequest) (*models.ApplicationDomain, error) {
	// First, verify the application exists and get its details
	application, err := s.applicationService.GetApplication(ctx, req.ApplicationSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	// Generate random slug for application domain
	slug, err := utils.GenerateRandomSlug()
	if err != nil {
		return nil, fmt.Errorf("failed to generate application domain slug: %w", err)
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
			return nil, fmt.Errorf("failed to generate application domain slug: %w", err)
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

	// Set default values
	domainType := req.Type
	if domainType == "" {
		domainType = models.ApplicationDomainTypeDefault
	}

	// Create internal application domain model
	applicationDomain := models.NewApplicationDomain(
		application.UUID,
		application.Slug,
		application.ProjectUUID,
		slug,
		req.Domain,
		req.Port,
		domainType,
		req.Default,
		req.TLSEnabled,
	)

	// Create Kubernetes ApplicationDomain CRD
	crd := s.convertToApplicationDomainCRD(applicationDomain, application)

	err = s.client.Create(ctx, crd)
	if err != nil {
		return nil, fmt.Errorf("failed to create ApplicationDomain CRD: %w", err)
	}

	// Update application domain with CRD information
	applicationDomain.Phase = models.ApplicationDomainPhase(crd.Status.Phase)

	return applicationDomain, nil
}

// GetApplicationDomain retrieves an application domain by UUID
func (s *ApplicationDomainService) GetApplicationDomain(ctx context.Context, uuid string) (*models.ApplicationDomain, error) {
	// List all application domains and find by UUID label only
	var domainList v1alpha1.ApplicationDomainList
	err := s.client.List(ctx, &domainList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list application domains: %w", err)
	}

	if len(domainList.Items) == 0 {
		return nil, fmt.Errorf("application domain with UUID %s not found", uuid)
	}

	if len(domainList.Items) > 1 {
		return nil, fmt.Errorf("multiple application domains found with UUID %s", uuid)
	}

	crd := domainList.Items[0]

	// Get application to retrieve application slug
	applicationUUID := crd.GetLabels()[validation.LabelApplicationUUID]
	application, err := s.getApplicationByUUID(ctx, applicationUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	// Convert CRD to internal model
	applicationDomain := &models.ApplicationDomain{}
	applicationDomain.ConvertFromCRD(&crd, application.Slug)

	return applicationDomain, nil
}

// GetApplicationDomainsByApplication retrieves all application domains for a specific application
func (s *ApplicationDomainService) GetApplicationDomainsByApplication(ctx context.Context, applicationSlug string) ([]*models.ApplicationDomain, error) {
	// First, verify the application exists and get its details
	application, err := s.applicationService.GetApplication(ctx, applicationSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	// List all application domains for this application
	var domainList v1alpha1.ApplicationDomainList
	err = s.client.List(ctx, &domainList, client.MatchingLabels{
		validation.LabelApplicationUUID: application.UUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list application domains: %w", err)
	}

	applicationDomains := make([]*models.ApplicationDomain, 0, len(domainList.Items))
	for _, crd := range domainList.Items {
		applicationDomain := &models.ApplicationDomain{}
		applicationDomain.ConvertFromCRD(&crd, application.Slug)
		applicationDomains = append(applicationDomains, applicationDomain)
	}

	return applicationDomains, nil
}

// GetApplicationDomainsByApplicationUUIDNoValidate lists domains for an application UUID
// without calling back into ApplicationService (avoids circular dependency). The caller is
// responsible for providing the corresponding applicationSlug used for conversion.
func (s *ApplicationDomainService) GetApplicationDomainsByApplicationUUIDNoValidate(
	ctx context.Context,
	applicationUUID string,
	applicationSlug string,
) ([]*models.ApplicationDomain, error) {
	var domainList v1alpha1.ApplicationDomainList
	if err := s.client.List(ctx, &domainList, client.MatchingLabels{
		validation.LabelApplicationUUID: applicationUUID,
	}); err != nil {
		return nil, fmt.Errorf("failed to list application domains: %w", err)
	}

	applicationDomains := make([]*models.ApplicationDomain, 0, len(domainList.Items))
	for i := range domainList.Items {
		d := &models.ApplicationDomain{}
		d.ConvertFromCRD(&domainList.Items[i], applicationSlug)
		applicationDomains = append(applicationDomains, d)
	}
	return applicationDomains, nil
}

// DeleteApplicationDomain deletes an application domain by UUID only
func (s *ApplicationDomainService) DeleteApplicationDomain(ctx context.Context, uuid string) error {
	// First find the application domain by UUID
	var domainList v1alpha1.ApplicationDomainList
	err := s.client.List(ctx, &domainList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})

	if err != nil {
		return fmt.Errorf("failed to list application domains: %w", err)
	}

	if len(domainList.Items) == 0 {
		return fmt.Errorf("application domain with UUID %s not found", uuid)
	}

	if len(domainList.Items) > 1 {
		return fmt.Errorf("multiple application domains found with UUID %s", uuid)
	}

	// Delete the CRD
	crd := domainList.Items[0]
	err = s.client.Delete(ctx, &crd)
	if err != nil {
		return fmt.Errorf("failed to delete ApplicationDomain CRD: %w", err)
	}

	return nil
}

// slugExists checks if an application domain with the given slug already exists
func (s *ApplicationDomainService) slugExists(ctx context.Context, slug string) (bool, error) {
	var domainList v1alpha1.ApplicationDomainList
	err := s.client.List(ctx, &domainList, client.MatchingLabels{
		validation.LabelResourceSlug: slug,
	})
	if err != nil {
		return false, err
	}
	return len(domainList.Items) > 0, nil
}

// getApplicationByUUID retrieves an application by its UUID
func (s *ApplicationDomainService) getApplicationByUUID(ctx context.Context, uuid string) (*models.Application, error) {
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

// convertToApplicationDomainCRD converts internal application domain model to Kubernetes ApplicationDomain CRD
func (s *ApplicationDomainService) convertToApplicationDomainCRD(applicationDomain *models.ApplicationDomain, application *models.Application) *v1alpha1.ApplicationDomain {
	return &v1alpha1.ApplicationDomain{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.operator.kibaship.com/v1alpha1",
			Kind:       "ApplicationDomain",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.GetApplicationDomainResourceName(applicationDomain.UUID),
			Namespace: "default",
			Labels: map[string]string{
				validation.LabelResourceUUID:    applicationDomain.UUID,
				validation.LabelResourceSlug:    applicationDomain.Slug,
				validation.LabelProjectUUID:     applicationDomain.ProjectUUID,
				validation.LabelApplicationUUID: applicationDomain.ApplicationUUID,
			},
			Annotations: map[string]string{
				validation.AnnotationResourceName: fmt.Sprintf("Domain %s for %s", applicationDomain.Domain, application.Name),
			},
		},
		Spec: v1alpha1.ApplicationDomainSpec{
			ApplicationRef: corev1.LocalObjectReference{
				Name: utils.GetApplicationResourceName(applicationDomain.ApplicationUUID),
			},
			Domain:     applicationDomain.Domain,
			Port:       applicationDomain.Port,
			Type:       v1alpha1.ApplicationDomainType(applicationDomain.Type),
			Default:    applicationDomain.Default,
			TLSEnabled: applicationDomain.TLSEnabled,
		},
	}
}
