package collection

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

  apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

)

// Set up logging so that the log statements in the product code come out in the test output
type testLogger struct {}
func (t testLogger) Info(msg string, keysAndValues ...interface{}) { fmt.Printf("Info: %v \n", msg) }
func (t testLogger) Enabled() bool { return true }
func (t testLogger) Error(err error, msg string, keysAndValues ...interface{}) { fmt.Printf("Error: %v: %v\n", msg, err.Error()) }
func (t testLogger) V(level int) logr.InfoLogger { return t }
func (t testLogger) WithValues(keysAndValues ...interface{}) logr.Logger { return t }
func (t testLogger) WithName(name string) logr.Logger { return t }

func init() {
	logf.SetLogger(testLogger{})
}

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

// -------------------------------------------------------------------------------
// Asset reuse tests
// -------------------------------------------------------------------------------

type unitTestClient struct {
	// Objects that the client knows about.  This is real simple.... for now.  We just
	// keep the name, and any owner references.
	objs map[string][]metav1.OwnerReference
}

func (c unitTestClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	fmt.Printf("Received Get() for %v\n", key.Name)
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid target object for get: %v\n", obj)
		return errors.New("Get only supports setting into Unstructured")
	}
	owners := c.objs[key.Name]
	if len(owners) == 0 {
		return apierrors.NewNotFound(schema.GroupResource{}, key.Name)
	}
	u.SetName(key.Name)
	u.SetNamespace(key.Namespace)
	u.SetOwnerReferences(owners)
	return nil
}
func (c unitTestClient) List(ctx context.Context, opts *client.ListOptions, list runtime.Object) error { return nil }
func (c unitTestClient) Create(ctx context.Context, obj runtime.Object) error {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid create: %v\n", obj)
		return errors.New("Create only supports Unstructured")
	}

	fmt.Printf("Received Create() for %v\n", u.GetName())
	owners := c.objs[u.GetName()]
	if len(owners) > 0 {
		fmt.Printf("Receive create object already exists: %v\n", u.GetName())
		return apierrors.NewAlreadyExists(schema.GroupResource{}, u.GetName())
	}

	gvk := u.GroupVersionKind()
	if gvk.Kind == "BadTask" {
		message := fmt.Sprintf("Receive create for invalid kind: %v", gvk.Kind)
		fmt.Printf(message + "\n")
		return errors.New(message)
	}
	
	c.objs[u.GetName()] = u.GetOwnerReferences()
	return nil
}
func (c unitTestClient)	Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOptionFunc) error {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid delete: %v\n", obj)
		return errors.New("Delete only supports Unstructured")
	}

	fmt.Printf("Received Delete() for %v\n", u.GetName())
	owners := c.objs[u.GetName()]
	if len(owners) == 0 {
		fmt.Printf("Received delete for an object that does not exist: %v\n", obj)
		return apierrors.NewNotFound(schema.GroupResource{}, u.GetName())
	}
	delete(c.objs, u.GetName())
	return nil
}
func (c unitTestClient) Update(ctx context.Context, obj runtime.Object) error {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid update: %v\n", obj)
		return errors.New("Update only supports Unstructured")
	}

	fmt.Printf("Received Update() for %v\n", u.GetName())
	owners := c.objs[u.GetName()]
	if len(owners) == 0 {
		fmt.Printf("Received update for object that does not exist: %v\n", obj)
		return apierrors.NewNotFound(schema.GroupResource{}, u.GetName())
	}
	c.objs[u.GetName()] = u.GetOwnerReferences()
	return nil
}
func (c unitTestClient) Status() client.StatusWriter { return c }

// HTTP handler that serves pipeline zips
type collectionHandler struct {
}

func (ch collectionHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	filename := fmt.Sprintf("testdata/%v", req.URL.String())
	fmt.Printf("Serving %v\n", filename)
	d, err := ioutil.ReadFile(filename)
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
	} else {
		rw.Write(d)
	}
}

