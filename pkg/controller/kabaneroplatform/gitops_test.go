package kabaneroplatform

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"testing"
)

// Set up logging so that the log statements in the product code come out in the test output
type testLogger struct{}

func (t testLogger) Info(msg string, keysAndValues ...interface{}) { fmt.Printf("Info: %v \n", msg) }
func (t testLogger) Enabled() bool                                 { return true }
func (t testLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	fmt.Printf("Error: %v: %v\n", msg, err.Error())
}
func (t testLogger) V(level int) logr.InfoLogger                         { return t }
func (t testLogger) WithValues(keysAndValues ...interface{}) logr.Logger { return t }
func (t testLogger) WithName(name string) logr.Logger                    { return t }

func init() {
	logf.SetLogger(testLogger{})
}

var klog = logf.Log.WithName("gitops_test")

// Unit test Kube client
type gitopsTestClient struct {
	// Objects that the client knows about.
	objs map[client.ObjectKey]bool
}

func (c gitopsTestClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	fmt.Printf("Received Get() for %v\n", key.Name)
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid target object for get: %v\n", obj)
		return errors.New("Get only supports setting into Unstructured")
	}
	_, ok = c.objs[key]
	if !ok {
		return apierrors.NewNotFound(schema.GroupResource{}, key.Name)
	}
	u.SetName(key.Name)
	u.SetNamespace(key.Namespace)
	return nil
}
func (c gitopsTestClient) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	return nil
}
func (c gitopsTestClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid create: %v\n", obj)
		return errors.New("Create only supports Unstructured")
	}

	fmt.Printf("Received Create() for %v\n", u.GetName())
	key := client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}
	_, ok = c.objs[key]
	if ok {
		fmt.Printf("Receive create object already exists: %v/%v\n", u.GetNamespace(), u.GetName())
		return apierrors.NewAlreadyExists(schema.GroupResource{}, u.GetName())
	}

	gvk := u.GroupVersionKind()
	if gvk.Kind == "BadTask" {
		message := fmt.Sprintf("Receive create for invalid kind: %v", gvk.Kind)
		fmt.Printf(message + "\n")
		return errors.New(message)
	}

	c.objs[key] = true
	return nil
}
func (c gitopsTestClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid delete: %v\n", obj)
		return errors.New("Delete only supports Unstructured")
	}

	fmt.Printf("Received Delete() for %v\n", u.GetName())
	key := client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}
	_, ok = c.objs[key]
	if !ok {
		fmt.Printf("Received delete for an object that does not exist: %v\n", obj)
		return apierrors.NewNotFound(schema.GroupResource{}, u.GetName())
	}
	delete(c.objs, key)
	return nil
}
func (c gitopsTestClient) DeleteAllOf(ctx context.Context, obj runtime.Object, opts ...client.DeleteAllOfOption) error {
	return errors.New("DeleteAllOf is not supported")
}
func (c gitopsTestClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		fmt.Printf("Received invalid update: %v\n", obj)
		return errors.New("Update only supports Unstructured")
	}

	fmt.Printf("Received Update() for %v\n", u.GetName())
	key := client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}
	_, ok = c.objs[key]
	if !ok {
		fmt.Printf("Received update for object that does not exist: %v\n", obj)
		return apierrors.NewNotFound(schema.GroupResource{}, u.GetName())
	}
	return nil
}
func (c gitopsTestClient) Status() client.StatusWriter { return c }

func (c gitopsTestClient) Patch(ctx context.Context, obj runtime.Object, patch client.Patch, opts ...client.PatchOption) error {
	return errors.New("Patch is not supported")
}

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

var digest1Pipeline = fileInfo{
	name:   "/digest1.pipeline.tar.gz",
	sha256: "0238ff31f191396ca4bf5e0ebeea323d012d5dbc7e3f0997e1bf66b017228aaf"}

