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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/webhooks"
)

// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments/status,verbs=get;update;patch

// PipelineRunWatcherReconciler watches Tekton PipelineRuns and mirrors their status into Deployments
// (correlated via label deployment.kibaship.com/name), then emits webhooks.
type PipelineRunWatcherReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Notifier webhooks.Notifier
}

const (
	condTrue    = "True"
	condFalse   = "False"
	condUnknown = "Unknown"
)

func (r *PipelineRunWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the PipelineRun as unstructured
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{Group: "tekton.dev", Version: "v1", Kind: "PipelineRun"})
	if err := r.Get(ctx, req.NamespacedName, u); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	labels := u.GetLabels()
	depName := ""
	if labels != nil {
		depName = labels["deployment.kibaship.com/name"]
	}
	if depName == "" {
		// not ours
		return ctrl.Result{}, nil
	}

	// Load the Deployment
	var dep platformv1alpha1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Name: depName, Namespace: u.GetNamespace()}, &dep); err != nil {
		if errors.IsNotFound(err) {
			// deployment might have been deleted; ignore
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get Deployment for PipelineRun", "deployment", depName)
		return ctrl.Result{}, err
	}

	// Check if we've already processed this PipelineRun status to avoid infinite loops
	lastProcessedGeneration := ""
	if dep.Annotations != nil {
		lastProcessedGeneration = dep.Annotations["platform.kibaship.com/last-processed-pipelinerun-generation"]
	}
	currentGeneration := fmt.Sprintf("%d", u.GetGeneration())

	// Extract current PipelineRun status for comparison
	currentStatus, currentReason, currentMessage := extractPRSucceeded(u)
	lastProcessedStatus := ""
	if dep.Annotations != nil {
		lastProcessedStatus = dep.Annotations["platform.kibaship.com/last-processed-pipelinerun-status"]
	}

	// Skip processing if we've already handled this generation and status hasn't changed
	if lastProcessedGeneration == currentGeneration && lastProcessedStatus == currentStatus {
		logger.V(1).Info("Skipping PipelineRun processing - already processed this generation and status",
			"generation", currentGeneration, "status", currentStatus)
		return ctrl.Result{}, nil
	}

	prev := string(dep.Status.Phase)

	// Derive deployment phase from PipelineRun condition Succeeded
	status, reason, message := currentStatus, currentReason, currentMessage
	switch status {
	case condTrue:
		dep.Status.Phase = platformv1alpha1.DeploymentPhaseSucceeded
	case condFalse:
		dep.Status.Phase = platformv1alpha1.DeploymentPhaseFailed
	default:
		dep.Status.Phase = platformv1alpha1.DeploymentPhaseRunning
	}

	// Update a condition reflecting PR state
	cond := metav1.Condition{
		Type:               "PipelineRunReady",
		Status:             toConditionStatus(status),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	upsertCondition(&dep.Status.Conditions, cond)

	// Mark this generation and status as processed to prevent infinite loops
	if dep.Annotations == nil {
		dep.Annotations = make(map[string]string)
	}
	dep.Annotations["platform.kibaship.com/last-processed-pipelinerun-generation"] = currentGeneration
	dep.Annotations["platform.kibaship.com/last-processed-pipelinerun-status"] = status

	// Persist status
	if err := r.Status().Update(ctx, &dep); err != nil {
		logger.Error(err, "update Deployment status failed", "dep", fmt.Sprintf("%s/%s", dep.Namespace, dep.Name))
		return ctrl.Result{}, err
	}

	// Emit optimized webhook if phase changed
	if r.Notifier != nil && prev != string(dep.Status.Phase) {
		evt := webhooks.OptimizedDeploymentStatusEvent{
			Type:          "deployment.status.changed",
			PreviousPhase: prev,
			NewPhase:      string(dep.Status.Phase),
			DeploymentRef: struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
				UUID      string `json:"uuid"`
				Phase     string `json:"phase"`
				Slug      string `json:"slug"`
			}{
				Name:      dep.Name,
				Namespace: dep.Namespace,
				UUID:      dep.GetUUID(),
				Phase:     string(dep.Status.Phase),
				Slug:      dep.GetSlug(),
			},
			PipelineRunRef: &struct {
				Name   string `json:"name"`
				Status string `json:"status"`
				Reason string `json:"reason"`
			}{
				Name:   u.GetName(),
				Status: status,
				Reason: reason,
			},
			Timestamp: time.Now().UTC(),
		}
		_ = r.Notifier.NotifyOptimizedDeploymentStatusChange(ctx, evt)
	}

	return ctrl.Result{}, nil
}

func (r *PipelineRunWatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{Group: "tekton.dev", Version: "v1", Kind: "PipelineRun"})
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool { return false },
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(u).
		WithEventFilter(pred).
		Complete(r)
}

// extractPRSucceeded pulls .status.conditions[type=="Succeeded"]
func extractPRSucceeded(u *unstructured.Unstructured) (status, reason, message string) {
	conds, found, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	if !found {
		return condUnknown, "", ""
	}
	for _, c := range conds {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := m["type"].(string); t == "Succeeded" {
			status, _ = m["status"].(string)
			reason, _ = m["reason"].(string)
			message, _ = m["message"].(string)
			if status == "" {
				status = condUnknown
			}
			return
		}
	}
	return condUnknown, "", ""
}

func toConditionStatus(s string) metav1.ConditionStatus {
	switch s {
	case condTrue:
		return metav1.ConditionTrue
	case condFalse:
		return metav1.ConditionFalse
	default:
		return metav1.ConditionUnknown
	}
}

func upsertCondition(conds *[]metav1.Condition, c metav1.Condition) {
	list := *conds
	for i := range list {
		if list[i].Type == c.Type {
			list[i] = c
			*conds = list
			return
		}
	}
	*conds = append(*conds, c)
}
