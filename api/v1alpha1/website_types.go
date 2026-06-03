/*
Package v1alpha1 defines the Website CRD. A Website is a small, opinionated
abstraction: the user declares an image, replica count, and hostname, and the
controller reconciles the underlying Deployment + Service to match — the
classic "operator" pattern of trading low-level objects for a high-level one.
*/
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WebsiteSpec is the desired state.
type WebsiteSpec struct {
	// Image is the container image to serve.
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// Replicas is the desired number of pods.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`

	// ContainerPort the app listens on.
	// +kubebuilder:default=8080
	ContainerPort int32 `json:"containerPort,omitempty"`

	// Host is an optional DNS name recorded in status (and usable by an Ingress).
	Host string `json:"host,omitempty"`
}

// WebsiteStatus is the observed state.
type WebsiteStatus struct {
	// ReadyReplicas mirrors the managed Deployment's ready replicas.
	ReadyReplicas int32 `json:"readyReplicas"`

	// Phase is a coarse human-readable state.
	Phase string `json:"phase,omitempty"`

	// Conditions follow the standard Kubernetes condition convention.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// Website is the Schema for the websites API.
type Website struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebsiteSpec   `json:"spec,omitempty"`
	Status WebsiteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WebsiteList contains a list of Website.
type WebsiteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Website `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Website{}, &WebsiteList{})
}
