package stack

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
)

func TestResolveIndex(t *testing.T) {
	repoConfig := kabanerov1alpha2.RepositoryConfig{
		Name: "name",
		Https: kabanerov1alpha2.HttpsProtocolFile{
			Url:                  "https://github.com/kabanero-io/stacks/releases/download/v0.0.1/incubator-index.yaml",
			SkipCertVerification: true,
		},
	}

	index, err := ResolveIndex(repoConfig, []Pipelines{}, []Trigger{}, "")
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
	repoConfig := kabanerov1alpha2.RepositoryConfig{
		Name:  "openLibertyTest",
		Https: kabanerov1alpha2.HttpsProtocolFile{Url: "https://github.com/appsody/stacks/releases/download/java-spring-boot2-v0.3.23/incubator-index.yaml"},
	}

	pipelines := []Pipelines{{Id: "testPipeline", Sha256: "1234567890", Url: "https://github.com/kabanero-io/collections/releases/download/0.5.0-rc.2/incubator.common.pipeline.default.tar.gz"}}
	triggers := []Trigger{{Id: "testTrigger", Sha256: "0987654321", Url: "https://github.com/kabanero-io/collections/releases/download/0.5.0-rc.2/incubator.trigger.tar.gz"}}
	index, err := ResolveIndex(repoConfig, pipelines, triggers, "kabanerobeta")

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

// HTTP handler that serves pipeline zips
type githubHandler struct {
}

func (ch githubHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	filename := fmt.Sprintf("testdata/%v", req.URL.String())
	fmt.Printf("Serving %v\n", filename)
	rw.WriteHeader(http.StatusNotFound)
}

// Tests that an index in a public Git hub repository is able to be read using the information provided
// in the under the gitRelease element of the Kabanero CR instance yaml.
func TestResolveIndexForStacksInPublicGit(t *testing.T) {
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	a, _ := url.Parse(server.URL)
	
	repoConfig := kabanerov1alpha2.RepositoryConfig{
		Name:       "openLibertyTest",
		GitRelease: kabanerov1alpha2.GitReleaseSpec{Hostname: a.Host, Organization: "appsody", Project: "stacks", Release: "java-spring-boot2-v0.3.23", AssetName: "incubator-index.yaml"},
	}

	pipelines := []Pipelines{{Id: "testPipeline", Sha256: "1234567890", Url: "https://github.com/kabanero-io/collections/releases/download/0.5.0-rc.2/incubator.common.pipeline.default.tar.gz"}}
	triggers := []Trigger{{Id: "testTrigger", Sha256: "0987654321", Url: "https://github.com/kabanero-io/collections/releases/download/0.5.0-rc.2/incubator.trigger.tar.gz"}}
	index, err := ResolveIndex(repoConfig, pipelines, triggers, "kabanerobeta")

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
	repoConfig := kabanerov1alpha2.RepositoryConfig{
		Name: "openLibertyTest",
	}

	pipelines := []Pipelines{{Id: "testPipeline", Sha256: "1234567890", Url: "https://github.com/kabanero-io/collections/releases/download/0.5.0-rc.2/incubator.common.pipeline.default.tar.gz"}}
	triggers := []Trigger{{Id: "testTrigger", Sha256: "0987654321", Url: "https://github.com/kabanero-io/collections/releases/download/0.5.0-rc.2/incubator.trigger.tar.gz"}}
	index, err := ResolveIndex(repoConfig, pipelines, triggers, "kabanerobeta")

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
