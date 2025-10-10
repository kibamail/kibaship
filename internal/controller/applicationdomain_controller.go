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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/webhooks"
)

const (
	// ApplicationDomainFinalizerName is the finalizer added to ApplicationDomain resources
	ApplicationDomainFinalizerName = "platform.operator.kibaship.com/applicationdomain-finalizer"
	// ApplicationDomainLabelApplication is the label key for the parent application
	ApplicationDomainLabelApplication = "platform.operator.kibaship.com/application"
	// ApplicationDomainLabelDomainType is the label key for the domain type
	ApplicationDomainLabelDomainType = "platform.operator.kibaship.com/domain-type"
)
const (
	certificatesNamespace = "certificates"
	clusterIssuerName     = "certmanager-acme-issuer"
)

// ApplicationDomainReconciler reconciles an ApplicationDomain object
type ApplicationDomainReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Notifier webhooks.Notifier
}

// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applicationdomains,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applicationdomains/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=applicationdomains/finalizers,verbs=update
// Access cert-manager.io Certificates to provision TLS for domains
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ApplicationDomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the ApplicationDomain instance
	var appDomain platformv1alpha1.ApplicationDomain
	if err := r.Get(ctx, req.NamespacedName, &appDomain); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("ApplicationDomain resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get ApplicationDomain")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling ApplicationDomain", "domain", appDomain.Spec.Domain, "phase", appDomain.Status.Phase)

	// Handle deletion
	if !appDomain.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &appDomain)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&appDomain, ApplicationDomainFinalizerName) {
		logger.Info("Adding finalizer to ApplicationDomain")
		controllerutil.AddFinalizer(&appDomain, ApplicationDomainFinalizerName)
		if err := r.Update(ctx, &appDomain); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Validate the domain configuration
	if err := r.validateDomain(ctx, &appDomain); err != nil {
		logger.Error(err, "Domain validation failed")
		return r.updateStatus(ctx, &appDomain, platformv1alpha1.ApplicationDomainPhaseFailed,
			fmt.Sprintf("Domain validation failed: %v", err))
	}

	// Handle certificate provisioning based on domain type
	var certName, certNS string
	var err error

	if appDomain.Spec.Type == platformv1alpha1.ApplicationDomainTypeCustom {
		// Custom domains: provision individual certificate via ACME/Let's Encrypt
		certName, certNS, err = r.ensureCertificateForDomain(ctx, &appDomain)
		if err != nil {
			logger.Error(err, "Failed to provision Certificate for custom ApplicationDomain")
			return r.updateStatus(ctx, &appDomain, platformv1alpha1.ApplicationDomainPhaseFailed,
				fmt.Sprintf("Certificate provisioning failed: %v", err))
		}
		logger.Info("Provisioned individual certificate for custom domain", "certificate", certName, "namespace", certNS)
	} else {
		// Default domains: reference the wildcard certificate
		certName = "tenant-wildcard-certificate"
		certNS = certificatesNamespace
		logger.Info("Using wildcard certificate for default domain", "certificate", certName, "namespace", certNS)
	}

	appDomain.Status.CertificateRef = &platformv1alpha1.NamespacedRef{Name: certName, Namespace: certNS}

	// Update status to indicate domain is ready (certificate issuance will progress asynchronously)
	return r.updateStatus(ctx, &appDomain, platformv1alpha1.ApplicationDomainPhaseReady,
		"Domain is configured and certificate requested")
}

// handleDeletion handles the cleanup when an ApplicationDomain is being deleted
func (r *ApplicationDomainReconciler) handleDeletion(ctx context.Context, appDomain *platformv1alpha1.ApplicationDomain) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(appDomain, ApplicationDomainFinalizerName) {
		logger.Info("ApplicationDomain is being deleted but finalizer not found, allowing deletion")
		return ctrl.Result{}, nil
	}

	logger.Info("Cleaning up ApplicationDomain resources", "domain", appDomain.Spec.Domain)

	// TODO: In future phases, clean up ingress and certificate resources here

	// Remove the finalizer to allow deletion
	controllerutil.RemoveFinalizer(appDomain, ApplicationDomainFinalizerName)
	if err := r.Update(ctx, appDomain); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully cleaned up ApplicationDomain", "domain", appDomain.Spec.Domain)
	return ctrl.Result{}, nil
}

