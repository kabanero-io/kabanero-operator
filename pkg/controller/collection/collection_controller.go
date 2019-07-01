package collection

import (
	"context"
	"fmt"
	"time"

	mf "github.com/jcrossley3/manifestival"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_collection")

// Add creates a new Collection Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCollection{client: mgr.GetClient(), scheme: mgr.GetScheme(), indexResolver: resolveIndex}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("collection-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Collection
	err = c.Watch(&source.Kind{Type: &kabanerov1alpha1.Collection{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Collection
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kabanerov1alpha1.Collection{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCollection implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCollection{}

// ReconcileCollection reconciles a Collection object
type ReconcileCollection struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	//The indexResolver which will be used during reconciliation
	indexResolver func(string) (*CollectionV1Index, error)
}

// Reconcile reads that state of the cluster for a Collection object and makes changes based on the state read
// and what is in the Collection.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCollection) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Collection")

	// Fetch the Collection instance
	instance := &kabanerov1alpha1.Collection{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Resolve the kabanero instance
	var k *kabanerov1alpha1.Kabanero
	l := kabanerov1alpha1.KabaneroList{}
	err = r.client.List(context.Background(), &client.ListOptions{}, &l)
	for _, _k := range l.Items {
		if _k.GetNamespace() == instance.GetNamespace() {
			k = &_k
		}
	}
	reqLogger.Info("Resolved Kabanero", "kabanero", k)

	rr, err := r.ReconcileCollection(instance, k)

	//Update the status
	if !rr.Requeue {
		r.client.Status().Update(ctx, instance)
	}

	return rr, err
}

func (r *ReconcileCollection) ReconcileCollection(c *kabanerov1alpha1.Collection, k *kabanerov1alpha1.Kabanero) (reconcile.Result, error) {
	r_log := log.WithValues("Request.Namespace", c.GetNamespace()).WithValues("Request.Name", c.GetName())

	//The collection name can be either the spec.name or the resource name. The
	//spec.name has precedence
	var collectionName string
	if c.Spec.Name != "" {
		collectionName = c.Spec.Name
	} else {
		collectionName = c.Name
	}
	r_log = r_log.WithValues("Collection.Name", collectionName)

	//Retreive the remote index
	var collection *CollectionV1
	if len(k.Spec.Collections.Repositories) > 0 {
		for _, repo := range k.Spec.Collections.Repositories {
			_collection, err := r.searchCollection(collectionName, repo.Url)
			r_log.Error(err, "Could not search the provided index")
			if _collection != nil {
				collection = _collection
			}
		}
	}

	//If not found, search the default
	//TODO: incorporate this default into a webhook
	default_url := "https://raw.githubusercontent.com/kabanero-io/kabanero-collection/master/experimental/index.yaml"
	collection, err := r.searchCollection(collectionName, default_url)

	if collection == nil {
		return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, fmt.Errorf("Collection could not be found")
	}

	err = activate(c, collection, r.client)
	if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCollection) searchCollection(collectionName string, url string) (*CollectionV1, error) {
	fmt.Println("Search collection ", collectionName, url, r)
	index, err := r.indexResolver(url)
	if err != nil {
		return nil, err
	}

	//Locate the desired collection in the index
	var collectionRef *IndexedCollectionV1
	for _, collectionList := range index.Collections {
		for _, _collectionRef := range collectionList {
			if _collectionRef.Name == collectionName {
				collectionRef = &_collectionRef
			}
		}
	}

	if collectionRef == nil {
		//The collection referenced in the Collection resource has no match in the index
		return nil, nil
	}

	collection, err := resolveCollection(collectionRef.CollectionUrls...)
	if err != nil {
		return nil, err
	}

	return collection, nil
}

func activate(collectionResource *kabanerov1alpha1.Collection, collection *CollectionV1, c client.Client) error {
	manifest := collection.Manifest

	for _, asset := range manifest.Assets {
		if asset.Type == "kubernetes-resource" {
			log.Info(fmt.Sprintf("Applying asset %v", asset.Url))

			m, err := mf.NewManifest(asset.Url, false, c)
			if err != nil {
				return err
			}

			log.Info(fmt.Sprintf("Resources: %v", m.Resources))

			err = m.Transform(func(u *unstructured.Unstructured) error {
				u.SetNamespace(collectionResource.GetNamespace())
				return nil
			})
			if err != nil {
				return err
			}

			err = m.ApplyAll()
		}
	}

	return nil
}
