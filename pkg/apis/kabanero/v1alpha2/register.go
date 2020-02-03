// NOTE: Boilerplate only.  Ignore this file.

// Package v1alpha2 contains API Schema definitions for the kabanero v1alpha2 API group
// +k8s:deepcopy-gen=package,register
// +groupName=kabanero.io
package v1alpha2

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "kabanero.io", Version: "v1alpha2"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
