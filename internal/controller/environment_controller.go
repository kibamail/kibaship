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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/validation"
	"github.com/kibamail/kibaship/pkg/webhooks"
)

const (
	// EnvironmentFinalizerName is the finalizer name for Environment resources
	EnvironmentFinalizerName = "platform.operator.kibaship.com/environment-finalizer"
)

// EnvironmentReconciler reconciles an Environment object
type EnvironmentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Notifier webhooks.Notifier
}

// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=environments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=environments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=environments/finalizers,verbs=update
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applications,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *EnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Environment instance
	var environment platformv1alpha1.Environment
	if err := r.Get(ctx, req.NamespacedName, &environment); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Environment not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Environment")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if environment.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, &environment)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&environment, EnvironmentFinalizerName) {
		controllerutil.AddFinalizer(&environment, EnvironmentFinalizerName)
		if err := r.Update(ctx, &environment); err != nil {
			log.Error(err, "Failed to add finalizer to Environment")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Ensure UUID labels are set correctly
	labelsUpdated, err := r.ensureUUIDLabels(ctx, &environment)
	if err != nil {
		log.Error(err, "Failed to ensure UUID labels")
		return ctrl.Result{}, err
	}

	if labelsUpdated {
		if err := r.Update(ctx, &environment); err != nil {
			log.Error(err, "Failed to update Environment with labels")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Track previous phase
	prevPhase := environment.Status.Phase

	// Handle Environment reconciliation
	return r.handleEnvironmentReconcile(ctx, &environment, prevPhase)
}

// handleDeletion handles the deletion of an Environment and its associated Applications
func (r *EnvironmentReconciler) handleDeletion(ctx context.Context, environment *platformv1alpha1.Environment) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("environment", environment.Name, "namespace", environment.Namespace)

	if !controllerutil.ContainsFinalizer(environment, EnvironmentFinalizerName) {
		return ctrl.Result{}, nil
	}

	log.Info("Handling Environment deletion")

	// Delete all Applications associated with this Environment
	if err := r.deleteAssociatedApplications(ctx, environment); err != nil {
		log.Error(err, "Failed to delete associated Applications")
		return ctrl.Result{}, err
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(environment, EnvironmentFinalizerName)
	if err := r.Update(ctx, environment); err != nil {
		log.Error(err, "Failed to remove finalizer from Environment")
		return ctrl.Result{}, err
	}

	log.Info("Successfully handled Environment deletion")
	return ctrl.Result{}, nil
}

// deleteAssociatedApplications deletes all Applications associated with the Environment
func (r *EnvironmentReconciler) deleteAssociatedApplications(ctx context.Context, environment *platformv1alpha1.Environment) error {
	log := logf.FromContext(ctx).WithValues("environment", environment.Name, "namespace", environment.Namespace)

	// Get environment UUID
	envUUID := environment.Labels[validation.LabelResourceUUID]

	// Use label selector to find Applications with this environment UUID
	var applicationList platformv1alpha1.ApplicationList
	if err := r.List(ctx, &applicationList,
		client.InNamespace(environment.Namespace),
		client.MatchingLabels{validation.LabelEnvironmentUUID: envUUID}); err != nil {
		return fmt.Errorf("failed to list Applications: %w", err)
	}

	// Delete all matching Applications
	for i := range applicationList.Items {
		app := &applicationList.Items[i]
		log.Info("Deleting associated Application", "application", app.Name)
		if err := r.Delete(ctx, app); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete Application %s: %w", app.Name, err)
		}
	}

	return nil
}

// handleEnvironmentReconcile handles the reconciliation of an Environment
func (r *EnvironmentReconciler) handleEnvironmentReconcile(ctx context.Context, environment *platformv1alpha1.Environment, prevPhase string) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("environment", environment.Name, "namespace", environment.Namespace)

	log.Info("Reconciling Environment")

	// Update environment status
	if err := r.updateEnvironmentStatus(ctx, environment); err != nil {
		log.Error(err, "Failed to update Environment status")
		return ctrl.Result{}, err
	}

	// Emit webhook on phase transition
	r.emitEnvironmentPhaseChange(ctx, environment, prevPhase, environment.Status.Phase)

	log.Info("Successfully reconciled Environment")
	return ctrl.Result{}, nil
}

