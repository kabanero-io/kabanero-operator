package kabaneroplatform

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"testing"
)

// -----------------------------------------------------------------------------------------------
// Client that creates/deletes stacks.
// -----------------------------------------------------------------------------------------------
type unitTestClient struct {
	objs map[string]*kabanerov1alpha2.Stack
}

func (c unitTestClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	fmt.Printf("Received Get() for %v\n", key.Name)
	u, ok := obj.(*kabanerov1alpha2.Stack)
	if !ok {
		fmt.Printf("Received invalid target object for get: %v\n", obj)
		return errors.New("Get only supports stacks")
	}
	stack := c.objs[key.Name]
	if stack == nil {
		return apierrors.NewNotFound(schema.GroupResource{}, key.Name)
	}
	stack.DeepCopyInto(u)
	return nil
}

func (c unitTestClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	l, ok := list.(*kabanerov1alpha2.StackList)
	if !ok {
		fmt.Printf("Received an invalid list object: %v\n", list)
		return errors.New("Get only supports stacks")
	}

	stackList := &kabanerov1alpha2.StackList{}
	items := []kabanerov1alpha2.Stack{}
	for _, stack := range c.objs {
		items = append(items, *stack)
	}

	stackList.Items = items
	stackList.DeepCopyInto(l)

	return nil
}
func (c unitTestClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	u, ok := obj.(*kabanerov1alpha2.Stack)
	if !ok {
		fmt.Printf("Received invalid create: %v\n", obj)
		return errors.New("Create only supports Stacks")
	}

	fmt.Printf("Received Create() for %v\n", u.Name)
	stack := c.objs[u.Name]
	if stack != nil {
		fmt.Printf("Receive create object already exists: %v\n", u.Name)
		return apierrors.NewAlreadyExists(schema.GroupResource{}, u.Name)
	}

	c.objs[u.Name] = u
	return nil
}
func (c unitTestClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	u, ok := obj.(*kabanerov1alpha2.Stack)
	if !ok {
		fmt.Printf("Received an invalid delete object: %v\n", obj)
		return errors.New("Update only supports Stack")
	}

	delete(c.objs, u.Name)
	return nil
}
func (c unitTestClient) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	return errors.New("DeleteAllOf is not supported")
}
func (c unitTestClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	u, ok := obj.(*kabanerov1alpha2.Stack)
	if !ok {
		fmt.Printf("Received invalid update: %v\n", obj)
		return errors.New("Update only supports Stack")
	}

	fmt.Printf("Received Update() for %v\n", u.Name)
	stack := c.objs[u.Name]
	if stack == nil {
		fmt.Printf("Received update for object that does not exist: %v\n", obj)
		return apierrors.NewNotFound(schema.GroupResource{}, u.Name)
	}
	c.objs[u.Name] = u
	return nil
}
func (c unitTestClient) Status() client.StatusWriter { return c }
func (c unitTestClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return errors.New("Patch is not supported")
}

// -----------------------------------------------------------------------------------------------
// HTTP handler that serves pipeline zips
// -----------------------------------------------------------------------------------------------
type stackIndexHandler struct {
}

func (ch stackIndexHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	filename := fmt.Sprintf("testdata/%v", req.URL.String())
	fmt.Printf("Serving %v\n", filename)
	d, err := ioutil.ReadFile(filename)
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
	} else {
		rw.Write(d)
	}
}

var defaultIndexName = "/kabanero-index.yaml"
var secondIndexName = "/kabanero-index-two.yaml"

var appsodyIndexName = "/appsody-index.yaml"

var defaultIndexPipeline = "https://github.com/kabanero-io/collections/releases/download/0.4.0/incubator.common.pipeline.default.tar.gz"
var defaultIndexPipelineDigest = "0123456789012345678901234567890123456789012345678901234567890123"
var secondIndexPipeline = "https://github.com/kabanero-io/collections/releases/download/0.6.0/incubator.common.pipeline.default.tar.gz"
var secondIndexPipelineDigest = "1234567890123456789012345678901234567890123456789012345678901234"
var featuredTestLogger logr.Logger = log.WithValues("Request.Namespace", "test", "Request.Name", "featured_stacks_test")

