package collection

import (
	"testing"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Base collection with collection.Spec.Versions[0] defined.
var validatingCollectionVersions kabanerov1alpha1.Collection = kabanerov1alpha1.Collection{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "java-microprofile",
		Namespace: "Kabanero",
		UID:       "1",
		OwnerReferences: []metav1.OwnerReference{
			metav1.OwnerReference{
				APIVersion: "a/1",
				Kind:       "Kabanero",
				Name:       "kabanero",
				UID:        "1",
			},
		},
	},
	Spec: kabanerov1alpha1.CollectionSpec{
		Name:          "java-microprofile",
		DesiredState:  "active",
		RepositoryUrl: "https://github.com/some/collection/kabanero-index.yaml",
		Version:       "1.2.3",
		Versions: []kabanerov1alpha1.CollectionVersion{{
			DesiredState:  "active",
			RepositoryUrl: "https://github.com/some/collection/kabanero-index.yaml",
			Version:       "1.2.3",
		}},
	},
	Status: kabanerov1alpha1.CollectionStatus{
		ActiveVersion:   "1.2.3",
		ActiveLocation:  "https://github.com/some/collection/kabanero-index.yaml",
		ActivePipelines: []kabanerov1alpha1.PipelineStatus{},
		Status:          "active",
		Images: []kabanerov1alpha1.Image{{
			Id:    "java-microprofile",
			Image: "kabanero/java-microprofile:1.2.3"}},
		Versions: []kabanerov1alpha1.CollectionVersionStatus{{
			Version:   "1.2.3",
			Location:  "https://github.com/some/collection/kabanero-index.yaml",
			Pipelines: []kabanerov1alpha1.PipelineStatus{},
			Status:    "active",
			Images: []kabanerov1alpha1.Image{{
				Id:    "java-microprofile",
				Image: "kabanero/java-microprofile:1.2.3"}},
		}},
	},
}

// Collection.Spec = New collection.Spec.Versions
func validationSuccess(t *testing.T) {
	newCollection := validatingCollectionVersions.DeepCopy()
	cv := collectionValidator{}
	allowed, msg, err := cv.validateCollectionFn(nil, newCollection)

	if !allowed {
		t.Fatal("Validation should have passed. The validation was not allowed")
	}

	if len(msg) != 0 {
		t.Fatal("Validation succeeded. A message was not expected. Message: ", msg)
	}

	if err != nil {
		t.Fatal("Validation succeeded. An error was not expected. Error: ", err)
	}
}

// Collection.Spec != New collection.Spec.Versions
func validationFailure(t *testing.T) {
	newCollection := validatingCollectionVersions.DeepCopy()
	newCollection.Spec.Version = "4.5.6"

	cv := collectionValidator{}
	allowed, msg, err := cv.validateCollectionFn(nil, newCollection)

	if allowed {
		t.Fatal("Validation should have failed. The validation was allowed instead.")
	}

	if len(msg) == 0 {
		t.Fatal("Validation failed. A message was expected.")
	}

	if err == nil {
		t.Fatal("Validation failed. An error was expected")
	}
}