// Apply some pipelines and make sure the status reflects what was activated
func TestReconcileGitopsPipelines(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(stackHandler{})
	defer server.Close()

	pipelineZipUrl := server.URL + digest1Pipeline.name

	kabaneroResource := kabanerov1alpha2.Kabanero{
		ObjectMeta: metav1.ObjectMeta{Name: "kabanero", Namespace: "kabanero"},
		Spec: kabanerov1alpha2.KabaneroSpec{
			Gitops: kabanerov1alpha2.GitopsSpec{
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: "default",
					Sha256: digest1Pipeline.sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: pipelineZipUrl, SkipCertVerification: true},
				}},
			},
		},
	}

	client := gitopsTestClient{map[client.ObjectKey]bool{}}
	err := reconcileGitopsPipelines(context.TODO(), &kabaneroResource, client, klog)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the kabanero resource was updated with asset information
	if len(kabaneroResource.Status.Gitops.Pipelines) != 1 {
		t.Fatal(fmt.Sprintf("Kabanero status should have 1 pipeline, but has %v", len(kabaneroResource.Status.Gitops.Pipelines)))
	}

	// Make sure the assets were created in the stack status
	pipeline := kabaneroResource.Status.Gitops.Pipelines[0]
	if len(pipeline.ActiveAssets) != 2 {
		t.Fatal(fmt.Sprintf("Pipeline should have 2 assets, but has %v", len(pipeline.ActiveAssets)))
	}

	for _, asset := range pipeline.ActiveAssets {
		if asset.Status != utils.AssetStatusActive {
			t.Fatal(fmt.Sprintf("Asset %v should have status active, but is %v", asset.Name, asset.Status))
		}
		if asset.StatusMessage != "" {
			t.Fatal(fmt.Sprintf("Asset %v should have no status message, but has %v", asset.Name, asset.StatusMessage))
		}
	}

	if pipeline.Name != kabaneroResource.Spec.Gitops.Pipelines[0].Id {
		t.Fatal(fmt.Sprintf("Pipeline name should be %v, but is %v", kabaneroResource.Spec.Gitops.Pipelines[0].Id, pipeline.Name))
	}

	// Make sure the client has the correct objects.
	if len(client.objs) != 2 {
		t.Fatal(fmt.Sprintf("Client map should have 2 entries, but has %v: %v", len(client.objs), client.objs))
	}

	// Make sure the Gitops message is cleared
	if len(kabaneroResource.Status.Gitops.Message) != 0 {
		t.Fatal(fmt.Sprintf("A message is present in the Gitops status: %v", kabaneroResource.Status.Gitops.Message))
	}

	if kabaneroResource.Status.Gitops.Ready != "True" {
		t.Fatal(fmt.Sprintf("Kabanero Gitops ready status is not \"True\": %v", kabaneroResource.Status.Gitops.Ready))
	}
}

// Make sure we can clean stuff up.
func TestCleanupGitopsPipelines(t *testing.T) {
	kabaneroResource := kabanerov1alpha2.Kabanero{
		ObjectMeta: metav1.ObjectMeta{Name: "kabanero", Namespace: "kabanero"},
		Spec: kabanerov1alpha2.KabaneroSpec{
			Gitops: kabanerov1alpha2.GitopsSpec{
				Pipelines: []kabanerov1alpha2.PipelineSpec{{
					Id: "default",
					Sha256: digest1Pipeline.sha256,
					Https: kabanerov1alpha2.HttpsProtocolFile{Url: "bogus"},
				}},
			},
		},
		Status: kabanerov1alpha2.KabaneroStatus{
			Gitops: kabanerov1alpha2.GitopsStatus{
				Pipelines: []kabanerov1alpha2.PipelineStatus{{
					Name: "default",
					Digest: digest1Pipeline.sha256,
					ActiveAssets: []kabanerov1alpha2.RepositoryAssetStatus{{
						Name: "my-pipeline",
						Namespace: "kabanero",
					}, {
						Name: "my-task",
						Namespace: "kabanero",
					}},
				}},
			},
		},
	}

	clientMap := make(map[client.ObjectKey]bool)
	clientMap[client.ObjectKey{Name: "my-pipeline", Namespace: "kabanero"}] = true
	clientMap[client.ObjectKey{Name: "my-task", Namespace: "kabanero"}] = true
	client := gitopsTestClient{clientMap}
	
	err := cleanupGitopsPipelines(context.TODO(), &kabaneroResource, client, klog)

	if err != nil {
		t.Fatal("Returned error: " + err.Error())
	}

	// Make sure the client has the correct objects.
	if len(client.objs) != 0 {
		t.Fatal(fmt.Sprintf("Client map should have 0 entries, but has %v: %v", len(client.objs), client.objs))
	}
}

