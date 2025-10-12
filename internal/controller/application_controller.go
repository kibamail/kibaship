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
	"github.com/kibamail/kibaship-operator/pkg/validation"
	"github.com/kibamail/kibaship-operator/pkg/webhooks"
)

const (
	// ApplicationFinalizerName is the finalizer name for Application resources
	ApplicationFinalizerName = "platform.operator.kibaship.com/application-finalizer"

	// DefaultDomainType is the default domain type for ApplicationDomains
	DefaultDomainType = "default"
)

const applicationPhaseReady = "Ready"

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Notifier webhooks.Notifier
}

// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applications/finalizers,verbs=update
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applicationdomains,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=environments,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

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

	// Delete all ApplicationDomains associated with this Application
	if err := r.deleteAssociatedDomains(ctx, app); err != nil {
		log.Error(err, "Failed to delete associated ApplicationDomains")
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

	// Ensure environment variables secret exists for all applications
	secretRefUpdated, err := r.ensureApplicationEnvSecret(ctx, app)
	if err != nil {
		log.Error(err, "Failed to ensure application env secret")
		return ctrl.Result{}, err
	}
	if secretRefUpdated {
		// Update the Application spec with the secret reference
		if err := r.Update(ctx, app); err != nil {
			log.Error(err, "Failed to update Application with env secret ref")
			return ctrl.Result{}, err
		}
		log.Info("Updated Application with env secret reference")
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle ApplicationDomain creation for GitRepository applications
	if err := r.handleApplicationDomains(ctx, app); err != nil {
		log.Error(err, "Failed to handle ApplicationDomains")
		return ctrl.Result{}, err
	}

	// Track previous phase before updating status
	prevPhase := app.Status.Phase

	// Update Application status
	if err := r.updateApplicationStatus(ctx, app); err != nil {
		log.Error(err, "Failed to update Application status")
		return ctrl.Result{}, err
	}

	// Emit webhook on phase transition
	r.emitApplicationPhaseChange(ctx, app, prevPhase, app.Status.Phase)

	log.Info("Successfully reconciled Application")
	return ctrl.Result{}, nil
}

// ensureApplicationEnvSecret ensures that an environment variables secret exists for the application
func (r *ApplicationReconciler) ensureApplicationEnvSecret(ctx context.Context, app *platformv1alpha1.Application) (bool, error) {
	log := logf.FromContext(ctx).WithValues("application", app.Name, "namespace", app.Namespace)

	appUUID := app.Labels[validation.LabelResourceUUID]
	if appUUID == "" {
		return false, fmt.Errorf("application UUID label not found")
	}

	secretName := fmt.Sprintf("env-%s", appUUID)

	// Get current env ref based on application type
	currentEnvRef := r.getCurrentEnvRef(app)

	// Check if secret ref is already set correctly
	if currentEnvRef != nil && currentEnvRef.Name == secretName {
		// Secret ref already set correctly, check if secret exists
		var existingSecret corev1.Secret
		err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: app.Namespace}, &existingSecret)
		if err == nil {
			// Secret exists and ref is set correctly
			return false, nil
		}
		if !errors.IsNotFound(err) {
			return false, fmt.Errorf("failed to get secret: %w", err)
		}
		// Secret doesn't exist, create it
	} else if currentEnvRef != nil {
		// Secret ref is set but to wrong name, update it
		log.Info("Updating env secret ref", "oldRef", currentEnvRef.Name, "newRef", secretName)
	}

	// Create the secret if it doesn't exist
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: app.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":        "kibaship-operator",
				validation.LabelApplicationUUID:       appUUID,
				"platform.operator.kibaship.com/type": "application-env-vars",
				validation.LabelProjectUUID:           app.Labels[validation.LabelProjectUUID],
				validation.LabelEnvironmentUUID:       app.Labels[validation.LabelEnvironmentUUID],
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{},
	}

	// Set owner reference to the application
	if err := controllerutil.SetControllerReference(app, secret, r.Scheme); err != nil {
		return false, fmt.Errorf("failed to set owner reference on secret: %w", err)
	}

	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: app.Namespace}, &corev1.Secret{})
	if errors.IsNotFound(err) {
		if err := r.Create(ctx, secret); err != nil {
			return false, fmt.Errorf("failed to create env secret: %w", err)
		}
		log.Info("Created env secret for application", "secret", secretName)
	} else if err != nil {
		return false, fmt.Errorf("failed to check for existing secret: %w", err)
	}

	// Set the secret ref on the application based on type
	secretRef := &corev1.LocalObjectReference{Name: secretName}
	updated, err := r.setEnvRefOnApplication(app, secretRef, secretName)
	if err != nil {
		return false, err
	}

	return updated, nil
}

