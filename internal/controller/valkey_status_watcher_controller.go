package controller

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:rbac:groups=hyperspike.io,resources=valkeys,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments/status,verbs=get;update;patch

// ValkeyStatusWatcherReconciler watches Valkey resources and mirrors their ready status
// into Deployment CR conditions (correlated via owner references)
type ValkeyStatusWatcherReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ValkeyStatusWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Valkey resource
	valkeyResource := &unstructured.Unstructured{}
	valkeyResource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "hyperspike.io",
		Version: "v1",
		Kind:    "Valkey",
	})

	if err := r.Get(ctx, req.NamespacedName, valkeyResource); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Find the owning Deployment CR via owner references
	ownerRefs := valkeyResource.GetOwnerReferences()
	var deploymentName string
	for _, ownerRef := range ownerRefs {
		if ownerRef.Kind == "Deployment" && ownerRef.APIVersion == "platform.operator.kibaship.com/v1alpha1" {
			deploymentName = ownerRef.Name
			break
		}
	}

	if deploymentName == "" {
		// Not owned by our Deployment CR, ignore
		return ctrl.Result{}, nil
	}

	// Get the Deployment CR
	var dep platformv1alpha1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: req.Namespace}, &dep); err != nil {
		if errors.IsNotFound(err) {
			// Deployment CR might have been deleted; ignore
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to get Deployment CR for Valkey resource", "deployment", deploymentName)
		return ctrl.Result{}, err
	}

	// Check if we've already processed this status to avoid infinite loops
	lastProcessedStatus := ""
	if dep.Annotations != nil {
		lastProcessedStatus = dep.Annotations["platform.kibaship.com/last-processed-valkey-status"]
	}

	// Get current Valkey status
	ready, found, err := unstructured.NestedBool(valkeyResource.Object, "status", "ready")
	if err != nil {
		logger.Error(err, "failed to get Valkey ready status")
		return ctrl.Result{}, err
	}

	currentStatus := fmt.Sprintf("ready=%t,found=%t", ready, found)

	// Skip if we've already processed this status
	if lastProcessedStatus == currentStatus {
		logger.V(1).Info("Skipping Valkey status processing - already processed",
			"status", currentStatus)
		return ctrl.Result{}, nil
	}

	// Determine condition status based on Valkey readiness
	var conditionStatus metav1.ConditionStatus
	var reason, message string

	if !found {
		conditionStatus = metav1.ConditionUnknown
		reason = "StatusNotAvailable"
		message = "Valkey status not yet available"
	} else if ready {
		conditionStatus = metav1.ConditionTrue
		reason = "ValkeyReady"
		message = "Valkey instance is ready and accepting connections"
	} else {
		conditionStatus = metav1.ConditionFalse
		reason = "ValkeyNotReady"
		message = "Valkey instance is not ready"
	}

	// Update condition reflecting Valkey state
	cond := metav1.Condition{
		Type:               "ValkeyReady",
		Status:             conditionStatus,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	upsertCondition(&dep.Status.Conditions, cond)

	// Mark this status as processed
	if dep.Annotations == nil {
		dep.Annotations = make(map[string]string)
	}
	dep.Annotations["platform.kibaship.com/last-processed-valkey-status"] = currentStatus

	// Persist status
	if err := r.Status().Update(ctx, &dep); err != nil {
		logger.Error(err, "update Deployment status failed", "deployment", fmt.Sprintf("%s/%s", dep.Namespace, dep.Name))
		return ctrl.Result{}, err
	}

	logger.V(1).Info("Updated Deployment CR with Valkey status",
		"ready", ready, "condition", conditionStatus)

	return ctrl.Result{}, nil
}

func (r *ValkeyStatusWatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Watch Valkey resources and trigger on status changes
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Trigger on create to set initial status
			return hasValkeyOwnerReference(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Only trigger if owned by our Deployment CR
			if !hasValkeyOwnerReference(e.ObjectNew) {
				return false
			}

			// Check if status.ready changed
			oldReady, oldFound, _ := unstructured.NestedBool(e.ObjectOld.(*unstructured.Unstructured).Object, "status", "ready")
			newReady, newFound, _ := unstructured.NestedBool(e.ObjectNew.(*unstructured.Unstructured).Object, "status", "ready")

			// Trigger if ready status or found status changed
			return oldReady != newReady || oldFound != newFound
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}

	// Create an unstructured object for Valkey resources
	valkeyObj := &unstructured.Unstructured{}
	valkeyObj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "hyperspike.io",
		Version: "v1",
		Kind:    "Valkey",
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(valkeyObj).
		WithEventFilter(pred).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 50, // Higher concurrency for status watching
		}).
		Named("valkey-status-watcher").
		Complete(r)
}

func hasValkeyOwnerReference(obj client.Object) bool {
	ownerRefs := obj.GetOwnerReferences()
	for _, ownerRef := range ownerRefs {
		if ownerRef.Kind == "Deployment" &&
			strings.Contains(ownerRef.APIVersion, "platform.operator.kibaship.com") {
			return true
		}
	}
	return false
}