// ensureUUIDLabels ensures that the Environment has the correct UUID and slug labels
func (r *EnvironmentReconciler) ensureUUIDLabels(ctx context.Context, environment *platformv1alpha1.Environment) (bool, error) {
	log := logf.FromContext(ctx).WithValues("environment", environment.Name, "namespace", environment.Namespace)

	if environment.Labels == nil {
		environment.Labels = make(map[string]string)
	}

	labelsUpdated := false

	// Validate that Environment has its own UUID label (should be set by PaaS)
	if _, exists := environment.Labels[validation.LabelResourceUUID]; !exists {
		return false, fmt.Errorf("environment must have label '%s' set by PaaS system", validation.LabelResourceUUID)
	}

	// Validate that Environment has its own slug label (should be set by PaaS)
	if _, exists := environment.Labels[validation.LabelResourceSlug]; !exists {
		return false, fmt.Errorf("environment must have label '%s' set by PaaS system", validation.LabelResourceSlug)
	}

	// Get project UUID and set project UUID label if not present
	if _, exists := environment.Labels[validation.LabelProjectUUID]; !exists {
		projectUUID, err := r.getProjectUUID(ctx, environment)
		if err != nil {
			return false, fmt.Errorf("failed to get project UUID: %w", err)
		}
		environment.Labels[validation.LabelProjectUUID] = projectUUID
		labelsUpdated = true
		log.Info("Set project UUID label", "projectUUID", projectUUID)
	}

	return labelsUpdated, nil
}

// getProjectUUID retrieves the UUID of the referenced project
func (r *EnvironmentReconciler) getProjectUUID(ctx context.Context, environment *platformv1alpha1.Environment) (string, error) {
	// Get the referenced project
	var project platformv1alpha1.Project
	err := r.Get(ctx, types.NamespacedName{
		Name: environment.Spec.ProjectRef.Name,
	}, &project)

	if err != nil {
		return "", fmt.Errorf("failed to get referenced project %s: %w", environment.Spec.ProjectRef.Name, err)
	}

	// Extract UUID from project labels
	projectUUID, exists := project.Labels[validation.LabelResourceUUID]
	if !exists {
		return "", fmt.Errorf("referenced project %s does not have required UUID label", project.Name)
	}

	return projectUUID, nil
}

// updateEnvironmentStatus updates the Environment status
func (r *EnvironmentReconciler) updateEnvironmentStatus(ctx context.Context, environment *platformv1alpha1.Environment) error {
	// Count applications in this environment
	envUUID := environment.Labels[validation.LabelResourceUUID]

	var applicationList platformv1alpha1.ApplicationList
	if err := r.List(ctx, &applicationList,
		client.InNamespace(environment.Namespace),
		client.MatchingLabels{validation.LabelEnvironmentUUID: envUUID}); err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	environment.Status.ApplicationCount = int32(len(applicationList.Items))

	// Set phase
	environment.Status.Phase = "Ready"
	environment.Status.Message = "Environment is ready"

	// Update last reconcile time
	now := metav1.Now()
	environment.Status.LastReconcileTime = &now

	// Set condition
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "EnvironmentReady",
		Message:            "Environment is ready",
	}

	updated := false
	for i, existingCondition := range environment.Status.Conditions {
		if existingCondition.Type == condition.Type {
			environment.Status.Conditions[i] = condition
			updated = true
			break
		}
	}
	if !updated {
		environment.Status.Conditions = append(environment.Status.Conditions, condition)
	}

	// Update the status
	if err := r.Status().Update(ctx, environment); err != nil {
		return fmt.Errorf("failed to update Environment status: %w", err)
	}

	return nil
}

// emitEnvironmentPhaseChange sends a webhook if Notifier is configured and the phase actually changed
func (r *EnvironmentReconciler) emitEnvironmentPhaseChange(ctx context.Context, environment *platformv1alpha1.Environment, prev, next string) {
	if r.Notifier == nil {
		return
	}
	if prev == next {
		return
	}
	evt := webhooks.EnvironmentStatusEvent{
		Type:          "environment.status.changed",
		PreviousPhase: prev,
		NewPhase:      next,
		Environment:   *environment,
		Timestamp:     time.Now().UTC(),
	}
	_ = r.Notifier.NotifyEnvironmentStatusChange(ctx, evt)
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Environment{}).
		Named("environment").
		Complete(r)
}
