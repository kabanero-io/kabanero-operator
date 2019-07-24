package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CollectionSpec defines the desired state of Collection
// +k8s:openapi-gen=true
type CollectionSpec struct {
	RepositoryUrl string `json:"repositoryUrl,omitempty"`
	Name          string `json:"name,omitempty"`
	Version       string `json:"version,omitempty"`
}

// CollectionStatus defines the observed state of Collection
// +k8s:openapi-gen=true
type CollectionStatus struct {
	ActiveVersion string `json:"activeVersion,omitempty"`
	ActiveDigest  string `json:"activeDigest,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Collection is the Schema for the collections API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Collection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CollectionSpec   `json:"spec,omitempty"`
	Status CollectionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CollectionList contains a list of Collection
type CollectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Collection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Collection{}, &CollectionList{})
}
