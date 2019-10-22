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

	index, err := ResolveIndex(repoConfig)
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