// validateDomain performs validation of the ApplicationDomain
func (r *ApplicationDomainReconciler) validateDomain(ctx context.Context, appDomain *platformv1alpha1.ApplicationDomain) error {
	logger := log.FromContext(ctx)

	// Validate domain format
	if err := ValidateDomainFormat(appDomain.Spec.Domain); err != nil {
		return fmt.Errorf("invalid domain format: %v", err)
	}

	// Validate domain uniqueness
	if err := r.validateDomainUniqueness(ctx, appDomain); err != nil {
		return fmt.Errorf("domain uniqueness validation failed: %v", err)
	}

	// Validate application reference exists
	if err := r.validateApplicationReference(ctx, appDomain); err != nil {
		return fmt.Errorf("application reference validation failed: %v", err)
	}

	// Validate default domain constraints
	if appDomain.Spec.Default {
		if err := r.validateDefaultDomainUniqueness(ctx, appDomain); err != nil {
			return fmt.Errorf("default domain validation failed: %v", err)
		}
	}

	logger.Info("Domain validation passed", "domain", appDomain.Spec.Domain)
	return nil
}

// validateDomainUniqueness ensures the domain is unique across all ApplicationDomains
func (r *ApplicationDomainReconciler) validateDomainUniqueness(ctx context.Context, appDomain *platformv1alpha1.ApplicationDomain) error {
	var domains platformv1alpha1.ApplicationDomainList
	if err := r.List(ctx, &domains); err != nil {
		return fmt.Errorf("failed to list existing domains: %v", err)
	}

	for _, d := range domains.Items {
		if d.UID != appDomain.UID && d.Spec.Domain == appDomain.Spec.Domain {
			return fmt.Errorf("domain %s already exists in ApplicationDomain %s/%s",
				appDomain.Spec.Domain, d.Namespace, d.Name)
		}
	}

	return nil
}

// validateApplicationReference validates that the referenced application exists
func (r *ApplicationDomainReconciler) validateApplicationReference(ctx context.Context, appDomain *platformv1alpha1.ApplicationDomain) error {
	var app platformv1alpha1.Application
	appKey := types.NamespacedName{
		Name:      appDomain.Spec.ApplicationRef.Name,
		Namespace: appDomain.Namespace,
	}

	if err := r.Get(ctx, appKey, &app); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("referenced application %s not found in namespace %s",
				appDomain.Spec.ApplicationRef.Name, appDomain.Namespace)
		}
		return fmt.Errorf("failed to get referenced application: %v", err)
	}

	// Validate that the application is of a supported type
	supportedTypes := []platformv1alpha1.ApplicationType{
		platformv1alpha1.ApplicationTypeGitRepository,
		platformv1alpha1.ApplicationTypeImageFromRegistry,
		platformv1alpha1.ApplicationTypeValkey,
		platformv1alpha1.ApplicationTypeValkeyCluster,
	}

	isSupported := false
	for _, supportedType := range supportedTypes {
		if app.Spec.Type == supportedType {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return fmt.Errorf("ApplicationDomain is currently only supported for GitRepository and ImageFromRegistry applications, got %s", app.Spec.Type)
	}

	return nil
}

// validateDefaultDomainUniqueness ensures only one default domain exists per application
func (r *ApplicationDomainReconciler) validateDefaultDomainUniqueness(ctx context.Context, appDomain *platformv1alpha1.ApplicationDomain) error {
	var domains platformv1alpha1.ApplicationDomainList
	if err := r.List(ctx, &domains, client.InNamespace(appDomain.Namespace)); err != nil {
		return fmt.Errorf("failed to list domains in namespace: %v", err)
	}

	for _, d := range domains.Items {
		if d.UID != appDomain.UID &&
			d.Spec.ApplicationRef.Name == appDomain.Spec.ApplicationRef.Name &&
			d.Spec.Default {
			return fmt.Errorf("application %s already has a default domain: %s",
				appDomain.Spec.ApplicationRef.Name, d.Spec.Domain)
		}
	}

	return nil
}

