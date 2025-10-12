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
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/validation"
)

// ResourceLabeler handles labeling operations for all resources
type ResourceLabeler struct {
	client client.Client
}

// NewResourceLabeler creates a new ResourceLabeler
func NewResourceLabeler(c client.Client) *ResourceLabeler {
	return &ResourceLabeler{client: c}
}

// ProjectLabeling handles labeling for Project resources
type ProjectLabeling struct {
	UUID          string
	Slug          string
	WorkspaceUUID string
}

// ValidateProjectLabeling validates Project labeling requirements
func (rl *ResourceLabeler) ValidateProjectLabeling(ctx context.Context, project *platformv1alpha1.Project) error {
	labels := project.GetLabels()
	if labels == nil {
		return fmt.Errorf("project must have labels")
	}

	// Validate UUID
	resourceUUID, exists := labels[validation.LabelResourceUUID]
	if !exists {
		return fmt.Errorf("project must have label %s", validation.LabelResourceUUID)
	}
	if !validation.ValidateUUID(resourceUUID) {
		return fmt.Errorf("project UUID must be valid: %s", resourceUUID)
	}

	// Validate Slug
	resourceSlug, exists := labels[validation.LabelResourceSlug]
	if !exists {
		return fmt.Errorf("project must have label %s", validation.LabelResourceSlug)
	}
	if !validation.ValidateSlug(resourceSlug) {
		return fmt.Errorf("project slug must be valid: %s", resourceSlug)
	}

	// Validate Workspace UUID if present
	if workspaceUUID, exists := labels[validation.LabelWorkspaceUUID]; exists {
		if !validation.ValidateUUID(workspaceUUID) {
			return fmt.Errorf("workspace UUID must be valid: %s", workspaceUUID)
		}
	}

	// Check uniqueness
	return rl.CheckProjectUniqueness(ctx, project, resourceUUID, resourceSlug)
}

// CheckProjectUniqueness ensures Project UUID and slug are unique cluster-wide
func (rl *ResourceLabeler) CheckProjectUniqueness(ctx context.Context, currentProject *platformv1alpha1.Project, uuid, slug string) error {
	projectList := &platformv1alpha1.ProjectList{}
	if err := rl.client.List(ctx, projectList); err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	for _, project := range projectList.Items {
		// Skip current project in update operations
		if project.GetName() == currentProject.GetName() &&
			project.GetNamespace() == currentProject.GetNamespace() &&
			project.GetUID() == currentProject.GetUID() {
			continue
		}

		labels := project.GetLabels()
		if labels == nil {
			continue
		}

		// Check UUID uniqueness
		if existingUUID, exists := labels[validation.LabelResourceUUID]; exists && existingUUID == uuid {
			return fmt.Errorf("project UUID %s already exists in project %s", uuid, project.Name)
		}

		// Check slug uniqueness
		if existingSlug, exists := labels[validation.LabelResourceSlug]; exists && existingSlug == slug {
			return fmt.Errorf("project slug %s already exists in project %s", slug, project.Name)
		}
	}

	return nil
}

// ApplicationLabeling handles labeling for Application resources
type ApplicationLabeling struct {
	UUID        string
	Slug        string
	ProjectUUID string
}

// ValidateApplicationLabeling validates Application labeling requirements
func (rl *ResourceLabeler) ValidateApplicationLabeling(ctx context.Context, application *platformv1alpha1.Application) error {
	labels := application.GetLabels()
	if labels == nil {
		return fmt.Errorf("application must have labels")
	}

	// Validate UUID
	resourceUUID, exists := labels[validation.LabelResourceUUID]
	if !exists {
		return fmt.Errorf("application must have label %s", validation.LabelResourceUUID)
	}
	if !validation.ValidateUUID(resourceUUID) {
		return fmt.Errorf("application UUID must be valid: %s", resourceUUID)
	}

	// Validate Slug
	resourceSlug, exists := labels[validation.LabelResourceSlug]
	if !exists {
		return fmt.Errorf("application must have label %s", validation.LabelResourceSlug)
	}
	if !validation.ValidateSlug(resourceSlug) {
		return fmt.Errorf("application slug must be valid: %s", resourceSlug)
	}

	// Validate Project UUID
	projectUUID, exists := labels[validation.LabelProjectUUID]
	if !exists {
		return fmt.Errorf("application must have label %s", validation.LabelProjectUUID)
	}
	if !validation.ValidateUUID(projectUUID) {
		return fmt.Errorf("project UUID must be valid: %s", projectUUID)
	}

	// Check uniqueness
	return rl.CheckApplicationUniqueness(ctx, application, resourceUUID, resourceSlug, projectUUID)
}

