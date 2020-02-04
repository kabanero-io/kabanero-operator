package stack

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
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
type testLogger struct{}

func (t testLogger) Info(msg string, keysAndValues ...interface{}) { fmt.Printf("Info: %v \n", msg) }
func (t testLogger) Enabled() bool                                 { return true }
func (t testLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	fmt.Printf("Error: %v: %v\n", msg, err.Error())
}
func (t testLogger) V(level int) logr.InfoLogger                         { return t }
func (t testLogger) WithValues(keysAndValues ...interface{}) logr.Logger { return t }
func (t testLogger) WithName(name string) logr.Logger                    { return t }

func init() {
	logf.SetLogger(testLogger{})
}

func TestReconcileStack(t *testing.T) {
	r := &ReconcileStack{indexResolver: func(kabanerov1alpha2.RepositoryConfig, []Pipelines, []Trigger, string) (*Index, error) {
		return &Index{
			URL:        "http://some/URL/to/V2/stack/index",
			APIVersion: "v2",
			Stacks: []Stack{
				Stack{
					DefaultImage:    "java-microprofile",
					DefaultPipeline: "default",
					DefaultTemplate: "default",
					Description:     "Test stack",
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

	c := &kabanerov1alpha2.Stack{
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
		Spec: kabanerov1alpha2.StackSpec{
			Name:         "somename",
			Versions:     []kabanerov1alpha2.StackVersion{
				{
					DesiredState:   "active",
				},
			},
		},
	}

	r.ReconcileStack(c)
}

// Test that failed assets are detected in the Stack instance status
func TestFailedAssets(t *testing.T) {
	var sampleAsset = []kabanerov1alpha2.RepositoryAssetStatus{{Name: "myAsset", Digest: "678910", Status: "active"},
		{Name: "myAsset2", Digest: "678911", Status: "failed", StatusMessage: "some failure"},
	}

	var samplePipelineStatus = []kabanerov1alpha2.PipelineStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", ActiveAssets: sampleAsset}}
	var sampleStackVersionStatus = []kabanerov1alpha2.StackVersionStatus{{Version: "", Location: "", Pipelines: samplePipelineStatus, Status: "", StatusMessage: ""}}
	status := kabanerov1alpha2.StackStatus{Versions: sampleStackVersionStatus}
	
	if failedAssets(status) == false {
		t.Fatal("Should be one failed asset in the status")
	}
}

// Test that no failed assets are detected in the Stack instance status
func TestNoFailedAssets(t *testing.T) {
	var sampleAsset = []kabanerov1alpha2.RepositoryAssetStatus{{Name: "myAsset", Digest: "678910", Status: "active"},
		{Name: "myAsset2", Digest: "678911", Status: "active"},
	}

	var samplePipelineStatus = []kabanerov1alpha2.PipelineStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", ActiveAssets: sampleAsset}}
	var sampleStackVersionStatus = []kabanerov1alpha2.StackVersionStatus{{Version: "", Location: "", Pipelines: samplePipelineStatus, Status: "", StatusMessage: ""}}
	status := kabanerov1alpha2.StackStatus{Versions: sampleStackVersionStatus}

	if failedAssets(status) {
		t.Fatal("Should be no failed asset in the status")
	}
}

// Test that an empty status yields no failed assets
func TestNoFailedAssetsEmptyStatus(t *testing.T) {
	var samplePipelineStatus = []kabanerov1alpha2.PipelineStatus{{Name: "myAsset", Url: "http://myurl.com", Digest: "1234", ActiveAssets: []kabanerov1alpha2.RepositoryAssetStatus{}}}
	var sampleStackVersionStatus = []kabanerov1alpha2.StackVersionStatus{{Version: "", Location: "", Pipelines: samplePipelineStatus, Status: "", StatusMessage: ""}}
	status := kabanerov1alpha2.StackStatus{Versions: sampleStackVersionStatus}

	if failedAssets(status) {
		t.Fatal("Should be no failed asset in the status")
	}
}


// -------------------------------------------------------------------------------
// Asset reuse tests
// -------------------------------------------------------------------------------

type unitTestClient struct {
	// Objects that the client knows about.  This is real simple.... for now.  We just
	// keep the name, and any owner references.
	objs map[client.ObjectKey][]metav1.OwnerReference
}

func (c unitTestClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	fmt.Printf("Received Get() for %v\n", key.Name)
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid target object for get: %v\n", obj)
		return errors.New("Get only supports setting into Unstructured")
	}
	owners := c.objs[key]
	if len(owners) == 0 {
		return apierrors.NewNotFound(schema.GroupResource{}, key.Name)
	}
	u.SetName(key.Name)
	u.SetNamespace(key.Namespace)
	u.SetOwnerReferences(owners)
	return nil
}
func (c unitTestClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	return nil
}
func (c unitTestClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid create: %v\n", obj)
		return errors.New("Create only supports Unstructured")
	}

	fmt.Printf("Received Create() for %v\n", u.GetName())
	key := client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}
	owners := c.objs[key]
	if len(owners) > 0 {
		fmt.Printf("Receive create object already exists: %v/%v\n", u.GetNamespace(), u.GetName())
		return apierrors.NewAlreadyExists(schema.GroupResource{}, u.GetName())
	}

	gvk := u.GroupVersionKind()
	if gvk.Kind == "BadTask" {
		message := fmt.Sprintf("Receive create for invalid kind: %v", gvk.Kind)
		fmt.Printf(message + "\n")
		return errors.New(message)
	}

	c.objs[key] = u.GetOwnerReferences()
	return nil
}
func (c unitTestClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid delete: %v\n", obj)
		return errors.New("Delete only supports Unstructured")
	}

	fmt.Printf("Received Delete() for %v\n", u.GetName())
	key := client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}
	owners := c.objs[key]
	if len(owners) == 0 {
		fmt.Printf("Received delete for an object that does not exist: %v\n", obj)
		return apierrors.NewNotFound(schema.GroupResource{}, u.GetName())
	}
	delete(c.objs, key)
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
	key := client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}
	owners := c.objs[key]
	if len(owners) == 0 {
		fmt.Printf("Received update for object that does not exist: %v\n", obj)
		return apierrors.NewNotFound(schema.GroupResource{}, u.GetName())
	}
	c.objs[key] = u.GetOwnerReferences()
	return nil
}
func (c unitTestClient) Status() client.StatusWriter { return c }

func (c unitTestClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return errors.New("Patch is not supported")
}

// HTTP handler that serves pipeline zips
type stackHandler struct {
}

func (ch stackHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
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
	name   string
	sha256 string
}

const (
	myuid    = "MYUID"
	otheruid = "OTHERUID"
)