// getCurrentEnvRef returns the current environment secret reference for the application
func (r *ApplicationReconciler) getCurrentEnvRef(app *platformv1alpha1.Application) *corev1.LocalObjectReference {
	switch app.Spec.Type {
	case platformv1alpha1.ApplicationTypeGitRepository:
		if app.Spec.GitRepository != nil {
			return app.Spec.GitRepository.Env
		}
	case platformv1alpha1.ApplicationTypeDockerImage:
		if app.Spec.DockerImage != nil {
			return app.Spec.DockerImage.Env
		}
	case platformv1alpha1.ApplicationTypeMySQL:
		if app.Spec.MySQL != nil {
			return app.Spec.MySQL.Env
		}
	case platformv1alpha1.ApplicationTypeMySQLCluster:
		if app.Spec.MySQLCluster != nil {
			return app.Spec.MySQLCluster.Env
		}
	case platformv1alpha1.ApplicationTypePostgres:
		if app.Spec.Postgres != nil {
			return app.Spec.Postgres.Env
		}
	case platformv1alpha1.ApplicationTypePostgresCluster:
		if app.Spec.PostgresCluster != nil {
			return app.Spec.PostgresCluster.Env
		}
	}
	return nil
}

// setEnvRefOnApplication sets the environment secret reference on the application and returns whether it was updated
func (r *ApplicationReconciler) setEnvRefOnApplication(app *platformv1alpha1.Application, secretRef *corev1.LocalObjectReference, secretName string) (bool, error) {
	switch app.Spec.Type {
	case platformv1alpha1.ApplicationTypeGitRepository:
		if app.Spec.GitRepository == nil {
			return false, fmt.Errorf("git repository config is nil")
		}
		if app.Spec.GitRepository.Env == nil || app.Spec.GitRepository.Env.Name != secretName {
			app.Spec.GitRepository.Env = secretRef
			return true, nil
		}
	case platformv1alpha1.ApplicationTypeDockerImage:
		if app.Spec.DockerImage == nil {
			return false, fmt.Errorf("docker image config is nil")
		}
		if app.Spec.DockerImage.Env == nil || app.Spec.DockerImage.Env.Name != secretName {
			app.Spec.DockerImage.Env = secretRef
			return true, nil
		}
	case platformv1alpha1.ApplicationTypeMySQL:
		if app.Spec.MySQL == nil {
			return false, fmt.Errorf("mysql config is nil")
		}
		if app.Spec.MySQL.Env == nil || app.Spec.MySQL.Env.Name != secretName {
			app.Spec.MySQL.Env = secretRef
			return true, nil
		}
	case platformv1alpha1.ApplicationTypeMySQLCluster:
		if app.Spec.MySQLCluster == nil {
			return false, fmt.Errorf("mysql cluster config is nil")
		}
		if app.Spec.MySQLCluster.Env == nil || app.Spec.MySQLCluster.Env.Name != secretName {
			app.Spec.MySQLCluster.Env = secretRef
			return true, nil
		}
	case platformv1alpha1.ApplicationTypePostgres:
		if app.Spec.Postgres == nil {
			return false, fmt.Errorf("postgres config is nil")
		}
		if app.Spec.Postgres.Env == nil || app.Spec.Postgres.Env.Name != secretName {
			app.Spec.Postgres.Env = secretRef
			return true, nil
		}
	case platformv1alpha1.ApplicationTypePostgresCluster:
		if app.Spec.PostgresCluster == nil {
			return false, fmt.Errorf("postgres cluster config is nil")
		}
		if app.Spec.PostgresCluster.Env == nil || app.Spec.PostgresCluster.Env.Name != secretName {
			app.Spec.PostgresCluster.Env = secretRef
			return true, nil
		}
	}
	return false, nil
}

