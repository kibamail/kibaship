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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

// PipelineRunStatusController watches PipelineRun status and updates Deployment conditions
type PipelineRunStatusController struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments/status,verbs=get;update;patch

func (r *PipelineRunStatusController) SetupWithManager(mgr ctrl.Manager) error {
	// Only watch PipelineRun status changes
	return ctrl.NewControllerManagedBy(mgr).
		For(&tektonv1.PipelineRun{}).
		WithEventFilter(predicate.ResourceVersionChangedPredicate{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 50, // High concurrency for status updates
		}).
		Named("pipelinerun-status").
		Complete(r)
}

func (r *PipelineRunStatusController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var pipelineRun tektonv1.PipelineRun
	if err := r.Get(ctx, req.NamespacedName, &pipelineRun); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Find owner Deployment via OwnerReferences (Kubernetes-native way)
	deploymentName := ""
	for _, owner := range pipelineRun.GetOwnerReferences() {
		if owner.Kind == DeploymentKind && owner.APIVersion == platformv1alpha1.GroupVersion.String() {
			deploymentName = owner.Name
			break
		}
	}

	if deploymentName == "" {
		// Not owned by our Deployment CRD
		return ctrl.Result{}, nil
	}

	var deployment platformv1alpha1.Deployment
	if err := r.Get(ctx, client.ObjectKey{
		Name:      deploymentName,
		Namespace: pipelineRun.Namespace,
	}, &deployment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Extract PipelineRun status
	succeededCondition := pipelineRun.Status.GetCondition("Succeeded")
	if succeededCondition == nil {
		// No status yet
		return ctrl.Result{}, nil
	}

	// CRITICAL: Idempotency check - compare ResourceVersion
	lastProcessedVersion := deployment.Annotations["platform.kibaship.com/last-pipelinerun-version"]
	currentVersion := pipelineRun.ResourceVersion

	if lastProcessedVersion == currentVersion {
		log.V(1).Info("Already processed this PipelineRun version",
			"version", currentVersion)
		return ctrl.Result{}, nil
	}

	// Update Deployment condition (not phase - that's DeploymentProgressController's job)
	condition := metav1.Condition{
		Type:               "PipelineRunReady",
		Status:             metav1.ConditionStatus(succeededCondition.Status),
		ObservedGeneration: deployment.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             succeededCondition.Reason,
		Message:            succeededCondition.Message,
	}

	meta.SetStatusCondition(&deployment.Status.Conditions, condition)

	// Update annotation to prevent reprocessing
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}
	deployment.Annotations["platform.kibaship.com/last-pipelinerun-version"] = currentVersion

	// Atomic update: annotation + condition
	if err := r.Update(ctx, &deployment); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Status().Update(ctx, &deployment); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Updated Deployment condition from PipelineRun",
		"deployment", deploymentName,
		"condition", condition.Type,
		"status", condition.Status)

	return ctrl.Result{}, nil
}
