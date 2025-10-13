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

// ApplicationService handles CRUD operations for applications
type ApplicationService struct {
	client             client.Client
	scheme             *runtime.Scheme
	projectService     *ProjectService
	environmentService *EnvironmentService
	domainService      *ApplicationDomainService
	deploymentService  *DeploymentService
}

// NewApplicationService creates a new ApplicationService
func NewApplicationService(k8sClient client.Client, scheme *runtime.Scheme, projectService *ProjectService, environmentService *EnvironmentService) *ApplicationService {
	return &ApplicationService{
		client:             k8sClient,
		scheme:             scheme,
		projectService:     projectService,
		environmentService: environmentService,
		domainService:      nil, // Will be set later to avoid circular dependency
		deploymentService:  nil, // Will be set later to avoid circular dependency
	}
}

// SetDomainService sets the domain service dependency (called after both services are created)
func (s *ApplicationService) SetDomainService(domainService *ApplicationDomainService) {
	s.domainService = domainService
}

// SetDeploymentService sets the deployment service dependency (called after both services are created)
func (s *ApplicationService) SetDeploymentService(deploymentService *DeploymentService) {
	s.deploymentService = deploymentService
}

// CreateApplication creates a new application
func (s *ApplicationService) CreateApplication(ctx context.Context, req *models.ApplicationCreateRequest) (*models.Application, error) {
	// First, verify the environment exists and get its details
	environment, err := s.environmentService.GetEnvironment(ctx, req.EnvironmentUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	// Get the project to check application type enablement
	project, err := s.projectService.GetProject(ctx, environment.ProjectUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Check if the application type is enabled for this project
	if !s.isApplicationTypeEnabled(project, req.Type) {
		return nil, fmt.Errorf("application type '%s' is not enabled for project '%s'", req.Type, environment.ProjectSlug)
	}

	// Generate random slug
	slug, err := utils.GenerateRandomSlug()
	if err != nil {
		return nil, fmt.Errorf("failed to generate application slug: %w", err)
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
			return nil, fmt.Errorf("failed to generate application slug: %w", err)
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

	// Create internal application model
	application := models.NewApplication(
		req.Name,
		project.UUID,
		project.Slug,
		req.Type,
		slug,
	)

	// Set type-specific configuration
	s.setApplicationConfiguration(application, req)

	// Create Kubernetes Application CRD
	crd := s.convertToApplicationCRD(application, environment)

	err = s.client.Create(ctx, crd)
	if err != nil {
		return nil, fmt.Errorf("failed to create Application CRD: %w", err)
	}

	// Update application with CRD information
	application.Status = "Pending" // Will be updated by the operator

	return application, nil
}

// GetApplication retrieves an application by UUID with domains auto-loaded
func (s *ApplicationService) GetApplication(ctx context.Context, uuid string) (*models.Application, error) {
	// List all applications and find by UUID label
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

	application := s.convertFromApplicationCRD(&applicationList.Items[0])

	// Auto-load domains if domain service is available, without circular dependency
	if s.domainService != nil {
		domains, err := s.domainService.GetApplicationDomainsByApplicationUUIDNoValidate(ctx, application.UUID, application.Slug)
		if err != nil {
			return nil, fmt.Errorf("failed to load application domains: %w", err)
		}
		application.Domains = domains
	}

	return application, nil
}

// UpdateApplication updates an application by UUID with partial updates (PATCH)
func (s *ApplicationService) UpdateApplication(ctx context.Context, uuid string, req *models.ApplicationUpdateRequest) (*models.Application, error) {
	// First get the existing application
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

	// Get the existing CRD
	existingCRD := &applicationList.Items[0]

	// Apply updates to annotations and spec
	s.applyApplicationUpdates(existingCRD, req)

	// Update the CRD in Kubernetes with a simple conflict retry loop
	var lastErr error
	for i := 0; i < 3; i++ {
		if err = s.client.Update(ctx, existingCRD); err == nil {
			break
		}
		if apierrors.IsConflict(err) {
			// Fetch latest, re-apply updates, and retry
			var latest v1alpha1.Application
			if getErr := s.client.Get(ctx, client.ObjectKey{Namespace: existingCRD.Namespace, Name: existingCRD.Name}, &latest); getErr != nil {
				lastErr = fmt.Errorf("failed to refetch Application for conflict resolution: %w", getErr)
				break
			}
			existingCRD = latest.DeepCopy()
			s.applyApplicationUpdates(existingCRD, req)
			lastErr = err
			continue
		}
		lastErr = err
		break
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update Application CRD: %w", lastErr)
	}

	// Convert back to internal model and return
	updatedApplication := s.convertFromApplicationCRD(existingCRD)
	return updatedApplication, nil
}

// UpdateApplicationEnv updates environment variables for an application
func (s *ApplicationService) UpdateApplicationEnv(ctx context.Context, uuid string, req *models.ApplicationEnvUpdateRequest) error {
	// First get the existing application
	var applicationList v1alpha1.ApplicationList
	err := s.client.List(ctx, &applicationList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})
	if err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	if len(applicationList.Items) == 0 {
		return fmt.Errorf("application with UUID %s not found", uuid)
	}

	if len(applicationList.Items) > 1 {
		return fmt.Errorf("multiple applications found with UUID %s", uuid)
	}

	app := &applicationList.Items[0]

	// Get the env secret ref based on application type
	var secretName string
	switch app.Spec.Type {
	case v1alpha1.ApplicationTypeGitRepository:
		if app.Spec.GitRepository == nil || app.Spec.GitRepository.Env == nil {
			return fmt.Errorf("application does not have an environment variables secret configured")
		}
		secretName = app.Spec.GitRepository.Env.Name
	case v1alpha1.ApplicationTypeDockerImage:
		if app.Spec.DockerImage == nil || app.Spec.DockerImage.Env == nil {
			return fmt.Errorf("application does not have an environment variables secret configured")
		}
		secretName = app.Spec.DockerImage.Env.Name
	case v1alpha1.ApplicationTypeMySQL:
		if app.Spec.MySQL == nil || app.Spec.MySQL.Env == nil {
			return fmt.Errorf("application does not have an environment variables secret configured")
		}
		secretName = app.Spec.MySQL.Env.Name
	case v1alpha1.ApplicationTypeMySQLCluster:
		if app.Spec.MySQLCluster == nil || app.Spec.MySQLCluster.Env == nil {
			return fmt.Errorf("application does not have an environment variables secret configured")
		}
		secretName = app.Spec.MySQLCluster.Env.Name
	case v1alpha1.ApplicationTypePostgres:
		if app.Spec.Postgres == nil || app.Spec.Postgres.Env == nil {
			return fmt.Errorf("application does not have an environment variables secret configured")
		}
		secretName = app.Spec.Postgres.Env.Name
	case v1alpha1.ApplicationTypePostgresCluster:
		if app.Spec.PostgresCluster == nil || app.Spec.PostgresCluster.Env == nil {
			return fmt.Errorf("application does not have an environment variables secret configured")
		}
		secretName = app.Spec.PostgresCluster.Env.Name
	default:
		return fmt.Errorf("unsupported application type: %s", app.Spec.Type)
	}

	// Fetch the secret
	var secret corev1.Secret
	err = s.client.Get(ctx, client.ObjectKey{
		Name:      secretName,
		Namespace: app.Namespace,
	}, &secret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("environment variables secret %s not found", secretName)
		}
		return fmt.Errorf("failed to get secret: %w", err)
	}

	// Merge incoming variables with existing ones
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	for key, value := range req.Variables {
		secret.Data[key] = []byte(value)
	}

	// Update the secret
	err = s.client.Update(ctx, &secret)
	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	return nil
}

// DeleteApplication deletes an application by UUID
func (s *ApplicationService) DeleteApplication(ctx context.Context, uuid string) error {
	// First check if application exists
	var applicationList v1alpha1.ApplicationList
	err := s.client.List(ctx, &applicationList, client.MatchingLabels{
		validation.LabelResourceUUID: uuid,
	})
	if err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	if len(applicationList.Items) == 0 {
		return fmt.Errorf("application with UUID %s not found", uuid)
	}

	if len(applicationList.Items) > 1 {
		return fmt.Errorf("multiple applications found with UUID %s", uuid)
	}

	// Delete the application CRD
	application := &applicationList.Items[0]
	err = s.client.Delete(ctx, application)
	if err != nil {
		return fmt.Errorf("failed to delete Application CRD: %w", err)
	}

	return nil
}

// GetApplicationsByProject retrieves all applications for a project with domains batch-loaded
func (s *ApplicationService) GetApplicationsByProject(ctx context.Context, projectUUID string) ([]*models.Application, error) {
	// First get the project to get its details
	project, err := s.projectService.GetProject(ctx, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// List all applications for this project
	var applicationList v1alpha1.ApplicationList
	err = s.client.List(ctx, &applicationList, client.MatchingLabels{
		validation.LabelProjectUUID: project.UUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	applications := make([]*models.Application, 0, len(applicationList.Items))
	for _, item := range applicationList.Items {
		app := s.convertFromApplicationCRD(&item)
		applications = append(applications, app)
	}

	// Batch-load domains and latest deployments for all applications
	if len(applications) > 0 {
		// Load domains if domain service is available
		if s.domainService != nil {
			err = s.batchLoadDomains(ctx, applications)
			if err != nil {
				return nil, fmt.Errorf("failed to batch load domains: %w", err)
			}
		}

		// Load latest deployments if deployment service is available
		if s.deploymentService != nil {
			err = s.batchLoadLatestDeployments(ctx, applications)
			if err != nil {
				return nil, fmt.Errorf("failed to batch load latest deployments: %w", err)
			}
		}
	}

	return applications, nil
}

// GetApplicationsByEnvironment retrieves all applications for an environment with domains batch-loaded
func (s *ApplicationService) GetApplicationsByEnvironment(ctx context.Context, environmentUUID string) ([]*models.Application, error) {
	// First get the environment to get its details
	environment, err := s.environmentService.GetEnvironment(ctx, environmentUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	// List all applications for this environment
	var applicationList v1alpha1.ApplicationList
	err = s.client.List(ctx, &applicationList, client.MatchingLabels{
		validation.LabelEnvironmentUUID: environment.UUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	applications := make([]*models.Application, 0, len(applicationList.Items))
	for _, item := range applicationList.Items {
		app := s.convertFromApplicationCRD(&item)
		applications = append(applications, app)
	}

	// Batch-load domains and latest deployments for all applications
	if len(applications) > 0 {
		// Load domains if domain service is available
		if s.domainService != nil {
			err = s.batchLoadDomains(ctx, applications)
			if err != nil {
				return nil, fmt.Errorf("failed to batch load domains: %w", err)
			}
		}

		// Load latest deployments if deployment service is available
		if s.deploymentService != nil {
			err = s.batchLoadLatestDeployments(ctx, applications)
			if err != nil {
				return nil, fmt.Errorf("failed to batch load latest deployments: %w", err)
			}
		}
	}

	return applications, nil
}

// batchLoadDomains efficiently loads domains for multiple applications in a single query
func (s *ApplicationService) batchLoadDomains(ctx context.Context, applications []*models.Application) error {
	if len(applications) == 0 {
		return nil
	}

	// Collect all application UUIDs
	applicationUUIDs := make([]string, len(applications))
	applicationMap := make(map[string]*models.Application)
	for i, app := range applications {
		applicationUUIDs[i] = app.UUID
		applicationMap[app.UUID] = app
	}

	// Query all domains for these applications in a single call
	var domainList v1alpha1.ApplicationDomainList
	err := s.client.List(ctx, &domainList, client.MatchingLabels{
		validation.LabelProjectUUID: applications[0].ProjectUUID, // All apps are in same project
	})
	if err != nil {
		return fmt.Errorf("failed to list application domains: %w", err)
	}

	// Group domains by application UUID
	for _, domainCRD := range domainList.Items {
		applicationUUID := domainCRD.GetLabels()[validation.LabelApplicationUUID]
		if app, exists := applicationMap[applicationUUID]; exists {
			// Convert CRD to internal model
			domain := &models.ApplicationDomain{}
			domain.ConvertFromCRD(&domainCRD, app.Slug)

			// Add to application's domains
			if app.Domains == nil {
				app.Domains = []*models.ApplicationDomain{}
			}
			app.Domains = append(app.Domains, domain)
		}
	}

	return nil
}

// batchLoadLatestDeployments efficiently loads the latest deployment for multiple applications
func (s *ApplicationService) batchLoadLatestDeployments(ctx context.Context, applications []*models.Application) error {
	if len(applications) == 0 {
		return nil
	}

	// Create a map to track applications by UUID for easy lookup
	applicationMap := make(map[string]*models.Application)
	for _, app := range applications {
		applicationMap[app.UUID] = app
	}

	// Query all deployments for these applications in a single call
	var deploymentList v1alpha1.DeploymentList
	err := s.client.List(ctx, &deploymentList, client.MatchingLabels{
		validation.LabelProjectUUID: applications[0].ProjectUUID, // All apps are in same project
	})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	// Group deployments by application UUID and find the latest for each
	latestDeployments := make(map[string]*v1alpha1.Deployment)
	for i := range deploymentList.Items {
		deploymentCRD := &deploymentList.Items[i]
		applicationUUID := deploymentCRD.GetLabels()[validation.LabelApplicationUUID]

		// Only process deployments for applications we're interested in
		if _, exists := applicationMap[applicationUUID]; !exists {
			continue
		}

		// Check if this is the latest deployment for this application
		if existing, found := latestDeployments[applicationUUID]; !found ||
			deploymentCRD.CreationTimestamp.After(existing.CreationTimestamp.Time) {
			latestDeployments[applicationUUID] = deploymentCRD
		}
	}

	// Convert latest deployments to internal models and attach to applications
	for applicationUUID, latestCRD := range latestDeployments {
		app := applicationMap[applicationUUID]

		// Convert CRD to internal model
		deployment := &models.Deployment{}
		deployment.ConvertFromCRD(latestCRD, app.Slug)

		// Attach to application
		app.LatestDeployment = deployment
	}

	return nil
}

// Helper methods

// isApplicationTypeEnabled checks if an application type is enabled for a project
func (s *ApplicationService) isApplicationTypeEnabled(project *models.Project, appType models.ApplicationType) bool {
	switch appType {
	case models.ApplicationTypeMySQL:
		return project.EnabledApplicationTypes.MySQL != nil && *project.EnabledApplicationTypes.MySQL
	case models.ApplicationTypeMySQLCluster:
		return project.EnabledApplicationTypes.MySQLCluster != nil && *project.EnabledApplicationTypes.MySQLCluster
	case models.ApplicationTypePostgres:
		return project.EnabledApplicationTypes.Postgres != nil && *project.EnabledApplicationTypes.Postgres
	case models.ApplicationTypePostgresCluster:
		return project.EnabledApplicationTypes.PostgresCluster != nil && *project.EnabledApplicationTypes.PostgresCluster
	case models.ApplicationTypeDockerImage:
		return project.EnabledApplicationTypes.DockerImage != nil && *project.EnabledApplicationTypes.DockerImage
	case models.ApplicationTypeGitRepository:
		return project.EnabledApplicationTypes.GitRepository != nil && *project.EnabledApplicationTypes.GitRepository
	case models.ApplicationTypeImageFromRegistry:
		return project.EnabledApplicationTypes.ImageFromRegistry != nil && *project.EnabledApplicationTypes.ImageFromRegistry
	default:
		return false
	}
}

// setApplicationConfiguration sets type-specific configuration on the application
func (s *ApplicationService) setApplicationConfiguration(app *models.Application, req *models.ApplicationCreateRequest) {
	switch req.Type {
	case models.ApplicationTypeGitRepository:
		app.GitRepository = req.GitRepository
	case models.ApplicationTypeDockerImage:
		app.DockerImage = req.DockerImage
	case models.ApplicationTypeImageFromRegistry:
		app.ImageFromRegistry = req.ImageFromRegistry
	case models.ApplicationTypeMySQL:
		app.MySQL = req.MySQL
	case models.ApplicationTypeMySQLCluster:
		app.MySQLCluster = req.MySQLCluster
	case models.ApplicationTypePostgres:
		app.Postgres = req.Postgres
	case models.ApplicationTypePostgresCluster:
		app.PostgresCluster = req.PostgresCluster
	}
}

// slugExists checks if an application with the given slug already exists
func (s *ApplicationService) slugExists(ctx context.Context, slug string) (bool, error) {
	var applicationList v1alpha1.ApplicationList
	err := s.client.List(ctx, &applicationList, client.MatchingLabels{
		validation.LabelResourceSlug: slug,
	})
	if err != nil {
		return false, err
	}
	return len(applicationList.Items) > 0, nil
}

// convertToApplicationCRD converts internal application model to Kubernetes Application CRD
func (s *ApplicationService) convertToApplicationCRD(app *models.Application, environment *models.Environment) *v1alpha1.Application {
	return &v1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.operator.kibaship.com/v1alpha1",
			Kind:       "Application",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.GetApplicationResourceName(app.UUID),
			Namespace: "default",
			Labels: map[string]string{
				validation.LabelResourceUUID:    app.UUID,
				validation.LabelResourceSlug:    app.Slug,
				validation.LabelProjectUUID:     app.ProjectUUID,
				validation.LabelEnvironmentUUID: environment.UUID,
				validation.LabelApplicationUUID: app.UUID,
			},
			Annotations: map[string]string{
				validation.AnnotationResourceName: app.Name,
			},
		},
		Spec: v1alpha1.ApplicationSpec{
			EnvironmentRef: corev1.LocalObjectReference{
				Name: utils.GetEnvironmentResourceName(environment.UUID),
			},
			Type:            s.convertApplicationType(app.Type),
			GitRepository:   s.convertGitRepositoryConfig(app.GitRepository),
			DockerImage:     s.convertDockerImageConfig(app.DockerImage),
			MySQL:           s.convertMySQLConfig(app.MySQL),
			MySQLCluster:    s.convertMySQLClusterConfig(app.MySQLCluster),
			Postgres:        s.convertPostgresConfig(app.Postgres),
			PostgresCluster: s.convertPostgresClusterConfig(app.PostgresCluster),
		},
	}
}

// convertFromApplicationCRD converts Kubernetes Application CRD to internal application model
func (s *ApplicationService) convertFromApplicationCRD(crd *v1alpha1.Application) *models.Application {
	labels := crd.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	annotations := crd.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Get project slug from project UUID label (stored on the application)
	projectSlug := ""
	// Note: ProjectSlug is derived from labels, not from a direct reference anymore
	// The actual project relationship is maintained through the project-uuid label

	return &models.Application{
		UUID:            labels[validation.LabelResourceUUID],
		Name:            annotations[validation.AnnotationResourceName],
		Slug:            labels[validation.LabelResourceSlug],
		ProjectUUID:     labels[validation.LabelProjectUUID],
		ProjectSlug:     projectSlug,
		EnvironmentUUID: labels[validation.LabelEnvironmentUUID],
		Type:            s.convertApplicationTypeFromCRD(crd.Spec.Type),
		GitRepository:   s.convertGitRepositoryConfigFromCRD(crd.Spec.GitRepository),
		DockerImage:     s.convertDockerImageConfigFromCRD(crd.Spec.DockerImage),
		MySQL:           s.convertMySQLConfigFromCRD(crd.Spec.MySQL),
		MySQLCluster:    s.convertMySQLClusterConfigFromCRD(crd.Spec.MySQLCluster),
		Postgres:        s.convertPostgresConfigFromCRD(crd.Spec.Postgres),
		PostgresCluster: s.convertPostgresClusterConfigFromCRD(crd.Spec.PostgresCluster),
		Status:          crd.Status.Phase,
		CreatedAt:       crd.CreationTimestamp.Time,
		UpdatedAt:       crd.CreationTimestamp.Time, // Would need to track updates
	}
}

// applyApplicationUpdates applies patch updates to the existing CRD
func (s *ApplicationService) applyApplicationUpdates(crd *v1alpha1.Application, req *models.ApplicationUpdateRequest) {
	annotations := crd.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Update name in annotations
	if req.Name != nil {
		annotations[validation.AnnotationResourceName] = *req.Name
		crd.SetAnnotations(annotations)
	}

	// Update type-specific configurations
	if req.GitRepository != nil {
		crd.Spec.GitRepository = s.convertGitRepositoryConfig(req.GitRepository)
	}
	if req.DockerImage != nil {
		crd.Spec.DockerImage = s.convertDockerImageConfig(req.DockerImage)
	}
	if req.MySQL != nil {
		crd.Spec.MySQL = s.convertMySQLConfig(req.MySQL)
	}
	if req.MySQLCluster != nil {
		crd.Spec.MySQLCluster = s.convertMySQLClusterConfig(req.MySQLCluster)
	}
	if req.Postgres != nil {
		crd.Spec.Postgres = s.convertPostgresConfig(req.Postgres)
	}
	if req.PostgresCluster != nil {
		crd.Spec.PostgresCluster = s.convertPostgresClusterConfig(req.PostgresCluster)
	}
}

// Type conversion methods

func (s *ApplicationService) convertApplicationType(appType models.ApplicationType) v1alpha1.ApplicationType {
	switch appType {
	case models.ApplicationTypeMySQL:
		return v1alpha1.ApplicationTypeMySQL
	case models.ApplicationTypeMySQLCluster:
		return v1alpha1.ApplicationTypeMySQLCluster
	case models.ApplicationTypePostgres:
		return v1alpha1.ApplicationTypePostgres
	case models.ApplicationTypePostgresCluster:
		return v1alpha1.ApplicationTypePostgresCluster
	case models.ApplicationTypeDockerImage:
		return v1alpha1.ApplicationTypeDockerImage
	case models.ApplicationTypeGitRepository:
		return v1alpha1.ApplicationTypeGitRepository
	case models.ApplicationTypeImageFromRegistry:
		return v1alpha1.ApplicationTypeImageFromRegistry
	default:
		return v1alpha1.ApplicationTypeDockerImage // Default fallback
	}
}

func (s *ApplicationService) convertApplicationTypeFromCRD(appType v1alpha1.ApplicationType) models.ApplicationType {
	switch appType {
	case v1alpha1.ApplicationTypeMySQL:
		return models.ApplicationTypeMySQL
	case v1alpha1.ApplicationTypeMySQLCluster:
		return models.ApplicationTypeMySQLCluster
	case v1alpha1.ApplicationTypePostgres:
		return models.ApplicationTypePostgres
	case v1alpha1.ApplicationTypePostgresCluster:
		return models.ApplicationTypePostgresCluster
	case v1alpha1.ApplicationTypeDockerImage:
		return models.ApplicationTypeDockerImage
	case v1alpha1.ApplicationTypeGitRepository:
		return models.ApplicationTypeGitRepository
	default:
		return models.ApplicationTypeDockerImage // Default fallback
	}
}

// Configuration conversion methods (simplified implementations)

func (s *ApplicationService) convertHealthCheckConfig(config *models.HealthCheckConfig) *v1alpha1.HealthCheckConfig {
	if config == nil {
		return nil
	}

	return &v1alpha1.HealthCheckConfig{
		Path:                config.Path,
		Port:                config.Port,
		InitialDelaySeconds: config.InitialDelaySeconds,
		PeriodSeconds:       config.PeriodSeconds,
		TimeoutSeconds:      config.TimeoutSeconds,
		SuccessThreshold:    config.SuccessThreshold,
		FailureThreshold:    config.FailureThreshold,
	}
}

func (s *ApplicationService) convertHealthCheckConfigFromCRD(config *v1alpha1.HealthCheckConfig) *models.HealthCheckConfig {
	if config == nil {
		return nil
	}

	return &models.HealthCheckConfig{
		Path:                config.Path,
		Port:                config.Port,
		InitialDelaySeconds: config.InitialDelaySeconds,
		PeriodSeconds:       config.PeriodSeconds,
		TimeoutSeconds:      config.TimeoutSeconds,
		SuccessThreshold:    config.SuccessThreshold,
		FailureThreshold:    config.FailureThreshold,
	}
}

func (s *ApplicationService) convertDockerfileBuildConfig(config *models.DockerfileBuildConfig) *v1alpha1.DockerfileBuildConfig {
	if config == nil {
		return nil
	}

	return &v1alpha1.DockerfileBuildConfig{
		DockerfilePath: config.DockerfilePath,
		BuildContext:   config.BuildContext,
	}
}

func (s *ApplicationService) convertDockerfileBuildConfigFromCRD(config *v1alpha1.DockerfileBuildConfig) *models.DockerfileBuildConfig {
	if config == nil {
		return nil
	}

	return &models.DockerfileBuildConfig{
		DockerfilePath: config.DockerfilePath,
		BuildContext:   config.BuildContext,
	}
}

func (s *ApplicationService) convertGitRepositoryConfig(config *models.GitRepositoryConfig) *v1alpha1.GitRepositoryConfig {
	if config == nil {
		return nil
	}

	var secretRef *corev1.LocalObjectReference
	if config.SecretRef != nil {
		secretRef = &corev1.LocalObjectReference{Name: *config.SecretRef}
	}

	return &v1alpha1.GitRepositoryConfig{
		Provider:           v1alpha1.GitProvider(config.Provider),
		Repository:         config.Repository,
		PublicAccess:       config.PublicAccess,
		SecretRef:          secretRef,
		Branch:             config.Branch,
		Path:               config.Path,
		RootDirectory:      config.RootDirectory,
		BuildCommand:       config.BuildCommand,
		StartCommand:       config.StartCommand,
		SpaOutputDirectory: config.SpaOutputDirectory,
		BuildType:          v1alpha1.BuildType(config.BuildType),
		DockerfileBuild:    s.convertDockerfileBuildConfig(config.DockerfileBuild),
		HealthCheck:        s.convertHealthCheckConfig(config.HealthCheck),
		// Env is automatically set by the application controller
	}
}

func (s *ApplicationService) convertGitRepositoryConfigFromCRD(config *v1alpha1.GitRepositoryConfig) *models.GitRepositoryConfig {
	if config == nil {
		return nil
	}

	var secretRef *string
	if config.SecretRef != nil {
		secretRef = &config.SecretRef.Name
	}

	return &models.GitRepositoryConfig{
		Provider:           models.GitProvider(config.Provider),
		Repository:         config.Repository,
		PublicAccess:       config.PublicAccess,
		SecretRef:          secretRef,
		Branch:             config.Branch,
		Path:               config.Path,
		RootDirectory:      config.RootDirectory,
		BuildCommand:       config.BuildCommand,
		StartCommand:       config.StartCommand,
		SpaOutputDirectory: config.SpaOutputDirectory,
		BuildType:          models.BuildType(config.BuildType),
		DockerfileBuild:    s.convertDockerfileBuildConfigFromCRD(config.DockerfileBuild),
		HealthCheck:        s.convertHealthCheckConfigFromCRD(config.HealthCheck),
		// Env is automatically managed by the application controller
	}
}

func (s *ApplicationService) convertDockerImageConfig(config *models.DockerImageConfig) *v1alpha1.DockerImageConfig {
	if config == nil {
		return nil
	}

	var imagePullSecretRef *corev1.LocalObjectReference
	if config.ImagePullSecretRef != nil {
		imagePullSecretRef = &corev1.LocalObjectReference{Name: *config.ImagePullSecretRef}
	}

	return &v1alpha1.DockerImageConfig{
		Image:              config.Image,
		ImagePullSecretRef: imagePullSecretRef,
		Tag:                config.Tag,
		HealthCheck:        s.convertHealthCheckConfig(config.HealthCheck),
	}
}

func (s *ApplicationService) convertDockerImageConfigFromCRD(config *v1alpha1.DockerImageConfig) *models.DockerImageConfig {
	if config == nil {
		return nil
	}

	var imagePullSecretRef *string
	if config.ImagePullSecretRef != nil {
		imagePullSecretRef = &config.ImagePullSecretRef.Name
	}

	return &models.DockerImageConfig{
		Image:              config.Image,
		ImagePullSecretRef: imagePullSecretRef,
		Tag:                config.Tag,
		HealthCheck:        s.convertHealthCheckConfigFromCRD(config.HealthCheck),
	}
}

func (s *ApplicationService) convertMySQLConfig(config *models.MySQLConfig) *v1alpha1.MySQLConfig {
	if config == nil {
		return nil
	}

	var secretRef *corev1.LocalObjectReference
	if config.SecretRef != nil {
		secretRef = &corev1.LocalObjectReference{Name: *config.SecretRef}
	}

	return &v1alpha1.MySQLConfig{
		Version:   config.Version,
		Database:  config.Database,
		SecretRef: secretRef,
	}
}

func (s *ApplicationService) convertMySQLConfigFromCRD(config *v1alpha1.MySQLConfig) *models.MySQLConfig {
	if config == nil {
		return nil
	}

	var secretRef *string
	if config.SecretRef != nil {
		secretRef = &config.SecretRef.Name
	}

	return &models.MySQLConfig{
		Version:   config.Version,
		Database:  config.Database,
		SecretRef: secretRef,
	}
}

func (s *ApplicationService) convertMySQLClusterConfig(config *models.MySQLClusterConfig) *v1alpha1.MySQLClusterConfig {
	if config == nil {
		return nil
	}

	var secretRef *corev1.LocalObjectReference
	if config.SecretRef != nil {
		secretRef = &corev1.LocalObjectReference{Name: *config.SecretRef}
	}

	return &v1alpha1.MySQLClusterConfig{
		Version:   config.Version,
		Replicas:  config.Replicas,
		Database:  config.Database,
		SecretRef: secretRef,
	}
}

func (s *ApplicationService) convertMySQLClusterConfigFromCRD(config *v1alpha1.MySQLClusterConfig) *models.MySQLClusterConfig {
	if config == nil {
		return nil
	}

	var secretRef *string
	if config.SecretRef != nil {
		secretRef = &config.SecretRef.Name
	}

	return &models.MySQLClusterConfig{
		Version:   config.Version,
		Replicas:  config.Replicas,
		Database:  config.Database,
		SecretRef: secretRef,
	}
}

func (s *ApplicationService) convertPostgresConfig(config *models.PostgresConfig) *v1alpha1.PostgresConfig {
	if config == nil {
		return nil
	}

	var secretRef *corev1.LocalObjectReference
	if config.SecretRef != nil {
		secretRef = &corev1.LocalObjectReference{Name: *config.SecretRef}
	}

	return &v1alpha1.PostgresConfig{
		Version:   config.Version,
		Database:  config.Database,
		SecretRef: secretRef,
	}
}

func (s *ApplicationService) convertPostgresConfigFromCRD(config *v1alpha1.PostgresConfig) *models.PostgresConfig {
	if config == nil {
		return nil
	}

	var secretRef *string
	if config.SecretRef != nil {
		secretRef = &config.SecretRef.Name
	}

	return &models.PostgresConfig{
		Version:   config.Version,
		Database:  config.Database,
		SecretRef: secretRef,
	}
}

func (s *ApplicationService) convertPostgresClusterConfig(config *models.PostgresClusterConfig) *v1alpha1.PostgresClusterConfig {
	if config == nil {
		return nil
	}

	var secretRef *corev1.LocalObjectReference
	if config.SecretRef != nil {
		secretRef = &corev1.LocalObjectReference{Name: *config.SecretRef}
	}

	return &v1alpha1.PostgresClusterConfig{
		Version:   config.Version,
		Replicas:  config.Replicas,
		Database:  config.Database,
		SecretRef: secretRef,
	}
}

func (s *ApplicationService) convertPostgresClusterConfigFromCRD(config *v1alpha1.PostgresClusterConfig) *models.PostgresClusterConfig {
	if config == nil {
		return nil
	}

	var secretRef *string
	if config.SecretRef != nil {
		secretRef = &config.SecretRef.Name
	}

	return &models.PostgresClusterConfig{
		Version:   config.Version,
		Replicas:  config.Replicas,
		Database:  config.Database,
		SecretRef: secretRef,
	}
}
