package stack

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Unit test client.
type resolverTestClient struct {
}

func (c resolverTestClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	return errors.New("Get is not implemented")
}
func (c resolverTestClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	return errors.New("List is not implemented")
}
func (c resolverTestClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	return errors.New("Create is not implemented")
}
func (c resolverTestClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	return errors.New("Delete is not implemented")
}
func (c resolverTestClient) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	return errors.New("DeleteAllOf is not implemented")
}
func (c resolverTestClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	return errors.New("Update is not implemented")
}
func (c resolverTestClient) Status() client.StatusWriter { return c }
func (c resolverTestClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return errors.New("Patch is not implemented")
}

var resolverTestLogger logr.Logger = log.WithValues("Request.Namespace", "test", "Request.Name", "resolver_test")

func TestResolveIndex(t *testing.T) {
	// The server that will host the stack hub index
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	repoConfig := kabanerov1alpha2.RepositoryConfig{
		Name: "name",
		Https: kabanerov1alpha2.HttpsProtocolFile{
			Url:                  server.URL + "/incubator-index-collections.yaml",
			SkipCertVerification: true,
		},
	}

	index, err := ResolveIndex(resolverTestClient{}, repoConfig, "kabanero", []Pipelines{}, []Trigger{}, "", resolverTestLogger)
	if err != nil {
		t.Fatal(err)
	}

	if index == nil {
		t.Fatal("Returned index was nil")
	}

	if index.APIVersion != "v2" {
		t.Fatal("Expected apiVersion == v2")
	}
}

func TestResolveIndexForStacks(t *testing.T) {
	// The server that will host the stack hub index
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	repoConfig := kabanerov1alpha2.RepositoryConfig{
		Name:  "openLibertyTest",
		Https: kabanerov1alpha2.HttpsProtocolFile{Url: server.URL + "/incubator-index.yaml", SkipCertVerification: true},
	}

	pipelines := []Pipelines{{Id: "testPipeline", Sha256: "513090b303ba8711c93ab1e2eacc66769086e0e18fe11a10140aaf6a70c8be78", Url: server.URL + "/0.5.0-rc.2/incubator.common.pipeline.default.tar.gz"}}
	triggers := []Trigger{{Id: "testTrigger", Sha256: "9b11091f295fb6706a8dbca62f57adf26b55d6f35eb0d5b0988129db91d295c0", Url: server.URL + "/0.5.0-rc.2/incubator.trigger.tar.gz"}}
	index, err := ResolveIndex(resolverTestClient{}, repoConfig, "kabanero", pipelines, triggers, "kabanerobeta", resolverTestLogger)

	if err != nil {
		t.Fatal(err)
	}

	if index == nil {
		t.Fatal("The resulting index structure was nil")
	}

	// Validate pipeline entries.
	numStacks := len(index.Stacks)

	if len(index.Stacks[numStacks-numStacks].Pipelines) == 0 {
		t.Fatal("Index.Stacks[0].Pipelines is empty. An entry was expected.")
	}

	c0p0 := index.Stacks[numStacks-numStacks].Pipelines[0]
	if c0p0.Id != "testPipeline" {
		t.Fatal("Expected Index.Stacks[umStacks-numStacks].Pipelines[0] to have a pipeline name of testPipeline. Instead it was: " + c0p0.Id)
	}

	if len(index.Stacks[numStacks-1].Pipelines) == 0 {
		t.Fatal("Index.Stacks[numStacks-1].Pipelines is empty. An entry was expected")
	}

	cLastP0 := index.Stacks[numStacks-1].Pipelines[0]
	if cLastP0.Id != "testPipeline" {
		t.Fatal("Expected Index.Stacks[0].Pipelines[0] to have a pipeline name of testPipeline. Instead it was: " + cLastP0.Id)
	}

	// Validate trigger entry.
	if len(index.Triggers) == 0 {
		t.Fatal("Index.Triggers is empty. An entry was expected")
	}
	trgr := index.Triggers[0]
	if trgr.Id != "testTrigger" {
		t.Fatal("Expected Index.Triggers[0] to have a trigger name of testTrigger. Instead it was: " + trgr.Id)
	}

	// Validate image entry.
	if len(index.Stacks[0].Images) == 0 {
		t.Fatal("index.Stacks[0].Images is empty. An entry was expected")
	}

	image := index.Stacks[0].Images[0]
	if len(image.Image) == 0 {
		t.Fatal("Expected index.Stacks[0].Images[0].Image to have a non-empty value.")
	}

	if len(image.Id) == 0 {
		t.Fatal("Expected index.Stacks[0].Images[0].Id to have a non-empty value.")
	}
}

// Tests that stack index resolution fails if both Git release information Http URL info is not configured in
// the Kabanero CR instance yaml.
func TestResolveIndexForStacksInPublicGitFailure1(t *testing.T) {
	// The server that will host the stack hub index
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	repoConfig := kabanerov1alpha2.RepositoryConfig{
		Name: "openLibertyTest",
	}

	pipelines := []Pipelines{{Id: "testPipeline", Sha256: "513090b303ba8711c93ab1e2eacc66769086e0e18fe11a10140aaf6a70c8be78", Url: server.URL + "/0.5.0-rc.2/incubator.common.pipeline.default.tar.gz"}}
	triggers := []Trigger{{Id: "testTrigger", Sha256: "9b11091f295fb6706a8dbca62f57adf26b55d6f35eb0d5b0988129db91d295c0", Url: server.URL + "/0.5.0-rc.2/incubator.trigger.tar.gz"}}
	index, err := ResolveIndex(resolverTestClient{}, repoConfig, "kabanero", pipelines, triggers, "kabanerobeta", resolverTestLogger)

	if err == nil {
		t.Fatal("No Git release or Http url were specified. An error was expected. Index: ", index)
	}
}
func TestSearchStack(t *testing.T) {
	index := &Index{
		APIVersion: "v2",
		Stacks: []Stack{
			Stack{
				DefaultImage:    "java-microprofile",
				DefaultPipeline: "default",
				DefaultTemplate: "default",
				Description:     "Test stack",
				Id:              "java-microprofile",
				Images: []Images{
					Images{},
				},
				Maintainers: []Maintainers{
					Maintainers{},
				},
				Name: "Eclipse Microprofile",
				Pipelines: []Pipelines{
					Pipelines{},
				},
			},
			Stack{
				DefaultImage:    "java-microprofile2",
				DefaultPipeline: "default2",
				DefaultTemplate: "default2",
				Description:     "Test stack 2",
				Id:              "java-microprofile2",
				Images: []Images{
					Images{},
				},
				Maintainers: []Maintainers{
					Maintainers{},
				},
				Name: "Eclipse Microprofile 2",
				Pipelines: []Pipelines{
					Pipelines{},
				},
			},
		},
	}

	stacks, err := SearchStack("java-microprofile2", index)
	if err != nil {
		t.Fatal(err)
	}

	if len(stacks) != 1 {
		t.Fatal("The expected number of stacks is 1, but found: ", len(stacks))
	}

	t.Log(stacks)
}
