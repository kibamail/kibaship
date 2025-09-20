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

package v1alpha1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kibamail/kibaship-operator/pkg/validation"
)

// ApplicationDomainType defines the type of domain
// +kubebuilder:validation:Enum=default;custom
type ApplicationDomainType string

const (
	// ApplicationDomainTypeDefault represents an auto-generated default domain
	ApplicationDomainTypeDefault ApplicationDomainType = "default"
	// ApplicationDomainTypeCustom represents a user-defined custom domain
	ApplicationDomainTypeCustom ApplicationDomainType = "custom"
)

// ApplicationDomainPhase defines the phase of an ApplicationDomain
// +kubebuilder:validation:Enum=Pending;Ready;Failed
type ApplicationDomainPhase string

const (
	// ApplicationDomainPhasePending indicates the domain is being configured
	ApplicationDomainPhasePending ApplicationDomainPhase = "Pending"
	// ApplicationDomainPhaseReady indicates the domain is ready for use
	ApplicationDomainPhaseReady ApplicationDomainPhase = "Ready"
	// ApplicationDomainPhaseFailed indicates the domain configuration failed
	ApplicationDomainPhaseFailed ApplicationDomainPhase = "Failed"
)

// ApplicationDomainSpec defines the desired state of ApplicationDomain
type ApplicationDomainSpec struct {
	// ApplicationRef references the parent application
	// +kubebuilder:validation:Required
	ApplicationRef corev1.LocalObjectReference `json:"applicationRef"`

	// Domain is the full domain name (e.g., "my-app-abc123.myapps.kibaship.com" or "custom.example.com")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`
	Domain string `json:"domain"`

	// Port is the application port for ingress routing
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=3000
	Port int32 `json:"port"`

	// Type indicates if this is a default generated domain or custom domain
	// +kubebuilder:validation:Enum=default;custom
	// +kubebuilder:default=default
	Type ApplicationDomainType `json:"type,omitempty"`

	// Default indicates if this is the default domain for the application
	// Only one domain per application can be marked as default
	// +kubebuilder:default=false
	Default bool `json:"default,omitempty"`

	// TLSEnabled indicates if TLS/SSL should be enabled for this domain
	// +kubebuilder:default=true
	// Note: omit 'omitempty' so that false is preserved over the default.
	TLSEnabled bool `json:"tlsEnabled"`
}

// ApplicationDomainStatus defines the observed state of ApplicationDomain
type ApplicationDomainStatus struct {
	// Phase indicates the current phase of the domain
	// +kubebuilder:validation:Enum=Pending;Ready;Failed
	Phase ApplicationDomainPhase `json:"phase,omitempty"`

	// CertificateReady indicates if the TLS certificate is ready
	CertificateReady bool `json:"certificateReady,omitempty"`

	// IngressReady indicates if the ingress is configured and ready
	IngressReady bool `json:"ingressReady,omitempty"`

	// DNSConfigured indicates if DNS is properly configured (for custom domains)
	DNSConfigured bool `json:"dnsConfigured,omitempty"`

	// LastReconcileTime is the last time the domain was reconciled
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`

	// Message provides human-readable status information
	Message string `json:"message,omitempty"`

	// Conditions represent the latest available observations of the domain state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:webhook:path=/validate-platform-operator-kibaship-com-v1alpha1-applicationdomain,mutating=false,failurePolicy=fail,sideEffects=None,groups=platform.operator.kibaship.com,resources=applicationdomains,verbs=create;update,versions=v1alpha1,name=vapplicationdomain.kb.io,admissionReviewVersions=v1
// +kubebuilder:printcolumn:name="Domain",type=string,JSONPath=".spec.domain"
// +kubebuilder:printcolumn:name="Port",type=integer,JSONPath=".spec.port"
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Default",type=boolean,JSONPath=".spec.default"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Certificate Ready",type=boolean,JSONPath=".status.certificateReady"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
type ApplicationDomain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationDomainSpec   `json:"spec,omitempty"`
	Status ApplicationDomainStatus `json:"status,omitempty"`
}

// ApplicationDomainList contains a list of ApplicationDomain
// +kubebuilder:object:root=true
type ApplicationDomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationDomain `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApplicationDomain{}, &ApplicationDomainList{})
}

