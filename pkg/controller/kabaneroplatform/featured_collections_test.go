package kabaneroplatform

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"testing"
)

// -----------------------------------------------------------------------------------------------
// Client that creates/deletes collections.
// -----------------------------------------------------------------------------------------------
type unitTestClient struct {
	objs map[string]*kabanerov1alpha1.Collection
}

func (c unitTestClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	fmt.Printf("Received Get() for %v\n", key.Name)
	u, ok := obj.(*kabanerov1alpha1.Collection)
	if !ok {
		fmt.Printf("Received invalid target object for get: %v\n", obj)
		return errors.New("Get only supports Collections")
	}
	collection := c.objs[key.Name]
	if collection == nil {
		return apierrors.NewNotFound(schema.GroupResource{}, key.Name)
	}
	collection.DeepCopyInto(u)
	return nil
}
func (c unitTestClient) List(ctx context.Context, opts *client.ListOptions, list runtime.Object) error {
	return errors.New("List is not supported")
}
func (c unitTestClient) Create(ctx context.Context, obj runtime.Object) error {
	u, ok := obj.(*kabanerov1alpha1.Collection)
	if !ok {
		fmt.Printf("Received invalid create: %v\n", obj)
		return errors.New("Create only supports Collections")
	}

	fmt.Printf("Received Create() for %v\n", u.Name)
	collection := c.objs[u.Name]
	if collection != nil {
		fmt.Printf("Receive create object already exists: %v\n", u.Name)
		return apierrors.NewAlreadyExists(schema.GroupResource{}, u.Name)
	}

	c.objs[u.Name] = u
	return nil
}
func (c unitTestClient)	Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOptionFunc) error {
	return errors.New("Delete is not supported")
}
func (c unitTestClient) Update(ctx context.Context, obj runtime.Object) error {
	u, ok := obj.(*kabanerov1alpha1.Collection)
	if !ok {
		fmt.Printf("Received invalid update: %v\n", obj)
		return errors.New("Update only supports Collections")
	}

	fmt.Printf("Received Update() for %v\n", u.Name)
	collection := c.objs[u.Name]
	if collection == nil {
		fmt.Printf("Received update for object that does not exist: %v\n", obj)
		return apierrors.NewNotFound(schema.GroupResource{}, u.Name)
	}
	c.objs[u.Name] = u
	return nil
}
func (c unitTestClient) Status() client.StatusWriter { return c }


// -----------------------------------------------------------------------------------------------
// HTTP handler that serves pipeline zips
// -----------------------------------------------------------------------------------------------
type collectionIndexHandler struct {
}

func (ch collectionIndexHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	filename := fmt.Sprintf("testdata/%v", req.URL.String())
	fmt.Printf("Serving %v\n", filename)
	d, err := ioutil.ReadFile(filename)
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
	} else {
		rw.Write(d)
	}
}

var defaultIndexName = "/kabanero-index.yaml"
var secondIndexName = "/kabanero-index-two.yaml"

// -----------------------------------------------------------------------------------------------
// Test cases
// -----------------------------------------------------------------------------------------------
func createKabanero(repositoryUrl string, activateDefaultCollections bool) *kabanerov1alpha1.Kabanero {
	return &kabanerov1alpha1.Kabanero{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kabanero.io/v1alpha1",
			Kind:       "Kabanero",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "kabanero",
			UID:       "12345",
		},
		Spec: kabanerov1alpha1.KabaneroSpec{
			Collections: kabanerov1alpha1.InstanceCollectionConfig{
				Repositories: []kabanerov1alpha1.RepositoryConfig{
					kabanerov1alpha1.RepositoryConfig{
						Name:                       "default",
						Url:                        repositoryUrl,
						ActivateDefaultCollections: activateDefaultCollections,
					},
				},
			},
		},
	}
}

func TestReconcileFeaturedCollections(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionIndexHandler{})
	defer server.Close()

	ctx := context.Background()
	cl := unitTestClient{make(map[string]*kabanerov1alpha1.Collection)}
	collectionUrl := server.URL + defaultIndexName
	k := createKabanero(collectionUrl, true)

	err := reconcileFeaturedCollections(ctx, k, cl)
	if err != nil {
		t.Fatal(err)
	}

	// Should have been two collections created
	javaMicroprofileCollection := &kabanerov1alpha1.Collection{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile"}, javaMicroprofileCollection)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile collection", err)
	}

	nodejsCollection := &kabanerov1alpha1.Collection{}
	err = cl.Get(ctx, types.NamespacedName{Name: "nodejs"}, nodejsCollection)
	if err != nil {
		t.Fatal("Could not resolve the nodejs collection", err)
	}

	// Make sure the collection has an owner set
	if len(nodejsCollection.OwnerReferences) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 owner, but found %v: %v", len(nodejsCollection.OwnerReferences), nodejsCollection))
	}

	if nodejsCollection.OwnerReferences[0].UID != k.UID {
		t.Fatal(fmt.Sprintf("Expected owner UID to be %v, but was %v", k.UID, nodejsCollection.OwnerReferences[0].UID))
	}

	// Make sure the collection is active
	if len(nodejsCollection.Spec.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 collection version, but found %v: %v", len(nodejsCollection.Spec.Versions), nodejsCollection.Spec.Versions))
	}

	if nodejsCollection.Spec.Versions[0].Version != "0.2.6" {
		t.Fatal(fmt.Sprintf("Expected nodejs collection version \"0.2.6\", but found %v", nodejsCollection.Spec.Versions[0].Version))
	}

	if nodejsCollection.Spec.Versions[0].DesiredState != kabanerov1alpha1.CollectionDesiredStateActive {
		t.Fatal(fmt.Sprintf("Expected nodejs collection to be active, but was %v", nodejsCollection.Spec.Versions[0].DesiredState))
	}

	if nodejsCollection.Spec.Versions[0].RepositoryUrl != collectionUrl {
		t.Fatal(fmt.Sprintf("Expected nodejs URL to be %v, but was %v", collectionUrl, nodejsCollection.Spec.Versions[0].RepositoryUrl))
	}
}