var stackResource kabanerov1alpha2.Stack = kabanerov1alpha2.Stack{
	ObjectMeta: metav1.ObjectMeta{Name: "nodejs", UID: "myuid", Namespace: "kabanero"},
	Spec: kabanerov1alpha2.StackSpec{
		Name: "nodejs",
		Versions: []kabanerov1alpha2.StackVersion{
			kabanerov1alpha2.StackVersion{
				Version: "0.2.4",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id:     "trigger.pipeline.0.2.4.tar.gz",
					Sha256: "2e8ff2e5c6ce8526edc9ce413876c450383814d4fa6f5f37b690d167433da363",
					Https:  kabanerov1alpha2.HttpsProtocolFile{Url: "https://pipelines/default/0.2.4"},
				}},
			},
			kabanerov1alpha2.StackVersion{
				Version: "0.2.5",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id:     "trigger.pipeline.0.2.5.tar.gz",
					Sha256: "2e8ff2e5c6ce8526edc9ce413876c450383814d4fa6f5f37b690d167433da363",
					Https:  kabanerov1alpha2.HttpsProtocolFile{Url: "https://pipelines/default/0.2.5"},
				}},
			},
			kabanerov1alpha2.StackVersion{
				Version: "0.2.6",
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id:     "trigger.pipeline.0.2.6.tar.gz",
					Sha256: "2e8ff2e5c6ce8526edc9ce413876c450383814d4fa6f5f37b690d167433da363",
					Https:  kabanerov1alpha2.HttpsProtocolFile{Url: "https://pipelines/default/0.2.6"},
				}},
			},
		},
	},
	Status: kabanerov1alpha2.StackStatus{},
}

// -----------------------------------------------------------------------------------------------
// Test cases
// -----------------------------------------------------------------------------------------------
func createKabanero(repositoryUrl string) *kabanerov1alpha2.Kabanero {
	return &kabanerov1alpha2.Kabanero{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kabanero.io/v1alpha2",
			Kind:       "Kabanero",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "kabanero",
			UID:       "12345",
		},
		Spec: kabanerov1alpha2.KabaneroSpec{
			Stacks: kabanerov1alpha2.InstanceStackConfig{
				Repositories: []kabanerov1alpha2.RepositoryConfig{
					kabanerov1alpha2.RepositoryConfig{
						Name:  "default",
						Https: kabanerov1alpha2.HttpsProtocolFile{Url: repositoryUrl},
					},
				},
			},
		},
	}
}

// Test that we can read a legacy CollectionHub that contains embedded
// pipeline and image data.
func TestReconcileFeaturedStacks(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackIndexHandler{})
	defer server.Close()

	ctx := context.Background()
	cl := unitTestClient{make(map[string]*kabanerov1alpha2.Stack)}
	stackUrl := server.URL + defaultIndexName
	k := createKabanero(stackUrl)

	err := reconcileFeaturedStacks(ctx, k, cl, featuredTestLogger)
	if err != nil {
		t.Fatal(err)
	}

	// Should have been two stacks created
	javaMicroprofileStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile"}, javaMicroprofileStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	nodejsStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "nodejs"}, nodejsStack)
	if err != nil {
		t.Fatal("Could not resolve the nodejs stack", err)
	}

	// Make sure the stack has an owner set
	if len(nodejsStack.OwnerReferences) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 owner, but found %v: %v", len(nodejsStack.OwnerReferences), nodejsStack))
	}

	if nodejsStack.OwnerReferences[0].UID != k.UID {
		t.Fatal(fmt.Sprintf("Expected owner UID to be %v, but was %v", k.UID, nodejsStack.OwnerReferences[0].UID))
	}

	// Make sure the stack is active
	if len(nodejsStack.Spec.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 stack version, but found %v: %v", len(nodejsStack.Spec.Versions), nodejsStack.Spec.Versions))
	}

	if nodejsStack.Spec.Versions[0].Version != "0.2.6" {
		t.Fatal(fmt.Sprintf("Expected nodejs stack version \"0.2.6\", but found %v", nodejsStack.Spec.Versions[0].Version))
	}

	if len(nodejsStack.Spec.Versions[0].DesiredState) != 0 {
		t.Fatal(fmt.Sprintf("Expected nodejs stack desiredState to be empty, but was %v", nodejsStack.Spec.Versions[0].DesiredState))
	}

	if len(nodejsStack.Spec.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Expected nodejs stack to have 1 pipeline zip, but had %v: %v", len(nodejsStack.Spec.Versions[0].Pipelines), nodejsStack.Spec.Versions[0].Pipelines))
	}

	if nodejsStack.Spec.Versions[0].Pipelines[0].Https.Url != defaultIndexPipeline {
		t.Fatal(fmt.Sprintf("Expected nodejs stack pipeline zip name to be %v, but was %v", defaultIndexPipeline, nodejsStack.Spec.Versions[0].Pipelines[0].Https.Url))
	}

	if len(nodejsStack.Spec.Versions[0].Images) != 1 {
		t.Fatal(fmt.Sprintf("Expected nodejs stack to have one image, but has %v", len(nodejsStack.Spec.Versions[0].Images)))
	}

	njsExpectedImage := "kabanero/nodejs"
	if nodejsStack.Spec.Versions[0].Images[0].Image != njsExpectedImage {
		t.Fatal(fmt.Sprintf("Expected nodejs stack image of %v, but was %v", njsExpectedImage, nodejsStack.Spec.Versions[0].Images[0].Image))
	}

	jmpExpectedImage := "kabanero/java-microprofile"
	if javaMicroprofileStack.Spec.Versions[0].Images[0].Image != jmpExpectedImage {
		t.Fatal(fmt.Sprintf("Expected nodejs stack image of %v, but was %v", jmpExpectedImage, javaMicroprofileStack.Spec.Versions[0].Images[0].Image))
	}
}