type fileInfo struct {
	name string
	sha256 string
}

const (
	myuid = "MYUID"
	otheruid = "OTHERUID"
)

var basicPipeline = fileInfo{
	name: "/basic.pipeline.tar.gz",
	sha256: "89277396f1f7083ae4c16633949f45860d164534212720b17f25e438e00f66af"}

var badPipeline = fileInfo{
	name: "/bad.pipeline.tar.gz",
	sha256: "6e2d419d7971b2ea60148a995914acda4e42c89cc4b38513aeba0aee18c944f3"}

// --------------------------------------------------------------------------------------------------
// Test that initial collection activation works
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInitial(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()
	
	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{Name: "java-microprofile", Version: "0.2.5", DesiredState: "active"},
		Status: kabanerov1alpha1.CollectionStatus{},
	}

	pipelineZipUrl := server.URL + basicPipeline.name
	desiredCollection := Collection{
		Name: "java-microprofile",
		Id: "java-microprofile",
		Version: "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}

	client := unitTestClient{map[string][]metav1.OwnerReference{}}

	err := reconcileActiveVersions(&collectionResource, &desiredCollection, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the collection resource was updated with asset information
	if len(collectionResource.Status.ActivePipelines) != 1 {
		t.Fatal(fmt.Sprintf("Collection status should have 1 pipeline, but has %v", len(collectionResource.Status.ActivePipelines)))
	}

	if collectionResource.Status.ActiveVersion != "0.2.5" {
		t.Fatal(fmt.Sprintf("Collection active version should be 0.2.5, but is %v", collectionResource.Status.ActiveVersion))
	}

	// Make sure the assets were created in the collection status
	pipeline := collectionResource.Status.ActivePipelines[0]
	if len(pipeline.ActiveAssets) != 2 {
		t.Fatal(fmt.Sprintf("Pipeline should have 2 assets, but has %v", len(pipeline.ActiveAssets)))
	}

	for _, asset := range pipeline.ActiveAssets {
		if asset.Status != assetStatusActive {
			t.Fatal(fmt.Sprintf("Asset %v should have status active, but is %v", asset.Name, asset.Status))
		}
		if asset.StatusMessage != "" {
			t.Fatal(fmt.Sprintf("Asset %v should have no status message, but has %v", asset.Name, asset.StatusMessage))
		}
	}

	// Make sure the client has the correct objects.
	if len(client.objs) != 2 {
		t.Fatal(fmt.Sprintf("Client map should have 2 entries, but has %v: %v", len(client.objs), client.objs))
	}

	// Make sure the client's objects have an owner set.
	for key, obj := range client.objs {
		if len(obj) != 1 {
			t.Fatal(fmt.Sprintf("Client object %v should have 1 owner, but has %v: %v", key, len(obj), obj))
		}
		if obj[0].UID != collectionResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, collectionResource.UID, obj[0].UID))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that a migration from one version to another works
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsUpgrade(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{Name: "java-microprofile", Version: "0.2.5", DesiredState: "active"},
		Status: kabanerov1alpha1.CollectionStatus{
			ActiveVersion: "0.2.4",
			ActivePipelines: []kabanerov1alpha1.PipelineStatus{{
				Url: "https://somewhere.com/v1/pipeline.tar.gz",
				Digest: "1234567",
				Name: "default",
				ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{{
					Name: "java-microprofile-build-task",
					Status: assetStatusActive,
				}, {
					Name: "java-microprofile-build-pipeline",
					Status: assetStatusActive,
				}, {
					Name: "java-microprofile-old-asset",
					Status: assetStatusActive,
				}},
			}},
		},
	}

	pipelineZipUrl := server.URL + basicPipeline.name
	desiredCollection := Collection{
		Name: "java-microprofile",
		Id: "java-microprofile",
		Version: "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}
		
	// Tell the client what should currently be there.
	client := unitTestClient{map[string][]metav1.OwnerReference{
		"java-microprofile-build-task": []metav1.OwnerReference{{UID: myuid}},
		"java-microprofile-build-pipeline": []metav1.OwnerReference{{UID: myuid}},
		"java-microprofile-old-asset": []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&collectionResource, &desiredCollection, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the collection resource was updated with asset information
	if len(collectionResource.Status.ActivePipelines) != 1 {
		t.Fatal(fmt.Sprintf("Collection status should have 1 pipeline, but has %v", len(collectionResource.Status.ActivePipelines)))
	}

	if collectionResource.Status.ActivePipelines[0].Url != desiredCollection.Pipelines[0].Url {
		t.Fatal(fmt.Sprintf("Collection status should have URL %v, but has %v", desiredCollection.Pipelines[0].Url, collectionResource.Status.ActivePipelines[0].Url))
	}

	if collectionResource.Status.ActivePipelines[0].Digest != desiredCollection.Pipelines[0].Sha256 {
		t.Fatal(fmt.Sprintf("Collection status should have digest %v, but has %v", desiredCollection.Pipelines[0].Sha256, collectionResource.Status.ActivePipelines[0].Digest))
	}
	
	if collectionResource.Status.ActiveVersion != "0.2.5" {
		t.Fatal(fmt.Sprintf("Collection active version should be 0.2.5, but is %v", collectionResource.Status.ActiveVersion))
	}

	// Make sure the actual assets are correct
	pipeline := collectionResource.Status.ActivePipelines[0]
	if len(pipeline.ActiveAssets) != 2 {
		t.Fatal(fmt.Sprintf("Pipeline should have 2 assets, but has %v", len(pipeline.ActiveAssets)))
	}

	for _, asset := range pipeline.ActiveAssets {
		if asset.Status != assetStatusActive {
			t.Fatal(fmt.Sprintf("Asset %v should have status active, but is %v", asset.Name, asset.Status))
		}
		if asset.StatusMessage != "" {
			t.Fatal(fmt.Sprintf("Asset %v should have no status message, but has %v", asset.Name, asset.StatusMessage))
		}
	}

	// Make sure the client has the correct objects.
	if len(client.objs) != 2 {
		t.Fatal(fmt.Sprintf("Client map should have 2 entries, but has %v: %v", len(client.objs), client.objs))
	}

	// Make sure the client's objects have an owner set.
	for key, obj := range client.objs {
		if len(obj) != 1 {
			t.Fatal(fmt.Sprintf("Client object %v should have 1 owner, but has %v: %v", key, len(obj), obj))
		}
		if obj[0].UID != collectionResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, collectionResource.UID, obj[0].UID))
		}
	}
	
}

