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
	"github.com/kibamail/kibaship-operator/pkg/validation"
)

const (
	// NamespacePrefix is the prefix used for project namespaces
	NamespacePrefix = "project-"

	// NamespaceSuffix is the suffix used for project namespaces
	NamespaceSuffix = ""

	// ProjectNameLabel is the label key for project name
	ProjectNameLabel = "platform.kibaship.com/project-name"

	// ManagedByLabel indicates this namespace is managed by the kibaship operator
	ManagedByLabel = "app.kubernetes.io/managed-by"

	// ManagedByValue is the value for the managed-by label
	ManagedByValue = "kibaship-operator"

	// ServiceAccountNamePrefix is the prefix for the service account name
	ServiceAccountNamePrefix = "project-"
	// ServiceAccountNameSuffix is the suffix for the service account name
	ServiceAccountNameSuffix = "-sa"

	// RoleNamePrefix is the prefix for the role name
	RoleNamePrefix = "project-"
	// RoleNameSuffix is the suffix for the role name
	RoleNameSuffix = "-admin-role"

	// RoleBindingNamePrefix is the prefix for the role binding name
	RoleBindingNamePrefix = "project-"
	// RoleBindingNameSuffix is the suffix for the role binding name
	RoleBindingNameSuffix = "-admin-binding"

	// Tekton constants
	TektonNamespace = "tekton-pipelines"
	TektonRoleName  = "tekton-tasks-reader"

	// TektonRoleBindingNamePrefix is the prefix for the tekton role binding name
	TektonRoleBindingNamePrefix = "project-"
	// TektonRoleBindingNameSuffix is the suffix for the tekton role binding name
	TektonRoleBindingNameSuffix = "-tekton-tasks-reader-binding"
)

// NamespaceManager handles namespace operations for projects
type NamespaceManager struct {
	client.Client
}

// NewNamespaceManager creates a new NamespaceManager
func NewNamespaceManager(k8sClient client.Client) *NamespaceManager {
	return &NamespaceManager{
		Client: k8sClient,
	}
}

// CreateProjectNamespace creates a namespace for the given project
func (nm *NamespaceManager) CreateProjectNamespace(ctx context.Context, project *platformv1alpha1.Project) (*corev1.Namespace, error) {
	log := logf.FromContext(ctx)

	projectUUID := project.Labels[validation.LabelResourceUUID]
	namespaceName := nm.GenerateNamespaceName(projectUUID)

	log.Info("Creating namespace for project", "project", project.Name, "namespace", namespaceName)

	// Check if namespace already exists
	existingNamespace := &corev1.Namespace{}
	err := nm.Get(ctx, types.NamespacedName{Name: namespaceName}, existingNamespace)
	if err == nil {
		// Namespace already exists, verify it belongs to this project
		if existingNamespace.Labels[validation.LabelResourceUUID] == project.Labels[validation.LabelResourceUUID] {
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
		_ = nm.Delete(ctx, namespace)
		return nil, fmt.Errorf("failed to create service account for project: %w", err)
	}

	log.Info("Successfully created namespace for project", "project", project.Name, "namespace", namespaceName)
	return namespace, nil
}

// DeleteProjectNamespace deletes the namespace for the given project
func (nm *NamespaceManager) DeleteProjectNamespace(ctx context.Context, project *platformv1alpha1.Project) error {
	log := logf.FromContext(ctx)

	projectUUID := project.Labels[validation.LabelResourceUUID]
	namespaceName := nm.GenerateNamespaceName(projectUUID)

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
	projectUUID := project.Labels[validation.LabelResourceUUID]
	namespaceName := nm.GenerateNamespaceName(projectUUID)

	namespace := &corev1.Namespace{}
	err := nm.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)
	if err != nil {
		return nil, err
	}

	return namespace, nil
}

// GenerateNamespaceName generates the namespace name for a project
func (nm *NamespaceManager) GenerateNamespaceName(projectUUID string) string {
	return NamespacePrefix + projectUUID + NamespaceSuffix
}

// generateServiceAccountName generates the service account name for a project
func (nm *NamespaceManager) generateServiceAccountName(projectUUID string) string {
	return ServiceAccountNamePrefix + projectUUID + ServiceAccountNameSuffix
}

// generateRoleName generates the role name for a project
func (nm *NamespaceManager) generateRoleName(projectUUID string) string {
	return RoleNamePrefix + projectUUID + RoleNameSuffix
}

// generateRoleBindingName generates the role binding name for a project
func (nm *NamespaceManager) generateRoleBindingName(projectUUID string) string {
	return RoleBindingNamePrefix + projectUUID + RoleBindingNameSuffix
}

// generateTektonRoleBindingName generates the Tekton role binding name for a project
func (nm *NamespaceManager) generateTektonRoleBindingName(projectUUID string) string {
	return TektonRoleBindingNamePrefix + projectUUID + TektonRoleBindingNameSuffix
}

