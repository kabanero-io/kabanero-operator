package stack

import (
	"testing"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
)

func TestResolveIndex(t *testing.T) {
	repoConfig := kabanerov1alpha2.RepositoryConfig{
		Name:                       "name",
		Url:                        "https://github.com/kabanero-io/stacks/releases/download/v0.0.1/incubator-index.yaml",
		SkipCertVerification: true,
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

func TestSearchStack(t *testing.T) {
	index := &Index{
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