var basicPipeline = fileInfo{
	name:   "/basic.pipeline.tar.gz",
	sha256: "8080076acd8f54ecbb7de132df148d964e5e93921cce983a0f781418b0871573"}

var badPipeline = fileInfo{
	name:   "/bad.pipeline.tar.gz",
	sha256: "eca24c909ee2b463abcae7c3b8d1be406297e0e1958e43dff1185dc765af985b"}

var digest1Pipeline = fileInfo{
	name:   "/digest1.pipeline.tar.gz",
	sha256: "0238ff31f191396ca4bf5e0ebeea323d012d5dbc7e3f0997e1bf66b017228aaf"}

var digest2Pipeline = fileInfo{
	name:   "/digest2.pipeline.tar.gz",
	sha256: "c3f28ffca707942a8b351000722f1aebda080e3706aa006650a29d10f4aa226b"}

var triggerPipeline = fileInfo{
	name:   "/trigger.pipeline.tar.gz",
	sha256: "2e8ff2e5c6ce8526edc9ce413876c450383814d4fa6f5f37b690d167433da363"}

// --------------------------------------------------------------------------------------------------
// Test stack/stack id validation.
// --------------------------------------------------------------------------------------------------
func TestStackIDValidation(t *testing.T) {
	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{{Version: "0.2.5", DesiredState: "active"}}},
		Status: kabanerov1alpha2.StackStatus{},
	}

	// Test invalid Id ending in "-"
	invalidID := "java-microprofile-"
	stackResource.Spec.Name = invalidID
	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{}}
	err := reconcileActiveVersions(&stackResource, client)

	if err == nil {
		t.Fatal(fmt.Sprintf("An error was expected because stack id %v is invalid. No error was issued.", invalidID))
	} else {
		if !strings.Contains(err.Error(), invalidID) {
			t.Fatal(fmt.Sprintf("The error message should have contained the name of the invalid stack ID: %v. Error: %v", invalidID, err))
		}
	}

	// Test invalid id containing an upper case char.
	invalidID = "java-Microprofile"
	stackResource.Spec.Name = invalidID
	err = reconcileActiveVersions(&stackResource, client)

	if err == nil {
		t.Fatal(fmt.Sprintf("An error was expected because stack id %v is invalid. No error was issued.", invalidID))
	} else {
		if !strings.Contains(err.Error(), invalidID) {
			t.Fatal(fmt.Sprintf("The error message should have contained the name of the invalid stack ID: %v. Error: %v", invalidID, err))
		}
	}

	// Test invalid id staritng with a number.
	invalidID = "0-java-microprofile"
	stackResource.Spec.Name = invalidID
	err = reconcileActiveVersions(&stackResource, client)

	if err == nil {
		t.Fatal(fmt.Sprintf("An error was expected because stack id %v is invalid. No error was issued.", invalidID))
	} else {
		if !strings.Contains(err.Error(), invalidID) {
			t.Fatal(fmt.Sprintf("The error message should have contained the name of the invalid stack ID: %v. Error: %v", invalidID, err))
		}
	}

	// Test invalid id staritng with a dot char.
	invalidID = "java-microprofile.1-0"
	stackResource.Spec.Name = invalidID
	err = reconcileActiveVersions(&stackResource, client)

	if err == nil {
		t.Fatal(fmt.Sprintf("An error was expected because stack id %v is invalid. No error was issued.", invalidID))
	} else {
		if !strings.Contains(err.Error(), invalidID) {
			t.Fatal(fmt.Sprintf("The error message should have contained the name of the invalid stack ID: %v. Error: %v", invalidID, err))
		}
	}

	// Test invalid id starting with invalid chars.
	invalidID = "java#-microprofile@1-0"
	stackResource.Spec.Name = invalidID
	err = reconcileActiveVersions(&stackResource, client)

	if err == nil {
		t.Fatal(fmt.Sprintf("An error was expected because stack id %v is invalid. No error was issued.", invalidID))
	} else {
		if !strings.Contains(err.Error(), invalidID) {
			t.Fatal(fmt.Sprintf("The error message should have contained the name of the invalid stack ID: %v. Error: %v", invalidID, err))
		}
	}

	// Test invalid id containing a single '-'.
	invalidID = "-"
	stackResource.Spec.Name = invalidID
	err = reconcileActiveVersions(&stackResource, client)

	if err == nil {
		t.Fatal(fmt.Sprintf("An error was expected because stack id %v is invalid. No error was issued.", invalidID))
	} else {
		if !strings.Contains(err.Error(), invalidID) {
			t.Fatal(fmt.Sprintf("The error message should have contained the name of the invalid stack ID: %v. Error: %v", invalidID, err))
		}
	}

		// Test invalid id containing a single number.
		invalidID = "9"
  	stackResource.Spec.Name = invalidID
		err = reconcileActiveVersions(&stackResource, client)
	
		if err == nil {
			t.Fatal(fmt.Sprintf("An error was expected because stack id %v is invalid. No error was issued.", invalidID))
		} else {
			if !strings.Contains(err.Error(), invalidID) {
				t.Fatal(fmt.Sprintf("The error message should have contained the name of the invalid stack ID: %v. Error: %v", invalidID, err))
			}
		}

	// Test invalid id with a length greater than 68 characters.
	invalidID = "abcdefghij-abcdefghij-abcdefghij-abcdefghij-abcdefghij-abcdefghij-69c"
	stackResource.Spec.Name = invalidID
	err = reconcileActiveVersions(&stackResource, client)

	if err == nil {
		t.Fatal(fmt.Sprintf("An error was expected because stack id %v is invalid. No error was issued.", invalidID))
	} else {
		if !strings.Contains(err.Error(), invalidID) {
			t.Fatal(fmt.Sprintf("The error message should have contained the name of the invalid stack ID: %v. Error: %v", invalidID, err))
		}
	}

	// Test a valid id containing multiple [a-z0-9-] chars.
	validID := "j-m-1-2-3"
	stackResource.Spec.Name = validID
	err = reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal(fmt.Sprintf("An error was NOT expected. Stack Id: %v is valid. Error: %v", validID, err))
	}

	// Test a valid id containing several '-' chars.
	validID = "n---0"
	stackResource.Spec.Name = validID
	err = reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal(fmt.Sprintf("An error was NOT expected. Stack Id: %v is valid. Error: %v", validID, err))
	}

	// Test a valid id containing only one valid char.
	validID = "x"
	stackResource.Spec.Name = validID
	err = reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal(fmt.Sprintf("An error was NOT expected. Stack Id: %v is valid. Error: %v", validID, err))
	}
}