// --------------------------------------------------------------------------------------------------
// Test that a collection can be deactivated
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsDeactivate(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + basicPipeline.name
	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{Name: "java-microprofile", Version: "0.2.5", DesiredState: "inactive"},
		Status: kabanerov1alpha1.CollectionStatus{
			ActiveVersion: "0.2.5",
			ActivePipelines: []kabanerov1alpha1.PipelineStatus{{
				Url: pipelineZipUrl,
				Digest: basicPipeline.sha256,
				Name: "default",
				ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{{
					Name: "java-microprofile-build-task",
					Status: assetStatusActive,
				}, {
					Name: "java-microprofile-build-pipeline",
					Status: assetStatusActive,
				}},
			}},
		},
	}

	desiredCollection := Collection{
		Name: "java-microprofile",
		Id: "java-microprofile",
		Version: "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}
		
	// Tell the client what should currently be there.
	client := unitTestClient{map[string][]metav1.OwnerReference{
		"java-microprofile-build-task": []metav1.OwnerReference{{UID: myuid}},
		"java-microprofile-build-pipeline": []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&collectionResource, &desiredCollection, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the collection resource was updated with asset information
	if len(collectionResource.Status.ActivePipelines) != 0 {
		t.Fatal(fmt.Sprintf("Collection status should have 0 pipelines, but has %v", len(collectionResource.Status.ActivePipelines)))
	}

	if collectionResource.Status.ActiveVersion != "" {
		t.Fatal(fmt.Sprintf("Collection active version should be empty, but is %v", collectionResource.Status.ActiveVersion))
	}

	if collectionResource.Status.StatusMessage == "" {
		t.Fatal("Collection status message should not be empty for an inactive collection")
	}
	
	// Make sure the client has the correct objects.
	if len(client.objs) != 0 {
		t.Fatal(fmt.Sprintf("Client map should have 0 entries, but has %v: %v", len(client.objs), client.objs))
	}
}

// --------------------------------------------------------------------------------------------------
// Test that an activate for shared assets adds an object owner
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsSharedAsset(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{Name: "java-microprofile", Version: "0.2.5", DesiredState: "active"},
		Status: kabanerov1alpha1.CollectionStatus{},
	}

	pipelineZipUrl := server.URL + basicPipeline.name
	desiredCollection := Collection{
		Name: "java-microprofile",
		Id: "java-microprofile",
		Version: "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}
		
	// Tell the client what should currently be there.
	client := unitTestClient{map[string][]metav1.OwnerReference{
		"java-microprofile-build-task": []metav1.OwnerReference{{UID: otheruid}},
		"java-microprofile-build-pipeline": []metav1.OwnerReference{{UID: otheruid}}}}

	err := reconcileActiveVersions(&collectionResource, &desiredCollection, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the collection resource was updated with asset information
	if len(collectionResource.Status.ActivePipelines) != 1 {
		t.Fatal(fmt.Sprintf("Collection status should have 1 pipeline, but has %v", len(collectionResource.Status.ActivePipelines)))
	}

	if collectionResource.Status.ActivePipelines[0].Url != desiredCollection.Pipelines[0].Url {
		t.Fatal(fmt.Sprintf("Collection status should have URL %v, but has %v", desiredCollection.Pipelines[0].Url, collectionResource.Status.ActivePipelines[0].Url))
	}

	if collectionResource.Status.ActivePipelines[0].Digest != desiredCollection.Pipelines[0].Sha256 {
		t.Fatal(fmt.Sprintf("Collection status should have digest %v, but has %v", desiredCollection.Pipelines[0].Sha256, collectionResource.Status.ActivePipelines[0].Digest))
	}
	
	if collectionResource.Status.ActiveVersion != "0.2.5" {
		t.Fatal(fmt.Sprintf("Collection active version should be 0.2.5, but is %v", collectionResource.Status.ActiveVersion))
	}

	// Make sure the actual assets are correct
	pipeline := collectionResource.Status.ActivePipelines[0]
	if len(pipeline.ActiveAssets) != 2 {
		t.Fatal(fmt.Sprintf("Pipeline should have 2 assets, but has %v", len(pipeline.ActiveAssets)))
	}

	for _, asset := range pipeline.ActiveAssets {
		if asset.Status != assetStatusActive {
			t.Fatal(fmt.Sprintf("Asset %v should have status active, but is %v", asset.Name, asset.Status))
		}
		if asset.StatusMessage != "" {
			t.Fatal(fmt.Sprintf("Asset %v should have no status message, but has %v", asset.Name, asset.StatusMessage))
		}
	}

	// Make sure the client has the correct objects.
	if len(client.objs) != 2 {
		t.Fatal(fmt.Sprintf("Client map should have 2 entries, but has %v: %v", len(client.objs), client.objs))
	}

	// Make sure the client's objects have two owners set.
	for key, obj := range client.objs {
		if len(obj) != 2 {
			t.Fatal(fmt.Sprintf("Client object %v should have 2 owners, but has %v: %v", key, len(obj), obj))
		}
		foundMe, foundOther := false, false
		for _, owner := range obj {
			if owner.UID == myuid {
				foundMe = true
			}
			if owner.UID == otheruid {
				foundOther = true
			}
		}
		if (foundMe == false) || (foundOther == false) {
			t.Fatal(fmt.Sprintf("Did not find correct collection owners in %v: %v", key, obj))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that a deactivate for shared assets removes an object owner
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsSharedAssetDeactivate(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + basicPipeline.name
	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{Name: "java-microprofile", Version: "0.2.5", DesiredState: "inactive"},
		Status: kabanerov1alpha1.CollectionStatus{
			ActiveVersion: "0.2.5",
			ActivePipelines: []kabanerov1alpha1.PipelineStatus{{
				Url: pipelineZipUrl,
				Digest: basicPipeline.sha256,
				Name: "default",
				ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{{
					Name: "java-microprofile-build-task",
					Status: assetStatusActive,
				}, {
					Name: "java-microprofile-build-pipeline",
					Status: assetStatusActive,
				}},
			}},
		},
	}

	desiredCollection := Collection{
		Name: "java-microprofile",
		Id: "java-microprofile",
		Version: "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}
		
	// Tell the client what should currently be there.
	client := unitTestClient{map[string][]metav1.OwnerReference{
		"java-microprofile-build-task": []metav1.OwnerReference{{UID: otheruid},{UID: myuid}},
		"java-microprofile-build-pipeline": []metav1.OwnerReference{{UID: otheruid},{UID: myuid}}}}

	err := reconcileActiveVersions(&collectionResource, &desiredCollection, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the collection resource was updated with asset information
	if len(collectionResource.Status.ActivePipelines) != 0 {
		t.Fatal(fmt.Sprintf("Collection status should have 0 pipelines, but has %v", len(collectionResource.Status.ActivePipelines)))
	}

	if collectionResource.Status.ActiveVersion != "" {
		t.Fatal(fmt.Sprintf("Collection active version should be empty, but is %v", collectionResource.Status.ActiveVersion))
	}

	if collectionResource.Status.StatusMessage == "" {
		t.Fatal("Collection status message should not be empty for an inactive collection")
	}

	// Make sure the client has the correct objects.
	if len(client.objs) != 2 {
		t.Fatal(fmt.Sprintf("Client map should have 2 entries, but has %v: %v", len(client.objs), client.objs))
	}

	// Make sure the client's objects have one owner set (the other owner).
	for key, obj := range client.objs {
		if len(obj) != 1 {
			t.Fatal(fmt.Sprintf("Client object %v should have 1 owner, but has %v: %v", key, len(obj), obj))
		}

		if obj[0].UID != otheruid {
			t.Fatal(fmt.Sprintf("Client object %v should be owned by %v but is owned by %v", key, otheruid, obj[0].UID))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that a reconcile will re-create assets that had been deleted.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsRecreatedDeletedAssets(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + basicPipeline.name
	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{Name: "java-microprofile", Version: "0.2.5", DesiredState: "active"},
		Status: kabanerov1alpha1.CollectionStatus{
			ActiveVersion: "0.2.5",
			ActivePipelines: []kabanerov1alpha1.PipelineStatus{{
				Url: pipelineZipUrl,
				Digest: basicPipeline.sha256,
				Name: "default",
				ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{{
					Name: "java-microprofile-build-task",
					Status: assetStatusActive,
				}, {
					Name: "java-microprofile-build-pipeline",
					Status: assetStatusActive,
				}},
			}},
		},
	}

	desiredCollection := Collection{
		Name: "java-microprofile",
		Id: "java-microprofile",
		Version: "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}
		
	// Tell the client what should currently be there.
	client := unitTestClient{map[string][]metav1.OwnerReference{
		"java-microprofile-build-task": []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&collectionResource, &desiredCollection, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the collection resource was updated with asset information
	if len(collectionResource.Status.ActivePipelines) != 1 {
		t.Fatal(fmt.Sprintf("Collection status should have 1 pipeline, but has %v", len(collectionResource.Status.ActivePipelines)))
	}

	if collectionResource.Status.ActivePipelines[0].Url != desiredCollection.Pipelines[0].Url {
		t.Fatal(fmt.Sprintf("Collection status should have URL %v, but has %v", desiredCollection.Pipelines[0].Url, collectionResource.Status.ActivePipelines[0].Url))
	}

	if collectionResource.Status.ActivePipelines[0].Digest != desiredCollection.Pipelines[0].Sha256 {
		t.Fatal(fmt.Sprintf("Collection status should have digest %v, but has %v", desiredCollection.Pipelines[0].Sha256, collectionResource.Status.ActivePipelines[0].Digest))
	}
	
	if collectionResource.Status.ActiveVersion != "0.2.5" {
		t.Fatal(fmt.Sprintf("Collection active version should be 0.2.5, but is %v", collectionResource.Status.ActiveVersion))
	}

	// Make sure the actual assets are correct
	pipeline := collectionResource.Status.ActivePipelines[0]
	if len(pipeline.ActiveAssets) != 2 {
		t.Fatal(fmt.Sprintf("Pipeline should have 2 assets, but has %v", len(pipeline.ActiveAssets)))
	}

	for _, asset := range pipeline.ActiveAssets {
		if asset.Status != assetStatusActive {
			t.Fatal(fmt.Sprintf("Asset %v should have status active, but is %v", asset.Name, asset.Status))
		}
		if asset.StatusMessage != "" {
			t.Fatal(fmt.Sprintf("Asset %v should have no status message, but has %v", asset.Name, asset.StatusMessage))
		}
	}

	// Make sure the client has the correct objects.
	if len(client.objs) != 2 {
		t.Fatal(fmt.Sprintf("Client map should have 2 entries, but has %v: %v", len(client.objs), client.objs))
	}

	// Make sure the client's objects have an owner set.
	for key, obj := range client.objs {
		if len(obj) != 1 {
			t.Fatal(fmt.Sprintf("Client object %v should have 1 owner, but has %v: %v", key, len(obj), obj))
		}
		if obj[0].UID != collectionResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, collectionResource.UID, obj[0].UID))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that a reconcile will attempt to re-create assets that had been deleted, but since the
// manifests are gone, it can't.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsRecreatedDeletedAssetsNoManifest(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	deletedPipeline := fileInfo{
		name: "/deleted.pipeline.tar.gz",
		sha256: "aaaabbbbccccdddd"}
	
	pipelineZipUrl := server.URL + deletedPipeline.name
	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{Name: "java-microprofile", Version: "0.2.5", DesiredState: "active"},
		Status: kabanerov1alpha1.CollectionStatus{
			ActiveVersion: "0.2.5",
			ActivePipelines: []kabanerov1alpha1.PipelineStatus{{
				Url: pipelineZipUrl,
				Digest: deletedPipeline.sha256,
				Name: "default",
				ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{{
					Name: "java-microprofile-build-task",
					Status: assetStatusActive,
				}, {
					Name: "java-microprofile-build-pipeline",
					Status: assetStatusActive,
				}},
			}},
		},
	}

	desiredCollection := Collection{
		Name: "java-microprofile",
		Id: "java-microprofile",
		Version: "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: deletedPipeline.sha256, Url: pipelineZipUrl}},
	}
		
	// Tell the client what should currently be there.
	client := unitTestClient{map[string][]metav1.OwnerReference{
		"java-microprofile-build-task": []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&collectionResource, &desiredCollection, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the collection resource was updated with asset information
	if len(collectionResource.Status.ActivePipelines) != 1 {
		t.Fatal(fmt.Sprintf("Collection status should have 1 pipeline, but has %v", len(collectionResource.Status.ActivePipelines)))
	}

	if collectionResource.Status.ActivePipelines[0].Url != desiredCollection.Pipelines[0].Url {
		t.Fatal(fmt.Sprintf("Collection status should have URL %v, but has %v", desiredCollection.Pipelines[0].Url, collectionResource.Status.ActivePipelines[0].Url))
	}

	if collectionResource.Status.ActivePipelines[0].Digest != desiredCollection.Pipelines[0].Sha256 {
		t.Fatal(fmt.Sprintf("Collection status should have digest %v, but has %v", desiredCollection.Pipelines[0].Sha256, collectionResource.Status.ActivePipelines[0].Digest))
	}
	
	if collectionResource.Status.ActiveVersion != "0.2.5" {
		t.Fatal(fmt.Sprintf("Collection active version should be 0.2.5, but is %v", collectionResource.Status.ActiveVersion))
	}

	// Make sure the actual assets are correct
	pipeline := collectionResource.Status.ActivePipelines[0]
	if len(pipeline.ActiveAssets) != 2 {
		t.Fatal(fmt.Sprintf("Pipeline should have 2 assets, but has %v", len(pipeline.ActiveAssets)))
	}

	foundPipeline, foundTask := false, false
	for _, asset := range pipeline.ActiveAssets {
		if asset.Name == "java-microprofile-build-task" {
			if asset.Status != assetStatusActive {
				t.Fatal(fmt.Sprintf("Asset %v should have status active, but is %v", asset.Name, asset.Status))
			}
			if asset.StatusMessage != "" {
				t.Fatal(fmt.Sprintf("Asset %v should have no status message, but has %v", asset.Name, asset.StatusMessage))
			}
			foundTask = true
		}
		if asset.Name == "java-microprofile-build-pipeline" {
			if asset.Status != assetStatusFailed {
				t.Fatal(fmt.Sprintf("Asset %v should have status failed, but is %v", asset.Name, asset.Status))
			}
			if asset.StatusMessage == "" {
				t.Fatal(fmt.Sprintf("Asset %v should have a status message, but has none", asset.Name))
			}
			foundPipeline = true
		}
	}

	if foundTask == false || foundPipeline == false {
		t.Fatal(fmt.Sprintf("Did not find expected assets: %v", pipeline.ActiveAssets))
	}
	
	// Make sure the client has the correct objects.
	if len(client.objs) != 1 {
		t.Fatal(fmt.Sprintf("Client map should have 1 entry, but has %v: %v", len(client.objs), client.objs))
	}

	// Make sure the client's objects have an owner set.
	for key, obj := range client.objs {
		if len(obj) != 1 {
			t.Fatal(fmt.Sprintf("Client object %v should have 1 owner, but has %v: %v", key, len(obj), obj))
		}
		if obj[0].UID != collectionResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, collectionResource.UID, obj[0].UID))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that a collection with a bad asset gets an appropriate error message.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsBadAsset(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()
	
	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{Name: "java-microprofile", Version: "0.2.5", DesiredState: "active"},
		Status: kabanerov1alpha1.CollectionStatus{},
	}

	pipelineZipUrl := server.URL + badPipeline.name
	desiredCollection := Collection{
		Name: "java-microprofile",
		Id: "java-microprofile",
		Version: "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: badPipeline.sha256, Url: pipelineZipUrl}},
	}

	client := unitTestClient{map[string][]metav1.OwnerReference{}}

	err := reconcileActiveVersions(&collectionResource, &desiredCollection, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the collection resource was updated with asset information
	if len(collectionResource.Status.ActivePipelines) != 1 {
		t.Fatal(fmt.Sprintf("Collection status should have 1 pipeline, but has %v", len(collectionResource.Status.ActivePipelines)))
	}

	if collectionResource.Status.ActiveVersion != "0.2.5" {
		t.Fatal(fmt.Sprintf("Collection active version should be 0.2.5, but is %v", collectionResource.Status.ActiveVersion))
	}

	// Make sure the assets were created in the collection status
	pipeline := collectionResource.Status.ActivePipelines[0]
	if len(pipeline.ActiveAssets) != 2 {
		t.Fatal(fmt.Sprintf("Pipeline should have 2 assets, but has %v", len(pipeline.ActiveAssets)))
	}

	foundPipeline, foundTask := false, false
	for _, asset := range pipeline.ActiveAssets {
		if asset.Name == "java-microprofile-build-pipeline" {
			if asset.Status != assetStatusActive {
				t.Fatal(fmt.Sprintf("Asset %v should have status active, but is %v", asset.Name, asset.Status))
			}
			if asset.StatusMessage != "" {
				t.Fatal(fmt.Sprintf("Asset %v should have no status message, but has %v", asset.Name, asset.StatusMessage))
			}
			foundTask = true
		}
		if asset.Name == "java-microprofile-build-task" {
			if asset.Status != assetStatusFailed {
				t.Fatal(fmt.Sprintf("Asset %v should have status failed, but is %v", asset.Name, asset.Status))
			}
			if asset.StatusMessage == "" {
				t.Fatal(fmt.Sprintf("Asset %v should have a status message, but has none", asset.Name))
			}
			foundPipeline = true
		}
	}

	if foundTask == false || foundPipeline == false {
		t.Fatal(fmt.Sprintf("Did not find expected assets: %v", pipeline.ActiveAssets))
	}

	// Make sure the client has the correct objects.
	if len(client.objs) != 1 {
		t.Fatal(fmt.Sprintf("Client map should have 1 entry, but has %v: %v", len(client.objs), client.objs))
	}

	// Make sure the client's objects have an owner set.
	for key, obj := range client.objs {
		if len(obj) != 1 {
			t.Fatal(fmt.Sprintf("Client object %v should have 1 owner, but has %v: %v", key, len(obj), obj))
		}
		if obj[0].UID != collectionResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, collectionResource.UID, obj[0].UID))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that a collection can be deactivated when the collection is no longer in the collection hub.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsDeactivateNotInHub(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	deletedPipeline := fileInfo{
		name: "/deleted.pipeline.tar.gz",
		sha256: "aaaabbbbccccdddd"}

	pipelineZipUrl := server.URL + deletedPipeline.name
	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{Name: "java-microprofile", Version: "0.2.5", DesiredState: "inactive"},
		Status: kabanerov1alpha1.CollectionStatus{
			ActiveVersion: "0.2.5",
			ActivePipelines: []kabanerov1alpha1.PipelineStatus{{
				Url: pipelineZipUrl,
				Digest: deletedPipeline.sha256,
				Name: "default",
				ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{{
					Name: "java-microprofile-build-task",
					Status: assetStatusActive,
				}, {
					Name: "java-microprofile-build-pipeline",
					Status: assetStatusActive,
				}},
			}},
		},
	}

	// Tell the client what should currently be there.
	client := unitTestClient{map[string][]metav1.OwnerReference{
		"java-microprofile-build-task": []metav1.OwnerReference{{UID: myuid}},
		"java-microprofile-build-pipeline": []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&collectionResource, nil, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the collection resource was updated with asset information
	if len(collectionResource.Status.ActivePipelines) != 0 {
		t.Fatal(fmt.Sprintf("Collection status should have 0 pipelines, but has %v", len(collectionResource.Status.ActivePipelines)))
	}

	if collectionResource.Status.ActiveVersion != "" {
		t.Fatal(fmt.Sprintf("Collection active version should be empty, but is %v", collectionResource.Status.ActiveVersion))
	}

	if collectionResource.Status.StatusMessage == "" {
		t.Fatal("Collection status message should not be empty for an inactive collection")
	}
	
	// Make sure the client has the correct objects.
	if len(client.objs) != 0 {
		t.Fatal(fmt.Sprintf("Client map should have 0 entries, but has %v: %v", len(client.objs), client.objs))
	}
}
