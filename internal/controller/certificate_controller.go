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

	meta "k8s.io/apimachinery/pkg/api/meta"
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
	"github.com/kibamail/kibaship/pkg/validation"
	"github.com/kibamail/kibaship/pkg/webhooks"
)

// CertificateWatcherReconciler watches cert-manager.io/v1 Certificates and mirrors their status into
// owning ApplicationDomains (correlated via shared labels), then emits enriched webhooks.
type CertificateWatcherReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Notifier webhooks.Notifier
}

const (
	reasonReadyStr   = "Ready"
	reasonFailedStr  = "Failed"
	reasonPendingStr = "Pending"
)

// RBAC: read Certificates; update ApplicationDomain status
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applicationdomains,verbs=get;list;watch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applicationdomains/status,verbs=get;update;patch

func (r *CertificateWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Certificate as unstructured
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"})
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, u); err != nil {
		// Gone
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	labels := u.GetLabels()
	uuid, ok := labels[validation.LabelResourceUUID]
	if !ok || uuid == "" {
		// Not one of ours
		return ctrl.Result{}, nil
	}

	// Find owning ApplicationDomain by shared UUID label
	var adList platformv1alpha1.ApplicationDomainList
	if err := r.List(ctx, &adList, client.MatchingLabels(map[string]string{validation.LabelResourceUUID: uuid})); err != nil {
		logger.Error(err, "list ApplicationDomains by UUID failed", "uuid", uuid)
		return ctrl.Result{}, err
	}
	if len(adList.Items) == 0 {
		logger.Info("no ApplicationDomain found for Certificate", "uuid", uuid, "cert", req.NamespacedName)
		return ctrl.Result{}, nil
	}
	// If multiple match (shouldn't), pick the first deterministically
	ad := adList.Items[0]
	prevPhase := ad.Status.Phase

	// Extract Certificate Ready condition
	readyStatus, reason, message := extractCertReady(u)
	// Derive phase
	var newPhase platformv1alpha1.ApplicationDomainPhase
	switch readyStatus {
	case condTrue:
		newPhase = platformv1alpha1.ApplicationDomainPhaseReady
	case condFalse:
		newPhase = platformv1alpha1.ApplicationDomainPhaseFailed
	default:
		newPhase = platformv1alpha1.ApplicationDomainPhasePending
	}

	ad.Status.CertificateReady = (readyStatus == condTrue)
	ad.Status.Phase = newPhase
	ad.Status.Message = joinNonEmpty(reason, message)
	now := metav1.Now()
	ad.Status.LastReconcileTime = &now

	// Update Ready condition on AD
	cond := metav1.Condition{Type: "Ready", LastTransitionTime: now, Message: ad.Status.Message}
	switch readyStatus {
	case "True":
		cond.Status = metav1.ConditionTrue
		cond.Reason = reasonReadyStr
	case "False":
		cond.Status = metav1.ConditionFalse
		cond.Reason = reasonFailedStr
	default:
		cond.Status = metav1.ConditionUnknown
		cond.Reason = reasonPendingStr
	}
	meta.SetStatusCondition(&ad.Status.Conditions, cond)

	if err := r.Status().Update(ctx, &ad); err != nil {
		logger.Error(err, "update ApplicationDomain status failed", "ad", fmt.Sprintf("%s/%s", ad.Namespace, ad.Name))
		return ctrl.Result{}, err
	}

	// Emit webhook (enriched in notifier) if phase changed
	if r.Notifier != nil && string(prevPhase) != string(newPhase) {
		evt := webhooks.ApplicationDomainStatusEvent{
			Type:              "applicationdomain.status.changed",
			PreviousPhase:     string(prevPhase),
			NewPhase:          string(newPhase),
			ApplicationDomain: ad,
			Timestamp:         time.Now().UTC(),
		}
		_ = r.Notifier.NotifyApplicationDomainStatusChange(ctx, evt)
	}

	return ctrl.Result{}, nil
}

func (r *CertificateWatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"})
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

func extractCertReady(u *unstructured.Unstructured) (status, reason, message string) {
	conds, found, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	if !found {
		return "Unknown", "", ""
	}
	for _, c := range conds {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := m["type"].(string); t == "Ready" {
			status, _ = m["status"].(string)
			reason, _ = m["reason"].(string)
			message, _ = m["message"].(string)
			return
		}
	}
	return "Unknown", "", ""
}

func joinNonEmpty(parts ...string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out == "" {
			out = p
		} else {
			out = out + ": " + p
		}
	}
	return out
}