var _ webhook.CustomValidator = &ApplicationDomain{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *ApplicationDomain) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	domainlog := logf.Log.WithName("applicationdomain-resource")

	domain, ok := obj.(*ApplicationDomain)
	if !ok {
		return nil, fmt.Errorf("expected an ApplicationDomain object, but got %T", obj)
	}

	domainlog.Info("validate create", "name", domain.Name)

	return nil, domain.validateApplicationDomain(ctx)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *ApplicationDomain) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	domainlog := logf.Log.WithName("applicationdomain-resource")

	domain, ok := newObj.(*ApplicationDomain)
	if !ok {
		return nil, fmt.Errorf("expected an ApplicationDomain object, but got %T", newObj)
	}

	domainlog.Info("validate update", "name", domain.Name)

	return nil, domain.validateApplicationDomain(ctx)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *ApplicationDomain) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	domainlog := logf.Log.WithName("applicationdomain-resource")

	domain, ok := obj.(*ApplicationDomain)
	if !ok {
		return nil, fmt.Errorf("expected an ApplicationDomain object, but got %T", obj)
	}

	domainlog.Info("validate delete", "name", domain.Name)

	return nil, nil
}

// validateApplicationDomain validates the ApplicationDomain resource
func (r *ApplicationDomain) validateApplicationDomain(ctx context.Context) error {
	_ = ctx // context is not used in current validation but required for webhook interface
	var errors []string

	// Use the centralized labeling validation
	// Note: In webhook context, we don't have access to the client for uniqueness checks
	// Uniqueness will be validated in the controller reconcile loop
	labels := r.GetLabels()
	if labels == nil {
		errors = append(errors, "application domain must have labels")
	} else {
		// Validate UUID
		if resourceUUID, exists := labels[validation.LabelResourceUUID]; !exists {
			errors = append(errors, fmt.Sprintf("application domain must have label %s", validation.LabelResourceUUID))
		} else if !validation.ValidateUUID(resourceUUID) {
			errors = append(errors, fmt.Sprintf("application domain UUID must be valid: %s", resourceUUID))
		}

		// Validate Slug
		if resourceSlug, exists := labels[validation.LabelResourceSlug]; !exists {
			errors = append(errors, fmt.Sprintf("application domain must have label %s", validation.LabelResourceSlug))
		} else if !validation.ValidateSlug(resourceSlug) {
			errors = append(errors, fmt.Sprintf("application domain slug must be valid: %s", resourceSlug))
		}

		// Validate Project UUID
		if projectUUID, exists := labels[validation.LabelProjectUUID]; !exists {
			errors = append(errors, fmt.Sprintf("application domain must have label %s", validation.LabelProjectUUID))
		} else if !validation.ValidateUUID(projectUUID) {
			errors = append(errors, fmt.Sprintf("project UUID must be valid: %s", projectUUID))
		}

		// Validate Application UUID
		if applicationUUID, exists := labels[validation.LabelApplicationUUID]; !exists {
			errors = append(errors, fmt.Sprintf("application domain must have label %s", validation.LabelApplicationUUID))
		} else if !validation.ValidateUUID(applicationUUID) {
			errors = append(errors, fmt.Sprintf("application UUID must be valid: %s", applicationUUID))
		}

		// Deployment UUID is optional for ApplicationDomain
		if deploymentUUID, exists := labels[validation.LabelDeploymentUUID]; exists {
			if !validation.ValidateUUID(deploymentUUID) {
				errors = append(errors, fmt.Sprintf("deployment UUID must be valid: %s", deploymentUUID))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *ApplicationDomain) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(r).
		Complete()
}
