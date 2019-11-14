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

	var sampleAsset = []kabanerov1alpha1.RepositoryAssetStatus{{
		Name:   "myAsset",
		Digest: "678910",
		Status: "active"},
	}

	var samplePipeline = Pipelines{
		Id:     "myPipeline",
		Sha256: "12345",
		Url:    "http://myurl.com",
	}

	var sampleCollectionAsset = CollectionAsset{
		Name:   "myAsset",
		Sha256: "678910",
	}

	var samplePipelineStatus = []kabanerov1alpha1.PipelineStatus{{Name: "myPipeline", Url: "http://myurl.com", Digest: "12345", ActiveAssets: sampleAsset}}

	status := kabanerov1alpha1.CollectionStatus{ActivePipelines: samplePipelineStatus}

	updateAssetStatus(&status, samplePipeline, sampleCollectionAsset, "", "active")

	if len(status.ActivePipelines) != 1 {
		t.Fatal("Status should have one asset in it. Currentl Length: ", status.ActivePipelines)
	}

	newStatus := status.ActivePipelines[0]
	if newStatus.Name != samplePipeline.Id {
		t.Fatal("Status name does not equal asset name")
	}

	if newStatus.Url != samplePipeline.Url {
		t.Fatal("Status URL does not equal asset URL")
	}

	if newStatus.Digest != samplePipeline.Sha256 {
		t.Fatal("Status digest (" + newStatus.Digest + ") does not equal asset digest (" + samplePipeline.Sha256 + ")")
	}

	if newStatus.ActiveAssets[0].Status != "active" {
		t.Fatal("Asset status is not active: " + newStatus.ActiveAssets[0].Status)
	}

	if newStatus.ActiveAssets[0].StatusMessage != "" {
		t.Fatal("Asset status message is not empty: " + newStatus.ActiveAssets[0].StatusMessage)
	}
}

// Test that a previous asset failure can be resolved in the status.
func TestUpdateAssetStatusFromFailure(t *testing.T) {
	var sampleAsset = []kabanerov1alpha1.RepositoryAssetStatus{{
		Name:          "myAsset",
		Digest:        "678910",
		Status:        "failed",
		StatusMessage: "someError"},
	}

	var samplePipeline = Pipelines{
		Id:     "myPipeline",
		Sha256: "12345",
		Url:    "http://myurl.com",
	}

	var sampleCollectionAsset = CollectionAsset{
		Name:   "myAsset",
		Sha256: "678910",
	}
	var samplePipelineStatus = []kabanerov1alpha1.PipelineStatus{{Name: "myPipeline", Url: "http://myurl.com", Digest: "12345", ActiveAssets: sampleAsset}}
	status := kabanerov1alpha1.CollectionStatus{ActivePipelines: samplePipelineStatus}

	updateAssetStatus(&status, samplePipeline, sampleCollectionAsset, "", "active")

	if len(status.ActivePipelines) != 1 {
		t.Fatal("Status should have one asset in it")
	}

	newStatus := status.ActivePipelines[0]
	if newStatus.Name != samplePipeline.Id {
		t.Fatal("Status name does not equal asset name")
	}

	if newStatus.Url != samplePipeline.Url {
		t.Fatal("Status URL does not equal asset URL")
	}

	if newStatus.Digest != samplePipeline.Sha256 {
		t.Fatal("Status digest (" + newStatus.Digest + ") does not equal asset digest (" + samplePipeline.Sha256 + ")")
	}

	if newStatus.ActiveAssets[0].Status != "active" {
		t.Fatal("Asset status is not active: " + newStatus.ActiveAssets[0].Status)
	}

	if newStatus.ActiveAssets[0].StatusMessage != "" {
		t.Fatal("Asset status message is not empty: " + newStatus.ActiveAssets[0].StatusMessage)
	}
}

// Test that a previous active asset can be updated with a failure
func TestUpdateAssetStatusToFailure(t *testing.T) {
	var sampleAsset = []kabanerov1alpha1.RepositoryAssetStatus{{
		Name:   "myAsset",
		Digest: "678910",
		Status: "active"},
	}

	var samplePipeline = Pipelines{
		Id:     "myPipeline",
		Sha256: "12345",
		Url:    "http://myurl.com",
	}

	var sampleCollectionAsset = CollectionAsset{
		Name:   "myAsset",
		Sha256: "678910",
	}
	var samplePipelineStatus = []kabanerov1alpha1.PipelineStatus{{Name: "myPipeline", Url: "http://myurl.com", Digest: "12345", ActiveAssets: sampleAsset}}
	status := kabanerov1alpha1.CollectionStatus{ActivePipelines: samplePipelineStatus}

	errorMessage := "some failure happened"
	updateAssetStatus(&status, samplePipeline, sampleCollectionAsset, errorMessage, "failed")

	if len(status.ActivePipelines) != 1 {
		t.Fatal("Status should have one asset in it")
	}

	newStatus := status.ActivePipelines[0]
	if newStatus.Name != samplePipeline.Id {
		t.Fatal("Status name does not equal asset name")
	}

	if newStatus.Url != samplePipeline.Url {
		t.Fatal("Status URL does not equal asset URL")
	}

	if newStatus.Digest != samplePipeline.Sha256 {
		t.Fatal("Status digest (" + newStatus.Digest + ") does not equal asset digest (" + samplePipeline.Sha256 + ")")
	}

	if newStatus.ActiveAssets[0].Status != "failed" {
		t.Fatal("Asset status is not active: " + newStatus.ActiveAssets[0].Status)
	}

	if newStatus.ActiveAssets[0].StatusMessage != errorMessage {
		t.Fatal("Asset status message is not empty: " + newStatus.ActiveAssets[0].StatusMessage)
	}
}

// Test that failed assets are detected in the Collection instance status
func TestFailedAssets(t *testing.T) {
	var sampleAsset = []kabanerov1alpha1.RepositoryAssetStatus{{Name: "myAsset", Digest: "678910", Status: "active"},
		{Name: "myAsset2", Digest: "678911", Status: "failed", StatusMessage: "some failure"},
	}

	var samplePipelineStatus = []kabanerov1alpha1.PipelineStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", ActiveAssets: sampleAsset}}
	status := kabanerov1alpha1.CollectionStatus{ActivePipelines: samplePipelineStatus}

	if failedAssets(status) == false {
		t.Fatal("Should be one failed asset in the status")
	}
}

// Test that no failed assets are detected in the Collection instance status
func TestNoFailedAssets(t *testing.T) {
	var sampleAsset = []kabanerov1alpha1.RepositoryAssetStatus{{Name: "myAsset", Digest: "678910", Status: "active"},
		{Name: "myAsset2", Digest: "678911", Status: "active"},
	}

	var samplePipelineStatus = []kabanerov1alpha1.PipelineStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", ActiveAssets: sampleAsset}}
	status := kabanerov1alpha1.CollectionStatus{ActivePipelines: samplePipelineStatus}

	if failedAssets(status) {
		t.Fatal("Should be no failed asset in the status")
	}
}

// Test that an empty status yields no failed assets
func TestNoFailedAssetsEmptyStatus(t *testing.T) {
	var samplePipelineStatus = []kabanerov1alpha1.PipelineStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{}}}
	status := kabanerov1alpha1.CollectionStatus{ActivePipelines: samplePipelineStatus}

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
