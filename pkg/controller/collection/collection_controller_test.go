package collection

import (
	"testing"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileCollection(t *testing.T) {
	r := &ReconcileCollection{indexResolver: func(kabanerov1alpha1.RepositoryConfig) (*Index, error) {
		return &Index{
			URL:        "http://some/URL/to/V2/collection/index",
			APIVersion: "v2",
			Collections: []Collection{
				Collection{
					DefaultImage:    "java-microprofile",
					DefaultPipeline: "default",
					DefaultTemplate: "default",
					Description:     "Test collection",
					Id:              "java-microprofile",
					Images: []Images{
						Images{
							Id:    "java-microprofile",
							Image: "kabanero/java-microprofile:0.2",
						},
					},
					Maintainers: []Maintainers{
						Maintainers{
							Email:    "maintainer@someemail.ibm.com",
							GithubId: "maintainer",
							Name:     "Joe Maintainer",
						},
					},
					Name: "Eclipse Microprofile",
					Pipelines: []Pipelines{
						Pipelines{},
					},
				},
			},
		}, nil
	}}

	c := &kabanerov1alpha1.Collection{
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
			Name:         "somename",
			DesiredState: "active",
		},
	}

	k := &kabanerov1alpha1.Kabanero{
		ObjectMeta: metav1.ObjectMeta{UID: "1"},
	}

	r.ReconcileCollection(c, k)
}

// Test that a status is updated with a new digest
func TestUpdateAssetStatus(t *testing.T) {
	var sampleAssetStatus = []kabanerov1alpha1.RepositoryAssetStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", Status: "active"}}
	var status = kabanerov1alpha1.CollectionStatus{ActiveAssets: sampleAssetStatus}
	var sampleAsset = Pipelines{
		Id:     "myAsset",
		Sha256: "12345",
		Url:    "http://myurl.com",
	}

	updateAssetStatus(&status, sampleAsset, "")

	if len(status.ActiveAssets) != 1 {
		t.Fatal("Status should have one asset in it")
	}

	newStatus := status.ActiveAssets[0]
	if newStatus.Name != sampleAsset.Id {
		t.Fatal("Status name does not equal asset name")
	}

	if newStatus.Url != sampleAsset.Url {
		t.Fatal("Status URL does not equal asset URL")
	}

	if newStatus.Digest != sampleAsset.Sha256 {
		t.Fatal("Status digest (" + newStatus.Digest + ") does not equal asset digest (" + sampleAsset.Sha256 + ")")
	}

	if newStatus.Status != "active" {
		t.Fatal("Asset status is not active: " + newStatus.Status)
	}

	if newStatus.StatusMessage != "" {
		t.Fatal("Asset status message is not empty: " + newStatus.StatusMessage)
	}
}

// Test that a previous asset failure can be resolved in the status.
func TestUpdateAssetStatusFromFailure(t *testing.T) {
	var sampleAssetStatus = []kabanerov1alpha1.RepositoryAssetStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", Status: "failed", StatusMessage: "some error"}}
	var status = kabanerov1alpha1.CollectionStatus{ActiveAssets: sampleAssetStatus}
	var sampleAsset = Pipelines{
		Id:     "myAsset",
		Sha256: "12345",
		Url:    "http://myurl.com",
	}

	updateAssetStatus(&status, sampleAsset, "")

	if len(status.ActiveAssets) != 1 {
		t.Fatal("Status should have one asset in it")
	}

	newStatus := status.ActiveAssets[0]
	if newStatus.Name != sampleAsset.Id {
		t.Fatal("Status name does not equal asset name")
	}

	if newStatus.Url != sampleAsset.Url {
		t.Fatal("Status URL does not equal asset URL")
	}

	if newStatus.Digest != sampleAsset.Sha256 {
		t.Fatal("Status digest (" + newStatus.Digest + ") does not equal asset digest (" + sampleAsset.Sha256 + ")")
	}

	if newStatus.Status != "active" {
		t.Fatal("Asset status is not active: " + newStatus.Status)
	}

	if newStatus.StatusMessage != "" {
		t.Fatal("Asset status message is not empty: " + newStatus.StatusMessage)
	}
}

// Test that a previous active asset can be updated with a failure
func TestUpdateAssetStatusToFailure(t *testing.T) {
	var sampleAssetStatus = []kabanerov1alpha1.RepositoryAssetStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", Status: "active"}}
	var status = kabanerov1alpha1.CollectionStatus{ActiveAssets: sampleAssetStatus}
	var sampleAsset = Pipelines{
		Id:     "myAsset",
		Sha256: "12345",
		Url:    "http://myurl.com",
	}

	errorMessage := "some failure happened"
	updateAssetStatus(&status, sampleAsset, errorMessage)

	if len(status.ActiveAssets) != 1 {
		t.Fatal("Status should have one asset in it")
	}

	newStatus := status.ActiveAssets[0]
	if newStatus.Name != sampleAsset.Id {
		t.Fatal("Status name does not equal asset name")
	}

	if newStatus.Url != sampleAsset.Url {
		t.Fatal("Status URL does not equal asset URL")
	}

	if newStatus.Digest != sampleAsset.Sha256 {
		t.Fatal("Status digest (" + newStatus.Digest + ") does not equal asset digest (" + sampleAsset.Sha256 + ")")
	}

	if newStatus.Status != "failed" {
		t.Fatal("Asset status is not failed: " + newStatus.Status)
	}

	if newStatus.StatusMessage != errorMessage {
		t.Fatal("Asset status message is not correct, should be " + errorMessage + " but is: " + newStatus.StatusMessage)
	}
}

