package collection

import (
	"testing"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestGetManifests(t *testing.T) {
	reqLogger := logf.NullLogger{}
	sha256 := "8eacd2a6870c2b7c729ae1441cc58d6f1356bde08a022875f9f50bca8fc66543"
	manifests, err := GetManifests("https://github.com/kabanero-io/collections/releases/download/v0.0.1/incubator.java-microprofile.pipeline.default.tar.gz", sha256, map[string]interface{}{"CollectionName": "Eclipse Microprofile", "CollectionId": "java-microprofile"}, reqLogger)
	if err != nil {
		t.Fatal(err)
	}

	for _, manifest := range manifests {
		t.Log(manifest)
	}
}