// --------------------------------------------------------------------------------------------------
// Test that initial stack activation works
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInitial(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + basicPipeline.name

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{{
				Version: "0.2.5",
				DesiredState: "active",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: "default",
					Sha256: basicPipeline.sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: pipelineZipUrl},
				}},
				Images: []kabanerov1alpha2.Image{{
					Id: "default",
					Image: "kabanero/kabanero-image:latest",
				}},
			}},
		},
		Status: kabanerov1alpha2.StackStatus{},
	}

	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the stack resource was updated with asset information
	if len(stackResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Stack status should have 1 pipeline, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	if stackResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Stack active version should be 0.2.5, but is %v", stackResource.Status.Versions[0].Version))
	}

	// Make sure the assets were created in the stack status
	pipeline := stackResource.Status.Versions[0].Pipelines[0]
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

	if pipeline.Name != stackResource.Spec.Versions[0].Pipelines[0].Id {
		t.Fatal(fmt.Sprintf("Pipeline name should be %v, but is %v", stackResource.Spec.Versions[0].Pipelines[0].Id, pipeline.Name))
	}

	// Make sure the status versions array was created in the stack status
	if len(stackResource.Status.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Versions array should have 1 entry, but has %v: %v", len(stackResource.Status.Versions), stackResource.Status.Versions))
	}

	if len(stackResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Stack versions status should have 1 pipeline, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	if stackResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Stack versions active version should be 0.2.5, but is %v", stackResource.Status.Versions[0].Version))
	}

	pipeline = stackResource.Status.Versions[0].Pipelines[0]
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

	if pipeline.Name != stackResource.Spec.Versions[0].Pipelines[0].Id {
		t.Fatal(fmt.Sprintf("Pipeline name should be %v, but is %v", stackResource.Spec.Versions[0].Pipelines[0].Id, pipeline.Name))
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
		if obj[0].UID != stackResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, stackResource.UID, obj[0].UID))
		}
	}

	// Make sure the status lists the images
	if len(stackResource.Status.Versions[0].Images) != 1 {
		t.Fatal(fmt.Sprintf("Status should contain one image, but contains %v: %#v", len(stackResource.Status.Versions[0].Images), stackResource.Status))
	}

	if stackResource.Status.Versions[0].Images[0].Image != stackResource.Spec.Versions[0].Images[0].Image {
		t.Fatal(fmt.Sprintf("Image should be %v, but is %v", stackResource.Spec.Versions[0].Images[0].Image, stackResource.Status.Versions[0].Images[0].Image))
	}
}

// --------------------------------------------------------------------------------------------------
// Test that a migration from one version to another works
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsUpgrade(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + basicPipeline.name
	desiredStack := Stack{
		Name:      "java-microprofile",
		Id:        "java-microprofile",
		Version:   "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{{
				Version: "0.2.5",
				DesiredState: "active",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: desiredStack.Pipelines[0].Id,
					Sha256: desiredStack.Pipelines[0].Sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: desiredStack.Pipelines[0].Url},
				}},
			}},
		},
		Status: kabanerov1alpha2.StackStatus{
			Versions: []kabanerov1alpha2.StackVersionStatus{{
				Version: "0.2.4",
				Pipelines: []kabanerov1alpha2.PipelineStatus{{
					Url:    "https://somewhere.com/v1/pipeline.tar.gz",
					Digest: "1234567",
					Name:   "default",
					ActiveAssets: []kabanerov1alpha2.RepositoryAssetStatus{{
						Name:   "java-microprofile-build-task",
						Status: assetStatusActive,
					}, {
						Name:   "java-microprofile-build-pipeline",
						Status: assetStatusActive,
					}, {
						Name:   "java-microprofile-old-asset",
						Status: assetStatusActive,
					}},
				}},
			}},
		},
	}

	// Tell the client what should currently be there.
	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{
		client.ObjectKey{Name: "java-microprofile-build-task", Namespace: "kabanero"}:     []metav1.OwnerReference{{UID: myuid}},
		client.ObjectKey{Name: "java-microprofile-build-pipeline", Namespace: "kabanero"}: []metav1.OwnerReference{{UID: myuid}},
		client.ObjectKey{Name: "java-microprofile-old-asset", Namespace: "kabanero"}:      []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the stack resource was updated with asset information
	if len(stackResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Stack status should have 1 pipeline, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	if stackResource.Status.Versions[0].Pipelines[0].Url != desiredStack.Pipelines[0].Url {
		t.Fatal(fmt.Sprintf("Stack status should have URL %v, but has %v", desiredStack.Pipelines[0].Url, stackResource.Status.Versions[0].Pipelines[0].Url))
	}

	if stackResource.Status.Versions[0].Pipelines[0].Digest != desiredStack.Pipelines[0].Sha256 {
		t.Fatal(fmt.Sprintf("Stack status should have digest %v, but has %v", desiredStack.Pipelines[0].Sha256, stackResource.Status.Versions[0].Pipelines[0].Digest))
	}

	if stackResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Stack active version should be 0.2.5, but is %v", stackResource.Status.Versions[0].Version))
	}

	// Make sure the actual assets are correct
	pipeline := stackResource.Status.Versions[0].Pipelines[0]
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

	// Make sure the stack versions status array was updated with asset information
	if len(stackResource.Status.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Stack version status should have 1 version, but has %v: %v", len(stackResource.Status.Versions), stackResource.Status.Versions))
	}

	if len(stackResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Stack version status should have 1 pipeline, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	if stackResource.Status.Versions[0].Pipelines[0].Url != desiredStack.Pipelines[0].Url {
		t.Fatal(fmt.Sprintf("Stack version status should have URL %v, but has %v", desiredStack.Pipelines[0].Url, stackResource.Status.Versions[0].Pipelines[0].Url))
	}

	if stackResource.Status.Versions[0].Pipelines[0].Digest != desiredStack.Pipelines[0].Sha256 {
		t.Fatal(fmt.Sprintf("Stack version status should have digest %v, but has %v", desiredStack.Pipelines[0].Sha256, stackResource.Status.Versions[0].Pipelines[0].Digest))
	}

	if stackResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Stack version status version should be 0.2.5, but is %v", stackResource.Status.Versions[0].Version))
	}

	pipeline = stackResource.Status.Versions[0].Pipelines[0]
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
		if obj[0].UID != stackResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, stackResource.UID, obj[0].UID))
		}
	}

}

