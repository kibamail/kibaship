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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects/finalizers,verbs=update

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
			// Owned objects are automatically garbage collected.
			log.Info("Project resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Project")
		return ctrl.Result{}, err
	}

	// Validate required labels
	if err := r.validateLabels(&project); err != nil {
		log.Error(err, "Project label validation failed")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled Project", "name", project.Name)
	return ctrl.Result{}, nil
}

// validateLabels validates that the project has the required UUID labels
func (r *ProjectReconciler) validateLabels(project *platformv1alpha1.Project) error {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	// Check if platform.kibaship.com/uuid label exists and is valid
	if uuid, exists := project.Labels["platform.kibaship.com/uuid"]; !exists {
		return fmt.Errorf("required label 'platform.kibaship.com/uuid' is missing")
	} else if !uuidRegex.MatchString(uuid) {
		return fmt.Errorf("label 'platform.kibaship.com/uuid' must be a valid UUID, got: %s", uuid)
	}

	// Check if platform.kibaship.com/workspace-uuid label exists and is valid (if present)
	if workspaceUUID, exists := project.Labels["platform.kibaship.com/workspace-uuid"]; exists {
		if !uuidRegex.MatchString(workspaceUUID) {
			return fmt.Errorf("label 'platform.kibaship.com/workspace-uuid' must be a valid UUID, got: %s", workspaceUUID)
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Project{}).
		Named("project").
		Complete(r)
}
