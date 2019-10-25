package config

import (
	"testing"
)

func TestAssets(t *testing.T) {
	path := "orchestrations/cli-services/0.1/kabanero-cli.yaml"

	if assets == nil {
		t.Fatal("Assets were not initialized")
	}
	f, err := Open(path)
	if err != nil {
		t.Fatalf("Unexpected error reading %v: %v", path, err)
	}
	if f == nil {
		t.Fatal("Could not open the file")
	}
	f.Close()
}
