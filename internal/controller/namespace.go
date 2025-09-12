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

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
)

const (
	// NamespacePrefix is the prefix used for project namespaces
	NamespacePrefix = "project-"

	// NamespaceSuffix is the suffix used for project namespaces
	NamespaceSuffix = "-kibaship-com"

	// ProjectUUIDLabel is the label key for project UUID
	ProjectUUIDLabel = "platform.kibaship.com/uuid"

	// WorkspaceUUIDLabel is the label key for workspace UUID
	WorkspaceUUIDLabel = "platform.kibaship.com/workspace-uuid"

	// ProjectNameLabel is the label key for project name
	ProjectNameLabel = "platform.kibaship.com/project-name"

	// ManagedByLabel indicates this namespace is managed by the kibaship operator
	ManagedByLabel = "app.kubernetes.io/managed-by"

	// ManagedByValue is the value for the managed-by label
	ManagedByValue = "kibaship-operator"

	// ServiceAccountNamePrefix is the prefix for the service account name
	ServiceAccountNamePrefix = "project-"
	// ServiceAccountNameSuffix is the suffix for the service account name
	ServiceAccountNameSuffix = "-sa-kibaship-com"

	// RoleNamePrefix is the prefix for the role name
	RoleNamePrefix = "project-"
	// RoleNameSuffix is the suffix for the role name
	RoleNameSuffix = "-admin-role-kibaship-com"

	// RoleBindingNamePrefix is the prefix for the role binding name
	RoleBindingNamePrefix = "project-"
	// RoleBindingNameSuffix is the suffix for the role binding name
	RoleBindingNameSuffix = "-admin-binding-kibaship-com"
)

// NamespaceManager handles namespace operations for projects
type NamespaceManager struct {
	client.Client
}

// NewNamespaceManager creates a new NamespaceManager
func NewNamespaceManager(client client.Client) *NamespaceManager {
	return &NamespaceManager{
		Client: client,
	}
}

// CreateProjectNamespace creates a namespace for the given project
func (nm *NamespaceManager) CreateProjectNamespace(ctx context.Context, project *platformv1alpha1.Project) (*corev1.Namespace, error) {
	log := logf.FromContext(ctx)

	namespaceName := nm.GenerateNamespaceName(project.Name)

	log.Info("Creating namespace for project", "project", project.Name, "namespace", namespaceName)

	// Check if namespace already exists
	existingNamespace := &corev1.Namespace{}
	err := nm.Get(ctx, types.NamespacedName{Name: namespaceName}, existingNamespace)
	if err == nil {
		// Namespace already exists, verify it belongs to this project
		if existingNamespace.Labels[ProjectUUIDLabel] == project.Labels[ProjectUUIDLabel] {
			log.Info("Namespace already exists for project", "namespace", namespaceName)
			return existingNamespace, nil
		}
		// Namespace exists but belongs to different project
		return nil, fmt.Errorf("namespace %s already exists but belongs to different project", namespaceName)
	}

	if !errors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to check if namespace exists: %w", err)
	}

	// Create new namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespaceName,
			Labels: nm.generateNamespaceLabels(project),
			Annotations: map[string]string{
				"platform.kibaship.com/created-by": "kibaship-operator",
				"platform.kibaship.com/project":    project.Name,
			},
		},
	}

	// Note: Cannot set owner reference because namespace is cluster-scoped and project is namespace-scoped
	// Instead, we use labels for tracking and finalizers for cleanup

	if err := nm.Create(ctx, namespace); err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	// Create service account with all permissions in the namespace
	if err := nm.CreateProjectServiceAccount(ctx, namespace, project); err != nil {
		// Try to clean up the namespace if service account creation fails
		nm.Delete(ctx, namespace)
		return nil, fmt.Errorf("failed to create service account for project: %w", err)
	}

	log.Info("Successfully created namespace for project", "project", project.Name, "namespace", namespaceName)
	return namespace, nil
}

