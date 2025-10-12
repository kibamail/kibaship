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
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/validation"
)

// ProjectValidator handles validation for Project resources
type ProjectValidator struct {
	client.Client
	NamespaceManager *NamespaceManager
}

// NewProjectValidator creates a new ProjectValidator
func NewProjectValidator(k8sClient client.Client) *ProjectValidator {
	return &ProjectValidator{
		Client:           k8sClient,
		NamespaceManager: NewNamespaceManager(k8sClient),
	}
}

// ValidateProjectCreate validates a project during creation
func (pv *ProjectValidator) ValidateProjectCreate(ctx context.Context, project *platformv1alpha1.Project) error {
	log := logf.FromContext(ctx)

	log.Info("Validating project creation", "project", project.Name)

	// Validate required labels and uniqueness
	resourceLabeler := NewResourceLabeler(pv.Client)
	if err := resourceLabeler.ValidateProjectLabeling(ctx, project); err != nil {
		return fmt.Errorf("label validation failed: %w", err)
	}

	// Validate project name uniqueness (check if namespace would conflict)
	isUnique, err := pv.NamespaceManager.IsProjectNamespaceUnique(ctx, project.Name, nil)
	if err != nil {
		return fmt.Errorf("failed to check project name uniqueness: %w", err)
	}
	if !isUnique {
		return fmt.Errorf("project name '%s' would conflict with existing namespace", project.Name)
	}

	// Validate project name format (additional validation beyond CRD schema)
	if err := pv.validateProjectName(project.Name); err != nil {
		return fmt.Errorf("project name validation failed: %w", err)
	}

	log.Info("Project validation passed", "project", project.Name)
	return nil
}

// ValidateProjectUpdate validates a project during update
func (pv *ProjectValidator) ValidateProjectUpdate(ctx context.Context, oldProject, newProject *platformv1alpha1.Project) error {
	log := logf.FromContext(ctx)

	log.Info("Validating project update", "project", newProject.Name)

	// Validate required labels and uniqueness
	resourceLabeler := NewResourceLabeler(pv.Client)
	if err := resourceLabeler.ValidateProjectLabeling(ctx, newProject); err != nil {
		return fmt.Errorf("label validation failed: %w", err)
	}

	// Check if project name changed (not allowed after creation)
	if oldProject.Name != newProject.Name {
		return fmt.Errorf("project name cannot be changed after creation")
	}

	// Validate UUID labels haven't changed (not allowed after creation)
	if oldProject.Labels[validation.LabelResourceUUID] != newProject.Labels[validation.LabelResourceUUID] {
		return fmt.Errorf("project UUID label cannot be changed after creation")
	}

	// Validate slug labels haven't changed (not allowed after creation)
	if oldProject.Labels[validation.LabelResourceSlug] != newProject.Labels[validation.LabelResourceSlug] {
		return fmt.Errorf("project slug label cannot be changed after creation")
	}

	log.Info("Project update validation passed", "project", newProject.Name)
	return nil
}

// ValidateRequiredLabels validates that the project has required UUID and slug labels
// This is a legacy method - use ResourceLabeler.ValidateProjectLabeling instead
func (pv *ProjectValidator) ValidateRequiredLabels(project *platformv1alpha1.Project) error {
	var validationErrors []string

	// Validate required UUID label
	if uuid, exists := project.Labels[validation.LabelResourceUUID]; !exists {
		validationErrors = append(validationErrors, "required label 'platform.kibaship.com/uuid' is missing")
	} else if !validation.ValidateUUID(uuid) {
		validationErrors = append(validationErrors, fmt.Sprintf("label 'platform.kibaship.com/uuid' must be a valid UUID, got: %s", uuid))
	}

	// Validate required slug label
	if slug, exists := project.Labels[validation.LabelResourceSlug]; !exists {
		validationErrors = append(validationErrors, "required label 'platform.kibaship.com/slug' is missing")
	} else if !validation.ValidateSlug(slug) {
		validationErrors = append(validationErrors, fmt.Sprintf("label 'platform.kibaship.com/slug' must be a valid slug, got: %s", slug))
	}

	// Validate workspace UUID label if present
	if workspaceUUID, exists := project.Labels[validation.LabelWorkspaceUUID]; exists {
		if !validation.ValidateUUID(workspaceUUID) {
			validationErrors = append(validationErrors, fmt.Sprintf("label 'platform.kibaship.com/workspace-uuid' must be a valid UUID, got: %s", workspaceUUID))
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(validationErrors, "; "))
	}

	return nil
}

