package collection

import (
	"testing"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
)

func TestResolveIndex(t *testing.T) {
	repoConfig := kabanerov1alpha1.RepositoryConfig{
			Name: "name",
			Url: "https://raw.githubusercontent.com/kabanero-io/kabanero-collection/master/experimental",
			ActivateDefaultCollections: true,
	}

	index, err := ResolveIndex(repoConfig)
	if err != nil {
		t.Fatal(err)
	}

	if index == nil {
		t.Fatal("Returned index was nil")
	}

	if index.ApiVersion != "v1" {
		t.Fatal("Expected apiVersion == v1")
	}
}

func TestResolveIndexWithSkipCertVerify(t *testing.T) {
	repoConfig := kabanerov1alpha1.RepositoryConfig{
			Name: "name",
			Url: "https://raw.githubusercontent.com/kabanero-io/kabanero-collection/master/experimental",
			ActivateDefaultCollections: true,
			SkipCertVerification: true,
	}

	index, err := ResolveIndex(repoConfig)
	if err != nil {
		t.Fatal(err)
	}

	if index == nil {
		t.Fatal("Index not found.")
	}

	if index.ApiVersion != "v1" {
		t.Fatal("Expected apiVersion == v1")
	}
}

func TestResolveIndexV2(t *testing.T) {
	repoConfig := kabanerov1alpha1.RepositoryConfig{
			Name: "name",
			Url: "https://github.com/kabanero-io/collections/releases/download/v0.0.1/incubator-index.yaml",
			ActivateDefaultCollections: true,
        }

	index, err := ResolveIndex(repoConfig)
	if err != nil {
		t.Fatal(err)
	}

	if index == nil {
		t.Fatal("Returned index was nil")
	}

	if index.ApiVersion != "v2" {
		t.Fatal("Expected apiVersion == v2")
	}
}

func TestResolveCollection(t *testing.T) {
	repoConfig := kabanerov1alpha1.RepositoryConfig{
			Name: "name",
			Url: "https://github.com/kabanero-io/collections/releases/download/v0.0.1/incubator-index.yaml",
			ActivateDefaultCollections: true,
	}

	collection, err := ResolveCollection(repoConfig, "https://raw.githubusercontent.com/kabanero-io/kabanero-collection/master/experimental/java-microprofile-0.2.1/collection.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if collection == nil {
		t.Fatal("Collection was nil")
	}

	t.Log(collection)
}