// ensureUUIDLabels ensures that the Application has the correct UUID and slug labels
func (r *ApplicationReconciler) ensureUUIDLabels(ctx context.Context, app *platformv1alpha1.Application) (bool, error) {
	log := logf.FromContext(ctx).WithValues("application", app.Name, "namespace", app.Namespace)

	// First validate the labels using the centralized labeling system
	resourceLabeler := NewResourceLabeler(r.Client)
	if err := resourceLabeler.ValidateApplicationLabeling(ctx, app); err != nil {
		return false, fmt.Errorf("application label validation failed: %w", err)
	}

	if app.Labels == nil {
		app.Labels = make(map[string]string)
	}

	labelsUpdated := false

	// Validate that Application has its own UUID label (should be set by PaaS)
	if _, exists := app.Labels[validation.LabelResourceUUID]; !exists {
		return false, fmt.Errorf("application must have label '%s' set by PaaS system", validation.LabelResourceUUID)
	}

	// Validate that Application has its own slug label (should be set by PaaS)
	if _, exists := app.Labels[validation.LabelResourceSlug]; !exists {
		return false, fmt.Errorf("application must have label '%s' set by PaaS system", validation.LabelResourceSlug)
	}

	// Get environment UUID first (should already be set by PaaS)
	if _, exists := app.Labels[validation.LabelEnvironmentUUID]; !exists {
		return false, fmt.Errorf("application must have label '%s' set by PaaS system", validation.LabelEnvironmentUUID)
	}

	// Get project UUID from environment if not set
	if _, exists := app.Labels[validation.LabelProjectUUID]; !exists {
		projectUUID, err := r.getProjectUUIDFromEnvironment(ctx, app)
		if err != nil {
			return false, fmt.Errorf("failed to get project UUID from environment: %w", err)
		}
		app.Labels[validation.LabelProjectUUID] = projectUUID
		labelsUpdated = true
		log.Info("Set project UUID label from environment", "projectUUID", projectUUID)
	}

	return labelsUpdated, nil
}

// getProjectUUIDFromEnvironment retrieves the project UUID from the referenced environment
func (r *ApplicationReconciler) getProjectUUIDFromEnvironment(ctx context.Context, app *platformv1alpha1.Application) (string, error) {
	var environment platformv1alpha1.Environment
	err := r.Get(ctx, types.NamespacedName{
		Name:      app.Spec.EnvironmentRef.Name,
		Namespace: app.Namespace,
	}, &environment)

	if err != nil {
		return "", fmt.Errorf("failed to get referenced environment %s: %w", app.Spec.EnvironmentRef.Name, err)
	}

	projectUUID, exists := environment.Labels[validation.LabelProjectUUID]
	if !exists {
		return "", fmt.Errorf("referenced environment %s does not have required project UUID label", environment.Name)
	}

	return projectUUID, nil
}

// updateApplicationStatus updates the Application status
func (r *ApplicationReconciler) updateApplicationStatus(ctx context.Context, app *platformv1alpha1.Application) error {
	// Update Application status to reflect the current state

	// Update the Application status to reflect the current state
	app.Status.ObservedGeneration = app.Generation
	app.Status.Phase = applicationPhaseReady

	// Set condition for application readiness
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "ApplicationReady",
		Message:            "Application is ready",
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

// handleApplicationDomains handles the creation and management of ApplicationDomains for GitRepository applications
func (r *ApplicationReconciler) handleApplicationDomains(ctx context.Context, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx).WithValues("application", app.Name, "namespace", app.Namespace)

	// All application types get domains

	// Check if default domain already exists
	var domains platformv1alpha1.ApplicationDomainList
	if err := r.List(ctx, &domains,
		client.InNamespace(app.Namespace),
		client.MatchingLabels{ApplicationDomainLabelApplication: app.Name},
	); err != nil {
		return fmt.Errorf("failed to list existing domains: %v", err)
	}

	// Find existing default domain
	var defaultDomain *platformv1alpha1.ApplicationDomain
	for _, domain := range domains.Items {
		if domain.Spec.Default {
			defaultDomain = &domain
			break
		}
	}

	// Create default domain if it doesn't exist
	if defaultDomain == nil {
		log.Info("Creating default ApplicationDomain for GitRepository application")
		return r.createDefaultDomain(ctx, app)
	}

	log.V(1).Info("Default ApplicationDomain already exists", "domain", defaultDomain.Spec.Domain)
	return nil
}