// --------------------------------------------------------------------------------------------------
// Test that a stack can be deactivated
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsDeactivate(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + basicPipeline.name
	desiredStack := Stack{
		Name:      "java-microprofile",
		Id:        "java-microprofile",
		Version:   "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{{
				Version: "0.2.5",
				DesiredState: "inactive",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: desiredStack.Pipelines[0].Id,
					Sha256: desiredStack.Pipelines[0].Sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: desiredStack.Pipelines[0].Url},
				}},
			}},
		},
		Status: kabanerov1alpha2.StackStatus{
			Versions: []kabanerov1alpha2.StackVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha2.PipelineStatus{{
					Url:    pipelineZipUrl,
					Digest: basicPipeline.sha256,
					Name:   "default",
					ActiveAssets: []kabanerov1alpha2.RepositoryAssetStatus{{
						Name:   "java-microprofile-build-task",
						Status: assetStatusActive,
					}, {
						Name:   "java-microprofile-build-pipeline",
						Status: assetStatusActive,
					}},
				}},
			}},
		},
	}

	// Tell the client what should currently be there.
	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{
		client.ObjectKey{Name: "java-microprofile-build-task", Namespace: "kabanero"}:     []metav1.OwnerReference{{UID: myuid}},
		client.ObjectKey{Name: "java-microprofile-build-pipeline", Namespace: "kabanero"}: []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the stack resource was updated with asset information
	if len(stackResource.Status.Versions[0].Pipelines) != 0 {
		t.Fatal(fmt.Sprintf("Stack status should have 0 pipelines, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	// Stack retains Version when deactivated
	if stackResource.Status.Versions[0].Version != desiredStack.Version {
		t.Fatal(fmt.Sprintf("Stack deactive version should be %v, but is %v", desiredStack.Version, stackResource.Status.Versions[0].Version))
	}


	if stackResource.Status.Versions[0].StatusMessage == "" {
		t.Fatal("Stack status message should not be empty for an inactive stack")
	}

	// Make sure the stack version resource was updated with asset information
	if len(stackResource.Status.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Stack version status should have 1 entry, but has %v: %v", len(stackResource.Status.Versions), stackResource.Status.Versions))
	}

	if stackResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Stack version status should have version \"0.2.5\", but has %v", stackResource.Status.Versions[0].Version))
	}

	if stackResource.Status.Versions[0].StatusMessage == "" {
		t.Fatal("Stack version status message should not be empty for an inactive stack")
	}

	if stackResource.Status.Versions[0].Status != kabanerov1alpha2.StackDesiredStateInactive {
		t.Fatal(fmt.Sprintf("Stack version status should be inactive, but is %v", stackResource.Status.Versions[0].Status))
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
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + basicPipeline.name
	desiredStack := Stack{
		Name:      "java-microprofile",
		Id:        "java-microprofile",
		Version:   "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{{
				Version: "0.2.5",
				DesiredState: "active",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: desiredStack.Pipelines[0].Id,
					Sha256: desiredStack.Pipelines[0].Sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: desiredStack.Pipelines[0].Url},
				}},
			}},
		},
		Status: kabanerov1alpha2.StackStatus{},
	}

	// Tell the client what should currently be there.
	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{
		client.ObjectKey{Name: "java-microprofile-build-task", Namespace: "kabanero"}:     []metav1.OwnerReference{{UID: otheruid}},
		client.ObjectKey{Name: "java-microprofile-build-pipeline", Namespace: "kabanero"}: []metav1.OwnerReference{{UID: otheruid}}}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the stack resource was updated with asset information
	if len(stackResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Stack status should have 1 pipeline, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	if stackResource.Status.Versions[0].Pipelines[0].Url != desiredStack.Pipelines[0].Url {
		t.Fatal(fmt.Sprintf("Stack status should have URL %v, but has %v", desiredStack.Pipelines[0].Url, stackResource.Status.Versions[0].Pipelines[0].Url))
	}

	if stackResource.Status.Versions[0].Pipelines[0].Digest != desiredStack.Pipelines[0].Sha256 {
		t.Fatal(fmt.Sprintf("Stack status should have digest %v, but has %v", desiredStack.Pipelines[0].Sha256, stackResource.Status.Versions[0].Pipelines[0].Digest))
	}

	if stackResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Stack active version should be 0.2.5, but is %v", stackResource.Status.Versions[0].Version))
	}

	// Make sure the actual assets are correct
	pipeline := stackResource.Status.Versions[0].Pipelines[0]
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
			t.Fatal(fmt.Sprintf("Did not find correct stack owners in %v: %v", key, obj))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that a deactivate for shared assets removes an object owner
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsSharedAssetDeactivate(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + basicPipeline.name
	desiredStack := Stack{
		Name:      "java-microprofile",
		Id:        "java-microprofile",
		Version:   "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{{
				Version: "0.2.5",
				DesiredState: "inactive",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: desiredStack.Pipelines[0].Id,
					Sha256: desiredStack.Pipelines[0].Sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: desiredStack.Pipelines[0].Url},
				}},
			}},
		},
		Status: kabanerov1alpha2.StackStatus{
			Versions: []kabanerov1alpha2.StackVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha2.PipelineStatus{{
					Url:    pipelineZipUrl,
					Digest: basicPipeline.sha256,
					Name:   "default",
					ActiveAssets: []kabanerov1alpha2.RepositoryAssetStatus{{
						Name:   "java-microprofile-build-task",
						Status: assetStatusActive,
					}, {
						Name:   "java-microprofile-build-pipeline",
						Status: assetStatusActive,
					}},
				}},
			}},
		},
	}

	// Tell the client what should currently be there.
	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{
		client.ObjectKey{Name: "java-microprofile-build-task", Namespace: "kabanero"}:     []metav1.OwnerReference{{UID: otheruid}, {UID: myuid}},
		client.ObjectKey{Name: "java-microprofile-build-pipeline", Namespace: "kabanero"}: []metav1.OwnerReference{{UID: otheruid}, {UID: myuid}}}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the stack resource was updated with asset information
	if len(stackResource.Status.Versions[0].Pipelines) != 0 {
		t.Fatal(fmt.Sprintf("Stack status should have 0 pipelines, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	// Stack retains Version when deactivated
	if stackResource.Status.Versions[0].Version != desiredStack.Version {
		t.Fatal(fmt.Sprintf("Stack deactive version should be %v, but is %v", desiredStack.Version, stackResource.Status.Versions[0].Version))
	}

	if stackResource.Status.Versions[0].StatusMessage == "" {
		t.Fatal("Stack status message should not be empty for an inactive stack")
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
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + basicPipeline.name
	desiredStack := Stack{
		Name:      "java-microprofile",
		Id:        "java-microprofile",
		Version:   "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}},
	}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{{
				Version: "0.2.5",
				DesiredState: "active",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: desiredStack.Pipelines[0].Id,
					Sha256: desiredStack.Pipelines[0].Sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: desiredStack.Pipelines[0].Url},
				}},
			}},
		},
		Status: kabanerov1alpha2.StackStatus{
			Versions: []kabanerov1alpha2.StackVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha2.PipelineStatus{{
					Url:    pipelineZipUrl,
					Digest: basicPipeline.sha256,
					Name:   "default",
					ActiveAssets: []kabanerov1alpha2.RepositoryAssetStatus{{
						Name:   "java-microprofile-build-task",
						Status: assetStatusActive,
					}, {
						Name:   "java-microprofile-build-pipeline",
						Status: assetStatusActive,
					}},
				}},
			}},
		},
	}

	// Tell the client what should currently be there.
	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{
		client.ObjectKey{Name: "java-microprofile-build-task", Namespace: "kabanero"}: []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the stack resource was updated with asset information
	if len(stackResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Stack status should have 1 pipeline, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	if stackResource.Status.Versions[0].Pipelines[0].Url != desiredStack.Pipelines[0].Url {
		t.Fatal(fmt.Sprintf("Stack status should have URL %v, but has %v", desiredStack.Pipelines[0].Url, stackResource.Status.Versions[0].Pipelines[0].Url))
	}

	if stackResource.Status.Versions[0].Pipelines[0].Digest != desiredStack.Pipelines[0].Sha256 {
		t.Fatal(fmt.Sprintf("Stack status should have digest %v, but has %v", desiredStack.Pipelines[0].Sha256, stackResource.Status.Versions[0].Pipelines[0].Digest))
	}

	if stackResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Stack active version should be 0.2.5, but is %v", stackResource.Status.Versions[0].Version))
	}

	// Make sure the actual assets are correct
	pipeline :=stackResource.Status.Versions[0].Pipelines[0]
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
		if obj[0].UID != stackResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, stackResource.UID, obj[0].UID))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that a reconcile will attempt to re-create assets that had been deleted, but since the