// Test that failed assets are detected in the Collection instance status
func TestFailedAssets(t *testing.T) {
	var sampleAssetStatus = []kabanerov1alpha1.RepositoryAssetStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", Status: "active"}, {Name: "myOtherAsset", Url: "http://myotherurl.com", Digest: "2345", Status: "failed"}}
	var status = kabanerov1alpha1.CollectionStatus{ActiveAssets: sampleAssetStatus}

	if failedAssets(status) == false {
		t.Fatal("Should be one failed asset in the status")
	}
}

// Test that no failed assets are detected in the Collection instance status
func TestNoFailedAssets(t *testing.T) {
	var sampleAssetStatus = []kabanerov1alpha1.RepositoryAssetStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", Status: "active"}, {Name: "myOtherAsset", Url: "http://myotherurl.com", Digest: "2345", Status: "active"}}
	var status = kabanerov1alpha1.CollectionStatus{ActiveAssets: sampleAssetStatus}

	if failedAssets(status) {
		t.Fatal("Should be no failed asset in the status")
	}
}

// Test that an empty status yields no failed assets
func TestNoFailedAssetsEmptyStatus(t *testing.T) {
	var sampleAssetStatus = []kabanerov1alpha1.RepositoryAssetStatus{}
	var status = kabanerov1alpha1.CollectionStatus{ActiveAssets: sampleAssetStatus}

	if failedAssets(status) {
		t.Fatal("Should be no failed asset in the status")
	}
}

// Test that I can find the max version of a collection
func TestFindMaxVersionCollection(t *testing.T) {

	collection1 := Collection{Version: "0.0.1"}
	collection2 := Collection{Version: "0.0.2"}

	var collections []resolvedCollection
	collections = append(collections, resolvedCollection{collection: collection1})
	collections = append(collections, resolvedCollection{collection: collection2})

	maxCollection := findMaxVersionCollection(collections)

	if maxCollection == nil {
		t.Fatal("Did not find a max version")
	}

	if maxCollection.collection.Version != "0.0.2" {
		t.Fatal("Returned max version (" + maxCollection.collection.Version + ") was not 0.0.2")
	}
}

// Test that I can find the max version of a collection when there is only 1 collection
func TestFindMaxVersionCollectionOne(t *testing.T) {

	collection1 := Collection{Version: "0.0.1"}

	var collections []resolvedCollection
	collections = append(collections, resolvedCollection{collection: collection1})

	maxCollection := findMaxVersionCollection(collections)

	if maxCollection == nil {
		t.Fatal("Did not find a max version")
	}

	if maxCollection.collection.Version != "0.0.1" {
		t.Fatal("Returned max version (" + maxCollection.collection.Version + ") was not 0.0.1")
	}
}

// Test that I do not find a max version with no input collections.  Technically this is an invalid case.
func TestFindMaxVersionCollectionEmpty(t *testing.T) {

	var collections []resolvedCollection
	maxCollection := findMaxVersionCollection(collections)

	if maxCollection != nil {
		t.Fatal("Should not have found a max version.")
	}
}

// Test that I can specify just a major.minor semver (invalid) and still be OK
func TestFindMaxVersionCollectionMajorMinor(t *testing.T) {

	collection1 := Collection{Version: "0.1"}
	collection2 := Collection{Version: "0.2"}

	var collections []resolvedCollection
	collections = append(collections, resolvedCollection{collection: collection1})
	collections = append(collections, resolvedCollection{collection: collection2})

	maxCollection := findMaxVersionCollection(collections)

	if maxCollection == nil {
		t.Fatal("Did not find a max version")
	}

	if maxCollection.collection.Version != "0.2" {
		t.Fatal("Returned max version (" + maxCollection.collection.Version + ") was not 0.2")
	}
}

// Test that if I just have invalid semvers, I don't find a max version.
func TestFindMaxVersionCollectionInvalidSemver(t *testing.T) {

	collection1 := Collection{Version: "blah"}
	collection2 := Collection{Version: "5nope"}

	var collections []resolvedCollection
	collections = append(collections, resolvedCollection{collection: collection1})
	collections = append(collections, resolvedCollection{collection: collection2})

	maxCollection := findMaxVersionCollection(collections)

	if maxCollection != nil {
		t.Fatal("Should not have found a max version (" + maxCollection.collection.Version + ")")
	}
}
