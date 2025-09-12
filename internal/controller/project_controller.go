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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
)

const (
	// ProjectFinalizerName is the finalizer name for project cleanup
	ProjectFinalizerName = "platform.kibaship.com/project-finalizer"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	NamespaceManager *NamespaceManager
	Validator        *ProjectValidator
}

// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="*",resources="*",verbs="*"

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Project object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Project instance
	var project platformv1alpha1.Project
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			log.Info("Project resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Project")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if project.DeletionTimestamp != nil {
		return r.handleProjectDeletion(ctx, &project)
	}

	// Add finalizer as the very first step (critical for cleanup)
	if !controllerutil.ContainsFinalizer(&project, ProjectFinalizerName) {
		controllerutil.AddFinalizer(&project, ProjectFinalizerName)
		if err := r.Update(ctx, &project); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Validate project labels (always check these)
	if err := r.Validator.ValidateRequiredLabels(&project); err != nil {
		log.Error(err, "Project label validation failed")
		r.updateStatusWithError(ctx, &project, "Failed", err.Error())
		return ctrl.Result{}, err
	}

	// Check if this is a new project by looking at status
	isNewProject := project.Status.Phase == "" || project.Status.Phase == "Pending"

	if isNewProject {
		// Set status to Pending for new projects
		if project.Status.Phase == "" {
			project.Status.Phase = "Pending"
			project.Status.Message = "Initializing project"
			if err := r.Status().Update(ctx, &project); err != nil {
				log.Error(err, "Failed to update project status to Pending")
				return ctrl.Result{}, err
			}
		}

		// Validate uniqueness for new projects (exclude this project)
		if err := r.Validator.CheckProjectNameUniqueness(ctx, project.Name, &project); err != nil {
			log.Error(err, "Project name uniqueness validation failed")
			r.updateStatusWithError(ctx, &project, "Failed", err.Error())
			return ctrl.Result{}, err
		}
	}

	// Create or update project namespace
	namespace, err := r.NamespaceManager.CreateProjectNamespace(ctx, &project)
	if err != nil {
		log.Error(err, "Failed to create project namespace")
		r.updateStatusWithError(ctx, &project, "Failed", err.Error())
		return ctrl.Result{}, err
	}

	// Update status to indicate project is ready
	if project.Status.Phase != "Ready" {
		project.Status.Phase = "Ready"
		project.Status.NamespaceName = namespace.Name
		project.Status.Message = "Project is ready"
		now := metav1.Now()
		project.Status.LastReconcileTime = &now
		if err := r.Status().Update(ctx, &project); err != nil {
			log.Error(err, "Failed to update project status to Ready")
			return ctrl.Result{}, err
		}
	}

	log.Info("Successfully reconciled Project",
		"name", project.Name,
		"namespace", namespace.Name,
		"phase", project.Status.Phase,
		"uuid", project.Labels[ProjectUUIDLabel])
	return ctrl.Result{}, nil
}

// handleProjectDeletion handles the deletion of a project and its resources
func (r *ProjectReconciler) handleProjectDeletion(ctx context.Context, project *platformv1alpha1.Project) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	log.Info("Handling project deletion", "project", project.Name)

	// Delete the project namespace (ignore NotFound errors for idempotency)
	if err := r.NamespaceManager.DeleteProjectNamespace(ctx, project); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete project namespace")
			return ctrl.Result{}, err
		}
		log.Info("Project namespace was already deleted", "project", project.Name)
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(project, ProjectFinalizerName)
	if err := r.Update(ctx, project); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Successfully deleted project", "project", project.Name)
	return ctrl.Result{}, nil
}

// updateStatusWithError updates the project status with error information
func (r *ProjectReconciler) updateStatusWithError(ctx context.Context, project *platformv1alpha1.Project, phase, message string) {
	project.Status.Phase = phase
	project.Status.Message = message
	now := metav1.Now()
	project.Status.LastReconcileTime = &now

	if err := r.Status().Update(ctx, project); err != nil {
		log := logf.FromContext(ctx)
		log.Error(err, "Failed to update project status with error")
	}
}

// NewProjectReconciler creates a new ProjectReconciler with required dependencies
func NewProjectReconciler(client client.Client, scheme *runtime.Scheme) *ProjectReconciler {
	return &ProjectReconciler{
		Client:           client,
		Scheme:           scheme,
		NamespaceManager: NewNamespaceManager(client),
		Validator:        NewProjectValidator(client),
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Project{}).
		Named("project").
		Complete(r)
}