// manifests are gone, it can't.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsRecreatedDeletedAssetsNoManifest(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	deletedPipeline := fileInfo{
		name:   "/deleted.pipeline.tar.gz",
		sha256: "aaaabbbbccccdddd"}

	pipelineZipUrl := server.URL + deletedPipeline.name
	desiredStack := Stack{
		Name:      "java-microprofile",
		Id:        "java-microprofile",
		Version:   "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: deletedPipeline.sha256, Url: pipelineZipUrl}},
	}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{{
				Version: "0.2.5",
				DesiredState: "active",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: desiredStack.Pipelines[0].Id,
					Sha256: desiredStack.Pipelines[0].Sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: desiredStack.Pipelines[0].Url},
				}},
			}},
		},
		Status: kabanerov1alpha2.StackStatus{
			Versions: []kabanerov1alpha2.StackVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha2.PipelineStatus{{
					Url:    pipelineZipUrl,
					Digest: deletedPipeline.sha256,
					Name:   "default",
					ActiveAssets: []kabanerov1alpha2.RepositoryAssetStatus{{
						Name:   "java-microprofile-build-task",
						Status: assetStatusActive,
					}, {
						Name:   "java-microprofile-build-pipeline",
						Status: assetStatusActive,
					}},
				}},
			}},
		},
	}

	// Tell the client what should currently be there.
	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{
		client.ObjectKey{Name: "java-microprofile-build-task", Namespace: "kabanero"}: []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the stack resource was updated with asset information
	if len(stackResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Stack status should have 1 pipeline, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	if stackResource.Status.Versions[0].Pipelines[0].Url != desiredStack.Pipelines[0].Url {
		t.Fatal(fmt.Sprintf("Stack status should have URL %v, but has %v", desiredStack.Pipelines[0].Url, stackResource.Status.Versions[0].Pipelines[0].Url))
	}

	if stackResource.Status.Versions[0].Pipelines[0].Digest != desiredStack.Pipelines[0].Sha256 {
		t.Fatal(fmt.Sprintf("Stack status should have digest %v, but has %v", desiredStack.Pipelines[0].Sha256, stackResource.Status.Versions[0].Pipelines[0].Digest))
	}

	if stackResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Stack active version should be 0.2.5, but is %v", stackResource.Status.Versions[0].Version))
	}

	// Make sure the actual assets are correct
	pipeline := stackResource.Status.Versions[0].Pipelines[0]
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
		if obj[0].UID != stackResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, stackResource.UID, obj[0].UID))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that a stack with a bad asset gets an appropriate error message.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsBadAsset(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + badPipeline.name
	desiredStack := Stack{
		Name:      "java-microprofile",
		Id:        "java-microprofile",
		Version:   "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: badPipeline.sha256, Url: pipelineZipUrl}},
	}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{{
				Version: "0.2.5",
				DesiredState: "active",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: desiredStack.Pipelines[0].Id,
					Sha256: desiredStack.Pipelines[0].Sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: desiredStack.Pipelines[0].Url},
				}},
			}},
		},
		Status: kabanerov1alpha2.StackStatus{},
	}

	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the stack resource was updated with asset information
	if len(stackResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Stack status should have 1 pipeline, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	if stackResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Stack active version should be 0.2.5, but is %v", stackResource.Status.Versions[0].Version))
	}

	// Make sure the assets were created in the stack status
	pipeline := stackResource.Status.Versions[0].Pipelines[0]
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
		if obj[0].UID != stackResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, stackResource.UID, obj[0].UID))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that tekton triggers are created in the tekton-pipelines namespace
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsWithTriggers(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	defaultImage := Images{Id: "default", Image: "kabanero/kabanero-image:latest"}

	pipelineZipUrl := server.URL + triggerPipeline.name
	desiredStack := Stack{
		Name:      "java-microprofile",
		Id:        "java-microprofile",
		Version:   "0.2.5",
		Pipelines: []Pipelines{{Id: "default", Sha256: triggerPipeline.sha256, Url: pipelineZipUrl}},
		Images:    []Images{defaultImage},
	}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{{
				Version: "0.2.5",
				DesiredState: "active",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: desiredStack.Pipelines[0].Id,
					Sha256: desiredStack.Pipelines[0].Sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: desiredStack.Pipelines[0].Url},
				}},
				Images: []kabanerov1alpha2.Image{{
					Id: desiredStack.Images[0].Id,
					Image: desiredStack.Images[0].Image,
				}},
			}},
		},
		Status: kabanerov1alpha2.StackStatus{},
	}

	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the stack resource was updated with asset information
	if len(stackResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Stack status should have 1 pipeline, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	if stackResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Stack active version should be 0.2.5, but is %v", stackResource.Status.Versions[0].Version))
	}

	// Make sure the assets were created in the stack status
	pipeline := stackResource.Status.Versions[0].Pipelines[0]
	if len(pipeline.ActiveAssets) != 3 {
		t.Fatal(fmt.Sprintf("Pipeline should have 3 assets, but has %v", len(pipeline.ActiveAssets)))
	}

	for _, asset := range pipeline.ActiveAssets {
		if asset.Status != assetStatusActive {
			t.Fatal(fmt.Sprintf("Asset %v should have status active, but is %v", asset.Name, asset.Status))
		}
		if asset.StatusMessage != "" {
			t.Fatal(fmt.Sprintf("Asset %v should have no status message, but has %v", asset.Name, asset.StatusMessage))
		}
		// Check to make sure that the trigger was created in the tekton-pipelines namespace
		if asset.Name == "java-microprofile-build-trigger-template" {
			if asset.Namespace != "tekton-pipelines" {
				t.Fatal(fmt.Sprintf("Asset %v should have been in the tekton-pipelines namespace, but was in %v", asset.Name, asset.Namespace))
			}
		} else {
			if asset.Namespace != "kabanero" {
				t.Fatal(fmt.Sprintf("Asset %v should have been in the kabanero namespace, but was in %v", asset.Name, asset.Namespace))
			}
		}
	}

	if pipeline.Name != desiredStack.Pipelines[0].Id {
		t.Fatal(fmt.Sprintf("Pipeline name should be %v, but is %v", desiredStack.Pipelines[0].Id, pipeline.Name))
	}

	// Make sure the status versions array was created in the stack status
	if len(stackResource.Status.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Versions array should have 1 entry, but has %v: %v", len(stackResource.Status.Versions), stackResource.Status.Versions))
	}

	if len(stackResource.Status.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Stack versions status should have 1 pipeline, but has %v", len(stackResource.Status.Versions[0].Pipelines)))
	}

	if stackResource.Status.Versions[0].Version != "0.2.5" {
		t.Fatal(fmt.Sprintf("Stack versions active version should be 0.2.5, but is %v", stackResource.Status.Versions[0].Version))
	}

	pipeline = stackResource.Status.Versions[0].Pipelines[0]
	if len(pipeline.ActiveAssets) != 3 {
		t.Fatal(fmt.Sprintf("Pipeline should have 3 assets, but has %v", len(pipeline.ActiveAssets)))
	}

	for _, asset := range pipeline.ActiveAssets {
		if asset.Status != assetStatusActive {
			t.Fatal(fmt.Sprintf("Asset %v should have status active, but is %v", asset.Name, asset.Status))
		}
		if asset.StatusMessage != "" {
			t.Fatal(fmt.Sprintf("Asset %v should have no status message, but has %v", asset.Name, asset.StatusMessage))
		}
		// Check to make sure that the trigger was created in the tekton-pipelines namespace
		if asset.Name == "java-microprofile-build-trigger-template" {
			if asset.Namespace != "tekton-pipelines" {
				t.Fatal(fmt.Sprintf("Asset %v should have been in the tekton-pipelines namespace, but was in %v", asset.Name, asset.Namespace))
			}
		} else {
			if asset.Namespace != "kabanero" {
				t.Fatal(fmt.Sprintf("Asset %v should have been in the kabanero namespace, but was in %v", asset.Name, asset.Namespace))
			}
		}
	}

	if pipeline.Name != desiredStack.Pipelines[0].Id {
		t.Fatal(fmt.Sprintf("Pipeline name should be %v, but is %v", desiredStack.Pipelines[0].Id, pipeline.Name))
	}

	// Make sure the client has the correct objects.
	if len(client.objs) != 3 {
		t.Fatal(fmt.Sprintf("Client map should have 3 entries, but has %v: %v", len(client.objs), client.objs))
	}

	// Make sure the client's objects have an owner set.
	for key, obj := range client.objs {
		if key.Name != "java-microprofile-build-trigger-template" {
			if len(obj) != 1 {
				t.Fatal(fmt.Sprintf("Client object %v should have 1 owner, but has %v: %v", key, len(obj), obj))
			}
			if obj[0].UID != stackResource.UID {
				t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, stackResource.UID, obj[0].UID))
			}
			if key.Namespace != "kabanero" {
				t.Fatal(fmt.Sprintf("Client object %v should have been created in kabanero namespace, but was %v", key.Name, key.Namespace))
			}
		} else {
			if len(obj) != 0 {
				t.Fatal(fmt.Sprintf("Client object %v should have 0 owners, but has %v: %v", key, len(obj), obj))
			}
			if key.Namespace != "tekton-pipelines" {
				t.Fatal(fmt.Sprintf("Client object %v should have been created in tekton-pipelines namespace, but was %v", key.Name, key.Namespace))
			}
		}
	}

	// Make sure the status lists the images
	if len(stackResource.Status.Versions[0].Images) != 1 {
		t.Fatal(fmt.Sprintf("Status should contain one image, but contains %v: %#v", len(stackResource.Status.Versions[0].Images), stackResource.Status))
	}

	if stackResource.Status.Versions[0].Images[0].Image != defaultImage.Image {
		t.Fatal(fmt.Sprintf("Image should be %v, but is %v", defaultImage.Image, stackResource.Status.Versions[0].Images[0].Image))
	}
}

