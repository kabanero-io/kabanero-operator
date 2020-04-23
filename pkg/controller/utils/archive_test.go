package utils

import (
	"fmt"
	"io/ioutil"
	"testing"
	"net/http"
	"net/http/httptest"
	
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// HTTP handler that serves pipeline zips
type stackHandler struct {
}

func (ch stackHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	filename := fmt.Sprintf("testdata/%v", req.URL.String())
	fmt.Printf("Serving %v\n", filename)
	d, err := ioutil.ReadFile(filename)
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
	} else {
		rw.Write(d)
	}
}

type fileInfo struct {
	name   string
	sha256 string
}

var basicPipeline = fileInfo{
	name:   "/basic.pipeline.tar.gz",
	sha256: "8080076acd8f54ecbb7de132df148d964e5e93921cce983a0f781418b0871573"}

func TestGetManifests(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	reqLogger := logf.NullLogger{}
	pipelineStatus := kabanerov1alpha2.PipelineStatus{
		Url:        server.URL + basicPipeline.name,
		Digest:     basicPipeline.sha256,
		GitRelease: kabanerov1alpha2.GitReleaseSpec{}}

	manifests, err := GetManifests(nil, "kabanero", pipelineStatus, map[string]interface{}{"StackName": "Eclipse Microprofile", "StackId": "java-microprofile"}, reqLogger)
	if err != nil {
		t.Fatal(err)
	}

	for _, manifest := range manifests {
		t.Log(manifest)
	}
}

func TestGetManifestsQuery(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	reqLogger := logf.NullLogger{}
	pipelineStatus := kabanerov1alpha2.PipelineStatus{
		Url:        server.URL + basicPipeline.name,
		Digest:     basicPipeline.sha256,
		GitRelease: kabanerov1alpha2.GitReleaseSpec{}}

	manifests, err := GetManifests(nil, "kabanero", pipelineStatus, map[string]interface{}{"StackName": "Eclipse Microprofile", "StackId": "java-microprofile"}, reqLogger)
	if err != nil {
		t.Fatal(err)
	}

	for _, manifest := range manifests {
		t.Log(manifest)
	}
}

func TestGetManifestsYaml(t *testing.T) {
	// The server that will host the pipeline yaml
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	reqLogger := logf.NullLogger{}
	pipelineStatus := kabanerov1alpha2.PipelineStatus{
		Url: server.URL + "/good-pipeline.yaml",
		Digest: "3b34de594df82cac3cb67c556a416443f6fafc0bc79101613eaa7ae0d59dd462",
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