func TestReconcileFeaturedStacksTwoRepositories(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackIndexHandler{})
	defer server.Close()

	ctx := context.Background()
	cl := unitTestClient{make(map[string]*kabanerov1alpha2.Stack)}
	stackUrl := server.URL + defaultIndexName
	stackUrlTwo := server.URL + secondIndexName
	k := createKabanero(stackUrl)
	k.Spec.Stacks.Repositories = append(k.Spec.Stacks.Repositories, kabanerov1alpha2.RepositoryConfig{Name: "two", Https: kabanerov1alpha2.HttpsProtocolFile{Url: stackUrlTwo}})

	err := reconcileFeaturedStacks(ctx, k, cl, featuredTestLogger)
	if err != nil {
		t.Fatal(err)
	}

	// Should have been two stacks created
	javaMicroprofileStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile"}, javaMicroprofileStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	nodejsStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "nodejs"}, nodejsStack)
	if err != nil {
		t.Fatal("Could not resolve the nodejs stack", err)
	}

	// Make sure the stack has an owner set
	if len(nodejsStack.OwnerReferences) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 owner, but found %v: %v", len(nodejsStack.OwnerReferences), nodejsStack))
	}

	if nodejsStack.OwnerReferences[0].UID != k.UID {
		t.Fatal(fmt.Sprintf("Expected owner UID to be %v, but was %v", k.UID, nodejsStack.OwnerReferences[0].UID))
	}

	// Make sure the stack is in the correct state
	if len(nodejsStack.Spec.Versions) != 2 {
		t.Fatal(fmt.Sprintf("Expected 2 stack versions, but found %v: %v", len(nodejsStack.Spec.Versions), nodejsStack.Spec.Versions))
	}

	foundVersions := make(map[string]bool)
	for _, cur := range nodejsStack.Spec.Versions {
		foundVersions[cur.Version] = true
		if len(cur.Pipelines) != 1 {
			t.Fatal(fmt.Sprintf("Expected version %v to have 1 pipeline zip, but has %v: %v", cur.Version, len(cur.Pipelines), cur.Pipelines))
		}
		if len(cur.DesiredState) != 0 {
			t.Fatal(fmt.Sprintf("Expected version %v desiredState to be empty, but was %v", cur.Version, cur.DesiredState))
		}
		if cur.Version == "0.2.6" {
			if cur.Pipelines[0].Https.Url != defaultIndexPipeline {
				t.Fatal(fmt.Sprintf("Expected version \"0.2.6\" pipeline URL to be %v, but was %v", defaultIndexPipeline, cur.Pipelines[0].Https.Url))
			}
		} else if cur.Version == "0.4.1" {
			if cur.Pipelines[0].Https.Url != secondIndexPipeline {
				t.Fatal(fmt.Sprintf("Expected version \"0.4.1\" pipeline URL to be %v, but was %v", secondIndexPipeline, cur.Pipelines[0].Https.Url))
			}
		} else {
			t.Fatal(fmt.Sprintf("Found unexpected version %v", cur.Version))
		}
	}

	if foundVersions["0.2.6"] != true {
		t.Fatal("Did not find stack version \"0.2.6\"")
	}

	if foundVersions["0.4.1"] != true {
		t.Fatal("Did not find stack version \"0.4.1\"")
	}
}