// ==================================================================================================
// --------------------------------------------------------------------------------------------------
// The following tests activate multiple versions of a stack.
// --------------------------------------------------------------------------------------------------
// ==================================================================================================

// --------------------------------------------------------------------------------------------------
// Test that two versions of the same stack can be activated.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInternalTwoInitial(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + basicPipeline.name
	stacks := []resolvedStack{{
		repositoryURL: "",
		stack: Stack{
			Name:      "java-microprofile",
			Id:        "java-microprofile",
			Version:   "0.2.5",
			Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}}},
	}, {
		repositoryURL: "",
		stack: Stack{
			Name:      "java-microprofile",
			Id:        "java-microprofile",
			Version:   "0.2.6",
			Pipelines: []Pipelines{{Id: "default", Sha256: basicPipeline.sha256, Url: pipelineZipUrl}}},
	}}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{
				kabanerov1alpha2.StackVersion{
					Version: "0.2.5",
					DesiredState: "active",
					Pipelines: []kabanerov1alpha2.PipelineSpec{{
						Id: stacks[0].stack.Pipelines[0].Id,
						Sha256: stacks[0].stack.Pipelines[0].Sha256,
						Https: kabanerov1alpha2.HttpsProtocolFile{Url: stacks[0].stack.Pipelines[0].Url},
					}},
				},
				kabanerov1alpha2.StackVersion{
					Version: "0.2.6",
					DesiredState: "active",
					Pipelines: []kabanerov1alpha2.PipelineSpec{{
						Id: stacks[1].stack.Pipelines[0].Id,
						Sha256: stacks[1].stack.Pipelines[0].Sha256,
						Https: kabanerov1alpha2.HttpsProtocolFile{Url: stacks[1].stack.Pipelines[0].Url},
					}},
				},
			},
		},
		Status: kabanerov1alpha2.StackStatus{},
	}

	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure we got two status structs back
	if len(stackResource.Status.Versions) != 2 {
		t.Fatal(fmt.Sprintf("Expected two statuses, but got %v: %#v", len(stackResource.Status.Versions), stackResource.Status))
	}

	// Make sure the stack resource was updated with asset information
	versionsFound := make(map[string]bool)
	for _, curStatus := range stackResource.Status.Versions {
		versionsFound[curStatus.Version] = true

		if len(curStatus.Pipelines) != 1 {
			t.Fatal(fmt.Sprintf("Stack status should have 1 pipeline, but has %v: %v", len(curStatus.Pipelines), curStatus))
		}

		// Make sure the assets were created in the stack status
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
		t.Fatal(fmt.Sprintf("Did not find version 0.2.5 in the status: %v", stackResource.Status))
	}

	if versionsFound["0.2.6"] == false {
		t.Fatal(fmt.Sprintf("Did not find version 0.2.6 in the status: %v", stackResource.Status))
	}

	// Make sure that the singleton status matches the first element in the versions status
	if stackResource.Status.Versions[0].Version != stackResource.Status.Versions[0].Version {
		t.Fatal(fmt.Sprintf("Stack status activeVersion %v does not match stack status version[0] %v", stackResource.Status.Versions[0].Version, stackResource.Status.Versions[0].Version))
	}

	if stackResource.Status.Versions[0].Location != stackResource.Status.Versions[0].Location {
		t.Fatal(fmt.Sprintf("Stack status activeLocation %v does not match stack status version [0] location %v", stackResource.Status.Versions[0].Location, stackResource.Status.Versions[0].Location))
	}

	if stackResource.Status.Versions[0].Status != stackResource.Status.Versions[0].Status {
		t.Fatal(fmt.Sprintf("Stack status status %v does not match stack status version[0] status %v", stackResource.Status.Versions[0].Status, stackResource.Status.Versions[0].Status))
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
		if obj[0].UID != stackResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, stackResource.UID, obj[0].UID))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that two versions of the same stack using different pipelines can be activated.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInternalTwoInitialDiffPipelines(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipeline1ZipUrl := server.URL + digest1Pipeline.name
	pipeline2ZipUrl := server.URL + digest2Pipeline.name
	stacks := []resolvedStack{{
		repositoryURL: "",
		stack: Stack{
			Name:      "java-microprofile",
			Id:        "java-microprofile",
			Version:   "0.2.5",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest1Pipeline.sha256, Url: pipeline1ZipUrl}}},
	}, {
		repositoryURL: "",
		stack: Stack{
			Name:      "java-microprofile",
			Id:        "java-microprofile",
			Version:   "0.2.6",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest2Pipeline.sha256, Url: pipeline2ZipUrl}}},
	}}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{
				kabanerov1alpha2.StackVersion{
					Version: "0.2.5",
					DesiredState: "active",
					Pipelines: []kabanerov1alpha2.PipelineSpec{{
						Id: stacks[0].stack.Pipelines[0].Id,
						Sha256: stacks[0].stack.Pipelines[0].Sha256,
						Https: kabanerov1alpha2.HttpsProtocolFile{Url: stacks[0].stack.Pipelines[0].Url},
					}},
				},
				kabanerov1alpha2.StackVersion{
					Version: "0.2.6",
					DesiredState: "active",
					Pipelines: []kabanerov1alpha2.PipelineSpec{{
						Id: stacks[1].stack.Pipelines[0].Id,
						Sha256: stacks[1].stack.Pipelines[0].Sha256,
						Https: kabanerov1alpha2.HttpsProtocolFile{Url: stacks[1].stack.Pipelines[0].Url},
					}},
				},
			},
		},
		Status: kabanerov1alpha2.StackStatus{},
	}

	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure we got two status structs back
	if len(stackResource.Status.Versions) != 2 {
		t.Fatal(fmt.Sprintf("Expected two statuses, but got %v: %#v", len(stackResource.Status.Versions), stackResource.Status))
	}

	// Make sure the stack resource was updated with asset information
	versionsFound := make(map[string]bool)
	for _, curStatus := range stackResource.Status.Versions {
		versionsFound[curStatus.Version] = true

		if len(curStatus.Pipelines) != 1 {
			t.Fatal(fmt.Sprintf("Stack status should have 1 pipeline, but has %v: %v", len(curStatus.Pipelines), curStatus))
		}

		// Make sure the assets were created in the stack status
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
		t.Fatal(fmt.Sprintf("Did not find version 0.2.5 in the status: %v", stackResource.Status))
	}

	if versionsFound["0.2.6"] == false {
		t.Fatal(fmt.Sprintf("Did not find version 0.2.6 in the status: %v", stackResource.Status))
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
		if obj[0].UID != stackResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, stackResource.UID, obj[0].UID))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that one version of a stack can be deleted but the other remains active.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInternalTwoDeactivateOne(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipeline1ZipUrl := server.URL + digest1Pipeline.name
	pipeline2ZipUrl := server.URL + digest2Pipeline.name

	stacks := []resolvedStack{{
		repositoryURL: "",
		stack: Stack{
			Name:      "java-microprofile",
			Id:        "java-microprofile",
			Version:   "0.2.5",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest1Pipeline.sha256, Url: pipeline1ZipUrl}}},
	}, {
		repositoryURL: "",
		stack: Stack{
			Name:      "java-microprofile",
			Id:        "java-microprofile",
			Version:   "0.2.6",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest2Pipeline.sha256, Url: pipeline2ZipUrl}},
		}},
	}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name:     "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{{
				Version: "0.2.6",
				DesiredState: "active",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: stacks[1].stack.Pipelines[0].Id,
					Sha256: stacks[1].stack.Pipelines[0].Sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: stacks[1].stack.Pipelines[0].Url},
				}},
			}},
		},
		Status: kabanerov1alpha2.StackStatus{
			Versions: []kabanerov1alpha2.StackVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha2.PipelineStatus{{
					Url:    pipeline1ZipUrl,
					Digest: digest1Pipeline.sha256,
					Name:   "default",
					ActiveAssets: []kabanerov1alpha2.RepositoryAssetStatus{{
						Name:   "build-task-0238ff31",
						Status: assetStatusActive,
					}, {
						Name:   "build-pipeline-0238ff31",
						Status: assetStatusActive,
					}},
				}},
			}, {
				Version: "0.2.6",
				Pipelines: []kabanerov1alpha2.PipelineStatus{{
					Url:    pipeline2ZipUrl,
					Digest: digest2Pipeline.sha256,
					Name:   "default",
					ActiveAssets: []kabanerov1alpha2.RepositoryAssetStatus{{
						Name:   "build-task-c3f28ffc",
						Status: assetStatusActive,
					}, {
						Name:   "build-pipeline-c3f28ffc",
						Status: assetStatusActive,
					}},
				}},
			}},
		},
	}

	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{
		client.ObjectKey{Name: "build-task-0238ff31", Namespace: "kabanero"}:     []metav1.OwnerReference{{UID: myuid}},
		client.ObjectKey{Name: "build-pipeline-0238ff31", Namespace: "kabanero"}: []metav1.OwnerReference{{UID: myuid}},
		client.ObjectKey{Name: "build-task-c3f28ffc", Namespace: "kabanero"}:     []metav1.OwnerReference{{UID: myuid}},
		client.ObjectKey{Name: "build-pipeline-c3f28ffc", Namespace: "kabanero"}: []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure we got one status structs back
	if len(stackResource.Status.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Expected one status, but got %v: %#v", len(stackResource.Status.Versions), stackResource.Status.Versions))
	}

	// Make sure the stack resource was updated with asset information
	for _, curStatus := range stackResource.Status.Versions {
		if curStatus.Version != "0.2.6" {
			t.Fatal(fmt.Sprintf("Expected stack version 0.2.6, but found %v: %#v", curStatus.Version, curStatus))
		}

		if len(curStatus.Pipelines) != 1 {
			t.Fatal(fmt.Sprintf("Stack status should have 1 pipeline, but has %v: %v", len(curStatus.Pipelines), curStatus))
		}

		// Make sure the assets were created in the stack status
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
		if obj[0].UID != stackResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, stackResource.UID, obj[0].UID))
		}
	}
}

