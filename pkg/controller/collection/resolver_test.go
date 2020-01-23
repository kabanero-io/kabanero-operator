package collection

import (
	"testing"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
)

func TestResolveIndex(t *testing.T) {
	repoConfig := kabanerov1alpha1.RepositoryConfig{
		Name:                       "name",
		Url:                        "https://github.com/kabanero-io/collections/releases/download/v0.0.1/incubator-index.yaml",
		ActivateDefaultCollections: true,
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
	repoConfig := kabanerov1alpha1.RepositoryConfig{
		Name:                       "openLibertyTest",
		Url:                        "https://github.com/appsody/stacks/releases/download/java-openliberty-v0.1.2/incubator-index.yaml",
		ActivateDefaultCollections: true,
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
	numStacks := len(index.Collections)
	if len(index.Collections[numStacks-numStacks].Pipelines) == 0 {
		t.Fatal("Index.Collections[0].Pipelines is empty. An entry was expected")
	}

	c0p0 := index.Collections[numStacks-numStacks].Pipelines[0]
	if c0p0.Id != "testPipeline" {
		t.Fatal("Expected Index.Collections[umStacks-numStacks].Pipelines[0] to have a pipeline name of testPipeline. Instead it was: " + c0p0.Id)
	}

	if len(index.Collections[numStacks-1].Pipelines) == 0 {
		t.Fatal("Index.Collections[numStacks-1].Pipelines is empty. An entry was expected")
	}

	cLastP0 := index.Collections[numStacks-1].Pipelines[0]
	if cLastP0.Id != "testPipeline" {
		t.Fatal("Expected Index.Collections[0].Pipelines[0] to have a pipeline name of testPipeline. Instead it was: " + cLastP0.Id)
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
	if len(index.Collections[0].Images) == 0 {
		t.Fatal("index.Collections[0].Images is empty. An entry was expected")
	}

	image := index.Collections[0].Images[0]
	if len(image.Image) == 0 {
		t.Fatal("Expected index.Collections[0].Images[0].Image to have a non-empty value.")
	}

	if len(image.Id) == 0 {
		t.Fatal("Expected index.Collections[0].Images[0].Id to have a non-empty value.")
	}
}

func TestSearchCollection(t *testing.T) {
	index := &Index{
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
			Collection{
				DefaultImage:    "java-microprofile2",
				DefaultPipeline: "default2",
				DefaultTemplate: "default2",
				Description:     "Test collection 2",
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

	collections, err := SearchCollection("java-microprofile2", index)
	if err != nil {
		t.Fatal(err)
	}

	if len(collections) != 1 {
		t.Fatal("The expected number of collections is 1, but found: ", len(collections))
	}

	t.Log(collections)
}
