package config

import (
	"testing"
)

func TestAssets(t *testing.T) {
	if assets == nil {
		t.Fatal("Assets were not initialized")
	}
	f, err := Open("reconciler/knative-eventing/knative-eventing.yaml")
	if err != nil {
		t.Fatal("Unexpected error reading olm.yaml")
	}
	if f == nil {
		t.Fatal("Could not open the file")
	}
	f.Close()
}
