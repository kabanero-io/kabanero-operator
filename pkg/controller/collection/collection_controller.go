package collection

import (
	"context"
	"fmt"
	"time"

	"github.com/blang/semver"
	mf "github.com/jcrossley3/manifestival"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	// TODO: issue #92, when repo is added to the Collection, there will be no need for the Kabanero
	//       object here.
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

// Finds the collection with the highest semver.  Caller has validated that resolvedCollection
// contains at least one collection.
func findMaxVersionCollection(collections []resolvedCollection) *resolvedCollection {
	log.Info(fmt.Sprintf("findMaxVersionCollection: processing %v collections", len(collections)))

	var maxCollection *resolvedCollection = nil
	maxVersion, _ := semver.Make("0.0.0")

	for i, _ := range collections {
		curVersion, err := semver.ParseTolerant(collections[i].collection.Manifest.Version)
		if err == nil {
			if curVersion.Compare(maxVersion) > 0 {
				maxCollection = &collections[i]
				maxVersion = curVersion
			}
		} else {
			log.Info(fmt.Sprintf("findMaxVersionCollection: invalid semver " + collections[i].collection.Manifest.Version))
		}
	}

	// It's possible we didn't find a valid semver, in which case we'd return nil.
	return maxCollection
}


// Used internally by ReconcileCollection to store matching collections
type resolvedCollection struct {
	collection CollectionV1
	repositoryUrl string
}

func (r *ReconcileCollection) ReconcileCollection(c *kabanerov1alpha1.Collection, k *kabanerov1alpha1.Kabanero) (reconcile.Result, error) {
	r_log := log.WithValues("Request.Namespace", c.GetNamespace()).WithValues("Request.Name", c.GetName())

	// Clear the status message, we'll generate a new one if necessary
	c.Status.StatusMessage = ""

	//The collection name can be either the spec.name or the resource name. The
	//spec.name has precedence
	var collectionName string
	if c.Spec.Name != "" {
		collectionName = c.Spec.Name
	} else {
		collectionName = c.Name
	}
	r_log = r_log.WithValues("Collection.Name", collectionName)

	// Retreive all matching collection names from all remote indexes.  If none were specified,
	// use the default.
	var matchingCollections []resolvedCollection
	repositories := k.Spec.Collections.Repositories
	if len(repositories) == 0 {
		default_url := "https://raw.githubusercontent.com/kabanero-io/kabanero-collection/master/experimental/index.yaml"
		repositories = append(repositories, kabanerov1alpha1.RepositoryConfig{Name: "default", Url: default_url})
	}
	
	for _, repo := range repositories {
		index, err := r.indexResolver(repo.Url)
		if err != nil {
			// TODO: Issue #92, should just search the repository where the colleciton was loaded initially.
			return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
		}

		// Search for all versions of the collection in this repository index.
		_collections, err := SearchCollection(collectionName, index)
		if err != nil {
			r_log.Error(err, "Could not search the provided index")
		}

		// Build out the list of all collections across all repository indexes
		for _, collection := range _collections {
			matchingCollections = append(matchingCollections, resolvedCollection{collection: collection, repositoryUrl: repo.Url})
		}
	}

	// We have a list of all collections that match the name.  We'll use this list to see
	// if we have one at the requested version, as well as find the one at the highest level
	// to inform the user if an upgrade is available.
	if len(matchingCollections) > 0 {
		specVersion, semverErr := semver.ParseTolerant(c.Spec.Version)
		if (semverErr == nil) {
			// Search for the highest version.  Update the Status field if one is found that
			// is higher than the requested version.  This will only work if the Spec.Version adheres
			// to the semver standard.
			upgradeCollection := findMaxVersionCollection(matchingCollections)
			if upgradeCollection != nil {
				// The upgrade collection semver is valid, we tested it in findMaxVersionCollection
				upgradeVersion, _ := semver.ParseTolerant(upgradeCollection.collection.Manifest.Version)
				if upgradeVersion.Compare(specVersion) > 0 {
					c.Status.AvailableVersion = upgradeCollection.collection.Manifest.Version
					c.Status.AvailableLocation = upgradeCollection.repositoryUrl
				} else {
					// The spec version is the same or higher than the collection versions
					c.Status.AvailableVersion = ""
					c.Status.AvailableLocation = ""
				}
			} else {
				// None of the collections versions adher to semver standards
				c.Status.AvailableVersion = ""
				c.Status.AvailableLocation = ""
			}
		} else {
			r_log.Error(semverErr, "Could not determine upgrade availability for collection " + collectionName)
		}
		
		// Search for the correct version.  The list may have duplicates, we're just searching for
		// the first match.
		for _, matchingCollection := range matchingCollections {
			if matchingCollection.collection.Manifest.Version == c.Spec.Version {
				err := activate(c, &matchingCollection.collection, r.client)
				if err != nil {
					return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
				}
				return reconcile.Result{}, nil
			}
		}

		// No collection with a matching version could be found. Update the status
		// message so the user understands that the version they want is not available.
		c.Status.StatusMessage = "The requested version of the collection (" + c.Spec.Version + ") is not available."

		return reconcile.Result{}, nil
	}

	// No version of the collection could be found.  If there is no active version, update
	// the status message so the user knows that something needs to be done.
	if c.Status.ActiveVersion == "" {
		c.Status.StatusMessage = "No version of the collection is available."
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

	// Detect if the version is changing from the active version.  If it is, we need to clean up the
	// assets from the previous version.
	if (collectionResource.Status.ActiveVersion != "") && (collectionResource.Status.ActiveVersion != manifest.Version) {
		// Our version change strategy is going to be as follows:
		// 1) Attempt to load all of the known artifacts.  Any failure, status message and punt.
		// 2) Delete the artifacts.  If something goes wrong here, the state of the collection is unknown.
		type transformedRemoteManifest struct {
			m mf.Manifest
			assetUrl string
		}
		
		var transformedManifests []transformedRemoteManifest
		errorMessage := "Error during version change from " + collectionResource.Status.ActiveVersion + " to " + manifest.Version
		
		for _, asset := range collectionResource.Status.ActiveAssets {
			log.Info(fmt.Sprintf("Preparing to delete asset %v", asset.Url))

			m, err := mf.NewManifest(asset.Url, false, c)
			if err != nil {
				log.Error(err, errorMessage, "resource", asset.Url)
				collectionResource.Status.StatusMessage = errorMessage + ": " + err.Error()
				return nil // Forces status to be updated
			}

			log.Info(fmt.Sprintf("Resources: %v", m.Resources))

			err = m.Transform(func(u *unstructured.Unstructured) error {
				u.SetNamespace(collectionResource.GetNamespace())
				return nil
			})
			if err != nil {
				log.Error(err, errorMessage, "resource", asset.Url)
				collectionResource.Status.StatusMessage = errorMessage + ": " + err.Error()
				return nil // Forces status to be updated
			}

			// Queue the resolved manifest to be deleted in the next step.
			transformedManifests = append(transformedManifests, transformedRemoteManifest{m, asset.Url})
		}

		// Now delete the manifests
		for _, transformedManifest := range transformedManifests {
		  err := transformedManifest.m.DeleteAll()
			if err != nil {
				// It's hard to know what the state of things is now... log the error.
				log.Error(err, "Error deleting the resource", "resource", transformedManifest.assetUrl)
			}
		}

		// Indicate there is not currently an active version of this collection.
		collectionResource.Status.ActiveVersion = ""
		collectionResource.Status.ActiveAssets = nil
	}

	// Now apply the new version
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

				transforms := []mf.Transformer{
					mf.InjectOwner(collectionResource),
					mf.InjectNamespace(collectionResource.GetNamespace()),
				}

				err = m.Transform(transforms... )
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