// Read an appsody index and specify custom pipelines in the Kabanero CR instance.
func TestReconcileAppsodyStacksCustomPipelines(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackIndexHandler{})
	defer server.Close()

	ctx := context.Background()
	cl := unitTestClient{make(map[string]*kabanerov1alpha2.Stack)}
	stackUrl := server.URL + appsodyIndexName
	k := createKabanero(stackUrl)

	// Need to specify the pipelines information
	pipelineUrl := kabanerov1alpha2.HttpsProtocolFile{Url: defaultIndexPipeline}
	k.Spec.Stacks.Pipelines = append(k.Spec.Stacks.Pipelines, kabanerov1alpha2.PipelineSpec{Id: "default", Sha256: defaultIndexPipelineDigest, Https: pipelineUrl})

	customPipelineUrl := kabanerov1alpha2.HttpsProtocolFile{Url: secondIndexPipeline}
	k.Spec.Stacks.Repositories[0].Pipelines = append(k.Spec.Stacks.Repositories[0].Pipelines, kabanerov1alpha2.PipelineSpec{Id: "custom", Sha256: secondIndexPipelineDigest, Https: customPipelineUrl})

	err := reconcileFeaturedStacks(ctx, k, cl, featuredTestLogger)
	if err != nil {
		t.Fatal(err)
	}

	// Should have been two stacks created
	javaMicroprofileStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile"}, javaMicroprofileStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	nodejsStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "nodejs"}, nodejsStack)
	if err != nil {
		t.Fatal("Could not resolve the nodejs stack", err)
	}

	// Make sure the stack has an owner set
	if len(nodejsStack.OwnerReferences) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 owner, but found %v: %v", len(nodejsStack.OwnerReferences), nodejsStack))
	}

	if nodejsStack.OwnerReferences[0].UID != k.UID {
		t.Fatal(fmt.Sprintf("Expected owner UID to be %v, but was %v", k.UID, nodejsStack.OwnerReferences[0].UID))
	}

	// Make sure the stack is active
	if len(nodejsStack.Spec.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 stack version, but found %v: %v", len(nodejsStack.Spec.Versions), nodejsStack.Spec.Versions))
	}

	if nodejsStack.Spec.Versions[0].Version != "0.3.2" {
		t.Fatal(fmt.Sprintf("Expected nodejs stack version \"0.3.2\", but found %v", nodejsStack.Spec.Versions[0].Version))
	}

	if len(nodejsStack.Spec.Versions[0].DesiredState) != 0 {
		t.Fatal(fmt.Sprintf("Expected nodejs stack desiredState to be empty, but was %v", nodejsStack.Spec.Versions[0].DesiredState))
	}

	if len(nodejsStack.Spec.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Expected nodejs stack to have 1 pipeline zip, but had %v: %v", len(nodejsStack.Spec.Versions[0].Pipelines), nodejsStack.Spec.Versions[0].Pipelines))
	}

	if nodejsStack.Spec.Versions[0].Pipelines[0].Https.Url != secondIndexPipeline {
		t.Fatal(fmt.Sprintf("Expected nodejs stack pipeline zip name to be %v, but was %v", secondIndexPipeline, nodejsStack.Spec.Versions[0].Pipelines[0].Https.Url))
	}

	if nodejsStack.Spec.Versions[0].Pipelines[0].Sha256 != secondIndexPipelineDigest {
		t.Fatal(fmt.Sprintf("Expected nodejs stack pipeline zip name to be %v, but was %v", secondIndexPipelineDigest, nodejsStack.Spec.Versions[0].Pipelines[0].Sha256))
	}
}

// Read an appsody index and specify the pipelines in the Kabanero CR instance.
func TestReconcileAppsodyStacksDefaultPipelines(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackIndexHandler{})
	defer server.Close()

	ctx := context.Background()
	cl := unitTestClient{make(map[string]*kabanerov1alpha2.Stack)}
	stackUrl := server.URL + appsodyIndexName
	k := createKabanero(stackUrl)

	// Need to specify the pipelines information
	pipelineUrl := kabanerov1alpha2.HttpsProtocolFile{Url: defaultIndexPipeline}
	k.Spec.Stacks.Pipelines = append(k.Spec.Stacks.Pipelines, kabanerov1alpha2.PipelineSpec{Id: "default", Sha256: defaultIndexPipelineDigest, Https: pipelineUrl})

	err := reconcileFeaturedStacks(ctx, k, cl, featuredTestLogger)
	if err != nil {
		t.Fatal(err)
	}

	// Should have been two stacks created
	javaMicroprofileStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile"}, javaMicroprofileStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	nodejsStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "nodejs"}, nodejsStack)
	if err != nil {
		t.Fatal("Could not resolve the nodejs stack", err)
	}

	// Make sure the stack has an owner set
	if len(nodejsStack.OwnerReferences) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 owner, but found %v: %v", len(nodejsStack.OwnerReferences), nodejsStack))
	}

	if nodejsStack.OwnerReferences[0].UID != k.UID {
		t.Fatal(fmt.Sprintf("Expected owner UID to be %v, but was %v", k.UID, nodejsStack.OwnerReferences[0].UID))
	}

	// Make sure the stack is active
	if len(nodejsStack.Spec.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 stack version, but found %v: %v", len(nodejsStack.Spec.Versions), nodejsStack.Spec.Versions))
	}

	if nodejsStack.Spec.Versions[0].Version != "0.3.2" {
		t.Fatal(fmt.Sprintf("Expected nodejs stack version \"0.3.2\", but found %v", nodejsStack.Spec.Versions[0].Version))
	}

	if len(nodejsStack.Spec.Versions[0].DesiredState) != 0 {
		t.Fatal(fmt.Sprintf("Expected nodejs stack desiredState to be empty, but was %v", nodejsStack.Spec.Versions[0].DesiredState))
	}

	if len(nodejsStack.Spec.Versions[0].Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Expected nodejs stack to have 1 pipeline zip, but had %v: %v", len(nodejsStack.Spec.Versions[0].Pipelines), nodejsStack.Spec.Versions[0].Pipelines))
	}

	if nodejsStack.Spec.Versions[0].Pipelines[0].Https.Url != defaultIndexPipeline {
		t.Fatal(fmt.Sprintf("Expected nodejs stack pipeline zip name to be %v, but was %v", defaultIndexPipeline, nodejsStack.Spec.Versions[0].Pipelines[0].Https.Url))
	}

	if nodejsStack.Spec.Versions[0].Pipelines[0].Sha256 != defaultIndexPipelineDigest {
		t.Fatal(fmt.Sprintf("Expected nodejs stack pipeline zip name to be %v, but was %v", defaultIndexPipelineDigest, nodejsStack.Spec.Versions[0].Pipelines[0].Sha256))
	}
}

