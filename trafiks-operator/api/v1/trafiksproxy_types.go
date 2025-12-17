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

// KubernetesProxyConfig contains Kubernetes-specific service discovery configuration
type KubernetesProxyConfig struct {
	// Namespace is the Kubernetes namespace of the target service
	// +required
	Namespace string `json:"namespace"`

	// ServiceName is the name of the Kubernetes Service
	// +required
	ServiceName string `json:"serviceName"`

	// ServicePortName is the port name to use (e.g., "http", "https")
	// If not specified, the operator will use a port named "http" or the first available port
	// +optional
	ServicePortName string `json:"servicePortName,omitempty"`

	// Selector is an optional label selector
	// +optional
	Selector map[string]string `json:"selector,omitempty"`
}

// CacheConfig contains cache configuration
type CacheConfig struct {
	// Enabled enables caching for this service
	// +kubebuilder:default=false
	// +optional
	Enabled bool `json:"enabled"`

	// TTL is the cache time-to-live in seconds
	// +kubebuilder:default=300
	// +kubebuilder:validation:Minimum=1
	// +optional
	TTL int `json:"ttl,omitempty"`
}

// HeadersConfig contains header manipulation configuration
type HeadersConfig struct {
	// Remove is a list of headers to remove from requests
	// +optional
	Remove []string `json:"remove,omitempty"`

	// Add is a map of headers to add to requests
	// +optional
	Add map[string]string `json:"add,omitempty"`
}

// QueryParamsConfig contains query parameter configuration
type QueryParamsConfig struct {
	// Remove is a list of query parameters to remove from requests
	// +optional
	Remove []string `json:"remove,omitempty"`
}

// BackendReference references a TrafiksBackend resource
type BackendReference struct {
	// Name of the TrafiksBackend resource
	// +required
	Name string `json:"name"`

	// Namespace of the TrafiksBackend resource (optional, defaults to TrafiksProxy namespace)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// IngressReference references an Ingress resource
type IngressReference struct {
	// Name of the Ingress resource
	Name string `json:"name"`

	// Namespace of the Ingress resource
	Namespace string `json:"namespace"`
}

// TrafiksProxySpec defines the desired state of TrafiksProxy
type TrafiksProxySpec struct {
	// BackendRef references the TrafiksBackend resource to use for API calls
	// +required
	BackendRef BackendReference `json:"backendRef"`

	// ProxyURL is the domain name for the proxy (e.g., "nginx.traffiked.cloud")
	// +required
	ProxyURL string `json:"proxyURL"`

	// Scheme is the proxy URL scheme (http or https)
	// +kubebuilder:validation:Enum=http;https
	// +kubebuilder:default=http
	// +optional
	Scheme string `json:"scheme,omitempty"`

	// ProjectName is the required project name in Trafiks dashboard
	// +required
	ProjectName string `json:"projectName,omitempty"`

	// Kubernetes-specific configuration
	// +required
	Kubernetes KubernetesProxyConfig `json:"kubernetes"`

	// Cache configuration
	// +optional
	Cache *CacheConfig `json:"cache,omitempty"`

	// Headers configuration
	// +optional
	Headers *HeadersConfig `json:"headers,omitempty"`

	// Query parameters configuration
	// +optional
	QueryParams *QueryParamsConfig `json:"queryParams,omitempty"`

	// HTTPS redirect configuration
	// If true, redirect HTTP to HTTPS; if false, reject HTTP; if nil, default behavior
	// +optional
	HTTPSRedirect *bool `json:"httpsRedirect,omitempty"`

	// TLSCertResolver specifies how to obtain the TLS certificate
	// - "letsencrypt": Extract certificate from Ingress TLS spec (requires Ingress with TLS configuration)
	// - "selfsigned": Backend will generate a self-signed certificate (default)
	// +kubebuilder:validation:Enum=letsencrypt;selfsigned
	// +kubebuilder:default=selfsigned
	// +optional
	TLSCertResolver string `json:"tlsCertResolver,omitempty"`
}

// TrafiksProxyStatus defines the observed state of TrafiksProxy.
type TrafiksProxyStatus struct {
	// conditions represent the current state of the TrafiksProxy resource.
	// Conditions used:
	// - "Ready": Overall readiness (service synced and active in Trafiks)
	// - "Synced": Service successfully synced to Trafiks backend
	// - "BackendReachable": Trafiks backend is accessible
	// - "ServiceResolved": Kubernetes service endpoint resolved
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ProjectUID is the UID of the project in Trafiks (cached to avoid repeated lookups)
	// +optional
	ProjectUID string `json:"projectUID,omitempty"`

	// ProjectName is the name of the project in Trafiks
	// +optional
	ProjectName string `json:"projectName,omitempty"`

	// ServiceUID is the UID of the service in Trafiks
	// +optional
	ServiceUID string `json:"serviceUID,omitempty"`

	// LastSyncTime is the timestamp of the last successful sync
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// IngressRef tracks which Ingress "owns" this TrafiksProxy
	// +optional
	IngressRef *IngressReference `json:"ingressRef,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TrafiksProxy is the Schema for the trafiksproxies API
type TrafiksProxy struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of TrafiksProxy
	// +required
	Spec TrafiksProxySpec `json:"spec"`

	// status defines the observed state of TrafiksProxy
	// +optional
	Status TrafiksProxyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// TrafiksProxyList contains a list of TrafiksProxy
type TrafiksProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []TrafiksProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TrafiksProxy{}, &TrafiksProxyList{})
}