// createDefaultDomain creates a default ApplicationDomain for a GitRepository application
func (r *ApplicationReconciler) createDefaultDomain(ctx context.Context, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx).WithValues("application", app.Name, "namespace", app.Namespace)

	// Get operator configuration
	config, err := GetOperatorConfig()
	if err != nil {
		return fmt.Errorf("failed to get operator configuration: %v", err)
	}

	// Use the application UUID from labels for subdomain generation (not the CR name)
	// The CR name includes domain suffix which would result in duplicate domains
	appUUID := app.Labels[validation.LabelResourceUUID]
	if appUUID == "" {
		return fmt.Errorf("application missing required label %s", validation.LabelResourceUUID)
	}

	// Generate unique subdomain based on application UUID
	subdomain, err := GenerateSubdomain(appUUID)
	if err != nil {
		return fmt.Errorf("failed to generate subdomain: %v", err)
	}

	// Generate full domain based on application type
	fullDomain, port, err := GenerateFullDomainForApplicationType(subdomain, app.Spec.Type)
	if err != nil {
		return fmt.Errorf("failed to generate full domain: %v", err)
	}

	// Create ApplicationDomain resource with proper labeling
	domainName := GenerateApplicationDomainName(app.Name, DefaultDomainType)

	// Generate UUID and slug for the domain
	domainUUID := validation.GenerateUUID()
	domainSlug := DefaultDomainType

	// Prepare parent labels to inherit from application
	parentLabels := map[string]string{
		validation.LabelProjectUUID:       app.Labels[validation.LabelProjectUUID],
		validation.LabelWorkspaceUUID:     app.Labels[validation.LabelWorkspaceUUID],
		validation.LabelApplicationUUID:   app.Labels[validation.LabelResourceUUID], // Application's UUID becomes application-uuid for ApplicationDomain
		ApplicationDomainLabelApplication: app.Name,
		ApplicationDomainLabelDomainType:  DefaultDomainType,
	}

	// Debug logging to understand label propagation
	log.Info("Creating ApplicationDomain with labels",
		"applicationUUID", app.Labels[validation.LabelResourceUUID],
		"projectUUID", app.Labels[validation.LabelProjectUUID],
		"workspaceUUID", app.Labels[validation.LabelWorkspaceUUID],
		"domainUUID", domainUUID)

	domain := &platformv1alpha1.ApplicationDomain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      domainName,
			Namespace: app.Namespace,
		},
		Spec: platformv1alpha1.ApplicationDomainSpec{
			ApplicationRef: corev1.LocalObjectReference{Name: app.Name},
			Domain:         fullDomain,
			Port:           port,
			Type:           platformv1alpha1.ApplicationDomainTypeDefault,
			Default:        true,
			TLSEnabled:     true,
		},
	}

	// Apply comprehensive labeling using the centralized system
	// Note: ApplyLabelsToResource will set validation.LabelResourceUUID to domainUUID
	ApplyLabelsToResource(domain, domainUUID, domainSlug, parentLabels)

	// Debug: Log the labels that were applied to the ApplicationDomain
	log.Info("ApplicationDomain labels after ApplyLabelsToResource", "labels", domain.GetLabels())

	// Set owner reference to ensure cleanup when application is deleted
	if err := controllerutil.SetControllerReference(app, domain, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %v", err)
	}

	// Create the ApplicationDomain
	if err := r.Create(ctx, domain); err != nil {
		return fmt.Errorf("failed to create ApplicationDomain: %v", err)
	}

	log.Info("Successfully created default ApplicationDomain", "domain", fullDomain, "port", config.DefaultPort)
	return nil
}

// deleteAssociatedDomains deletes all ApplicationDomains associated with the Application
func (r *ApplicationReconciler) deleteAssociatedDomains(ctx context.Context, app *platformv1alpha1.Application) error {
	log := logf.FromContext(ctx).WithValues("application", app.Name, "namespace", app.Namespace)

	// Use label selector to efficiently find only ApplicationDomains associated with this Application
	var domainList platformv1alpha1.ApplicationDomainList
	if err := r.List(ctx, &domainList,
		client.InNamespace(app.Namespace),
		client.MatchingLabels{ApplicationDomainLabelApplication: app.Name}); err != nil {
		return fmt.Errorf("failed to list ApplicationDomains: %w", err)
	}

	// Delete all matching ApplicationDomains
	for _, domain := range domainList.Items {
		log.Info("Deleting associated ApplicationDomain", "domain", domain.Spec.Domain)

		if err := r.Delete(ctx, &domain); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete ApplicationDomain %s: %w", domain.Name, err)
		}
	}

	return nil
}

// emitApplicationPhaseChange sends a webhook if Notifier is configured and the phase actually changed.
func (r *ApplicationReconciler) emitApplicationPhaseChange(ctx context.Context, app *platformv1alpha1.Application, prev, next string) {
	if r.Notifier == nil {
		return
	}
	if prev == next {
		return
	}
	evt := webhooks.ApplicationStatusEvent{
		Type:          "application.status.changed",
		PreviousPhase: prev,
		NewPhase:      next,
		Application:   *app,
		Timestamp:     time.Now().UTC(),
	}
	_ = r.Notifier.NotifyApplicationStatusChange(ctx, evt)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Application{}).
		Named("application").
		Complete(r)
}