// specify ActivateDefaultCollections: false
func TestReconcileFeaturedCollectionsActivateDefaultCollectionsFalse(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionIndexHandler{})
	defer server.Close()

	ctx := context.Background()
	cl := unitTestClient{make(map[string]*kabanerov1alpha1.Collection)}
	collectionUrl := server.URL + defaultIndexName
	k := createKabanero(collectionUrl, false)

	err := reconcileFeaturedCollections(ctx, k, cl)
	if err != nil {
		t.Fatal(err)
	}

	// Should have been two collections created
	javaMicroprofileCollection := &kabanerov1alpha1.Collection{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile"}, javaMicroprofileCollection)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile collection", err)
	}

	nodejsCollection := &kabanerov1alpha1.Collection{}
	err = cl.Get(ctx, types.NamespacedName{Name: "nodejs"}, nodejsCollection)
	if err != nil {
		t.Fatal("Could not resolve the nodejs collection", err)
	}

	// Make sure the collection has an owner set
	if len(nodejsCollection.OwnerReferences) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 owner, but found %v: %v", len(nodejsCollection.OwnerReferences), nodejsCollection))
	}

	if nodejsCollection.OwnerReferences[0].UID != k.UID {
		t.Fatal(fmt.Sprintf("Expected owner UID to be %v, but was %v", k.UID, nodejsCollection.OwnerReferences[0].UID))
	}

	// Make sure the collection is active
	if len(nodejsCollection.Spec.Versions) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 collection version, but found %v: %v", len(nodejsCollection.Spec.Versions), nodejsCollection.Spec.Versions))
	}

	if nodejsCollection.Spec.Versions[0].Version != "0.2.6" {
		t.Fatal(fmt.Sprintf("Expected nodejs collection version \"0.2.6\", but found %v", nodejsCollection.Spec.Versions[0].Version))
	}

	if nodejsCollection.Spec.Versions[0].DesiredState != kabanerov1alpha1.CollectionDesiredStateInactive {
		t.Fatal(fmt.Sprintf("Expected nodejs collection to be inactive, but was %v", nodejsCollection.Spec.Versions[0].DesiredState))
	}

	if nodejsCollection.Spec.Versions[0].RepositoryUrl != collectionUrl {
		t.Fatal(fmt.Sprintf("Expected nodejs URL to be %v, but was %v", collectionUrl, nodejsCollection.Spec.Versions[0].RepositoryUrl))
	}
}