// CheckApplicationUniqueness ensures Application UUID and slug are unique within project scope
func (rl *ResourceLabeler) CheckApplicationUniqueness(ctx context.Context, currentApplication *platformv1alpha1.Application, uuid, slug, projectUUID string) error {
	applicationList := &platformv1alpha1.ApplicationList{}
	if err := rl.client.List(ctx, applicationList, client.InNamespace(currentApplication.GetNamespace())); err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	for _, application := range applicationList.Items {
		// Skip current application in update operations
		if application.GetName() == currentApplication.GetName() &&
			application.GetNamespace() == currentApplication.GetNamespace() &&
			application.GetUID() == currentApplication.GetUID() {
			continue
		}

		labels := application.GetLabels()
		if labels == nil {
			continue
		}

		// Check UUID uniqueness (globally)
		if existingUUID, exists := labels[validation.LabelResourceUUID]; exists && existingUUID == uuid {
			return fmt.Errorf("application UUID %s already exists in application %s", uuid, application.Name)
		}

		// Check slug uniqueness within the same project
		if existingSlug, exists := labels[validation.LabelResourceSlug]; exists && existingSlug == slug {
			if existingProjectUUID, exists := labels[validation.LabelProjectUUID]; exists && existingProjectUUID == projectUUID {
				return fmt.Errorf("application slug %s already exists in project %s for application %s", slug, projectUUID, application.Name)
			}
		}
	}

	return nil
}

// DeploymentLabeling handles labeling for Deployment resources
type DeploymentLabeling struct {
	UUID            string
	Slug            string
	ProjectUUID     string
	ApplicationUUID string
}

// ValidateDeploymentLabeling validates Deployment labeling requirements
func (rl *ResourceLabeler) ValidateDeploymentLabeling(ctx context.Context, deployment *platformv1alpha1.Deployment) error {
	labels := deployment.GetLabels()
	if labels == nil {
		return fmt.Errorf("deployment must have labels")
	}

	// Validate UUID
	resourceUUID, exists := labels[validation.LabelResourceUUID]
	if !exists {
		return fmt.Errorf("deployment must have label %s", validation.LabelResourceUUID)
	}
	if !validation.ValidateUUID(resourceUUID) {
		return fmt.Errorf("deployment UUID must be valid: %s", resourceUUID)
	}

	// Validate Slug
	resourceSlug, exists := labels[validation.LabelResourceSlug]
	if !exists {
		return fmt.Errorf("deployment must have label %s", validation.LabelResourceSlug)
	}
	if !validation.ValidateSlug(resourceSlug) {
		return fmt.Errorf("deployment slug must be valid: %s", resourceSlug)
	}

	// Validate Project UUID
	projectUUID, exists := labels[validation.LabelProjectUUID]
	if !exists {
		return fmt.Errorf("deployment must have label %s", validation.LabelProjectUUID)
	}
	if !validation.ValidateUUID(projectUUID) {
		return fmt.Errorf("project UUID must be valid: %s", projectUUID)
	}

	// Validate Application UUID
	applicationUUID, exists := labels[validation.LabelApplicationUUID]
	if !exists {
		return fmt.Errorf("deployment must have label %s", validation.LabelApplicationUUID)
	}
	if !validation.ValidateUUID(applicationUUID) {
		return fmt.Errorf("application UUID must be valid: %s", applicationUUID)
	}

	// Check uniqueness
	return rl.CheckDeploymentUniqueness(ctx, deployment, resourceUUID, resourceSlug, applicationUUID)
}

// CheckDeploymentUniqueness ensures Deployment UUID and slug are unique within application scope
func (rl *ResourceLabeler) CheckDeploymentUniqueness(ctx context.Context, currentDeployment *platformv1alpha1.Deployment, uuid, slug, applicationUUID string) error {
	deploymentList := &platformv1alpha1.DeploymentList{}
	if err := rl.client.List(ctx, deploymentList, client.InNamespace(currentDeployment.GetNamespace())); err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	for _, deployment := range deploymentList.Items {
		// Skip current deployment in update operations
		if deployment.GetName() == currentDeployment.GetName() &&
			deployment.GetNamespace() == currentDeployment.GetNamespace() &&
			deployment.GetUID() == currentDeployment.GetUID() {
			continue
		}

		labels := deployment.GetLabels()
		if labels == nil {
			continue
		}

		// Check UUID uniqueness (globally)
		if existingUUID, exists := labels[validation.LabelResourceUUID]; exists && existingUUID == uuid {
			return fmt.Errorf("deployment UUID %s already exists in deployment %s", uuid, deployment.Name)
		}

		// Check slug uniqueness within the same application
		if existingSlug, exists := labels[validation.LabelResourceSlug]; exists && existingSlug == slug {
			if existingApplicationUUID, exists := labels[validation.LabelApplicationUUID]; exists && existingApplicationUUID == applicationUUID {
				return fmt.Errorf("deployment slug %s already exists in application %s for deployment %s", slug, applicationUUID, deployment.Name)
			}
		}
	}

	return nil
}

