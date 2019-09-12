package collection

import (
	"testing"
)

func TestGetManifests(t *testing.T) {
	manifests, err := GetManifests("https://github.com/kabanero-io/collections/releases/download/v0.0.1/incubator.java-microprofile.pipeline.default.tar.gz", map[string]interface{}{"CollectionName": "Eclipse Microprofile", "CollectionId": "java-microprofile"})
	if err != nil {
		t.Fatal(err)
	}

	for _, manifest := range manifests {
		t.Log(manifest)
	}
}
