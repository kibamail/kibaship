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

	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:rbac:groups=mysql.oracle.com,resources=innodbclusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=deployments/status,verbs=get;update;patch

// MySQLStatusWatcherReconciler watches InnoDBCluster resources and mirrors their ready status
// into Deployment CR conditions (correlated via owner references)
type MySQLStatusWatcherReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *MySQLStatusWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the InnoDBCluster resource
	mysqlResource := &unstructured.Unstructured{}
	mysqlResource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "mysql.oracle.com",
		Version: "v2",
		Kind:    "InnoDBCluster",
	})

	if err := r.Get(ctx, req.NamespacedName, mysqlResource); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Find the owning Deployment CR via owner references
	ownerRefs := mysqlResource.GetOwnerReferences()
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
		logger.Error(err, "failed to get Deployment CR for MySQL resource", "deployment", deploymentName)
		return ctrl.Result{}, err
	}

	// Check if we've already processed this status to avoid infinite loops
	lastProcessedStatus := ""
	if dep.Annotations != nil {
		lastProcessedStatus = dep.Annotations["platform.kibaship.com/last-processed-mysql-status"]
	}

	// Get current MySQL status
	// InnoDBCluster uses status.cluster.status field which can be: PENDING, INITIALIZING, ONLINE, OFFLINE, etc.
	clusterStatus, found, err := unstructured.NestedString(mysqlResource.Object, "status", "cluster", "status")
	if err != nil {
		logger.Error(err, "failed to get InnoDBCluster status")
		return ctrl.Result{}, err
	}

	currentStatus := fmt.Sprintf("status=%s,found=%t", clusterStatus, found)

	// Skip if we've already processed this status
	if lastProcessedStatus == currentStatus {
		logger.V(1).Info("Skipping MySQL status processing - already processed",
			"status", currentStatus)
		return ctrl.Result{}, nil
	}

	// Determine condition status based on MySQL readiness
	var conditionStatus metav1.ConditionStatus
	var reason, message string

	if !found {
		conditionStatus = metav1.ConditionUnknown
		reason = "StatusNotAvailable"
		message = "MySQL status not yet available"
	} else if clusterStatus == "ONLINE" {
		conditionStatus = metav1.ConditionTrue
		reason = "MySQLReady"
		message = "MySQL cluster is online and ready"
	} else if clusterStatus == "PENDING" || clusterStatus == "INITIALIZING" {
		conditionStatus = metav1.ConditionFalse
		reason = MySQLNotReady
		message = fmt.Sprintf("MySQL cluster is in %s state", clusterStatus)
	} else {
		conditionStatus = metav1.ConditionFalse
		reason = MySQLNotReady
		message = fmt.Sprintf("MySQL cluster status: %s", clusterStatus)
	}

	// Update condition reflecting MySQL state
	cond := metav1.Condition{
		Type:               "MySQLReady",
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
	dep.Annotations["platform.kibaship.com/last-processed-mysql-status"] = currentStatus

	// Persist status
	if err := r.Status().Update(ctx, &dep); err != nil {
		logger.Error(err, "update Deployment status failed", "deployment", fmt.Sprintf("%s/%s", dep.Namespace, dep.Name))
		return ctrl.Result{}, err
	}

	logger.V(1).Info("Updated Deployment CR with MySQL status",
		"status", clusterStatus, "condition", conditionStatus)

	return ctrl.Result{}, nil
}

func (r *MySQLStatusWatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Watch InnoDBCluster resources and trigger on status changes
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// Trigger on create to set initial status
			return hasMySQLOwnerReference(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Only trigger if owned by our Deployment CR
			if !hasMySQLOwnerReference(e.ObjectNew) {
				return false
			}

			// Check if status.cluster.status changed
			oldStatus, _, _ := unstructured.NestedString(e.ObjectOld.(*unstructured.Unstructured).Object, "status", "cluster", "status")
			newStatus, _, _ := unstructured.NestedString(e.ObjectNew.(*unstructured.Unstructured).Object, "status", "cluster", "status")

			// Trigger if status changed
			return oldStatus != newStatus
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}

	// Create an unstructured object for InnoDBCluster resources
	mysqlObj := &unstructured.Unstructured{}
	mysqlObj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "mysql.oracle.com",
		Version: "v2",
		Kind:    "InnoDBCluster",
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(mysqlObj).
		WithEventFilter(pred).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 50, // Higher concurrency for status watching
		}).
		Named("mysql-status-watcher").
		Complete(r)
}

func hasMySQLOwnerReference(obj client.Object) bool {
	ownerRefs := obj.GetOwnerReferences()
	for _, ownerRef := range ownerRefs {
		if ownerRef.Kind == "Deployment" &&
			strings.Contains(ownerRef.APIVersion, "platform.operator.kibaship.com") {
			return true
		}
	}
	return false
}
