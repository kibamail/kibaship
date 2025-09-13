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

// InnoDBClusterSpec defines the desired state of InnoDBCluster
type InnoDBClusterSpec struct {
	// SecretName is the name of a generic type Secret containing root/default account password
	// +kubebuilder:validation:Required
	SecretName string `json:"secretName"`

	// TLSUseSelfSigned enables use of self-signed TLS certificates
	// +kubebuilder:default=true
	// +optional
	TLSUseSelfSigned bool `json:"tlsUseSelfSigned,omitempty"`

	// Version is the MySQL Server version
	// +optional
	Version string `json:"version,omitempty"`

	// Edition is the MySQL Server Edition (community or enterprise)
	// +kubebuilder:validation:Enum=community;enterprise
	// +optional
	Edition string `json:"edition,omitempty"`

	// Instances is the number of MySQL replica instances for the cluster
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=9
	// +kubebuilder:default=1
	// +optional
	Instances int32 `json:"instances,omitempty"`

	// DatadirVolumeClaimTemplate is the template for a PersistentVolumeClaim, to be used as datadir
	// +optional
	DatadirVolumeClaimTemplate *corev1.PersistentVolumeClaim `json:"datadirVolumeClaimTemplate,omitempty"`

	// Router configuration for MySQL Router instances
	// +optional
	Router *InnoDBClusterRouterSpec `json:"router,omitempty"`
}

// InnoDBClusterRouterSpec defines the router configuration for InnoDBCluster
type InnoDBClusterRouterSpec struct {
	// Instances is the number of MySQL Router instances
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// +optional
	Instances int32 `json:"instances,omitempty"`
}

// InnoDBClusterStatus defines the observed state of InnoDBCluster
type InnoDBClusterStatus struct {
	// Phase represents the current phase of the InnoDBCluster
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the cluster's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Message provides additional information about the current status
	// +optional
	Message string `json:"message,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed InnoDBCluster
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version"
// +kubebuilder:printcolumn:name="Instances",type="integer",JSONPath=".spec.instances"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// InnoDBCluster is the Schema for the innodbclusters API from MySQL Operator
// Note: This is a local representation of the mysql.oracle.com/v2 API for controller use
type InnoDBCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InnoDBClusterSpec   `json:"spec,omitempty"`
	Status InnoDBClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InnoDBClusterList contains a list of InnoDBCluster
type InnoDBClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InnoDBCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InnoDBCluster{}, &InnoDBClusterList{})
}