// generateNamespaceLabels creates the labels for a project namespace
func (nm *NamespaceManager) generateNamespaceLabels(project *platformv1alpha1.Project) map[string]string {
	labels := map[string]string{
		ManagedByLabel:   ManagedByValue,
		ProjectNameLabel: project.Name,
	}

	// Copy project UUID labels if they exist
	if projectUUID, exists := project.Labels[validation.LabelResourceUUID]; exists {
		labels[validation.LabelResourceUUID] = projectUUID
	}

	if workspaceUUID, exists := project.Labels[validation.LabelWorkspaceUUID]; exists {
		labels[validation.LabelWorkspaceUUID] = workspaceUUID
	}

	return labels
}

// IsProjectNamespaceUnique checks if the project UUID would result in a unique namespace
func (nm *NamespaceManager) IsProjectNamespaceUnique(ctx context.Context, projectUUID string, excludeProject *platformv1alpha1.Project) (bool, error) {
	namespaceName := nm.GenerateNamespaceName(projectUUID)

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
		if namespace.Labels[validation.LabelResourceUUID] == excludeProject.Labels[validation.LabelResourceUUID] {
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

	// Create Tekton integration
	if err := nm.createTektonIntegration(ctx, namespace, project); err != nil {
		return fmt.Errorf("failed to create Tekton integration: %w", err)
	}

	log.Info("Successfully created service account with admin permissions", "project", project.Name, "namespace", namespace.Name)
	return nil
}

// createServiceAccount creates the service account in the namespace
func (nm *NamespaceManager) createServiceAccount(ctx context.Context, namespace *corev1.Namespace, project *platformv1alpha1.Project) error {
	projectUUID := project.Labels[validation.LabelResourceUUID]
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nm.generateServiceAccountName(projectUUID),
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
	if projectUUID, exists := project.Labels[validation.LabelResourceUUID]; exists {
		serviceAccount.Labels[validation.LabelResourceUUID] = projectUUID
	}
	if workspaceUUID, exists := project.Labels[validation.LabelWorkspaceUUID]; exists {
		serviceAccount.Labels[validation.LabelWorkspaceUUID] = workspaceUUID
	}

	if err := nm.Create(ctx, serviceAccount); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createAdminRole creates a role with all permissions in the namespace
func (nm *NamespaceManager) createAdminRole(ctx context.Context, namespace *corev1.Namespace, project *platformv1alpha1.Project) error {
	projectUUID := project.Labels[validation.LabelResourceUUID]
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nm.generateRoleName(projectUUID),
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
	if projectUUID, exists := project.Labels[validation.LabelResourceUUID]; exists {
		role.Labels[validation.LabelResourceUUID] = projectUUID
	}
	if workspaceUUID, exists := project.Labels[validation.LabelWorkspaceUUID]; exists {
		role.Labels[validation.LabelWorkspaceUUID] = workspaceUUID
	}

	if err := nm.Create(ctx, role); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createRoleBinding creates a role binding between the service account and the admin role
func (nm *NamespaceManager) createRoleBinding(ctx context.Context, namespace *corev1.Namespace, project *platformv1alpha1.Project) error {
	projectUUID := project.Labels[validation.LabelResourceUUID]
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nm.generateRoleBindingName(projectUUID),
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
				Name:      nm.generateServiceAccountName(projectUUID),
				Namespace: namespace.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     nm.generateRoleName(projectUUID),
		},
	}

	// Copy project UUID labels if they exist
	if projectUUID, exists := project.Labels[validation.LabelResourceUUID]; exists {
		roleBinding.Labels[validation.LabelResourceUUID] = projectUUID
	}
	if workspaceUUID, exists := project.Labels[validation.LabelWorkspaceUUID]; exists {
		roleBinding.Labels[validation.LabelWorkspaceUUID] = workspaceUUID
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

	projectUUID := project.Labels[validation.LabelResourceUUID]

	// Delete role binding
	roleBindingName := nm.generateRoleBindingName(projectUUID)
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
	roleName := nm.generateRoleName(projectUUID)
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
	serviceAccountName := nm.generateServiceAccountName(projectUUID)
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace.Name,
		},
	}
	if err := nm.Delete(ctx, serviceAccount); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to delete service account", "name", serviceAccountName, "namespace", namespace.Name)
	}

	// Delete Tekton role binding
	tektonRoleBindingName := nm.generateTektonRoleBindingName(projectUUID)
	tektonRoleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tektonRoleBindingName,
			Namespace: TektonNamespace,
		},
	}
	if err := nm.Delete(ctx, tektonRoleBinding); err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to delete Tekton role binding", "name", tektonRoleBindingName, "namespace", TektonNamespace)
	}

	log.Info("Service account resources cleanup completed", "project", project.Name, "namespace", namespace.Name)
}

