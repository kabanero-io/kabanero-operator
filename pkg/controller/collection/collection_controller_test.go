package collection

import (
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestIt(t *testing.T) {
	r := &ReconcileCollection{indexResolver: func(url string) (*CollectionV1Index, error) {
		return &CollectionV1Index{
			Collections: map[string][]IndexedCollectionV1{
				"java-microprofile": []IndexedCollectionV1{
					IndexedCollectionV1{
						Name:           "java-microprofile",
						CollectionUrls: []string{},
					},
				},
			},
		}, nil
	}}

	c := &kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{Name: "java-microprofile"},
	}

	k := &kabanerov1alpha1.Kabanero{}

	r.ReconcileCollection(c, k)
}
