package stack

import (
	"fmt"
	"testing"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestGetManifests(t *testing.T) {
	reqLogger := logf.NullLogger{}
	pipelineStatus := kabanerov1alpha2.PipelineStatus{
		Url:        "https://github.com/kabanero-io/stacks/releases/download/v0.0.1/incubator.java-microprofile.pipeline.default.tar.gz",
		Digest:     "8eacd2a6870c2b7c729ae1441cc58d6f1356bde08a022875f9f50bca8fc66543",
		GitRelease: kabanerov1alpha2.GitReleaseSpec{}}

	manifests, err := GetManifests(nil, "kabanero", pipelineStatus, map[string]interface{}{"StackName": "Eclipse Microprofile", "StackId": "java-microprofile"}, reqLogger)
	if err != nil {
		t.Fatal(err)
	}

	for _, manifest := range manifests {
		t.Log(manifest)
	}
}

func TestCommTraceZero(t *testing.T) {
	out := commTrace(nil)
	if out != "" {
		t.Fatal(fmt.Sprintf("Trace of zero bytes should yield an empty string, but got: %v", out))
	}
}

func TestCommTraceSixteen(t *testing.T) {
	buffer := "1234567890123456"
	out := commTrace([]byte(buffer))
	if out != "00000000: 3132 3334 3536 3738 3930 3132 3334 3536 1234567890123456\n" {
		t.Fatal(fmt.Sprintf("Trace of 16 bytes incorrect output: %v", out))
	}
}

func TestCommTraceEight(t *testing.T) {
	buffer := "12345678"
	out := commTrace([]byte(buffer))
	if out != "00000000: 3132 3334 3536 3738                     12345678\n" {
		t.Fatal(fmt.Sprintf("Trace of 8 bytes incorrect output: %v", out))
	}
}

func TestCommTraceNine(t *testing.T) {
	buffer := "123456789"
	out := commTrace([]byte(buffer))
	if out != "00000000: 3132 3334 3536 3738 39                  123456789\n" {
		t.Fatal(fmt.Sprintf("Trace of 9 bytes incorrect output: %v", out))
	}
}

func TestCommTraceThirtyTwo(t *testing.T) {
	buffer := "12345678901234567890123456789012"
	out := commTrace([]byte(buffer))
	if out != "00000000: 3132 3334 3536 3738 3930 3132 3334 3536 1234567890123456\n00000010: 3738 3930 3132 3334 3536 3738 3930 3132 7890123456789012\n" {
		t.Fatal(fmt.Sprintf("Trace of 9 bytes incorrect output: %v", out))
	}
}