// createTektonIntegration creates Tekton role and role binding for the project
func (nm *NamespaceManager) createTektonIntegration(ctx context.Context, namespace *corev1.Namespace, project *platformv1alpha1.Project) error {
	log := logf.FromContext(ctx)

	log.Info("Creating Tekton integration for project", "project", project.Name, "tektonNamespace", TektonNamespace)

	// Ensure the tekton-tasks-reader role exists
	if err := nm.ensureTektonTasksReaderRole(ctx); err != nil {
		return fmt.Errorf("failed to ensure Tekton tasks reader role: %w", err)
	}

	// Create role binding from project service account to tekton role
	if err := nm.createTektonRoleBinding(ctx, namespace, project); err != nil {
		return fmt.Errorf("failed to create Tekton role binding: %w", err)
	}

	log.Info("Successfully created Tekton integration", "project", project.Name, "tektonNamespace", TektonNamespace)
	return nil
}

// ensureTektonTasksReaderRole ensures the tekton-tasks-reader role exists in tekton-pipelines namespace
func (nm *NamespaceManager) ensureTektonTasksReaderRole(ctx context.Context) error {
	log := logf.FromContext(ctx)

	// Ensure tekton-pipelines namespace exists
	if err := nm.ensureTektonNamespace(ctx); err != nil {
		return fmt.Errorf("failed to ensure Tekton namespace: %w", err)
	}

	// Check if role already exists
	existingRole := &rbacv1.Role{}
	err := nm.Get(ctx, types.NamespacedName{
		Name:      TektonRoleName,
		Namespace: TektonNamespace,
	}, existingRole)

	if err == nil {
		log.Info("Tekton tasks reader role already exists", "role", TektonRoleName, "namespace", TektonNamespace)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check if Tekton role exists: %w", err)
	}

	// Create the role
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TektonRoleName,
			Namespace: TektonNamespace,
			Labels: map[string]string{
				ManagedByLabel: ManagedByValue,
			},
			Annotations: map[string]string{
				"platform.kibaship.com/created-by": "kibaship-operator",
				"platform.kibaship.com/purpose":    "Allow projects to read Tekton tasks",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"tekton.dev"},
				Resources: []string{"tasks"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}

	if err := nm.Create(ctx, role); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create Tekton tasks reader role: %w", err)
	}

	log.Info("Created Tekton tasks reader role", "role", TektonRoleName, "namespace", TektonNamespace)
	return nil
}

// createTektonRoleBinding creates a role binding from project service account to tekton-tasks-reader role
func (nm *NamespaceManager) createTektonRoleBinding(ctx context.Context, namespace *corev1.Namespace, project *platformv1alpha1.Project) error {
	log := logf.FromContext(ctx)

	projectUUID := project.Labels[validation.LabelResourceUUID]
	roleBindingName := nm.generateTektonRoleBindingName(projectUUID)

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: TektonNamespace,
			Labels: map[string]string{
				ManagedByLabel:   ManagedByValue,
				ProjectNameLabel: project.Name,
			},
			Annotations: map[string]string{
				"platform.kibaship.com/created-by": "kibaship-operator",
				"platform.kibaship.com/project":    project.Name,
				"platform.kibaship.com/purpose":    "Allow project service account to read Tekton tasks",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      nm.generateServiceAccountName(projectUUID),
				Namespace: namespace.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     TektonRoleName,
		},
	}

	// Copy project UUID labels if they exist
	if projectUUID, exists := project.Labels[validation.LabelResourceUUID]; exists {
		roleBinding.Labels[validation.LabelResourceUUID] = projectUUID
	}
	if workspaceUUID, exists := project.Labels[validation.LabelWorkspaceUUID]; exists {
		roleBinding.Labels[validation.LabelWorkspaceUUID] = workspaceUUID
	}

	if err := nm.Create(ctx, roleBinding); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create Tekton role binding: %w", err)
	}

	log.Info("Created Tekton role binding", "roleBinding", roleBindingName, "namespace", TektonNamespace, "project", project.Name)
	return nil
}

// ensureTektonNamespace ensures the tekton-pipelines namespace exists
func (nm *NamespaceManager) ensureTektonNamespace(ctx context.Context) error {
	log := logf.FromContext(ctx)

	// Check if namespace already exists
	existingNamespace := &corev1.Namespace{}
	err := nm.Get(ctx, types.NamespacedName{Name: TektonNamespace}, existingNamespace)

	if err == nil {
		log.Info("Tekton namespace already exists", "namespace", TektonNamespace)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check if Tekton namespace exists: %w", err)
	}

	// Create the namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: TektonNamespace,
			Labels: map[string]string{
				ManagedByLabel: ManagedByValue,
			},
			Annotations: map[string]string{
				"platform.kibaship.com/created-by": "kibaship-operator",
				"platform.kibaship.com/purpose":    "Tekton Pipelines namespace for task management",
			},
		},
	}

	if err := nm.Create(ctx, namespace); err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create Tekton namespace: %w", err)
	}

	log.Info("Created Tekton namespace", "namespace", TektonNamespace)
	return nil
}
