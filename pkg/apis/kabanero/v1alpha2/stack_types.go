package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// NOTE: The +listType=set marker are required by OpenAPI generation for list types.

const (
	// StackDesiredStateActive represents a desired stack active state.
	// It indicates that the stack needs activation.
	StackDesiredStateActive = "active"

	// StackDesiredStateInactive represents a desired stack inactive state.
	// It indicates that the stack needs to be deactivated.
	StackDesiredStateInactive = "inactive"
)

// StackSpec defines the desired composition of a Stack
// +k8s:openapi-gen=true
type StackSpec struct {
	Name                 string `json:"name,omitempty"`
	// +listType=set
	Versions []StackVersion `json:"versions,omitempty"`
}

// StackVersion defines the desired composition of a specific stack version.
type StackVersion struct {
	// +listType=set
	Pipelines            []PipelineSpec `json:"pipelines,omitempty"`
	Version              string  `json:"version,omitempty"`
	DesiredState         string  `json:"desiredState,omitempty"`
	SkipCertVerification bool    `json:"skipCertVerification,omitempty"`
	// +listType=set
	Images               []Image `json:"images,omitempty"`
}

// PipelineStatus defines the observed state of the assets located within a single pipeline .tar.gz.
type PipelineStatus struct {
	Name   string `json:"name,omitEmpty"`
	Url    string `json:"url,omitEmpty"`
	Digest string `json:"digest,omitEmpty"`
	// +listType=set
	ActiveAssets []RepositoryAssetStatus `json:"activeAssets,omitempty"`
}

// RepositoryAssetStatus defines the observed state of a single asset in a respository, in the stack.
type RepositoryAssetStatus struct {
	Name          string `json:"assetName,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	Group         string `json:"group,omitempty"`
	Version       string `json:"version,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Digest        string `json:"assetDigest,omitempty"`
	Status        string `json:"status,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

// StackStatus defines the observed state of a stack
// +k8s:openapi-gen=true
type StackStatus struct {
	StatusMessage     string      `json:"statusMessage,omitempty"`
	// +listType=set
	Versions []StackVersionStatus `json:"versions,omitempty"`
}

// StackVersionStatus defines the observed state of a specific stack version.
type StackVersionStatus struct {
	Version  string `json:"version,omitempty"`
	Location string `json:"location,omitempty"`
	// +listType=set
	Pipelines     []PipelineStatus `json:"pipelines,omitempty"`
	Status        string           `json:"status,omitempty"`
	StatusMessage string           `json:"statusMessage,omitempty"`
	// +listType=set
	Images []Image `json:"images,omitempty"`
}

// Image defines a container image used by a stack
type Image struct {
	Id    string `json:"id,omitempty"`
	Image string `json:"image,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Stack is the Schema for the stack API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations."
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="Stack status."
// +kubebuilder:resource:path=stacks,scope=Namespaced
type Stack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StackSpec   `json:"spec,omitempty"`
	Status StackStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StackList contains a list of Stacks
type StackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=set
	Items []Stack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stack{}, &StackList{})
}