func TestReconcileFeaturedCollectionsTwoRepositories(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionIndexHandler{})
	defer server.Close()

	ctx := context.Background()
	cl := unitTestClient{make(map[string]*kabanerov1alpha1.Collection)}
	collectionUrl := server.URL + defaultIndexName
	collectionUrlTwo := server.URL + secondIndexName
	k := createKabanero(collectionUrl, true)
	k.Spec.Collections.Repositories = append(k.Spec.Collections.Repositories, kabanerov1alpha1.RepositoryConfig{Name: "two", Url: collectionUrlTwo, ActivateDefaultCollections: false})

	err := reconcileFeaturedCollections(ctx, k, cl)
	if err != nil {
		t.Fatal(err)
	}

	// Should have been two collections created
	javaMicroprofileCollection := &kabanerov1alpha1.Collection{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile"}, javaMicroprofileCollection)
	if err != nil {
		t.Fatal("Could not resolve the java-microprofile collection", err)
	}

	nodejsCollection := &kabanerov1alpha1.Collection{}
	err = cl.Get(ctx, types.NamespacedName{Name: "nodejs"}, nodejsCollection)
	if err != nil {
		t.Fatal("Could not resolve the nodejs collection", err)
	}

	// Make sure the collection has an owner set
	if len(nodejsCollection.OwnerReferences) != 1 {
		t.Fatal(fmt.Sprintf("Expected 1 owner, but found %v: %v", len(nodejsCollection.OwnerReferences), nodejsCollection))
	}

	if nodejsCollection.OwnerReferences[0].UID != k.UID {
		t.Fatal(fmt.Sprintf("Expected owner UID to be %v, but was %v", k.UID, nodejsCollection.OwnerReferences[0].UID))
	}

	// Make sure the collection is in the correct state
	if len(nodejsCollection.Spec.Versions) != 2 {
		t.Fatal(fmt.Sprintf("Expected 2 collection versions, but found %v: %v", len(nodejsCollection.Spec.Versions), nodejsCollection.Spec.Versions))
	}

	foundVersions := make(map[string]bool)
	for _, cur := range nodejsCollection.Spec.Versions {
		foundVersions[cur.Version] = true
		if cur.Version == "0.2.6" {
			if cur.DesiredState != kabanerov1alpha1.CollectionDesiredStateActive {
				t.Fatal(fmt.Sprintf("Expected version \"0.2.6\" to be active, but was %v", cur.DesiredState))
			}
			if cur.RepositoryUrl != collectionUrl {
				t.Fatal(fmt.Sprintf("Expected version \"0.2.6\" URL to be %v, but was %v", collectionUrl, cur.RepositoryUrl))
			}
		} else if cur.Version == "0.4.1" {
			if cur.DesiredState != kabanerov1alpha1.CollectionDesiredStateInactive {
				t.Fatal(fmt.Sprintf("Expected version \"0.4.1\" to be inactive, but was %v", cur.DesiredState))
			}
			if cur.RepositoryUrl != collectionUrlTwo {
				t.Fatal(fmt.Sprintf("Expected version \"0.4.1\" URL to be %v, but was %v", collectionUrlTwo, cur.RepositoryUrl))
			}
		} else {
			t.Fatal(fmt.Sprintf("Found unexpected version %v", cur.Version))
		}
	}

	if foundVersions["0.2.6"] != true {
		t.Fatal("Did not find collection version \"0.2.6\"")
	}

	if foundVersions["0.4.1"] != true {
		t.Fatal("Did not find collection version \"0.4.1\"")
	}
}

// Attempts to resolve the featured collections from the default repository
func TestResolveFeaturedCollections(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionIndexHandler{})
	defer server.Close()

	collection_index_url := server.URL + defaultIndexName
	k := createKabanero(collection_index_url, true)

	collections, err := featuredCollections(k)
	if err != nil {
		t.Fatal("Could not resolve the featured collections from the default index", err)
	}

	// Should be two collections
	if len(collections) != 2 {
		t.Fatal(fmt.Sprintf("Was expecting 2 collections to be found, but found %v: %v", len(collections), collections))
	}

	javaMicroprofileCollectionVersions, ok := collections["java-microprofile"]
	if !ok {
		t.Fatal(fmt.Sprintf("Could not find java-microprofile collection: %v", collections))
	}

	nodejsCollectionVersions, ok := collections["nodejs"]
	if !ok {
		t.Fatal(fmt.Sprintf("Could not find nodejs collection: %v", collections))
	}

	// Make sure each collection has one version
	if len(javaMicroprofileCollectionVersions) != 1 {
		t.Fatal(fmt.Sprintf("Expected one version of java-microprofile collection, but found %v: %v", len(javaMicroprofileCollectionVersions), javaMicroprofileCollectionVersions))
	}

	if len(nodejsCollectionVersions) != 1 {
		t.Fatal(fmt.Sprintf("Expected one version of nodejs collection, but found %v: %v", len(nodejsCollectionVersions), nodejsCollectionVersions))
	}
}

// Attempts to resolve the featured collections from two repositories
func TestResolveFeaturedCollectionsTwoRepositories(t *testing.T) {
	// The server that will host the pipeline zip
	server := httptest.NewServer(collectionIndexHandler{})
	defer server.Close()

	collection_index_url := server.URL + defaultIndexName
	collection_index_url_two := server.URL + secondIndexName
	k := createKabanero(collection_index_url, true)
	k.Spec.Collections.Repositories = append(k.Spec.Collections.Repositories, kabanerov1alpha1.RepositoryConfig{Name: "two", Url: collection_index_url_two})

	collections, err := featuredCollections(k)
	if err != nil {
		t.Fatal("Could not resolve the featured collections from the default index", err)
	}

	// Should be two collections
	if len(collections) != 2 {
		t.Fatal(fmt.Sprintf("Was expecting 2 collections to be found, but found %v: %v", len(collections), collections))
	}

	javaMicroprofileCollectionVersions, ok := collections["java-microprofile"]
	if !ok {
		t.Fatal(fmt.Sprintf("Could not find java-microprofile collection: %v", collections))
	}

	nodejsCollectionVersions, ok := collections["nodejs"]
	if !ok {
		t.Fatal(fmt.Sprintf("Could not find nodejs collection: %v", collections))
	}

	// Make sure each collection has two versions
	if len(javaMicroprofileCollectionVersions) != 2 {
		t.Fatal(fmt.Sprintf("Expected two versions of java-microprofile collection, but found %v: %v", len(javaMicroprofileCollectionVersions), javaMicroprofileCollectionVersions))
	}

	if len(nodejsCollectionVersions) != 2 {
		t.Fatal(fmt.Sprintf("Expected two versions of nodejs collection, but found %v: %v", len(nodejsCollectionVersions), nodejsCollectionVersions))
	}
}
