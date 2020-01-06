package collection

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

  apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

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
func (c unitTestClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error { return nil }
func (c unitTestClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
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
func (c unitTestClient)	Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
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
func (c unitTestClient) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	return errors.New("DeleteAllOf is not supported")
}
func (c unitTestClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
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

func (c unitTestClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return errors.New("Patch is not supported")
}

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

var digest1Pipeline = fileInfo{
	name: "/digest1.pipeline.tar.gz",
	sha256: "0238ff31f191396ca4bf5e0ebeea323d012d5dbc7e3f0997e1bf66b017228aaf"}

var digest2Pipeline = fileInfo{
	name: "/digest2.pipeline.tar.gz",
	sha256: "c3f28ffca707942a8b351000722f1aebda080e3706aa006650a29d10f4aa226b"}

// --------------------------------------------------------------------------------------------------
// Test that initial collection activation works
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInitial(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()
	
	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{{Version: "0.2.5", DesiredState: "active"}}},
		Status: kabanerov1alpha1.CollectionStatus{},
	}

	defaultImage := Images{Id: "default", Image: "kabanero/kabanero-image:latest"}
	
	pipelineZipUrl := server.URL + basicPipeline.name
	desiredCollection := Collection{
		Name: "java-microprofile",
		Id: "java-microprofile",
		Version: "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
		Images: []Images{defaultImage},
	}

	client := unitTestClient{map[string][]metav1.OwnerReference{}}

	err := reconcileActiveVersions(&collectionResource, []resolvedCollection{{"", desiredCollection}}, client)

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

	if pipeline.Name != desiredCollection.Pipelines[0].Id {
		t.Fatal(fmt.Sprintf("Pipeline name should be %v, but is %v", desiredCollection.Pipelines[0].Id, pipeline.Name))
	}

	// Make sure the status versions array was created in the collection status
	if len(collectionResource.Status.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Versions array should have 1 entry, but has %v: %v", len(collectionResource.Status.Versions), collectionResource.Status.Versions))
	}
	
	if len(collectionResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Collection versions status should have 1 pipeline, but has %v", len(collectionResource.Status.Versions[0].Pipelines)))
	}

	if collectionResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Collection versions active version should be 0.2.5, but is %v", collectionResource.Status.Versions[0].Version))
	}

	pipeline = collectionResource.Status.Versions[0].Pipelines[0]
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

	if pipeline.Name != desiredCollection.Pipelines[0].Id {
		t.Fatal(fmt.Sprintf("Pipeline name should be %v, but is %v", desiredCollection.Pipelines[0].Id, pipeline.Name))
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

	// Make sure the status lists the images
	if len(collectionResource.Status.Images) != 1 {
		t.Fatal(fmt.Sprintf("Status should contain one image, but contains %v: %#v", len(collectionResource.Status.Images), collectionResource.Status))
	}

	if collectionResource.Status.Images[0].Image != defaultImage.Image {
		t.Fatal(fmt.Sprintf("Image should be %v, but is %v", defaultImage.Image, collectionResource.Status.Images[0].Image))
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
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{{Version: "0.2.5", DesiredState: "active"}}},
		Status: kabanerov1alpha1.CollectionStatus{
			Versions: []kabanerov1alpha1.CollectionVersionStatus{{
				Version: "0.2.4",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
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

	err := reconcileActiveVersions(&collectionResource, []resolvedCollection{{"", desiredCollection}}, client)

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

	// Make sure the collection versions status array was updated with asset information
	if len(collectionResource.Status.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Collection version status should have 1 version, but has %v: %v", len(collectionResource.Status.Versions), collectionResource.Status.Versions))
	}
	
	if len(collectionResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Collection version status should have 1 pipeline, but has %v", len(collectionResource.Status.Versions[0].Pipelines)))
	}

	if collectionResource.Status.Versions[0].Pipelines[0].Url != desiredCollection.Pipelines[0].Url {
		t.Fatal(fmt.Sprintf("Collection version status should have URL %v, but has %v", desiredCollection.Pipelines[0].Url, collectionResource.Status.Versions[0].Pipelines[0].Url))
	}

	if collectionResource.Status.Versions[0].Pipelines[0].Digest != desiredCollection.Pipelines[0].Sha256 {
		t.Fatal(fmt.Sprintf("Collection version status should have digest %v, but has %v", desiredCollection.Pipelines[0].Sha256, collectionResource.Status.Versions[0].Pipelines[0].Digest))
	}
	
	if collectionResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Collection version status version should be 0.2.5, but is %v", collectionResource.Status.Versions[0].Version))
	}

	pipeline = collectionResource.Status.Versions[0].Pipelines[0]
	if len(pipeline.ActiveAssets) != 2 {
		t.Fatal(fmt.Sprintf("Pipeline in version status should have 2 assets, but has %v", len(pipeline.ActiveAssets)))
	}

	for _, asset := range pipeline.ActiveAssets {
		if asset.Status != assetStatusActive {
			t.Fatal(fmt.Sprintf("Asset %v in version status should have status active, but is %v", asset.Name, asset.Status))
		}
		if asset.StatusMessage != "" {
			t.Fatal(fmt.Sprintf("Asset %v in version status should have no status message, but has %v", asset.Name, asset.StatusMessage))
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
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{{Version: "0.2.5", DesiredState: "inactive"}}},
		Status: kabanerov1alpha1.CollectionStatus{
			Versions: []kabanerov1alpha1.CollectionVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
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

	err := reconcileActiveVersions(&collectionResource, []resolvedCollection{{"", desiredCollection}}, client)
	
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

	// Make sure the collection version resource was updated with asset information
	if len(collectionResource.Status.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Collection version status should have 1 entry, but has %v: %v", len(collectionResource.Status.Versions), collectionResource.Status.Versions))
	}

	if collectionResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Collection version status should have version \"0.2.5\", but has %v", collectionResource.Status.Versions[0].Version))
	}

	if collectionResource.Status.Versions[0].StatusMessage == "" {
		t.Fatal("Collection version status message should not be empty for an inactive collection")
	}

	if collectionResource.Status.Versions[0].Status != kabanerov1alpha1.CollectionDesiredStateInactive {
		t.Fatal(fmt.Sprintf("Collection version status should be inactive, but is %v", collectionResource.Status.Versions[0].Status))
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
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{{Version: "0.2.5", DesiredState: "active"}}},
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

	err := reconcileActiveVersions(&collectionResource, []resolvedCollection{{"", desiredCollection}}, client)

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
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{{Version: "0.2.5", DesiredState: "inactive"}}},
		Status: kabanerov1alpha1.CollectionStatus{
			Versions: []kabanerov1alpha1.CollectionVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
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

	err := reconcileActiveVersions(&collectionResource, []resolvedCollection{{"", desiredCollection}}, client)

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
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{{Version: "0.2.5", DesiredState: "active"}}},
		Status: kabanerov1alpha1.CollectionStatus{
			Versions: []kabanerov1alpha1.CollectionVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
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

	err := reconcileActiveVersions(&collectionResource, []resolvedCollection{{"", desiredCollection}}, client)

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
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{{Version: "0.2.5", DesiredState: "active"}}},
		Status: kabanerov1alpha1.CollectionStatus{
			Versions: []kabanerov1alpha1.CollectionVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
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

	err := reconcileActiveVersions(&collectionResource, []resolvedCollection{{"", desiredCollection}}, client)

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
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{{Version: "0.2.5", DesiredState: "active"}}},
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

	err := reconcileActiveVersions(&collectionResource, []resolvedCollection{{"", desiredCollection}}, client)

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
// Test that we can't activate a collection that's not in the hub.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsActivateNotInHub(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{{Version: "0.2.5", DesiredState: "active"}}},
		Status: kabanerov1alpha1.CollectionStatus{
			Versions: []kabanerov1alpha1.CollectionVersionStatus{{
				Version: "0.2.4",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
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
			}},
		},
	}

	// Note the "desired" collection version doesn't match the one that we want to activate.
	pipelineZipUrl := server.URL + basicPipeline.name
	desiredCollection := Collection{
		Name: "java-microprofile",
		Id: "java-microprofile",
		Version: "0.2.6",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}
		
	// Tell the client what should currently be there.
	client := unitTestClient{map[string][]metav1.OwnerReference{
		"java-microprofile-build-task": []metav1.OwnerReference{{UID: myuid}},
		"java-microprofile-build-pipeline": []metav1.OwnerReference{{UID: myuid}},
		"java-microprofile-old-asset": []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&collectionResource, []resolvedCollection{{"", desiredCollection}}, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the collection resource was updated with asset information
	if len(collectionResource.Status.ActivePipelines) != 0 {
		t.Fatal(fmt.Sprintf("Collection status should have 0 pipelines, but has %v", len(collectionResource.Status.ActivePipelines)))
	}

	if collectionResource.Status.ActiveVersion != "" {
		t.Fatal(fmt.Sprintf("There should be no active version of the collection, but we have %v", collectionResource.Status.ActiveVersion))
	}

	if collectionResource.Status.StatusMessage == "" {
		t.Fatal(fmt.Sprintf("There should be an error status set, but there is none: %#v", collectionResource.Status))
	}

	if strings.Contains(collectionResource.Status.StatusMessage, "is not available") == false {
		t.Fatal(fmt.Sprintf("The status message should say not available, but it says %v", collectionResource.Status.StatusMessage))
	}

	if collectionResource.Status.Status != kabanerov1alpha1.CollectionDesiredStateInactive {
		t.Fatal(fmt.Sprintf("The status should be inactive, but is %v", collectionResource.Status.Status))
	}

	// Make sure the collection version array status was updated with asset information
	if len(collectionResource.Status.Versions) != 1 {
		t.Fatal(fmt.Sprintf("There should be 1 version in the versions array, but there are %v: %v", len(collectionResource.Status.Versions), collectionResource.Status.Versions))
	}

	if collectionResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Version \"0.2.5\" should be present in the versions array, but it is %v", collectionResource.Status.Versions[0].Version))
	}

	if strings.Contains(collectionResource.Status.StatusMessage, "is not available") == false {
		t.Fatal(fmt.Sprintf("The status message in the versions array should say not available, but it says %v", collectionResource.Status.Versions[0].StatusMessage))
	}

	if collectionResource.Status.Versions[0].Status != kabanerov1alpha1.CollectionDesiredStateInactive {
		t.Fatal(fmt.Sprintf("The status should be inactive, but is %v", collectionResource.Status.Versions[0].Status))
	}
	
	// Make sure the client has the correct objects.
	if len(client.objs) != 0 {
		t.Fatal(fmt.Sprintf("Client map should have 0 entries, but has %v: %v", len(client.objs), client.objs))
	}

	// Reconcile it again and make sure we retain the information.
	err = reconcileActiveVersions(&collectionResource, []resolvedCollection{{"", desiredCollection}}, client)

	if len(collectionResource.Status.ActivePipelines) != 0 {
		t.Fatal(fmt.Sprintf("Collection status should have 0 pipelines, but has %v", len(collectionResource.Status.ActivePipelines)))
	}

	if collectionResource.Status.ActiveVersion != "" {
		t.Fatal(fmt.Sprintf("There should be no active version of the collection, but we have %v", collectionResource.Status.ActiveVersion))
	}

	if collectionResource.Status.StatusMessage == "" {
		t.Fatal(fmt.Sprintf("There should be an error status set, but there is none: %#v", collectionResource.Status))
	}

	if strings.Contains(collectionResource.Status.StatusMessage, "is not available") == false {
		t.Fatal(fmt.Sprintf("The status message should say not available, but it says %v", collectionResource.Status.StatusMessage))
	}

	if collectionResource.Status.Status != kabanerov1alpha1.CollectionDesiredStateInactive {
		t.Fatal(fmt.Sprintf("The status should be inactive, but is %v", collectionResource.Status.Status))
	}
	
	// Make sure the client has the correct objects.
	if len(client.objs) != 0 {
		t.Fatal(fmt.Sprintf("Client map should have 0 entries, but has %v: %v", len(client.objs), client.objs))
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
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{{Version: "0.2.5", DesiredState: "inactive"}}},
		Status: kabanerov1alpha1.CollectionStatus{
			Versions: []kabanerov1alpha1.CollectionVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
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

// ==================================================================================================
// --------------------------------------------------------------------------------------------------
// The following tests activate multiple versions of a collection.
// --------------------------------------------------------------------------------------------------
// ==================================================================================================

// --------------------------------------------------------------------------------------------------
// Test that two versions of the same collection can be activated.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInternalTwoInitial(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{
				{ Version: "0.2.5", DesiredState: "active" },
				{ Version: "0.2.6", DesiredState: "active" }}},
		Status: kabanerov1alpha1.CollectionStatus{},
	}
	
	pipelineZipUrl := server.URL + basicPipeline.name
	collections := []resolvedCollection{{
		repositoryURL: "",
		collection: Collection{
			Name: "java-microprofile",
			Id: "java-microprofile",
			Version: "0.2.5",
			Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}}},
	}, {
		repositoryURL: "",
		collection: Collection{
			Name: "java-microprofile",
			Id: "java-microprofile",
			Version: "0.2.6",
			Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}}},
	}}

	client := unitTestClient{map[string][]metav1.OwnerReference{}}

	err := reconcileActiveVersions(&collectionResource, collections, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure we got two status structs back
	if len(collectionResource.Status.Versions) != 2 {
		t.Fatal(fmt.Sprintf("Expected two statuses, but got %v: %#v", len(collectionResource.Status.Versions), collectionResource.Status))
	}
	
	// Make sure the collection resource was updated with asset information
	versionsFound := make(map[string]bool)
	for _, curStatus := range collectionResource.Status.Versions {
		versionsFound[curStatus.Version] = true
		
		if len(curStatus.Pipelines) != 1 {
			t.Fatal(fmt.Sprintf("Collection status should have 1 pipeline, but has %v: %v", len(curStatus.Pipelines), curStatus))
		}

		// Make sure the assets were created in the collection status
		pipeline := curStatus.Pipelines[0]
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
	}

	if versionsFound["0.2.5"] == false {
		t.Fatal(fmt.Sprintf("Did not find version 0.2.5 in the status: %v", collectionResource.Status))
	}

	if versionsFound["0.2.6"] == false {
		t.Fatal(fmt.Sprintf("Did not find version 0.2.6 in the status: %v", collectionResource.Status))
	}

	// Make sure that the singleton status matches the first element in the versions status
	if collectionResource.Status.Versions[0].Version != collectionResource.Status.ActiveVersion {
		t.Fatal(fmt.Sprintf("Collection status activeVersion %v does not match collection status version[0] %v", collectionResource.Status.ActiveVersion, collectionResource.Status.Versions[0].Version))
	}

	if collectionResource.Status.Versions[0].Location != collectionResource.Status.ActiveLocation {
		t.Fatal(fmt.Sprintf("Collection status activeLocation %v does not match collection status version [0] location %v", collectionResource.Status.ActiveLocation, collectionResource.Status.Versions[0].Location))
	}

	if collectionResource.Status.Versions[0].Status != collectionResource.Status.Status {
		t.Fatal(fmt.Sprintf("Collection status status %v does not match collection status version[0] status %v", collectionResource.Status.Status, collectionResource.Status.Versions[0].Status))
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
// Test that two versions of the same collection using different pipelines can be activated.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInternalTwoInitialDiffPipelines(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{
				{ Version: "0.2.5", DesiredState: "active" },
				{ Version: "0.2.6", DesiredState: "active" }}},
		Status: kabanerov1alpha1.CollectionStatus{},
	}

	pipeline1ZipUrl := server.URL + digest1Pipeline.name
	pipeline2ZipUrl := server.URL + digest2Pipeline.name
	collections := []resolvedCollection{{
		repositoryURL: "",
		collection: Collection{
			Name: "java-microprofile",
			Id: "java-microprofile",
			Version: "0.2.5",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest1Pipeline.sha256, Url: pipeline1ZipUrl}}},
	}, {
		repositoryURL: "",
		collection: Collection{
			Name: "java-microprofile",
			Id: "java-microprofile",
			Version: "0.2.6",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest2Pipeline.sha256, Url: pipeline2ZipUrl}}},
	}}
	
	client := unitTestClient{map[string][]metav1.OwnerReference{}}

	err := reconcileActiveVersions(&collectionResource, collections, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure we got two status structs back
	if len(collectionResource.Status.Versions) != 2 {
		t.Fatal(fmt.Sprintf("Expected two statuses, but got %v: %#v", len(collectionResource.Status.Versions), collectionResource.Status))
	}
	
	// Make sure the collection resource was updated with asset information
	versionsFound := make(map[string]bool)
	for _, curStatus := range collectionResource.Status.Versions {
		versionsFound[curStatus.Version] = true
		
		if len(curStatus.Pipelines) != 1 {
			t.Fatal(fmt.Sprintf("Collection status should have 1 pipeline, but has %v: %v", len(curStatus.Pipelines), curStatus))
		}

		// Make sure the assets were created in the collection status
		pipeline := curStatus.Pipelines[0]
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
	}

	if versionsFound["0.2.5"] == false {
		t.Fatal(fmt.Sprintf("Did not find version 0.2.5 in the status: %v", collectionResource.Status))
	}

	if versionsFound["0.2.6"] == false {
		t.Fatal(fmt.Sprintf("Did not find version 0.2.6 in the status: %v", collectionResource.Status))
	}

	// Make sure the client has the correct objects.
	if len(client.objs) != 4 {
		t.Fatal(fmt.Sprintf("Client map should have 4 entries, but has %v: %v", len(client.objs), client.objs))
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
// Test that two versions of the same collection using different pipelines can be activated.  One of
// the versions is currently active but no longer available in the collection hub.  The operator
// should continue to use the last-available version of that collection, along with an error message
// saying that it's not there anymore.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInternalTwoInitialDiffPipelinesOneDeletedFromHub(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	badRepositoryUrl := "https://bogus.com/kabanero_index.yaml"
	pipeline1ZipUrl := server.URL + digest1Pipeline.name
	pipeline2ZipUrl := server.URL + digest2Pipeline.name
	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{
				{ Version: "0.2.5", DesiredState: "active", RepositoryUrl: badRepositoryUrl },
				{ Version: "0.2.6", DesiredState: "active" }}},
		Status: kabanerov1alpha1.CollectionStatus{
			Versions: []kabanerov1alpha1.CollectionVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
					Url: pipeline1ZipUrl,
					Digest: digest1Pipeline.sha256,
					Name: "default",
					ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{{
						Name: "build-task-0238ff31",
						Status: assetStatusActive,
					}, {
						Name: "build-pipeline-0238ff31",
						Status: assetStatusActive,
					}},
				}},
			}},
		},
	}

	// Only one of the two collection versions will be found in the collection hub.  Only put the 0.2.6 version here.
	collections := []resolvedCollection{{
		repositoryURL: "",
		collection: Collection{
			Name: "java-microprofile",
			Id: "java-microprofile",
			Version: "0.2.6",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest2Pipeline.sha256, Url: pipeline2ZipUrl}},
		}}}

	client := unitTestClient{map[string][]metav1.OwnerReference{
		"build-task-0238ff31": []metav1.OwnerReference{{UID: myuid}},
		"build-pipeline-0238ff31": []metav1.OwnerReference{{UID: myuid}}}}
	
	err := reconcileActiveVersions(&collectionResource, collections, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure we got two status structs back
	if len(collectionResource.Status.Versions) != 2 {
		t.Fatal(fmt.Sprintf("Expected two statuses, but got %v: %#v", len(collectionResource.Status.Versions), collectionResource.Status))
	}
	
	// Make sure the collection resource was updated with asset information
	versionsFound := make(map[string]bool)
	for _, curStatus := range collectionResource.Status.Versions {
		versionsFound[curStatus.Version] = true
		
		if len(curStatus.Pipelines) != 1 {
			t.Fatal(fmt.Sprintf("Collection status should have 1 pipeline, but has %v: %v", len(curStatus.Pipelines), curStatus))
		}

		// Make sure the assets were created in the collection status
		pipeline := curStatus.Pipelines[0]
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

		// Version 0.2.5 was deleted from the collection hub.  Make sure there is an error set.
		if (curStatus.Version == "0.2.5") && (strings.Contains(curStatus.StatusMessage, badRepositoryUrl) == false) {
			t.Fatal(fmt.Sprintf("Collection version 0.2.5 should have an error message due to collection not being in hub, but has: %v", curStatus.StatusMessage))
		}

		// Make sure both statuses contain a pipeline name
		if curStatus.Pipelines[0].Name != "default" {
			t.Fatal(fmt.Sprintf("Collection version %v should contain a pipeline named \"default\", but is %v", curStatus.Version, curStatus.Pipelines[0].Name))
		}
	}

	if versionsFound["0.2.5"] == false {
		t.Fatal(fmt.Sprintf("Did not find version 0.2.5 in the status: %v", collectionResource.Status.Versions))
	}

	if versionsFound["0.2.6"] == false {
		t.Fatal(fmt.Sprintf("Did not find version 0.2.6 in the status: %v", collectionResource.Status.Versions))
	}

	// Make sure the client has the correct objects.
	if len(client.objs) != 4 {
		t.Fatal(fmt.Sprintf("Client map should have 4 entries, but has %v: %v", len(client.objs), client.objs))
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
// Test that one version of a collection can be deleted but the other remains active.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInternalTwoDeactivateOne(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	pipeline1ZipUrl := server.URL + digest1Pipeline.name
	pipeline2ZipUrl := server.URL + digest2Pipeline.name

	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{{ Version: "0.2.6", DesiredState: "active" }}},
		Status: kabanerov1alpha1.CollectionStatus{
			Versions: []kabanerov1alpha1.CollectionVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
					Url: pipeline1ZipUrl,
					Digest: digest1Pipeline.sha256,
					Name: "default",
					ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{{
						Name: "build-task-0238ff31",
						Status: assetStatusActive,
					}, {
						Name: "build-pipeline-0238ff31",
						Status: assetStatusActive,
					}},
				}},
			}, {
				Version: "0.2.6",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
					Url: pipeline2ZipUrl,
					Digest: digest2Pipeline.sha256,
					Name: "default",
					ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{{
						Name: "build-task-c3f28ffc",
						Status: assetStatusActive,
					}, {
						Name: "build-pipeline-c3f28ffc",
						Status: assetStatusActive,
					}},
				}},
			}},
		},
	}

	collections := []resolvedCollection{{
		repositoryURL: "",
		collection: Collection{
			Name: "java-microprofile",
			Id: "java-microprofile",
			Version: "0.2.5",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest1Pipeline.sha256, Url: pipeline1ZipUrl}}},
	}, {
		repositoryURL: "",
		collection: Collection{
			Name: "java-microprofile",
			Id: "java-microprofile",
			Version: "0.2.6",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest2Pipeline.sha256, Url: pipeline2ZipUrl}},
		}},
	}

	client := unitTestClient{map[string][]metav1.OwnerReference{
		"build-task-0238ff31": []metav1.OwnerReference{{UID: myuid}},
		"build-pipeline-0238ff31": []metav1.OwnerReference{{UID: myuid}},
		"build-task-c3f28ffc": []metav1.OwnerReference{{UID: myuid}},
		"build-pipeline-c3f28ffc": []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&collectionResource, collections, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure we got one status structs back
	if len(collectionResource.Status.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Expected one status, but got %v: %#v", len(collectionResource.Status.Versions), collectionResource.Status.Versions))
	}
	
	// Make sure the collection resource was updated with asset information
	for _, curStatus := range collectionResource.Status.Versions {
		if curStatus.Version != "0.2.6" {
			t.Fatal(fmt.Sprintf("Expected collection version 0.2.6, but found %v: %#v", curStatus.Version, curStatus))
		}
		
		if len(curStatus.Pipelines) != 1 {
			t.Fatal(fmt.Sprintf("Collection status should have 1 pipeline, but has %v: %v", len(curStatus.Pipelines), curStatus))
		}

		// Make sure the assets were created in the collection status
		pipeline := curStatus.Pipelines[0]
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
// Test that one version of a collection can be inactive but the other remains active.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInternalTwoDeleteOne(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionHandler{})
	defer server.Close()

	pipeline1ZipUrl := server.URL + digest1Pipeline.name
	pipeline2ZipUrl := server.URL + digest2Pipeline.name
	collectionResource := kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{UID: myuid},
		Spec: kabanerov1alpha1.CollectionSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha1.CollectionVersion{
				{ Version: "0.2.5", DesiredState: "inactive"},
				{ Version: "0.2.6", DesiredState: "active" }}},
		Status: kabanerov1alpha1.CollectionStatus{
			Versions: []kabanerov1alpha1.CollectionVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
					Url: pipeline1ZipUrl,
					Digest: digest1Pipeline.sha256,
					Name: "default",
					ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{{
						Name: "build-task-0238ff31",
						Status: assetStatusActive,
					}, {
						Name: "build-pipeline-0238ff31",
						Status: assetStatusActive,
					}},
				}},
			}, {
				Version: "0.2.6",
				Pipelines: []kabanerov1alpha1.PipelineStatus{{
					Url: pipeline2ZipUrl,
					Digest: digest2Pipeline.sha256,
					Name: "default",
					ActiveAssets: []kabanerov1alpha1.RepositoryAssetStatus{{
						Name: "build-task-c3f28ffc",
						Status: assetStatusActive,
					}, {
						Name: "build-pipeline-c3f28ffc",
						Status: assetStatusActive,
					}},
				}},
			}},
		},
	}

	collections := []resolvedCollection{{
		repositoryURL: "",
		collection: Collection{
			Name: "java-microprofile",
			Id: "java-microprofile",
			Version: "0.2.5",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest1Pipeline.sha256, Url: pipeline1ZipUrl}},
		},
	}, {
		repositoryURL: "",
		collection: Collection{
			Name: "java-microprofile",
			Id: "java-microprofile",
			Version: "0.2.6",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest2Pipeline.sha256, Url: pipeline2ZipUrl}},
		},
	}}

	client := unitTestClient{map[string][]metav1.OwnerReference{
		"build-task-0238ff31": []metav1.OwnerReference{{UID: myuid}},
		"build-pipeline-0238ff31": []metav1.OwnerReference{{UID: myuid}},
		"build-task-c3f28ffc": []metav1.OwnerReference{{UID: myuid}},
		"build-pipeline-c3f28ffc": []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&collectionResource, collections, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure we got one status structs back
	if len(collectionResource.Status.Versions) != 2 {
		t.Fatal(fmt.Sprintf("Expected two statuses, but got %v: %#v", len(collectionResource.Status.Versions), collectionResource.Status.Versions))
	}
	
	// Make sure the collection resource was updated with asset information
	versionsFound := make(map[string]bool)
	for _, curStatus := range collectionResource.Status.Versions {
		versionsFound[curStatus.Version] = true

		if curStatus.Version == "0.2.5" {
			if len(curStatus.Pipelines) != 0 {
				t.Fatal(fmt.Sprintf("Collection version 0.2.5 should not have any active pipelines: %#v", curStatus.Pipelines))
			}

			if curStatus.StatusMessage == "" {
				t.Fatal(fmt.Sprintf("Collection version 0.2.5 should have a status message, but has none."))
			}

			if curStatus.Status != kabanerov1alpha1.CollectionDesiredStateInactive {
				t.Fatal(fmt.Sprintf("Collection version 0.2.5 should be marked inactive, but is %v", curStatus.Status))
			}
		} else if curStatus.Version == "0.2.6" {
			if len(curStatus.Pipelines) != 1 {
				t.Fatal(fmt.Sprintf("Collection status should have 1 pipeline, but has %v: %v", len(curStatus.Pipelines), curStatus))
			}

			// Make sure the assets were created in the collection status
			pipeline := curStatus.Pipelines[0]
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
		} else {
			t.Fatal(fmt.Sprintf("Found an invalid version: %v", curStatus.Version))
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

// TODO: More "multiple collection" tests...

