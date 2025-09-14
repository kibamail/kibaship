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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	TLSEnabled bool `json:"tlsEnabled,omitempty"`
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

// ApplicationDomain is the Schema for the applicationdomains API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=applicationdomains,scope=Namespaced
// +kubebuilder:printcolumn:name="Domain",type=string,JSONPath=`.spec.domain`
// +kubebuilder:printcolumn:name="Port",type=integer,JSONPath=`.spec.port`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Default",type=boolean,JSONPath=`.spec.default`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Certificate Ready",type=boolean,JSONPath=`.status.certificateReady`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
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