// Attempts to resolve the featured stacks from the default repository
func TestResolveFeaturedStacks(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackIndexHandler{})
	defer server.Close()

	stack_index_url := server.URL + defaultIndexName
	k := createKabanero(stack_index_url)

	stacks, err := featuredStacks(k, nil, featuredTestLogger)
	if err != nil {
		t.Fatal("Could not resolve the featured stacks from the default index", err)
	}

	// Should be two stacks
	if len(stacks) != 2 {
		t.Fatal(fmt.Sprintf("Was expecting 2 stacks to be found, but found %v: %v", len(stacks), stacks))
	}

	javaMicroprofileStackVersions, ok := stacks["java-microprofile"]
	if !ok {
		t.Fatal(fmt.Sprintf("Could not find java-microprofile stack: %v", stacks))
	}

	nodejsStackVersions, ok := stacks["nodejs"]
	if !ok {
		t.Fatal(fmt.Sprintf("Could not find nodejs stack: %v", stacks))
	}

	// Make sure each stack has one version
	if len(javaMicroprofileStackVersions) != 1 {
		t.Fatal(fmt.Sprintf("Expected one version of java-microprofile stack, but found %v: %v", len(javaMicroprofileStackVersions), javaMicroprofileStackVersions))
	}

	if len(nodejsStackVersions) != 1 {
		t.Fatal(fmt.Sprintf("Expected one version of nodejs stack, but found %v: %v", len(nodejsStackVersions), nodejsStackVersions))
	}
}

// Attempts to resolve the featured stacks from two repositories
func TestResolveFeaturedStacksTwoRepositories(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackIndexHandler{})
	defer server.Close()

	stack_index_url := server.URL + defaultIndexName
	stack_index_url_two := server.URL + secondIndexName
	k := createKabanero(stack_index_url)
	k.Spec.Stacks.Repositories = append(k.Spec.Stacks.Repositories, kabanerov1alpha2.RepositoryConfig{Name: "two", Https: kabanerov1alpha2.HttpsProtocolFile{Url: stack_index_url_two}})

	stacks, err := featuredStacks(k, nil, featuredTestLogger)
	if err != nil {
		t.Fatal("Could not resolve the featured stacks from the default index", err)
	}

	// Should be two stacks
	if len(stacks) != 2 {
		t.Fatal(fmt.Sprintf("Was expecting 2 stacks to be found, but found %v: %v", len(stacks), stacks))
	}

	javaMicroprofileStackVersions, ok := stacks["java-microprofile"]
	if !ok {
		t.Fatal(fmt.Sprintf("Could not find java-microprofile stack: %v", stacks))
	}

	nodejsStackVersions, ok := stacks["nodejs"]
	if !ok {
		t.Fatal(fmt.Sprintf("Could not find nodejs stack: %v", stacks))
	}

	// Make sure each stack has two versions
	if len(javaMicroprofileStackVersions) != 2 {
		t.Fatal(fmt.Sprintf("Expected two versions of java-microprofile stack, but found %v: %v", len(javaMicroprofileStackVersions), javaMicroprofileStackVersions))
	}

	if len(nodejsStackVersions) != 2 {
		t.Fatal(fmt.Sprintf("Expected two versions of nodejs stack, but found %v: %v", len(nodejsStackVersions), nodejsStackVersions))
	}
}

