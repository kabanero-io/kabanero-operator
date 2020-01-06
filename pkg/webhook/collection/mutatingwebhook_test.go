package collection

import (
	"testing"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Base collection with no collection.Spec.Versions[] defined.
var mutatingBaseCollection kabanerov1alpha1.Collection = kabanerov1alpha1.Collection{
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
	},
	Status: kabanerov1alpha1.CollectionStatus{
		ActiveVersion:   "1.2.3",
		ActiveLocation:  "https://github.com/some/collection/kabanero-index.yaml",
		ActivePipelines: []kabanerov1alpha1.PipelineStatus{},
		Status:          "active",
		Images: []kabanerov1alpha1.Image{{
			Id:    "java-microprofile",
			Image: "kabanero/java-microprofile:1.2.3"}},
	},
}

// Base collection with no collection.Spec.Versions[0] defined.
var mutatingBaseCollectionVersions kabanerov1alpha1.Collection = kabanerov1alpha1.Collection{
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

// Current collection.Spec = New collection.Spec.
// Current collection.Spec.Versions[] (empty) =  and New collection.Spec.Versions[] (empty).
// Expectation: collection.Spec.versions[0] should be added with the contents of collection.Spec data.
func Test1(t *testing.T) {
	newCollection := mutatingBaseCollection.DeepCopy()
	err := processUpdate(&mutatingBaseCollection, newCollection)
	if err != nil {
		t.Fatal("Unexpected error during mutation.", err)
	}

	expectedversion0 := kabanerov1alpha1.CollectionVersion{
		RepositoryUrl: "https://github.com/some/collection/kabanero-index.yaml",
		Version:       "1.2.3",
		DesiredState:  "active"}

	//fmt.Println("Here is the base collection: ", mutatingBaseCollection)
	//fmt.Println("Here is the new collection: ", newCollection)

	if newCollection.Spec.Versions[0] != expectedversion0 {
		t.Fatal("Mutated versions[0] does not match expected versions[0] values. Mutated versions[0]: ", newCollection.Spec.Versions[0], "Expected versions[0]: ", expectedversion0)
	}

}

// Current collection.Spec != New collection.Spec.
// Current collection.Spec.Versions[] (empty) =  and New collection.Spec.Versions[] (empty).
// Expectation: collection.Spec.versions[0] should be added with the contents of collection.Spec data.
func Test2(t *testing.T) {
	newCollection := mutatingBaseCollection.DeepCopy()
	newCollection.Spec.RepositoryUrl = "https://github.com/some/collection/alternate-kabanero-index.yaml"
	newCollection.Spec.Version = "4.5.6"
	err := processUpdate(&mutatingBaseCollection, newCollection)
	if err != nil {
		t.Fatal("Unexpected error during mutation.", err)
	}

	expectedversion0 := kabanerov1alpha1.CollectionVersion{
		RepositoryUrl: "https://github.com/some/collection/alternate-kabanero-index.yaml",
		Version:       "4.5.6",
		DesiredState:  "active"}

	if newCollection.Spec.Versions[0] != expectedversion0 {
		t.Fatal("Mutated versions[0] does not match expected versions[0] values. Mutated versions[0]: ", newCollection.Spec.Versions[0], "Expected versions[0]: ", expectedversion0)
	}
}

// Current collection.Spec != New collection.Spec.
// Current collection.Spec.Versions[] (empty) =  and New collection.Spec.Versions[] (empty).
// Expectation: An error condition should be reported.
func Test3(t *testing.T) {
	newCollection := mutatingBaseCollection.DeepCopy()
	newCollection.Spec.RepositoryUrl = ""
	newCollection.Spec.Version = ""
	newCollection.Spec.DesiredState = ""
	err := processUpdate(&mutatingBaseCollection, newCollection)
	if err == nil {
		t.Fatal("An error condition should have been reported. Spec and Spec.versions were not properly defined.", err)
	}
}

// Current collection.Spec  != New collection.Spec.
// Current collection.Spec == New collection.Spec.versions[0]
// Current collection.Spec.Versions[] (empty).
// Expectation: New collection.Spec should have been copied to New collection.Spec.versions[0].
func Test4(t *testing.T) {
	newCollection := mutatingBaseCollectionVersions.DeepCopy()
	newCollection.Spec.RepositoryUrl = "https://github.com/some/collection/alternate-kabanero-index.yaml"
	newCollection.Spec.Version = "4.5.6"
	newCollection.Spec.DesiredState = "inactive"
	err := processUpdate(&mutatingBaseCollection, newCollection)
	if err != nil {
		t.Fatal("Unexpected error during mutation.", err)
	}

	expectedversion0 := kabanerov1alpha1.CollectionVersion{
		RepositoryUrl: "https://github.com/some/collection/alternate-kabanero-index.yaml",
		Version:       "4.5.6",
		DesiredState:  "inactive"}

	if newCollection.Spec.Versions[0] != expectedversion0 {
		t.Fatal("New collection.Spec.Versions[0] values do not match expected collection.Spec.Versions[0] values. New versions[0]: ", newCollection.Spec.Versions[0], "Expected versions[0]: ", expectedversion0)
	}
}

// Current collection.Spec == New collection.Spec.
// Current collection.Spec != New collection.Spec.versions[0]
// Current collection.Spec.Versions[] (empty).
// Expectation: New collection.Spec.versions[0] values should have been copied to New collection.Spec.
func Test5(t *testing.T) {
	newCollection := mutatingBaseCollectionVersions.DeepCopy()
	newCollection.Spec.Versions[0].RepositoryUrl = "https://github.com/some/collection/alternate-kabanero-index.yaml"
	newCollection.Spec.Versions[0].Version = "4.5.6"
	newCollection.Spec.Versions[0].DesiredState = "inactive"
	err := processUpdate(&mutatingBaseCollection, newCollection)
	if err != nil {
		t.Fatal("Unexpected error during mutation.", err)
	}

	if newCollection.Spec.RepositoryUrl != "https://github.com/some/collection/alternate-kabanero-index.yaml" {
		t.Fatal("New collection.Spec.RepositoryUrl values do not match expected value of https://github.com/some/collection/alternate-kabanero-index.yaml. RepositoryUrl found: ", newCollection.Spec.RepositoryUrl)
	}
	if newCollection.Spec.Version != "4.5.6" {
		t.Fatal("New collection.Spec.Version values do not match expected value of 4.5.6. Version found: ", newCollection.Spec.Version)
	}
	if newCollection.Spec.DesiredState != "inactive" {
		t.Fatal("New collection.Spec.DesiredState values do not match expected value of inactive. DesiredStateme found: ", newCollection.Spec.DesiredState)
	}
}

// Current collection.Spec != New collection.Spec.
// Current collection.Spec != New collection.Spec.versions[0]
// Current collection.Spec.Versions[] (empty).
// Expectation: An error condition should be reported.
func Test6(t *testing.T) {
	newCollection := mutatingBaseCollectionVersions.DeepCopy()
	newCollection.Spec.RepositoryUrl = "https://github.com/some/collection/other-alternate-kabanero-index.yaml"
	newCollection.Spec.Version = "7.8.9"
	newCollection.Spec.DesiredState = "active"
	newCollection.Spec.Versions[0].RepositoryUrl = "https://github.com/some/collection/alternate-kabanero-index.yaml"
	newCollection.Spec.Versions[0].Version = "4.5.6"
	newCollection.Spec.Versions[0].DesiredState = "inactive"
	err := processUpdate(&mutatingBaseCollection, newCollection)
	if err == nil {
		t.Fatal("An error condition should have been reported. New collection.Spec and new collection.Spec.versions[0] contain conflicting data.", err)
	}
}

// Current collection.Spec == New collection.Spec.
// Current collection.Spec == New collection.Spec.versions[0]
// Current collection.Spec.Versions[] (empty).
// Expectation: No change should have taken place. Everything should still be the same.
func Test7(t *testing.T) {
	newCollection := mutatingBaseCollectionVersions.DeepCopy()
	err := processUpdate(&mutatingBaseCollection, newCollection)
	if err != nil {
		t.Fatal("Unexpected error during mutation.", err)
	}

	expectedversion0 := kabanerov1alpha1.CollectionVersion{
		DesiredState:  "active",
		RepositoryUrl: "https://github.com/some/collection/kabanero-index.yaml",
		Version:       "1.2.3"}

	if newCollection.Spec.Versions[0] != expectedversion0 {
		t.Fatal("New collection.Spec.Versions[0] values do not match expected collection.Spec.Versions[0] values. New versions[0]: ", newCollection.Spec.Versions[0], "Expected versions[0]: ", expectedversion0)
	}

	if newCollection.Spec.RepositoryUrl != "https://github.com/some/collection/kabanero-index.yaml" {
		t.Fatal("New collection.Spec.RepositoryUrl values do not match expected value of https://github.com/some/collection/kabanero-index.yaml. RepositoryUrl found: ", newCollection.Spec.RepositoryUrl)
	}
	if newCollection.Spec.Version != "1.2.3" {
		t.Fatal("New collection.Spec.Version values do not match expected value of 1.2.3. Version found: ", newCollection.Spec.Version)
	}
	if newCollection.Spec.DesiredState != "active" {
		t.Fatal("New collection.Spec.DesiredState values do not match expected value of active. DesiredStateme found: ", newCollection.Spec.DesiredState)
	}
}

// Current collection.Spec == New collection.Spec.
// Current collection.Spec != New collection.Spec.versions[0] because collection.Spec.versions[0] values were cleared.
// Current collection.Spec.Versions[] (empty).
// Expectation: New collection.Spec values should have been copied to New collection.Spec.versions[0]. The same behavior
// should be applied as the case where collection.Spec.versions is empty.
func Test8(t *testing.T) {
	newCollection := mutatingBaseCollectionVersions.DeepCopy()
	newCollection.Spec.Versions[0].RepositoryUrl = ""
	newCollection.Spec.Versions[0].Version = ""
	newCollection.Spec.Versions[0].DesiredState = ""

	err := processUpdate(&mutatingBaseCollection, newCollection)
	if err != nil {
		t.Fatal("Unexpected error during mutation.", err)
	}

	expectedversion0 := kabanerov1alpha1.CollectionVersion{
		DesiredState:  "active",
		RepositoryUrl: "https://github.com/some/collection/kabanero-index.yaml",
		Version:       "1.2.3"}

	if newCollection.Spec.Versions[0] != expectedversion0 {
		t.Fatal("New collection.Spec.Versions[0] values do not match expected collection.Spec.Versions[0] values. New versions[0]: ", newCollection.Spec.Versions[0], "Expected versions[0]: ", expectedversion0)
	}
}

// New collection.Spec.Versions[0] == New collection.Spec.
// New collection.Spec.Versions[0] has all values cleared == New collection.Spec has all values cleared.
// Expectation: This is an invalis case.
func Test9(t *testing.T) {
	newCollection := mutatingBaseCollectionVersions.DeepCopy()
	newCollection.Spec.RepositoryUrl = ""
	newCollection.Spec.Version = ""
	newCollection.Spec.DesiredState = ""
	newCollection.Spec.Versions[0].RepositoryUrl = ""
	newCollection.Spec.Versions[0].Version = ""
	newCollection.Spec.Versions[0].DesiredState = ""

	err := processUpdate(&mutatingBaseCollection, newCollection)
	if err == nil {
		t.Fatal("An error condition should have been reported. New collection.Spec and new collection.Spec.versions[0] contain empty fields.", err)
	}
}

// New colleciton.Spec != new collection.Spec.Versions[0].
// Current colleciton.Spec != new collection.Spec.
// Current collection.Spec != new collection.Spec.Versions[0].
// Current collection.Spec.Versions[0] = new collection.Spec.Versions[0].
// Expectation: new collection.Spec values should be copied to new collection.Spec.Versions[0]
func Test10(t *testing.T) {
	custommutatingBaseCollection := mutatingBaseCollectionVersions.DeepCopy()
	newCollection := mutatingBaseCollectionVersions.DeepCopy()
	custommutatingBaseCollection.Spec.Version = "1.2.4"
	newCollection.Spec.Version = "1.2.5"
	custommutatingBaseCollection.Spec.Versions[0].Version = "2.0.0"
	newCollection.Spec.Versions[0].Version = "2.0.0"

	err := processUpdate(custommutatingBaseCollection, newCollection)
	if err != nil {
		t.Fatal("Unexpected error during mutation.", err)
	}

	expectedversion0 := kabanerov1alpha1.CollectionVersion{
		DesiredState:  "active",
		RepositoryUrl: "https://github.com/some/collection/kabanero-index.yaml",
		Version:       "1.2.5"}

	if newCollection.Spec.Versions[0] != expectedversion0 {
		t.Fatal("New collection.Spec.Versions[0] values do not match expected collection.Spec.Versions[0] values. New versions[0]: ", newCollection.Spec.Versions[0], "Expected versions[0]: ", expectedversion0)
	}
}

// New colleciton.Spec != new collection.Spec.Versions[0].
// Current colleciton.Spec == new collection.Spec.
// Current collection.Spec != new collection.Spec.Versions[0].
// Current collection.Spec.Versions[0] != new collection.Spec.Versions[0].
// Expectation: new collection.Spec.Versions[0] values should be copied to new collection.Spec.
func Test11(t *testing.T) {
	custommutatingBaseCollection := mutatingBaseCollectionVersions.DeepCopy()
	newCollection := mutatingBaseCollectionVersions.DeepCopy()
	custommutatingBaseCollection.Spec.Version = "1.2.4"
	newCollection.Spec.Version = "1.2.4"
	custommutatingBaseCollection.Spec.Versions[0].Version = "2.0.0"
	newCollection.Spec.Versions[0].Version = "2.0.1"

	err := processUpdate(custommutatingBaseCollection, newCollection)
	if err != nil {
		t.Fatal("Unexpected error during mutation.", err)
	}

	if newCollection.Spec.RepositoryUrl != "https://github.com/some/collection/kabanero-index.yaml" {
		t.Fatal("New collection.Spec.RepositoryUrl values do not match expected value of https://github.com/some/collection/kabanero-index.yaml. RepositoryUrl found: ", newCollection.Spec.RepositoryUrl)
	}
	if newCollection.Spec.Version != "2.0.1" {
		t.Fatal("New collection.Spec.Version values do not match expected value of 1.2.3. Version found: ", newCollection.Spec.Version)
	}
	if newCollection.Spec.DesiredState != "active" {
		t.Fatal("New collection.Spec.DesiredState values do not match expected value of active. DesiredStateme found: ", newCollection.Spec.DesiredState)
	}
}

// New colleciton.Spec == new collection.Spec.Versions[0].
// Current colleciton.Spec != new collection.Spec.
// Current collection.Spec.Versions[0] == new collection.Spec.Versions[0].
// Expectation: No updates are expected.
func Test12(t *testing.T) {
	custommutatingBaseCollection := mutatingBaseCollectionVersions.DeepCopy()
	newCollection := mutatingBaseCollectionVersions.DeepCopy()
	custommutatingBaseCollection.Spec.Version = "1.2.3"
	newCollection.Spec.Version = "1.2.4"
	custommutatingBaseCollection.Spec.Versions[0].Version = "1.2.4"
	newCollection.Spec.Versions[0].Version = "1.2.4"

	err := processUpdate(custommutatingBaseCollection, newCollection)
	if err != nil {
		t.Fatal("Unexpected error during mutation.", err)
	}

	expectedversion0 := kabanerov1alpha1.CollectionVersion{
		DesiredState:  "active",
		RepositoryUrl: "https://github.com/some/collection/kabanero-index.yaml",
		Version:       "1.2.4"}

	if newCollection.Spec.Versions[0] != expectedversion0 {
		t.Fatal("New collection.Spec.Versions[0] values do not match expected collection.Spec.Versions[0] values. New versions[0]: ", newCollection.Spec.Versions[0], "Expected versions[0]: ", expectedversion0)
	}

	if newCollection.Spec.RepositoryUrl != "https://github.com/some/collection/kabanero-index.yaml" {
		t.Fatal("New collection.Spec.RepositoryUrl values do not match expected value of https://github.com/some/collection/kabanero-index.yaml. RepositoryUrl found: ", newCollection.Spec.RepositoryUrl)
	}
	if newCollection.Spec.Version != "1.2.4" {
		t.Fatal("New collection.Spec.Version values do not match expected value of 1.2.3. Version found: ", newCollection.Spec.Version)
	}
	if newCollection.Spec.DesiredState != "active" {
		t.Fatal("New collection.Spec.DesiredState values do not match expected value of active. DesiredStateme found: ", newCollection.Spec.DesiredState)
	}
}

// New colleciton.Spec != new collection.Spec.Versions[0].
// Current colleciton.Spec != new collection.Spec.
// Current collection.Spec != new collection.Spec.Versions[0].
// Current collection.Spec.Versions[0] = new collection.Spec.Versions[0].
// Expectation: This is an invalis case and is unrecoverable. New and current version values are different.
// The instance may need to be re-deployed.
func Test13(t *testing.T) {
	custommutatingBaseCollection := mutatingBaseCollectionVersions.DeepCopy()
	newCollection := mutatingBaseCollectionVersions.DeepCopy()
	custommutatingBaseCollection.Spec.Version = "1.2.4"
	newCollection.Spec.Version = "1.2.5"
	custommutatingBaseCollection.Spec.Versions[0].Version = "2.0.0"
	newCollection.Spec.Versions[0].Version = "2.0.1"
	err := processUpdate(custommutatingBaseCollection, newCollection)
	if err == nil {
		t.Fatal("An error condition should have been reported. Current and new collection.Spec and current and new collection.Spec.versions[0] have different values.", err)
	}
}