// ApplicationDomainLabeling handles labeling for ApplicationDomain resources
type ApplicationDomainLabeling struct {
	UUID            string
	Slug            string
	ProjectUUID     string
	ApplicationUUID string
	DeploymentUUID  string
}

// ValidateApplicationDomainLabeling validates ApplicationDomain labeling requirements
func (rl *ResourceLabeler) ValidateApplicationDomainLabeling(ctx context.Context, domain *platformv1alpha1.ApplicationDomain) error {
	labels := domain.GetLabels()
	if labels == nil {
		return fmt.Errorf("application domain must have labels")
	}

	// Validate UUID
	resourceUUID, exists := labels[validation.LabelResourceUUID]
	if !exists {
		return fmt.Errorf("application domain must have label %s", validation.LabelResourceUUID)
	}
	if !validation.ValidateUUID(resourceUUID) {
		return fmt.Errorf("application domain UUID must be valid: %s", resourceUUID)
	}

	// Validate Slug
	resourceSlug, exists := labels[validation.LabelResourceSlug]
	if !exists {
		return fmt.Errorf("application domain must have label %s", validation.LabelResourceSlug)
	}
	if !validation.ValidateSlug(resourceSlug) {
		return fmt.Errorf("application domain slug must be valid: %s", resourceSlug)
	}

	// Validate Project UUID
	projectUUID, exists := labels[validation.LabelProjectUUID]
	if !exists {
		return fmt.Errorf("application domain must have label %s", validation.LabelProjectUUID)
	}
	if !validation.ValidateUUID(projectUUID) {
		return fmt.Errorf("project UUID must be valid: %s", projectUUID)
	}

	// Validate Application UUID
	applicationUUID, exists := labels[validation.LabelApplicationUUID]
	if !exists {
		return fmt.Errorf("application domain must have label %s", validation.LabelApplicationUUID)
	}
	if !validation.ValidateUUID(applicationUUID) {
		return fmt.Errorf("application UUID must be valid: %s", applicationUUID)
	}

	// Deployment UUID is optional for ApplicationDomain
	if deploymentUUID, exists := labels[validation.LabelDeploymentUUID]; exists {
		if !validation.ValidateUUID(deploymentUUID) {
			return fmt.Errorf("deployment UUID must be valid: %s", deploymentUUID)
		}
	}

	// Check uniqueness
	return rl.CheckApplicationDomainUniqueness(ctx, domain, resourceUUID, resourceSlug, applicationUUID)
}

// CheckApplicationDomainUniqueness ensures ApplicationDomain UUID and slug are unique within application scope
func (rl *ResourceLabeler) CheckApplicationDomainUniqueness(ctx context.Context, currentDomain *platformv1alpha1.ApplicationDomain, uuid, slug, applicationUUID string) error {
	domainList := &platformv1alpha1.ApplicationDomainList{}
	if err := rl.client.List(ctx, domainList, client.InNamespace(currentDomain.GetNamespace())); err != nil {
		return fmt.Errorf("failed to list application domains: %w", err)
	}

	for _, domain := range domainList.Items {
		// Skip current domain in update operations
		if domain.GetName() == currentDomain.GetName() &&
			domain.GetNamespace() == currentDomain.GetNamespace() &&
			domain.GetUID() == currentDomain.GetUID() {
			continue
		}

		labels := domain.GetLabels()
		if labels == nil {
			continue
		}

		// Check UUID uniqueness (globally)
		if existingUUID, exists := labels[validation.LabelResourceUUID]; exists && existingUUID == uuid {
			return fmt.Errorf("application domain UUID %s already exists in domain %s", uuid, domain.Name)
		}

		// Check slug uniqueness within the same application
		if existingSlug, exists := labels[validation.LabelResourceSlug]; exists && existingSlug == slug {
			if existingApplicationUUID, exists := labels[validation.LabelApplicationUUID]; exists && existingApplicationUUID == applicationUUID {
				return fmt.Errorf("application domain slug %s already exists in application %s for domain %s", slug, applicationUUID, domain.Name)
			}
		}
	}

	return nil
}

// ApplyLabelsToResource applies the standard labels to any resource
func ApplyLabelsToResource(obj metav1.Object, resourceUUID, resourceSlug string, parentLabels map[string]string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	// Set resource-specific labels
	labels[validation.LabelResourceUUID] = resourceUUID
	labels[validation.LabelResourceSlug] = resourceSlug

	// Apply parent labels
	for key, value := range parentLabels {
		labels[key] = value
	}

	obj.SetLabels(labels)
}
