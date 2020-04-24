package stack

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Unit test client.
type archiveTestClient struct {
}

func (c archiveTestClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	return errors.New("Get is not implemented")
}
func (c archiveTestClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	return errors.New("List is not implemented")
}
func (c archiveTestClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	return errors.New("Create is not implemented")
}
func (c archiveTestClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	return errors.New("Delete is not implemented")
}
func (c archiveTestClient) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	return errors.New("DeleteAllOf is not implemented")
}
func (c archiveTestClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	return errors.New("Update is not implemented")
}
func (c archiveTestClient) Status() client.StatusWriter { return c }
func (c archiveTestClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return errors.New("Patch is not implemented")
}

func TestGetManifests(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	reqLogger := logf.NullLogger{}
	pipelineStatus := kabanerov1alpha2.PipelineStatus{
		Url:        server.URL + basicPipeline.name,
		Digest:     basicPipeline.sha256,
		GitRelease: kabanerov1alpha2.GitReleaseInfo{}}

	manifests, err := GetManifests(archiveTestClient{}, "kabanero", pipelineStatus, map[string]interface{}{"StackName": "Eclipse Microprofile", "StackId": "java-microprofile"}, true, reqLogger)

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
		GitRelease: kabanerov1alpha2.GitReleaseInfo{}}

	manifests, err := GetManifests(archiveTestClient{}, "kabanero", pipelineStatus, map[string]interface{}{"StackName": "Eclipse Microprofile", "StackId": "java-microprofile"}, true, reqLogger)

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
		Url:        server.URL + "/good-pipeline.yaml",
		Digest:     "3b34de594df82cac3cb67c556a416443f6fafc0bc79101613eaa7ae0d59dd462",
		GitRelease: kabanerov1alpha2.GitReleaseInfo{}}

	manifests, err := GetManifests(archiveTestClient{}, "kabanero", pipelineStatus, map[string]interface{}{"StackName": "Eclipse Microprofile", "StackId": "java-microprofile"}, true, reqLogger)

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
