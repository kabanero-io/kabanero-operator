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

// Test equality between an asset with a name, and the kube status
func TestAssetMatch(t *testing.T) {
	var sampleStatus = kabanerov1alpha1.RepositoryAssetStatus{Name: "myAsset", Url: "http://myurl.com", Digest: "abcd", Status: "active"}
	var sampleAsset = AssetManifest{"myAsset", "type", "http://myurl.com", "abcd"}
	if !assetMatch(sampleStatus, sampleAsset) {
		t.Fatal("Status and asset do not match")
	}
}

// Test non-equality between assets that are not equal
func TestAssetNotMatch(t *testing.T) {
	var sampleStatus = kabanerov1alpha1.RepositoryAssetStatus{Name: "myOtherAsset", Url: "http://myurl.com", Digest: "abcd", Status: "active"}
	var sampleAsset = AssetManifest{"myAsset", "type", "http://myurl.com", "abcd"}
	if assetMatch(sampleStatus, sampleAsset) {
		t.Fatal("Status and asset should not match")
	}
}

// Test equality between asset and status when there is no name
func TestAssetMatchNoName(t *testing.T) {
	var sampleStatus = kabanerov1alpha1.RepositoryAssetStatus{Url: "http://myurl.com", Digest: "abcd", Status: "active"}
	var sampleAsset = AssetManifest{"", "type", "http://myurl.com", "abcd"}
	if !assetMatch(sampleStatus, sampleAsset) {
		t.Fatal("Status and asset do not match")
	}
}

// Test non-equality between assets that are not equal and do not have a name
func TestAssetNotMatchNoName(t *testing.T) {
	var sampleStatus = kabanerov1alpha1.RepositoryAssetStatus{Url: "http://myurl.com", Digest: "abcd", Status: "active"}
	var sampleAsset = AssetManifest{"", "type", "http://myurl.net", "abcd"}
	if assetMatch(sampleStatus, sampleAsset) {
		t.Fatal("Status and asset should not match")
	}
}

// Test that a new status is created for an asset that was not present in the status
func TestUpdateAssetStatusNotExist(t *testing.T) {
	var status = kabanerov1alpha1.CollectionStatus{}
	var sampleAsset = AssetManifest{"myAsset", "type", "http://myurl.com", "efgh"}

	updateAssetStatus(&status, sampleAsset, "");

	if len(status.ActiveAssets) != 1 {
		t.Fatal("Status should have one asset in it")
	}

	newStatus := status.ActiveAssets[0]
	if (newStatus.Name != sampleAsset.Name) {
		t.Fatal("Status name does not equal asset name")
	}

	if (newStatus.Url != sampleAsset.Url) {
		t.Fatal("Status URL does not equal asset URL")
	}

	if (newStatus.Digest != sampleAsset.Digest) {
		t.Fatal("Status digest does not equal asset digest")
	}

	if (newStatus.Status != "active") {
		t.Fatal("Status of asset is not active: " + newStatus.Status)
	}

	if (newStatus.StatusMessage != "") {
		t.Fatal("Status message of asset is not empty: " + newStatus.StatusMessage)
	}
}

// Test that a status is updated with a new digest
func TestUpdateAssetStatus(t *testing.T) {
	var sampleAssetStatus = []kabanerov1alpha1.RepositoryAssetStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", Status: "active"}}
	var status = kabanerov1alpha1.CollectionStatus{ActiveAssets: sampleAssetStatus}
	var sampleAsset = AssetManifest{"myAsset", "type", "http://myurl.com", "efgh"}

	updateAssetStatus(&status, sampleAsset, "");

	if len(status.ActiveAssets) != 1 {
		t.Fatal("Status should have one asset in it")
	}

	newStatus := status.ActiveAssets[0]
	if (newStatus.Name != sampleAsset.Name) {
		t.Fatal("Status name does not equal asset name")
	}

	if (newStatus.Url != sampleAsset.Url) {
		t.Fatal("Status URL does not equal asset URL")
	}

	if (newStatus.Digest != sampleAsset.Digest) {
		t.Fatal("Status digest (" + newStatus.Digest + ") does not equal asset digest (" + sampleAsset.Digest + ")")
	}

	if (newStatus.Status != "active") {
		t.Fatal("Asset status is not active: " + newStatus.Status)
	}

	if (newStatus.StatusMessage != "") {
		t.Fatal("Asset status message is not empty: " + newStatus.StatusMessage)
	}
}

