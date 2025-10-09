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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
)

const (
	// CrashLoopBackOffReason represents the reason for crash loop back off state
	CrashLoopBackOffReason = "CrashLoopBackOff"
)

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments/status,verbs=get;update;patch

// DeploymentStatusWatcherReconciler watches K8s Deployments and mirrors their ready status
// into Deployment CR conditions (correlated via label platform.kibaship.com/deployment-uuid)
type DeploymentStatusWatcherReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *DeploymentStatusWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the K8s Deployment
	var k8sDep appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &k8sDep); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Extract deployment UUID from labels
	labels := k8sDep.GetLabels()
	deploymentUUID := ""
	if labels != nil {
		deploymentUUID = labels["platform.kibaship.com/deployment-uuid"]
	}
	if deploymentUUID == "" {
		// Not ours
		return ctrl.Result{}, nil
	}

	// Find the Deployment CR
	deploymentName := fmt.Sprintf("deployment-%s", deploymentUUID)
	var dep platformv1alpha1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: k8sDep.Namespace}, &dep); err != nil {
		if errors.IsNotFound(err) {
			// Deployment CR might have been deleted; ignore
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get Deployment CR for K8s Deployment", "deployment", deploymentName)
		return ctrl.Result{}, err
	}

	// Check if we've already processed this status to avoid infinite loops
	lastProcessedGeneration := ""
	lastProcessedReady := ""
	if dep.Annotations != nil {
		lastProcessedGeneration = dep.Annotations["platform.kibaship.com/last-processed-k8s-generation"]
		lastProcessedReady = dep.Annotations["platform.kibaship.com/last-processed-k8s-ready"]
	}
	currentGeneration := fmt.Sprintf("%d", k8sDep.Generation)
	currentReady := fmt.Sprintf("%d/%d", k8sDep.Status.ReadyReplicas, k8sDep.Status.Replicas)

	// Skip if we've already processed this generation and ready status
	if lastProcessedGeneration == currentGeneration && lastProcessedReady == currentReady {
		logger.V(1).Info("Skipping K8s Deployment processing - already processed",
			"generation", currentGeneration, "ready", currentReady)
		return ctrl.Result{}, nil
	}

	// Check for crash loop before determining condition status
	crashLooping, crashMessage := r.isPodsCrashLooping(ctx, &k8sDep)

	// Derive condition status from K8s Deployment
	var conditionStatus metav1.ConditionStatus
	var reason, message string

	if crashLooping {
		// Pods are crash looping - mark as failed
		conditionStatus = metav1.ConditionFalse
		reason = CrashLoopBackOffReason
		message = crashMessage
	} else if k8sDep.Status.ReadyReplicas > 0 {
		conditionStatus = metav1.ConditionTrue
		reason = "PodsReady"
		message = fmt.Sprintf("%d/%d pods ready", k8sDep.Status.ReadyReplicas, k8sDep.Status.Replicas)
	} else if k8sDep.Status.UnavailableReplicas > 0 {
		conditionStatus = metav1.ConditionFalse
		reason = "PodsNotReady"
		message = fmt.Sprintf("%d pods unavailable", k8sDep.Status.UnavailableReplicas)
	} else {
		conditionStatus = metav1.ConditionUnknown
		reason = "Deploying"
		message = "Waiting for pods to be scheduled"
	}

	// Update condition reflecting K8s Deployment state
	cond := metav1.Condition{
		Type:               "K8sDeploymentReady",
		Status:             conditionStatus,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	upsertCondition(&dep.Status.Conditions, cond)

	// Mark this generation and ready status as processed
	if dep.Annotations == nil {
		dep.Annotations = make(map[string]string)
	}
	dep.Annotations["platform.kibaship.com/last-processed-k8s-generation"] = currentGeneration
	dep.Annotations["platform.kibaship.com/last-processed-k8s-ready"] = currentReady

	// Persist status
	if err := r.Status().Update(ctx, &dep); err != nil {
		logger.Error(err, "update Deployment status failed", "deployment", fmt.Sprintf("%s/%s", dep.Namespace, dep.Name))
		return ctrl.Result{}, err
	}

	logger.V(1).Info("Updated Deployment CR with K8s Deployment status",
		"ready", currentReady, "condition", conditionStatus)

	return ctrl.Result{}, nil
}

// isPodsCrashLooping checks if any pods are crash looping
// Returns true if restart count exceeds threshold (3 restarts)
func (r *DeploymentStatusWatcherReconciler) isPodsCrashLooping(ctx context.Context, k8sDep *appsv1.Deployment) (bool, string) {
	const restartThreshold = 3

	// List pods for this deployment
	var podList corev1.PodList
	if err := r.List(ctx, &podList, client.InNamespace(k8sDep.Namespace), client.MatchingLabels(k8sDep.Spec.Selector.MatchLabels)); err != nil {
		// If we can't list pods, don't consider it crash looping
		return false, ""
	}

	if len(podList.Items) == 0 {
		return false, ""
	}

	// Check each pod for crash loops
	for _, pod := range podList.Items {
		// Check container statuses
		for _, containerStatus := range pod.Status.ContainerStatuses {
			// Check restart count
			if containerStatus.RestartCount >= restartThreshold {
				return true, fmt.Sprintf("Container %s has restarted %d times (threshold: %d)",
					containerStatus.Name, containerStatus.RestartCount, restartThreshold)
			}

			// Check if currently in CrashLoopBackOff state
			if containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason == CrashLoopBackOffReason {
				return true, fmt.Sprintf("Container %s is in CrashLoopBackOff state", containerStatus.Name)
			}
		}
	}

	return false, ""
}

func (r *DeploymentStatusWatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Only watch K8s Deployments with our label
	// Only trigger on status changes (not spec changes)
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Trigger on create to set initial status
			return hasDeploymentUUIDLabel(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Only trigger if status changed (ReadyReplicas, etc.)
			if !hasDeploymentUUIDLabel(e.ObjectNew) {
				return false
			}
			oldDep, ok := e.ObjectOld.(*appsv1.Deployment)
			if !ok {
				return false
			}
			newDep, ok := e.ObjectNew.(*appsv1.Deployment)
			if !ok {
				return false
			}
			// Trigger only when status changes
			return oldDep.Status.ReadyReplicas != newDep.Status.ReadyReplicas ||
				oldDep.Status.Replicas != newDep.Status.Replicas ||
				oldDep.Status.UnavailableReplicas != newDep.Status.UnavailableReplicas
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(pred).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 50, // Higher concurrency for status watching
		}).
		Named("deployment-status-watcher").
		Complete(r)
}

func hasDeploymentUUIDLabel(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}
	return labels["platform.kibaship.com/deployment-uuid"] != ""
}