// updateStatus updates the ApplicationDomain status
func (r *ApplicationDomainReconciler) updateStatus(ctx context.Context, appDomain *platformv1alpha1.ApplicationDomain,
	phase platformv1alpha1.ApplicationDomainPhase, message string) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	// Update application domain status

	now := metav1.Now()
	prevPhase := appDomain.Status.Phase
	appDomain.Status.Phase = phase
	appDomain.Status.Message = message
	appDomain.Status.LastReconcileTime = &now

	// Update conditions based on phase
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		LastTransitionTime: now,
		Reason:             "Reconciling",
		Message:            message,
	}

	const (
		reasonReady   = "Ready"
		reasonFailed  = "Failed"
		reasonPending = "Pending"
	)

	switch phase {
	case platformv1alpha1.ApplicationDomainPhaseReady:
		condition.Status = metav1.ConditionTrue
		condition.Reason = reasonReady
		// For now, set certificate and ingress as ready since we're not implementing them yet
		appDomain.Status.CertificateReady = true
		appDomain.Status.IngressReady = true
		appDomain.Status.DNSConfigured = appDomain.Spec.Type == platformv1alpha1.ApplicationDomainTypeDefault
	case platformv1alpha1.ApplicationDomainPhaseFailed:
		condition.Status = metav1.ConditionFalse
		condition.Reason = reasonFailed
		appDomain.Status.CertificateReady = false
		appDomain.Status.IngressReady = false
		appDomain.Status.DNSConfigured = false
	case platformv1alpha1.ApplicationDomainPhasePending:
		condition.Reason = reasonPending
		appDomain.Status.CertificateReady = false
		appDomain.Status.IngressReady = false
		appDomain.Status.DNSConfigured = false
	}

	meta.SetStatusCondition(&appDomain.Status.Conditions, condition)

	if err := r.Status().Update(ctx, appDomain); err != nil {
		logger.Error(err, "Failed to update ApplicationDomain status")
		return ctrl.Result{}, err
	}

	// Emit webhook on phase transition
	r.emitApplicationDomainPhaseChange(ctx, appDomain, string(prevPhase), string(phase))

	logger.Info("Updated ApplicationDomain status", "phase", phase, "message", message)

	// Streaming removed; no external event emission

	// Requeue if still pending
	if phase == platformv1alpha1.ApplicationDomainPhasePending {

		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	return ctrl.Result{}, nil
}

// emitApplicationDomainPhaseChange sends a webhook if Notifier is configured and the phase actually changed.
func (r *ApplicationDomainReconciler) emitApplicationDomainPhaseChange(ctx context.Context, appDomain *platformv1alpha1.ApplicationDomain, prev, next string) {
	if r.Notifier == nil {
		return
	}
	if prev == next {
		return
	}
	evt := webhooks.ApplicationDomainStatusEvent{
		Type:              "applicationdomain.status.changed",
		PreviousPhase:     prev,
		NewPhase:          next,
		ApplicationDomain: *appDomain,
		Timestamp:         time.Now().UTC(),
	}
	_ = r.Notifier.NotifyApplicationDomainStatusChange(ctx, evt)

}

// ensureCertificateForDomain ensures a cert-manager.io Certificate exists for the given ApplicationDomain.
// It copies all labels from the ApplicationDomain onto the Certificate (including the domain UUID),
// and returns the created/existing certificate name and namespace.
func (r *ApplicationDomainReconciler) ensureCertificateForDomain(ctx context.Context, appDomain *platformv1alpha1.ApplicationDomain) (string, string, error) {
	logger := log.FromContext(ctx)
	certName := fmt.Sprintf("ad-%s", appDomain.Name)
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"})
	obj.SetNamespace(certificatesNamespace)
	obj.SetName(certName)

	// Try to get existing Certificate
	if err := r.Get(ctx, client.ObjectKey{Namespace: certificatesNamespace, Name: certName}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return "", "", err
		}
		// Create new Certificate
		labels := map[string]string{}
		if appDomain.Labels != nil {
			for k, v := range appDomain.Labels {
				labels[k] = v
			}
		}
		obj.SetLabels(labels)
		obj.Object["spec"] = map[string]any{
			"secretName": fmt.Sprintf("tls-%s", certName),
			"issuerRef":  map[string]any{"name": clusterIssuerName, "kind": "ClusterIssuer"},
			"dnsNames":   []any{appDomain.Spec.Domain},
		}
		if err := r.Create(ctx, obj); err != nil {
			return "", "", err
		}
		logger.Info("Created Certificate for ApplicationDomain", "certificate", certName, "namespace", certificatesNamespace)
	} else {
		// Ensure labels include those from ApplicationDomain
		labels := obj.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		changed := false
		for k, v := range appDomain.GetLabels() {
			if labels[k] != v {
				labels[k] = v
				changed = true
			}
		}
		if changed {
			obj.SetLabels(labels)
			_ = r.Update(ctx, obj)
		}
	}

	return certName, certificatesNamespace, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationDomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.ApplicationDomain{}).
		Complete(r)
}