// validateProjectName validates the project name format and constraints
func (pv *ProjectValidator) validateProjectName(name string) error {
	// Additional name validation beyond CRD schema
	if len(name) == 0 {
		return fmt.Errorf("project name cannot be empty")
	}

	if len(name) > 50 {
		return fmt.Errorf("project name cannot exceed 50 characters")
	}

	// Ensure name is DNS compliant (already validated by CRD but double-checking)
	dnsPattern := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !dnsPattern.MatchString(name) {
		return fmt.Errorf("project name must be DNS compliant (lowercase letters, numbers, and hyphens only)")
	}

	// Check for reserved names
	reservedNames := []string{"default", "kube-system", "kube-public", "kube-node-lease", "kibaship"}
	for _, reserved := range reservedNames {
		if name == reserved || strings.HasPrefix(name, reserved+"-") {
			return fmt.Errorf("project name '%s' conflicts with reserved namespace name", name)
		}
	}

	return nil
}

// CheckProjectNameUniqueness checks if a project name is unique across the cluster
func (pv *ProjectValidator) CheckProjectNameUniqueness(ctx context.Context, projectName string, excludeProject *platformv1alpha1.Project) error {
	log := logf.FromContext(ctx)

	// Check if any other project has the same name
	projectList := &platformv1alpha1.ProjectList{}
	if err := pv.List(ctx, projectList); err != nil {
		return fmt.Errorf("failed to list existing projects: %w", err)
	}

	for _, existingProject := range projectList.Items {
		// Skip the project we're validating (for updates)
		if excludeProject != nil && existingProject.Name == excludeProject.Name {
			continue
		}

		if existingProject.Name == projectName {
			return fmt.Errorf("project name '%s' already exists", projectName)
		}
	}

	// Also check if the generated namespace name would conflict
	isUnique, err := pv.NamespaceManager.IsProjectNamespaceUnique(ctx, projectName, excludeProject)
	if err != nil {
		return fmt.Errorf("failed to check namespace uniqueness: %w", err)
	}
	if !isUnique {
		return fmt.Errorf("project name '%s' would create conflicting namespace", projectName)
	}

	log.Info("Project name uniqueness validated", "project", projectName)
	return nil
}

// GetProjectByNamespace finds a project that owns the given namespace
func (pv *ProjectValidator) GetProjectByNamespace(ctx context.Context, namespaceName string) (*platformv1alpha1.Project, error) {
	// Extract project name from namespace name
	if !strings.HasPrefix(namespaceName, NamespacePrefix) || !strings.HasSuffix(namespaceName, NamespaceSuffix) {
		return nil, fmt.Errorf("namespace '%s' is not a project namespace", namespaceName)
	}

	projectName := strings.TrimPrefix(namespaceName, NamespacePrefix)
	projectName = strings.TrimSuffix(projectName, NamespaceSuffix)

	// Get the project
	project := &platformv1alpha1.Project{}
	err := pv.Get(ctx, types.NamespacedName{Name: projectName}, project)
	if err != nil {
		return nil, fmt.Errorf("failed to get project '%s': %w", projectName, err)
	}

	return project, nil
}

// ListProjectNamespaces returns all namespaces managed by projects
func (pv *ProjectValidator) ListProjectNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	namespaces := &corev1.NamespaceList{}

	// List namespaces with our managed-by label
	listOptions := []client.ListOption{
		client.MatchingLabels{ManagedByLabel: ManagedByValue},
	}

	err := pv.List(ctx, namespaces, listOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to list project namespaces: %w", err)
	}

	return namespaces.Items, nil
}
