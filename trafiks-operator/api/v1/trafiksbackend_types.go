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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SecretReference references a Kubernetes Secret containing Trafiks backend credentials
type SecretReference struct {
	// Name of the secret
	// +required
	Name string `json:"name"`

	// Namespace of the secret (optional, defaults to TrafiksBackend namespace if not specified)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// TrafiksBackendSpec defines the desired state of TrafiksBackend
type TrafiksBackendSpec struct {
	// Reference to the Kubernetes Secret containing baseURL and apiKey
	// +required
	SecretRef SecretReference `json:"secretRef"`

	// Optional: Custom key names in the secret (defaults: "baseURL" and "apiKey")
	// +optional
	BaseURLKey string `json:"baseURLKey,omitempty"`

	// +optional
	APIKeyKey string `json:"apiKeyKey,omitempty"`
}

// TrafiksBackendStatus defines the observed state of TrafiksBackend.
type TrafiksBackendStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the TrafiksBackend resource.
	// Conditions used:
	// - "Ready": Overall readiness (secret valid, backend reachable, authentication successful)
	// - "Available": Backend reachability (Trafiks backend is reachable)
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TrafiksBackend is the Schema for the trafiksbackends API
type TrafiksBackend struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of TrafiksBackend
	// +required
	Spec TrafiksBackendSpec `json:"spec"`

	// status defines the observed state of TrafiksBackend
	// +optional
	Status TrafiksBackendStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// TrafiksBackendList contains a list of TrafiksBackend
type TrafiksBackendList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []TrafiksBackend `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TrafiksBackend{}, &TrafiksBackendList{})
}