// Tests that if an existing stack version has desired state defined (any allowed string), it should not be deleted or modified.
// Tests that if an existing stack version has no desired state defined and it matches the version in the index, the existing
// stack's values are overriden by the ones in the index.
func TestResolveFeaturedStacksCleanup1(t *testing.T) {
	stack := stackResource.DeepCopy()
	stack.Spec.Versions[0].DesiredState = "inactive"
	stack.Spec.Versions[1].DesiredState = kabanerov1alpha2.StackDesiredStateActive

	deployedStacks := make(map[string]*kabanerov1alpha2.Stack)
	deployedStacks[stack.Name] = stack
	cl := unitTestClient{deployedStacks}

	server := httptest.NewServer(stackIndexHandler{})
	defer server.Close()
	stackUrl := server.URL + defaultIndexName
	k := createKabanero(stackUrl)

	ctx := context.Background()
	err := reconcileFeaturedStacks(ctx, k, cl, featuredTestLogger)
	if err != nil {
		t.Fatal(err)
	}

	// Two stacks should have been created.
	javaMicroprofileStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile"}, javaMicroprofileStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	nodejsStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "nodejs"}, nodejsStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	// Three nodejs versions are expected to be available.
	if len(nodejsStack.Spec.Versions) != 3 {
		t.Fatal(fmt.Sprintf("The nodejs stack did not have the number of expected versions: 3. It has: %v. Stack: %v", len(nodejsStack.Spec.Versions), nodejsStack))
	}

	// Iterate and validate that the expected versions and content match what we expect them to be.
	for _, njVersion := range nodejsStack.Spec.Versions {
		// Existing version 0.2.6 matches what is in the index. The content should have been overriden with what is in the index because
		// the existing stack version did not have its desired state set.
		if njVersion.Version == "0.2.6" {
			if njVersion.Pipelines[0].Https.Url != "https://github.com/kabanero-io/collections/releases/download/0.4.0/incubator.common.pipeline.default.tar.gz" {
				t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.6 should have been updated. Stack version: %v", njVersion))
			}
		}

		// Existing version 0.2.5 does exist in the new index. However it's desired state is set (active); therefore, this version must remain unchanged.
		if njVersion.Version == "0.2.5" {
			if njVersion.Pipelines[0].Https.Url != "https://pipelines/default/0.2.5" {
				t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.5 did not contain the expected Url. Url found: %v. Stack version: %v", njVersion.Pipelines[0].Https.Url, njVersion))
			}
			if njVersion.DesiredState != "active" {
				t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.5 did not contain the expected desired state of active. Desired state found: %v. Stack version: %v", njVersion.DesiredState, njVersion))
			}
		}

		// Existing version 0.2.4 should should not have been removed or modified. It defines a non-empty desired state, which under the new definition of a desired state, it
		// is equivalent to saying do not delete/modify the resource.
		if njVersion.Version == "0.2.4" {
			if njVersion.Pipelines[0].Https.Url != "https://pipelines/default/0.2.4" {
				t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.4 did not contain the expected Url. Url found: %v. Stack version: %v", njVersion.Pipelines[0].Https.Url, njVersion))
			}
			if njVersion.DesiredState != "inactive" {
				t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.4 did not contain the expected desired state of inactive. Desired state found: %v. Stack version: %v", njVersion.DesiredState, njVersion))
			}
		}
	}
}

// Tests that if an existing stack version with a set desired state matches the name/version of a stack in the index, the
// data associated with the index version is ignored.
// Tests that if an existing stack version with a set desired state (any string), the existing stack is not deleted/changed;
// even though, the existing/index stack versions do not match.
// Tests that a stack versions with an unset desired states are removed if they are not found in the index being deployed.
func TestResolveFeaturedStacksCleanup2(t *testing.T) {
	stack := stackResource.DeepCopy()
	stack.Spec.Versions[0].DesiredState = kabanerov1alpha2.StackDesiredStateActive
	stack.Spec.Versions[1].DesiredState = ""
	stack.Spec.Versions[2].DesiredState = kabanerov1alpha2.StackDesiredStateActive

	deployedStacks := make(map[string]*kabanerov1alpha2.Stack)
	deployedStacks[stack.Name] = stack
	cl := unitTestClient{deployedStacks}

	server := httptest.NewServer(stackIndexHandler{})
	defer server.Close()
	stackUrl := server.URL + defaultIndexName
	k := createKabanero(stackUrl)

	ctx := context.Background()
	err := reconcileFeaturedStacks(ctx, k, cl, featuredTestLogger)
	if err != nil {
		t.Fatal(err)
	}

	// Two stacks should have been created.
	javaMicroprofileStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile"}, javaMicroprofileStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	nodejsStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "nodejs"}, nodejsStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	// Only one nodejs versions are expected to be available.
	if len(nodejsStack.Spec.Versions) != 2 {
		t.Fatal(fmt.Sprintf("The nodejs stack did not have the number of expected versions: 2. It has: %v. Stack: %v", len(nodejsStack.Spec.Versions), nodejsStack))
	}

	// Iterate and validate that the expected versions and content match what we expect them to be.
	for _, njVersion := range nodejsStack.Spec.Versions {
		// Existing version 0.2.4 does not what is in the index; however, the existing version has a set desired state (active).
		// This means that the existing version 0.2.4 should be kept.
		if njVersion.Version == "0.2.4" {
			if njVersion.Pipelines[0].Https.Url != "https://pipelines/default/0.2.4" {
				t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.4 should not have been updated. Stack version: %v", njVersion))
			}
			if njVersion.DesiredState != "active" {
				t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.4 did not contain the expected desired state of active. Desired state found: %v. Stack version: %v", njVersion.DesiredState, njVersion))
			}
		}

		// Existing version 0.2.5 should have been deleted because it is not in the new index and its current desired state was not set.
		if njVersion.Version == "0.2.5" {
			t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.5 should have been deleted. Stack: %v", nodejsStack))
		}

		// Existing version 0.2.6 matches what is in the index; however, the existing version has a set desired state (active).
		// This means that the existing 0.2.6 values should not be overriden by the contents of the index.
		if njVersion.Version == "0.2.6" {
			if njVersion.Pipelines[0].Https.Url != "https://pipelines/default/0.2.6" {
				t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.6 should not have been updated. Stack version: %v", njVersion))
			}
			if njVersion.DesiredState != "active" {
				t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.6 did not contain the expected desired state of active. Desired state found: %v. Stack version: %v", njVersion.DesiredState, njVersion))
			}
		}
	}
}