// --------------------------------------------------------------------------------------------------
// Test that one version of a stack can be inactive but the other remains active.
// --------------------------------------------------------------------------------------------------
func TestReconcileActiveVersionsInternalTwoDeleteOne(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipeline1ZipUrl := server.URL + digest1Pipeline.name
	pipeline2ZipUrl := server.URL + digest2Pipeline.name
	stacks := []resolvedStack{{
		repositoryURL: "",
		stack: Stack{
			Name:      "java-microprofile",
			Id:        "java-microprofile",
			Version:   "0.2.5",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest1Pipeline.sha256, Url: pipeline1ZipUrl}},
		},
	}, {
		repositoryURL: "",
		stack: Stack{
			Name:      "java-microprofile",
			Id:        "java-microprofile",
			Version:   "0.2.6",
			Pipelines: []Pipelines{{Id: "default", Sha256: digest2Pipeline.sha256, Url: pipeline2ZipUrl}},
		},
	}}

	stackResource := kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{UID: myuid, Namespace: "kabanero"},
		Spec: kabanerov1alpha2.StackSpec{
			Name: "java-microprofile",
			Versions: []kabanerov1alpha2.StackVersion{
				kabanerov1alpha2.StackVersion{
					Version: "0.2.5",
					DesiredState: "inactive",
					Pipelines: []kabanerov1alpha2.PipelineSpec{{
						Id: stacks[0].stack.Pipelines[0].Id,
						Sha256: stacks[0].stack.Pipelines[0].Sha256,
						Https: kabanerov1alpha2.HttpsProtocolFile{Url: stacks[0].stack.Pipelines[0].Url},
					}},
				},
				kabanerov1alpha2.StackVersion{
					Version: "0.2.6",
					DesiredState: "active",
					Pipelines: []kabanerov1alpha2.PipelineSpec{{
						Id: stacks[1].stack.Pipelines[0].Id,
						Sha256: stacks[1].stack.Pipelines[0].Sha256,
						Https: kabanerov1alpha2.HttpsProtocolFile{Url: stacks[1].stack.Pipelines[0].Url},
					}},
				},
			},
		},
		Status: kabanerov1alpha2.StackStatus{
			Versions: []kabanerov1alpha2.StackVersionStatus{{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha2.PipelineStatus{{
					Url:    pipeline1ZipUrl,
					Digest: digest1Pipeline.sha256,
					Name:   "default",
					ActiveAssets: []kabanerov1alpha2.RepositoryAssetStatus{{
						Name:   "build-task-0238ff31",
						Status: assetStatusActive,
					}, {
						Name:   "build-pipeline-0238ff31",
						Status: assetStatusActive,
					}},
				}},
			}, {
				Version: "0.2.6",
				Pipelines: []kabanerov1alpha2.PipelineStatus{{
					Url:    pipeline2ZipUrl,
					Digest: digest2Pipeline.sha256,
					Name:   "default",
					ActiveAssets: []kabanerov1alpha2.RepositoryAssetStatus{{
						Name:   "build-task-c3f28ffc",
						Status: assetStatusActive,
					}, {
						Name:   "build-pipeline-c3f28ffc",
						Status: assetStatusActive,
					}},
				}},
			}},
		},
	}

	client := unitTestClient{map[client.ObjectKey][]metav1.OwnerReference{
		client.ObjectKey{Name: "build-task-0238ff31", Namespace: "kabanero"}:     []metav1.OwnerReference{{UID: myuid}},
		client.ObjectKey{Name: "build-pipeline-0238ff31", Namespace: "kabanero"}: []metav1.OwnerReference{{UID: myuid}},
		client.ObjectKey{Name: "build-task-c3f28ffc", Namespace: "kabanero"}:     []metav1.OwnerReference{{UID: myuid}},
		client.ObjectKey{Name: "build-pipeline-c3f28ffc", Namespace: "kabanero"}: []metav1.OwnerReference{{UID: myuid}}}}

	err := reconcileActiveVersions(&stackResource, client)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure we got one status structs back
	if len(stackResource.Status.Versions) != 2 {
		t.Fatal(fmt.Sprintf("Expected two statuses, but got %v: %#v", len(stackResource.Status.Versions), stackResource.Status.Versions))
	}

	// Make sure the stack resource was updated with asset information
	versionsFound := make(map[string]bool)
	for _, curStatus := range stackResource.Status.Versions {
		versionsFound[curStatus.Version] = true

		if curStatus.Version == "0.2.5" {
			if len(curStatus.Pipelines) != 0 {
				t.Fatal(fmt.Sprintf("Stack version 0.2.5 should not have any active pipelines: %#v", curStatus.Pipelines))
			}

			if curStatus.StatusMessage == "" {
				t.Fatal(fmt.Sprintf("Stack version 0.2.5 should have a status message, but has none."))
			}

			if curStatus.Status != kabanerov1alpha2.StackDesiredStateInactive {
				t.Fatal(fmt.Sprintf("Stack version 0.2.5 should be marked inactive, but is %v", curStatus.Status))
			}
		} else if curStatus.Version == "0.2.6" {
			if len(curStatus.Pipelines) != 1 {
				t.Fatal(fmt.Sprintf("Stack status should have 1 pipeline, but has %v: %v", len(curStatus.Pipelines), curStatus))
			}

			// Make sure the assets were created in the stack status
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
		if obj[0].UID != stackResource.UID {
			t.Fatal(fmt.Sprintf("Client object %v should have owner UID %v but has %v", key, stackResource.UID, obj[0].UID))
		}
	}
}

// TODO: More "multiple stack" tests...
