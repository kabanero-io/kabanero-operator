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
	return &ReconcileCollection{client: mgr.GetClient(), scheme: mgr.GetScheme(), indexResolver: ResolveIndex}
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

	// Force a requeue if there are failed assets.  These should be retried, and since
	// they are hosted outside of Kubernetes, the controller will not see when they
	// are updated.
	if (failedAssets(instance.Status) && (rr.Requeue == false)) {
		reqLogger.Info("Forcing requeue due to failed assets in the Collection")
		rr.Requeue = true
		rr.RequeueAfter = 60 * time.Second
	}
	
	return rr, err
}

// Check to see if the status contains any assets that are failed
func failedAssets(status kabanerov1alpha1.CollectionStatus) bool {
	for _, assetStatus := range status.ActiveAssets {
		if assetStatus.Status == asset_failed_status {
			return true
		}
	}

	return false
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
	if k.Spec.Collections.Repositories != nil && len(k.Spec.Collections.Repositories) > 0 {
		for _, repo := range k.Spec.Collections.Repositories {
			index, err := r.indexResolver(repo.Url)
			if err != nil {
				return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
			}

			_collection, err := SearchCollection(collectionName, index)
			r_log.Error(err, "Could not search the provided index")
			if _collection != nil {
				collection = _collection
			}
		}
	}

	//If not found, search the default
	//TODO: incorporate this default into a webhook
	default_url := "https://raw.githubusercontent.com/kabanero-io/kabanero-collection/master/experimental/index.yaml"
	index, err := r.indexResolver(default_url)
	if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
	}
	collection, err = SearchCollection(collectionName, index)
	if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
	}
	if collection == nil {
		return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, fmt.Errorf("Collection could not be found")
	}

	err = activate(c, collection, r.client)
	if err != nil {
		return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
	}

	return reconcile.Result{}, nil
}

// Check if the asset read from a manifest is equal to the asset in the status
// object.  They are equal if they have the same name, or if the name is
// nil, if the URLs are equal.
func assetMatch(assetStatus kabanerov1alpha1.RepositoryAssetStatus, asset AssetManifest) bool {
	if len(asset.Name) == 0 {
		return asset.Url == assetStatus.Url
	}

	return asset.Name == assetStatus.Name
}

// Some asset status states
const asset_active_status = "active"
const asset_failed_status = "failed"

func updateAssetStatus(status *kabanerov1alpha1.CollectionStatus, asset AssetManifest, message string) {
	log.Info(fmt.Sprintf("Updating status for asset %v", asset.Name))

	// Assume that if there is a message, it's an error condition
	assetStatusMessage := message
	assetStatus := asset_active_status
	if len(assetStatusMessage) > 0 {
		assetStatus = asset_failed_status
	}
	
	// First find the asset in the Collection status.
	for index, curAssetStatus := range status.ActiveAssets {
		if assetMatch(curAssetStatus, asset) {
			// We found it - update the digest and asset status.
			status.ActiveAssets[index].Digest = asset.Digest
			status.ActiveAssets[index].Status = assetStatus
			status.ActiveAssets[index].StatusMessage = assetStatusMessage
			return
		}
	}

	// If the asset was not found, create a status for it
	status.ActiveAssets = append(status.ActiveAssets, kabanerov1alpha1.RepositoryAssetStatus{asset.Name, asset.Url, asset.Digest, assetStatus, assetStatusMessage})
}

func activate(collectionResource *kabanerov1alpha1.Collection, collection *CollectionV1, c client.Client) error {
	manifest := collection.Manifest
	
	for _, asset := range manifest.Assets {
		if asset.Type == "kubernetes-resource" {
			// If the asset has a digest, see if the digest has changed.  Don't bother updating anything
			// if the digest is the same and the asset is active.
			log.Info(fmt.Sprintf("Checking digest for asset %v", asset.Url))
			applyAsset := true
			if len(asset.Digest) > 0 {
				for _, assetStatus := range collectionResource.Status.ActiveAssets {
					if (assetMatch(assetStatus, asset) && (assetStatus.Digest == asset.Digest) && (assetStatus.Status == asset_active_status)) {
						// The digest is equal and the asset is active - don't apply the asset.
						log.Info(fmt.Sprintf("Digest has not changed %v", asset.Digest))
						applyAsset = false
						break
					}
				}
			}

			if applyAsset {
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
				if err != nil {
					// Update the asset status with the error message
					log.Error(err, "Error installing the resource", "resource", asset.Url)
					updateAssetStatus(&collectionResource.Status, asset, err.Error())
				} else {
					// Update the digest for this asset in the status
					updateAssetStatus(&collectionResource.Status, asset, "")
				}
			}
		}
	}

	// Update the status of the Collection object to reflect the version we applied.
	collectionResource.Status.ActiveVersion = manifest.Version;

	return nil
}
