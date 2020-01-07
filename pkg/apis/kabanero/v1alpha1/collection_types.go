package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// NOTE: The +listType=set marker are required by OpenAPI generation for list types.

const (
	// CollectionDesiredStateActive represents a desired collection active state.
	// It indicates that the collection needs activation.
	CollectionDesiredStateActive = "active"

	// CollectionDesiredStateInactive represents a desired collection inactive state.
	// It indicates that the collection needs to be deactivated.
	CollectionDesiredStateInactive = "inactive"
)

// CollectionSpec defines the desired composition of a Collection
// +k8s:openapi-gen=true
type CollectionSpec struct {
	RepositoryUrl string              `json:"repositoryUrl,omitempty"`
	SkipCertVerification bool         `json:"skipCertVerification,omitempty"`
	Name          string              `json:"name,omitempty"`
	Version       string              `json:"version,omitempty"`
	DesiredState  string              `json:"desiredState,omitempty"`
	// +listType=set
	Versions      []CollectionVersion `json:"versions,omitempty"`
}

// CollectionVersion defines the desired composition of a specific collection version.
type CollectionVersion struct {
	RepositoryUrl string `json:"repositoryUrl,omitempty"`
	Version       string `json:"version,omitempty"`
	DesiredState  string `json:"desiredState,omitempty"`
	SkipCertVerification bool `json:"skipCertVerification,omitempty"`
}

// PipelineStatus defines the observed state of the assets located within a single pipeline .tar.gz.
type PipelineStatus struct {
	Name         string                  `json:"name,omitEmpty"`
	Url          string                  `json:"url,omitEmpty"`
	Digest       string                  `json:"digest,omitEmpty"`
	ActiveAssets []RepositoryAssetStatus `json:"activeAssets,omitempty"`
}

// RepositoryAssetStatus defines the observed state of a single asset in a respository, in the collection.
type RepositoryAssetStatus struct {
	Name          string `json:"assetName,omitempty"`
	Group         string `json:"group,omitempty"`
	Version       string `json:"version,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Digest        string `json:"assetDigest,omitempty"`
	Status        string `json:"status,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

// CollectionStatus defines the observed state of a collection
// +k8s:openapi-gen=true
type CollectionStatus struct {
	ActiveVersion     string                    `json:"activeVersion,omitempty"`
	ActiveLocation    string                    `json:"activeLocation,omitempty"`
	// +listType=set
	ActivePipelines   []PipelineStatus          `json:"activePipelines,omitempty"`
	AvailableVersion  string                    `json:"availableVersion,omitempty"`
	AvailableLocation string                    `json:"availableLocation,omitempty"`
	Status            string                    `json:"status,omitempty"`
	StatusMessage     string                    `json:"statusMessage,omitempty"`
	// +listType=set
	Images            []Image                   `json:"images,omitempty"`
	// +listType=set
	Versions          []CollectionVersionStatus `json:"versions,omitempty"`
}

// CollectionVersionStatus defines the observed state of a specific collection version.
type CollectionVersionStatus struct {
	Version       string           `json:"version,omitempty"`
	Location      string           `json:"location,omitempty"`
	Pipelines     []PipelineStatus `json:"pipelines,omitempty"`
	Status        string           `json:"status,omitempty"`
	StatusMessage string           `json:"statusMessage,omitempty"`
	Images        []Image          `json:"images,omitempty"`
}

// Image defines a container image used by a collection
type Image struct {
	Id    string `json:"id,omitempty"`
	Image string `json:"image,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Collection is the Schema for the collections API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp is a timestamp representing the server time when this object was created. It is not guaranteed to be set in happens-before order across separate operations."
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="Collection status."
// +kubebuilder:resource:path=collections,scope=Namespaced
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