// DeleteProjectNamespace deletes the namespace for the given project
func (nm *NamespaceManager) DeleteProjectNamespace(ctx context.Context, project *platformv1alpha1.Project) error {
	log := logf.FromContext(ctx)

	namespaceName := nm.GenerateNamespaceName(project.Name)

	log.Info("Deleting namespace for project", "project", project.Name, "namespace", namespaceName)

	namespace := &corev1.Namespace{}
	err := nm.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)
	if errors.IsNotFound(err) {
		log.Info("Namespace already deleted", "namespace", namespaceName)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get namespace: %w", err)
	}

	// Clean up service account resources first (optional, as they'll be deleted with namespace)
	nm.deleteServiceAccountResources(ctx, namespace, project)

	if err := nm.Delete(ctx, namespace); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Namespace was already deleted during deletion attempt", "namespace", namespaceName)
			return nil
		}
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	log.Info("Successfully deleted namespace for project", "project", project.Name, "namespace", namespaceName)
	return nil
}

// GetProjectNamespace retrieves the namespace for the given project
func (nm *NamespaceManager) GetProjectNamespace(ctx context.Context, project *platformv1alpha1.Project) (*corev1.Namespace, error) {
	namespaceName := nm.GenerateNamespaceName(project.Name)

	namespace := &corev1.Namespace{}
	err := nm.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)
	if err != nil {
		return nil, err
	}

	return namespace, nil
}

// GenerateNamespaceName generates the namespace name for a project
func (nm *NamespaceManager) GenerateNamespaceName(projectName string) string {
	return NamespacePrefix + projectName + NamespaceSuffix
}

// generateServiceAccountName generates the service account name for a project
func (nm *NamespaceManager) generateServiceAccountName(projectName string) string {
	return ServiceAccountNamePrefix + projectName + ServiceAccountNameSuffix
}

// generateRoleName generates the role name for a project
func (nm *NamespaceManager) generateRoleName(projectName string) string {
	return RoleNamePrefix + projectName + RoleNameSuffix
}

// generateRoleBindingName generates the role binding name for a project
func (nm *NamespaceManager) generateRoleBindingName(projectName string) string {
	return RoleBindingNamePrefix + projectName + RoleBindingNameSuffix
}

// generateNamespaceLabels creates the labels for a project namespace
func (nm *NamespaceManager) generateNamespaceLabels(project *platformv1alpha1.Project) map[string]string {
	labels := map[string]string{
		ManagedByLabel:   ManagedByValue,
		ProjectNameLabel: project.Name,
	}

	// Copy project UUID labels if they exist
	if projectUUID, exists := project.Labels[ProjectUUIDLabel]; exists {
		labels[ProjectUUIDLabel] = projectUUID
	}

	if workspaceUUID, exists := project.Labels[WorkspaceUUIDLabel]; exists {
		labels[WorkspaceUUIDLabel] = workspaceUUID
	}

	return labels
}

// IsProjectNamespaceUnique checks if the project name would result in a unique namespace
func (nm *NamespaceManager) IsProjectNamespaceUnique(ctx context.Context, projectName string, excludeProject *platformv1alpha1.Project) (bool, error) {
	namespaceName := nm.GenerateNamespaceName(projectName)

	// Check if namespace exists
	namespace := &corev1.Namespace{}
	err := nm.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)
	if errors.IsNotFound(err) {
		return true, nil // Namespace doesn't exist, so it's unique
	}
	if err != nil {
		return false, fmt.Errorf("failed to check namespace existence: %w", err)
	}

	// If we're updating an existing project, check if the namespace belongs to the same project
	if excludeProject != nil {
		if namespace.Labels[ProjectUUIDLabel] == excludeProject.Labels[ProjectUUIDLabel] {
			return true, nil // Same project, so it's fine
		}
	}

	return false, nil // Namespace exists and belongs to different project
}

// CreateProjectServiceAccount creates a service account with all permissions in the project namespace
func (nm *NamespaceManager) CreateProjectServiceAccount(ctx context.Context, namespace *corev1.Namespace, project *platformv1alpha1.Project) error {
	log := logf.FromContext(ctx)

	log.Info("Creating service account for project", "project", project.Name, "namespace", namespace.Name)

	// Create the service account
	if err := nm.createServiceAccount(ctx, namespace, project); err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}

	// Create the role with all permissions
	if err := nm.createAdminRole(ctx, namespace, project); err != nil {
		return fmt.Errorf("failed to create admin role: %w", err)
	}

	// Create the role binding
	if err := nm.createRoleBinding(ctx, namespace, project); err != nil {
		return fmt.Errorf("failed to create role binding: %w", err)
	}

	log.Info("Successfully created service account with admin permissions", "project", project.Name, "namespace", namespace.Name)
	return nil
}

