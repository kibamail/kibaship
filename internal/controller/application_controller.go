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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
)

const (
	// ApplicationFinalizerName is the finalizer name for Application resources
	ApplicationFinalizerName = "platform.operator.kibaship.com/application-finalizer"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applications/finalizers,verbs=update
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Application instance
	var app platformv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		if errors.IsNotFound(err) {
			// Application was deleted
			log.Info("Application not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Application")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if app.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, &app)
	}

	// Ensure UUID labels are set correctly
	labelsUpdated, err := r.ensureUUIDLabels(ctx, &app)
	if err != nil {
		log.Error(err, "Failed to ensure UUID labels")
		return ctrl.Result{}, err
	}

	// Add finalizer if not present
	finalizerAdded := false
	if !controllerutil.ContainsFinalizer(&app, ApplicationFinalizerName) {
		controllerutil.AddFinalizer(&app, ApplicationFinalizerName)
		finalizerAdded = true
	}

	// Update if labels or finalizer were changed
	if labelsUpdated || finalizerAdded {
		if err := r.Update(ctx, &app); err != nil {
			log.Error(err, "Failed to update Application with labels/finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle Application creation/update
	return r.handleApplicationReconcile(ctx, &app)
}

// handleDeletion handles the deletion of an Application and its associated Deployments
func (r *ApplicationReconciler) handleDeletion(ctx context.Context, app *platformv1alpha1.Application) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("application", app.Name, "namespace", app.Namespace)

	if !controllerutil.ContainsFinalizer(app, ApplicationFinalizerName) {
		// Finalizer already removed, nothing to do
		return ctrl.Result{}, nil
	}

	log.Info("Handling Application deletion")

	// Delete all Deployments associated with this Application
	if err := r.deleteAssociatedDeployments(ctx, app); err != nil {
		log.Error(err, "Failed to delete associated Deployments")
		return ctrl.Result{}, err
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(app, ApplicationFinalizerName)
	if err := r.Update(ctx, app); err != nil {
		log.Error(err, "Failed to remove finalizer from Application")
		return ctrl.Result{}, err
	}

	log.Info("Successfully handled Application deletion")
	return ctrl.Result{}, nil
}

// deleteAssociatedDeployments deletes all Deployments associated with the Application
func (r *ApplicationReconciler) deleteAssociatedDeployments(ctx context.Context, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx).WithValues("application", app.Name, "namespace", app.Namespace)

	// Use label selector to efficiently find only Deployments associated with this Application
	var deploymentList platformv1alpha1.DeploymentList
	if err := r.List(ctx, &deploymentList,
		client.InNamespace(app.Namespace),
		client.MatchingLabels{"platform.operator.kibaship.com/application": app.Name}); err != nil {
		return fmt.Errorf("failed to list Deployments: %w", err)
	}

	// Delete all matching Deployments
	for _, deployment := range deploymentList.Items {
		log.Info("Deleting associated Deployment", "deployment", deployment.Name)
		if err := r.Delete(ctx, &deployment); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete Deployment %s: %w", deployment.Name, err)
		}
	}

	return nil
}

// handleApplicationReconcile handles the reconciliation of an Application
func (r *ApplicationReconciler) handleApplicationReconcile(ctx context.Context, app *platformv1alpha1.Application) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("application", app.Name, "namespace", app.Namespace)

	log.Info("Reconciling Application")

	// Ensure Deployment exists for this Application
	deployment, err := r.ensureDeployment(ctx, app)
	if err != nil {
		log.Error(err, "Failed to ensure Deployment")
		return ctrl.Result{}, err
	}

	// Update Application status
	if err := r.updateApplicationStatus(ctx, app, deployment); err != nil {
		log.Error(err, "Failed to update Application status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled Application")
	return ctrl.Result{}, nil
}