// Tests that an existing stack is not deleted if the index does not have a matching stack and there is at least one existing stack version that defines
// a desired state. Furthermore, any other versions of the existing stack that do not define a desired state, should be deleted.
func TestResolveFeaturedStacksCleanup3(t *testing.T) {
	stack := stackResource.DeepCopy()
	stack.Spec.Name = "cleanuptest"
	stack.ObjectMeta.Name = "cleanuptest"
	stack.Spec.Versions[1].DesiredState = kabanerov1alpha2.StackDesiredStateActive

	deployedStacks := make(map[string]*kabanerov1alpha2.Stack)
	deployedStacks[stack.Name] = stack
	cl := unitTestClient{deployedStacks}

	server := httptest.NewServer(stackIndexHandler{})
	defer server.Close()
	stackUrl := server.URL + defaultIndexName
	k := createKabanero(stackUrl)

	ctx := context.Background()
	err := reconcileFeaturedStacks(ctx, k, cl, featuredTestLogger)
	if err != nil {
		t.Fatal(err)
	}

	// Three stacks should have been deployed. nodejs and java-microprofile were defined in index
	// and teststack was a pre-existing stack with one version set to active.
	javaMicroprofileStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile"}, javaMicroprofileStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	// Only one nodejs version is expected to be available.
	if len(javaMicroprofileStack.Spec.Versions) != 1 {
		t.Fatal(fmt.Sprintf("The java-microprofile stack did not have the number of expected versions: 1. It has: %v. Stack: %v", len(javaMicroprofileStack.Spec.Versions), javaMicroprofileStack))
	}

	// Iterate and validate that the expected versions and content match what we expect them to be.
	for _, jmVersion := range javaMicroprofileStack.Spec.Versions {
		// Existing version 0.2.19 matches what is in the index; however, the existing version has a set desired state (active).
		// This means that the existing values should not be overriden by the contents of the index.
		if jmVersion.Version == "0.2.19" {
			if jmVersion.Pipelines[0].Https.Url != "https://github.com/kabanero-io/collections/releases/download/0.4.0/incubator.common.pipeline.default.tar.gz" {
				t.Fatal(fmt.Sprintf("java-microprofile stack version 0.2.19 should not have been updated. Stack version: %v", jmVersion))
			}
		}
	}

	nodejsStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "nodejs"}, nodejsStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	// Only one nodejs version is expected to be available.
	if len(nodejsStack.Spec.Versions) != 1 {
		t.Fatal(fmt.Sprintf("The nodejs stack did not have the number of expected versions: 1. It has: %v. Stack: %v", len(nodejsStack.Spec.Versions), nodejsStack))
	}

	// Iterate and validate that the expected versions and content match what we expect them to be.
	for _, njVersion := range nodejsStack.Spec.Versions {
		// Existing version 0.2.6 matches what is in the index; however, the existing version has a set desired state (active).
		// This means that the existing values should not be overriden by the contents of the index.
		if njVersion.Version == "0.2.6" {
			if njVersion.Pipelines[0].Https.Url != "https://github.com/kabanero-io/collections/releases/download/0.4.0/incubator.common.pipeline.default.tar.gz" {
				t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.6 should not have been updated. Stack version: %v", njVersion))
			}
		}
	}

	cleanuptestStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "cleanuptest"}, cleanuptestStack)
	if err != nil {
		t.Fatal("Could not resolve the cleanuptest stack", err)
	}

	// Only one nodejs version is expected to be available.
	if len(cleanuptestStack.Spec.Versions) != 1 {
		t.Fatal(fmt.Sprintf("The cleanuptest stack did not have the number of expected versions: 1. It has: %v. Stack: %v", len(cleanuptestStack.Spec.Versions), cleanuptestStack))
	}

	// Iterate and validate that the expected versions and content match what we expect them to be.
	for _, ctVersion := range cleanuptestStack.Spec.Versions {
		// Existing version 0.2.4 should have been deleted because it is not in the new index and its current desired state is not set.
		if ctVersion.Version == "0.2.4" {
			t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.4 should have been deleted. Stack: %v", cleanuptestStack))
		}

		// Existing version 0.2.6 should have been deleted because it is not in the new index and its current desired state is not set.
		if ctVersion.Version == "0.2.6" {
			t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.6 should have been deleted. Stack: %v", cleanuptestStack))
		}

		// Existing version 0.2.5 matches what is in the index; however, the existing version has a desired state set (active).
		// This means that the existing values should not be overriden by the contents of the index.
		if ctVersion.Version == "0.2.5" {
			if ctVersion.Pipelines[0].Https.Url != "https://pipelines/default/0.2.5" {
				t.Fatal(fmt.Sprintf("Cleanuptest stack version 0.2.6 should not have been updated. Stack version: %v", ctVersion))
			}
			if ctVersion.DesiredState != "active" {
				t.Fatal(fmt.Sprintf("Cleanuptest stack version 0.2.6 did not contain the expected desired state of active. Desired state found: %v. Stack version: %v", ctVersion.DesiredState, ctVersion))
			}
		}
	}
}