// createServiceAccount creates the service account in the namespace
func (nm *NamespaceManager) createServiceAccount(ctx context.Context, namespace *corev1.Namespace, project *platformv1alpha1.Project) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nm.generateServiceAccountName(project.Name),
			Namespace: namespace.Name,
			Labels: map[string]string{
				ManagedByLabel:   ManagedByValue,
				ProjectNameLabel: project.Name,
			},
			Annotations: map[string]string{
				"platform.kibaship.com/created-by": "kibaship-operator",
				"platform.kibaship.com/project":    project.Name,
			},
		},
	}

	// Copy project UUID labels if they exist
	if projectUUID, exists := project.Labels[ProjectUUIDLabel]; exists {
		serviceAccount.Labels[ProjectUUIDLabel] = projectUUID
	}
	if workspaceUUID, exists := project.Labels[WorkspaceUUIDLabel]; exists {
		serviceAccount.Labels[WorkspaceUUIDLabel] = workspaceUUID
	}

	if err := nm.Create(ctx, serviceAccount); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createAdminRole creates a role with all permissions in the namespace
func (nm *NamespaceManager) createAdminRole(ctx context.Context, namespace *corev1.Namespace, project *platformv1alpha1.Project) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nm.generateRoleName(project.Name),
			Namespace: namespace.Name,
			Labels: map[string]string{
				ManagedByLabel:   ManagedByValue,
				ProjectNameLabel: project.Name,
			},
			Annotations: map[string]string{
				"platform.kibaship.com/created-by": "kibaship-operator",
				"platform.kibaship.com/project":    project.Name,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}

	// Copy project UUID labels if they exist
	if projectUUID, exists := project.Labels[ProjectUUIDLabel]; exists {
		role.Labels[ProjectUUIDLabel] = projectUUID
	}
	if workspaceUUID, exists := project.Labels[WorkspaceUUIDLabel]; exists {
		role.Labels[WorkspaceUUIDLabel] = workspaceUUID
	}

	if err := nm.Create(ctx, role); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createRoleBinding creates a role binding between the service account and the admin role
func (nm *NamespaceManager) createRoleBinding(ctx context.Context, namespace *corev1.Namespace, project *platformv1alpha1.Project) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nm.generateRoleBindingName(project.Name),
			Namespace: namespace.Name,
			Labels: map[string]string{
				ManagedByLabel:   ManagedByValue,
				ProjectNameLabel: project.Name,
			},
			Annotations: map[string]string{
				"platform.kibaship.com/created-by": "kibaship-operator",
				"platform.kibaship.com/project":    project.Name,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      nm.generateServiceAccountName(project.Name),
				Namespace: namespace.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     nm.generateRoleName(project.Name),
		},
	}

	// Copy project UUID labels if they exist
	if projectUUID, exists := project.Labels[ProjectUUIDLabel]; exists {
		roleBinding.Labels[ProjectUUIDLabel] = projectUUID
	}
	if workspaceUUID, exists := project.Labels[WorkspaceUUIDLabel]; exists {
		roleBinding.Labels[WorkspaceUUIDLabel] = workspaceUUID
	}

	if err := nm.Create(ctx, roleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// deleteServiceAccountResources cleans up service account, role, and role binding
// Note: These resources are namespace-scoped so they will be automatically deleted
// when the namespace is deleted, but we delete them explicitly for better logging
func (nm *NamespaceManager) deleteServiceAccountResources(ctx context.Context, namespace *corev1.Namespace, project *platformv1alpha1.Project) {
	log := logf.FromContext(ctx)

	log.Info("Cleaning up service account resources", "project", project.Name, "namespace", namespace.Name)

	// Delete role binding
	roleBindingName := nm.generateRoleBindingName(project.Name)
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: namespace.Name,
		},
	}
	if err := nm.Delete(ctx, roleBinding); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to delete role binding", "name", roleBindingName, "namespace", namespace.Name)
	}

	// Delete role
	roleName := nm.generateRoleName(project.Name)
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace.Name,
		},
	}
	if err := nm.Delete(ctx, role); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to delete role", "name", roleName, "namespace", namespace.Name)
	}

	// Delete service account
	serviceAccountName := nm.generateServiceAccountName(project.Name)
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace.Name,
		},
	}
	if err := nm.Delete(ctx, serviceAccount); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to delete service account", "name", serviceAccountName, "namespace", namespace.Name)
	}

	log.Info("Service account resources cleanup completed", "project", project.Name, "namespace", namespace.Name)
}
