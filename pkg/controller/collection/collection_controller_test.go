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
	var sampleStatus = kabanerov1alpha1.RepositoryAssetStatus{"myAsset", "http://myurl.com", "abcd"}
	var sampleAsset = AssetManifest{"myAsset", "type", "http://myurl.com", "abcd"}
	if !assetMatch(sampleStatus, sampleAsset) {
		t.Fatal("Status and asset do not match")
	}
}

// Test non-equality between assets that are not equal
func TestAssetNotMatch(t *testing.T) {
	var sampleStatus = kabanerov1alpha1.RepositoryAssetStatus{"myOtherAsset", "http://myurl.com", "abcd"}
	var sampleAsset = AssetManifest{"myAsset", "type", "http://myurl.com", "abcd"}
	if assetMatch(sampleStatus, sampleAsset) {
		t.Fatal("Status and asset should not match")
	}
}

// Test equality between asset and status when there is no name
func TestAssetMatchNoName(t *testing.T) {
	var sampleStatus = kabanerov1alpha1.RepositoryAssetStatus{"", "http://myurl.com", "abcd"}
	var sampleAsset = AssetManifest{"", "type", "http://myurl.com", "abcd"}
	if !assetMatch(sampleStatus, sampleAsset) {
		t.Fatal("Status and asset do not match")
	}
}

// Test non-equality between assets that are not equal and do not have a name
func TestAssetNotMatchNoName(t *testing.T) {
	var sampleStatus = kabanerov1alpha1.RepositoryAssetStatus{"", "http://myurl.com", "abcd"}
	var sampleAsset = AssetManifest{"", "type", "http://myurl.net", "abcd"}
	if assetMatch(sampleStatus, sampleAsset) {
		t.Fatal("Status and asset should not match")
	}
}

// Test that a new status is created for an asset that was not present in the status
func TestUpdateResourceDigestNotExist(t *testing.T) {
	var status = kabanerov1alpha1.CollectionStatus{"0.0.1", "abcd", nil}
	var sampleAsset = AssetManifest{"myAsset", "type", "http://myurl.com", "efgh"}

	updateResouceDigest(&status, sampleAsset);

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
}

// Test that a status is updated with a new digest
func TestUpdateResourceDigest(t *testing.T) {
	var sampleAssetStatus = []kabanerov1alpha1.RepositoryAssetStatus{{"myAsset", "http://myurl.com", "1234"}}
	var status = kabanerov1alpha1.CollectionStatus{"0.0.1", "abcd", sampleAssetStatus}
	var sampleAsset = AssetManifest{"myAsset", "type", "http://myurl.com", "efgh"}

	updateResouceDigest(&status, sampleAsset);

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
}
