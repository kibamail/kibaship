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

// DeploymentPhase represents the current phase of the deployment
type DeploymentPhase string

const (
	// DeploymentPhaseInitializing indicates the deployment is being set up
	DeploymentPhaseInitializing DeploymentPhase = "Initializing"
	// DeploymentPhaseRunning indicates a pipeline is currently running
	DeploymentPhaseRunning DeploymentPhase = "Running"
	// DeploymentPhaseSucceeded indicates the last pipeline run succeeded
	DeploymentPhaseSucceeded DeploymentPhase = "Succeeded"
	// DeploymentPhaseFailed indicates the last pipeline run failed
	DeploymentPhaseFailed DeploymentPhase = "Failed"
	// DeploymentPhaseWaiting indicates the deployment is waiting for trigger
	DeploymentPhaseWaiting DeploymentPhase = "Waiting"
)

// PipelineRunStatus tracks pipeline run information
type PipelineRunStatus struct {
	// Name of the pipeline run
	Name string `json:"name"`

	// Namespace of the pipeline run
	Namespace string `json:"namespace"`

	// Phase of the pipeline run
	Phase string `json:"phase"`

	// Start time
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Completion time
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Git commit SHA that triggered this run
	// +optional
	CommitSha string `json:"commitSha,omitempty"`

	// Built image from this run
	// +optional
	Image string `json:"image,omitempty"`

	// Message provides additional information about the pipeline run
	// +optional
	Message string `json:"message,omitempty"`
}

// ImageInfo contains information about built images
type ImageInfo struct {
	// Full image reference with digest
	Reference string `json:"reference"`

	// Image digest
	Digest string `json:"digest"`

	// Image tag
	Tag string `json:"tag"`

	// Build timestamp
	BuiltAt metav1.Time `json:"builtAt"`

	// Git commit SHA used for this image
	CommitSha string `json:"commitSha"`
}

// DeploymentSpec defines the desired state of Deployment.
type DeploymentSpec struct {
	// ApplicationRef references the Application this deployment belongs to
	// +kubebuilder:validation:Required
	ApplicationRef corev1.LocalObjectReference `json:"applicationRef"`
}

// DeploymentStatus defines the observed state of Deployment.
type DeploymentStatus struct {
	// Current phase of the deployment
	// +optional
	Phase DeploymentPhase `json:"phase,omitempty"`

	// Current pipeline run information
	// +optional
	CurrentPipelineRun *PipelineRunStatus `json:"currentPipelineRun,omitempty"`

	// Last successful pipeline run
	// +optional
	LastSuccessfulRun *PipelineRunStatus `json:"lastSuccessfulRun,omitempty"`

	// Built image information from the last successful run
	// +optional
	BuiltImage *ImageInfo `json:"builtImage,omitempty"`

	// Conditions represent the latest available observations of the deployment's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed Deployment
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Pipeline run history (last 5 runs)
	// +optional
	RunHistory []PipelineRunStatus `json:"runHistory,omitempty"`

	// Message provides additional information about the current status
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Application",type="string",JSONPath=".spec.applicationRef.name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Current Run",type="string",JSONPath=".status.currentPipelineRun.name"
// +kubebuilder:printcolumn:name="Last Success",type="string",JSONPath=".status.lastSuccessfulRun.name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Deployment is the Schema for the deployments API.
type Deployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentSpec   `json:"spec,omitempty"`
	Status DeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DeploymentList contains a list of Deployment.
type DeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Deployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Deployment{}, &DeploymentList{})
}