// Tests that an existing stack is deleted if the index does not have a matching stack and the existing stack's versions
// do not define a desired state.
func TestResolveFeaturedStacksCleanup4(t *testing.T) {
	stack := stackResource.DeepCopy()
	stack.Spec.Name = "cleanuptest"
	stack.ObjectMeta.Name = "cleanuptest"

	deployedStacks := make(map[string]*kabanerov1alpha2.Stack)
	deployedStacks[stack.Name] = stack
	cl := unitTestClient{deployedStacks}

	server := httptest.NewServer(stackIndexHandler{})
	defer server.Close()
	stackUrl := server.URL + defaultIndexName
	k := createKabanero(stackUrl)

	ctx := context.Background()
	err := reconcileFeaturedStacks(ctx, k, cl, featuredTestLogger)
	if err != nil {
		t.Fatal(err)
	}

	// Two stacks should have been deployed from the index: nodejs and java-microprofile.
	// The cleanuptest stack should have been deleted.
	javaMicroprofileStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile"}, javaMicroprofileStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	// Only one nodejs version is expected to be available.
	if len(javaMicroprofileStack.Spec.Versions) != 1 {
		t.Fatal(fmt.Sprintf("The java-microprofile stack did not have the number of expected versions: 1. It has: %v. Stack: %v", len(javaMicroprofileStack.Spec.Versions), javaMicroprofileStack))
	}

	// Iterate and validate that the expected versions and content match what we expect them to be.
	for _, jmVersion := range javaMicroprofileStack.Spec.Versions {
		// Existing version 0.2.19 matches what is in the index; however, the existing version has a set desired state (active).
		// This means that the existing values should not be overriden by the contents of the index.
		if jmVersion.Version == "0.2.19" {
			if jmVersion.Pipelines[0].Https.Url != "https://github.com/kabanero-io/collections/releases/download/0.4.0/incubator.common.pipeline.default.tar.gz" {
				t.Fatal(fmt.Sprintf("java-microprofile stack version 0.2.19 should not have been updated. Stack version: %v", jmVersion))
			}
		}
	}

	nodejsStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "nodejs"}, nodejsStack)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile stack", err)
	}

	// Only one nodejs version is expected to be available.
	if len(nodejsStack.Spec.Versions) != 1 {
		t.Fatal(fmt.Sprintf("The nodejsStack stack did not have the number of expected versions: 1. It has: %v. Stack: %v", len(nodejsStack.Spec.Versions), nodejsStack))
	}

	// Iterate and validate that the expected versions and content match what we expect them to be.
	for _, njVersion := range nodejsStack.Spec.Versions {
		// Existing version 0.2.6 matches what is in the index; however, the existing version has a set desired state (active).
		// This means that the existing values should not be overriden by the contents of the index.
		if njVersion.Version == "0.2.6" {
			if njVersion.Pipelines[0].Https.Url != "https://github.com/kabanero-io/collections/releases/download/0.4.0/incubator.common.pipeline.default.tar.gz" {
				t.Fatal(fmt.Sprintf("Nodejs stack version 0.2.6 should not have been updated. Stack version: %v", njVersion))
			}
		}
	}

	cleanuptestStack := &kabanerov1alpha2.Stack{}
	err = cl.Get(ctx, types.NamespacedName{Name: "cleanuptest"}, cleanuptestStack)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			t.Fatal("Could not resolve the cleanuptest stack", err)
		}
	}
}