// ensureDeployment ensures a Deployment exists for the Application
func (r *ApplicationReconciler) ensureDeployment(ctx context.Context, app *platformv1alpha1.Application) (*platformv1alpha1.Deployment, error) {
	log := logf.FromContext(ctx).WithValues("application", app.Name, "namespace", app.Namespace)

	// Generate Deployment name based on Application name
	deploymentName := fmt.Sprintf("%s-deployment", app.Name)

	// Check if Deployment already exists
	var existingDeployment platformv1alpha1.Deployment
	err := r.Get(ctx, types.NamespacedName{
		Name:      deploymentName,
		Namespace: app.Namespace,
	}, &existingDeployment)

	if err == nil {
		// Deployment exists, check if it needs to be updated
		if existingDeployment.Spec.ApplicationRef.Name != app.Name {
			// Update the Deployment to reference the correct Application
			existingDeployment.Spec.ApplicationRef.Name = app.Name
			if err := r.Update(ctx, &existingDeployment); err != nil {
				return nil, fmt.Errorf("failed to update existing Deployment: %w", err)
			}
			log.Info("Updated existing Deployment", "deployment", deploymentName)
		}
		return &existingDeployment, nil
	}

	if !errors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get Deployment: %w", err)
	}

	// Create new Deployment
	newDeployment := &platformv1alpha1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: app.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":                     app.Name,
				"app.kubernetes.io/component":                "deployment",
				"app.kubernetes.io/managed-by":               "kibaship-operator",
				"platform.operator.kibaship.com/application": app.Name,
			},
		},
		Spec: platformv1alpha1.DeploymentSpec{
			ApplicationRef: corev1.LocalObjectReference{
				Name: app.Name,
			},
		},
	}

	// Set owner reference so the Deployment is cleaned up when Application is deleted
	if err := controllerutil.SetControllerReference(app, newDeployment, r.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference: %w", err)
	}

	// Create the Deployment
	if err := r.Create(ctx, newDeployment); err != nil {
		return nil, fmt.Errorf("failed to create Deployment: %w", err)
	}

	log.Info("Created new Deployment", "deployment", deploymentName)
	return newDeployment, nil
}

// ensureUUIDLabels ensures that the Application has the correct UUID labels
func (r *ApplicationReconciler) ensureUUIDLabels(ctx context.Context, app *platformv1alpha1.Application) (bool, error) {
	log := logf.FromContext(ctx).WithValues("application", app.Name, "namespace", app.Namespace)

	if app.Labels == nil {
		app.Labels = make(map[string]string)
	}

	labelsUpdated := false

	// Validate that Application has its own UUID label (should be set by PaaS)
	uuidLabel := "platform.kibaship.com/uuid"
	if _, exists := app.Labels[uuidLabel]; !exists {
		return false, fmt.Errorf("application must have label 'platform.kibaship.com/uuid' set by PaaS system")
	}

	// Get project UUID and set project UUID label
	projectUUIDLabel := "platform.kibaship.com/project-uuid"
	if _, exists := app.Labels[projectUUIDLabel]; !exists {
		projectUUID, err := r.getProjectUUID(ctx, app)
		if err != nil {
			return false, fmt.Errorf("failed to get project UUID: %w", err)
		}
		app.Labels[projectUUIDLabel] = projectUUID
		labelsUpdated = true
		log.Info("Set project UUID label", "projectUUID", projectUUID)
	}

	return labelsUpdated, nil
}

// getProjectUUID retrieves the UUID of the referenced project
func (r *ApplicationReconciler) getProjectUUID(ctx context.Context, app *platformv1alpha1.Application) (string, error) {
	// Get the referenced project
	var project platformv1alpha1.Project
	err := r.Get(ctx, types.NamespacedName{
		Name:      app.Spec.ProjectRef.Name,
		Namespace: app.Namespace,
	}, &project)

	if err != nil {
		return "", fmt.Errorf("failed to get referenced project %s: %w", app.Spec.ProjectRef.Name, err)
	}

	// Extract UUID from project labels
	projectUUID, exists := project.Labels["platform.kibaship.com/uuid"]
	if !exists {
		return "", fmt.Errorf("referenced project %s does not have required UUID label", project.Name)
	}

	return projectUUID, nil
}

// updateApplicationStatus updates the Application status based on the Deployment
func (r *ApplicationReconciler) updateApplicationStatus(ctx context.Context, app *platformv1alpha1.Application, deployment *platformv1alpha1.Deployment) error {
	// Update the Application status to reflect the current state
	app.Status.ObservedGeneration = app.Generation

	// Set condition based on deployment existence
	condition := metav1.Condition{
		Type:               "DeploymentReady",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "DeploymentCreated",
		Message:            fmt.Sprintf("Deployment %s created successfully", deployment.Name),
	}

	// Update or add the condition
	updated := false
	for i, existingCondition := range app.Status.Conditions {
		if existingCondition.Type == condition.Type {
			app.Status.Conditions[i] = condition
			updated = true
			break
		}
	}
	if !updated {
		app.Status.Conditions = append(app.Status.Conditions, condition)
	}

	// Update the status
	if err := r.Status().Update(ctx, app); err != nil {
		return fmt.Errorf("failed to update Application status: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Application{}).
		Owns(&platformv1alpha1.Deployment{}).
		Named("application").
		Complete(r)
}
