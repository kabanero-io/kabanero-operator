package transforms

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func InjectOwnerReference(ownerReference metav1.OwnerReference) func(u *unstructured.Unstructured) error {
	return func(u *unstructured.Unstructured) error {
		u.SetOwnerReferences([]metav1.OwnerReference{ownerReference})
		return nil
	}
}