// Test that a previous asset failure can be resolved in the status.
func TestUpdateAssetStatusFromFailure(t *testing.T) {
	var sampleAssetStatus = []kabanerov1alpha1.RepositoryAssetStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", Status: "failed", StatusMessage: "some error"}}
	var status = kabanerov1alpha1.CollectionStatus{ActiveAssets: sampleAssetStatus}
	var sampleAsset = AssetManifest{"myAsset", "type", "http://myurl.com", "efgh"}

	updateAssetStatus(&status, sampleAsset, "");

	if len(status.ActiveAssets) != 1 {
		t.Fatal("Status should have one asset in it")
	}

	newStatus := status.ActiveAssets[0]
	if (newStatus.Name != sampleAsset.Name) {
		t.Fatal("Status name does not equal asset name")
	}

	if (newStatus.Url != sampleAsset.Url) {
		t.Fatal("Status URL does not equal asset URL")
	}

	if (newStatus.Digest != sampleAsset.Digest) {
		t.Fatal("Status digest (" + newStatus.Digest + ") does not equal asset digest (" + sampleAsset.Digest + ")")
	}

	if (newStatus.Status != "active") {
		t.Fatal("Asset status is not active: " + newStatus.Status)
	}

	if (newStatus.StatusMessage != "") {
		t.Fatal("Asset status message is not empty: " + newStatus.StatusMessage)
	}
}

// Test that a previous active asset can be updated with a failure
func TestUpdateAssetStatusToFailure(t *testing.T) {
	var sampleAssetStatus = []kabanerov1alpha1.RepositoryAssetStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", Status: "active"}}
	var status = kabanerov1alpha1.CollectionStatus{ActiveAssets: sampleAssetStatus}
	var sampleAsset = AssetManifest{"myAsset", "type", "http://myurl.com", "efgh"}

	errorMessage := "some failure happened"
	updateAssetStatus(&status, sampleAsset, errorMessage);

	if len(status.ActiveAssets) != 1 {
		t.Fatal("Status should have one asset in it")
	}

	newStatus := status.ActiveAssets[0]
	if (newStatus.Name != sampleAsset.Name) {
		t.Fatal("Status name does not equal asset name")
	}

	if (newStatus.Url != sampleAsset.Url) {
		t.Fatal("Status URL does not equal asset URL")
	}

	if (newStatus.Digest != sampleAsset.Digest) {
		t.Fatal("Status digest (" + newStatus.Digest + ") does not equal asset digest (" + sampleAsset.Digest + ")")
	}

	if (newStatus.Status != "failed") {
		t.Fatal("Asset status is not failed: " + newStatus.Status)
	}

	if (newStatus.StatusMessage != errorMessage) {
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

	collection1 := CollectionV1{Manifest: CollectionV1Manifest{Version: "0.0.1"}}
	collection2 := CollectionV1{Manifest: CollectionV1Manifest{Version: "0.0.2"}}
	
	var collections []resolvedCollection
	collections = append(collections, resolvedCollection{collection: collection1})
	collections = append(collections, resolvedCollection{collection: collection2})
	
	maxCollection := findMaxVersionCollection(collections)

	if maxCollection == nil {
		t.Fatal("Did not find a max version")
	}

	if maxCollection.collection.Manifest.Version != "0.0.2" {
		t.Fatal("Returned max version (" + maxCollection.collection.Manifest.Version + ") was not 0.0.2")
	}
}

// Test that I can find the max version of a collection when there is only 1 collection
func TestFindMaxVersionCollectionOne(t *testing.T) {

	collection1 := CollectionV1{Manifest: CollectionV1Manifest{Version: "0.0.1"}}
	
	var collections []resolvedCollection
	collections = append(collections, resolvedCollection{collection: collection1})
	
	maxCollection := findMaxVersionCollection(collections)

	if maxCollection == nil {
		t.Fatal("Did not find a max version")
	}

	if maxCollection.collection.Manifest.Version != "0.0.1" {
		t.Fatal("Returned max version (" + maxCollection.collection.Manifest.Version + ") was not 0.0.1")
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

	collection1 := CollectionV1{Manifest: CollectionV1Manifest{Version: "0.1"}}
	collection2 := CollectionV1{Manifest: CollectionV1Manifest{Version: "0.2"}}
	
	var collections []resolvedCollection
	collections = append(collections, resolvedCollection{collection: collection1})
	collections = append(collections, resolvedCollection{collection: collection2})
	
	maxCollection := findMaxVersionCollection(collections)

	if maxCollection == nil {
		t.Fatal("Did not find a max version")
	}

	if maxCollection.collection.Manifest.Version != "0.2" {
		t.Fatal("Returned max version (" + maxCollection.collection.Manifest.Version + ") was not 0.2")
	}
}

// Test that if I just have invalid semvers, I don't find a max version.
func TestFindMaxVersionCollectionInvalidSemver(t *testing.T) {

	collection1 := CollectionV1{Manifest: CollectionV1Manifest{Version: "blah"}}
	collection2 := CollectionV1{Manifest: CollectionV1Manifest{Version: "5nope"}}
	
	var collections []resolvedCollection
	collections = append(collections, resolvedCollection{collection: collection1})
	collections = append(collections, resolvedCollection{collection: collection2})
	
	maxCollection := findMaxVersionCollection(collections)

	if maxCollection != nil {
		t.Fatal("Should not have found a max version (" + maxCollection.collection.Manifest.Version + ")")
	}
}
