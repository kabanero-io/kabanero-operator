package collection

import (
	"context"
	me "errors"
	"fmt"
	"time"

	"github.com/blang/semver"
	mf "github.com/jcrossley3/manifestival"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
//	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
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

	// Create Collection predicate
	c_pred := predicate.Funcs{
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Returning true only when the metadata generation has changed, 
			// allows us to ignore events where only the object status has changed, 
			// since the generation is not incremented when only the status changes
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	}

	// Watch for changes to primary resource Collection
	err = c.Watch(&source.Kind{Type: &kabanerov1alpha1.Collection{}}, &handler.EnqueueRequestForObject{}, c_pred)
	if err != nil {
		return err
	}

	// Create a handler for handling Tekton Pipeline & Task events
	t_h := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kabanerov1alpha1.Collection{},
	}
	
	// Create Tekton predicate
	t_pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// ignore Create. Collection create applies the documents. Watch would unnecessarily requeue.
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Returning true only when the metadata generation has changed, 
			// allows us to ignore events where only the object status has changed, 
			// since the generation is not incremented when only the status changes
			return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
		},
	}
	
	// Watch for changes to Collection Tekton Pipeline objects
	err = c.Watch(&source.Kind{Type: &pipelinev1alpha1.Pipeline{}}, t_h, t_pred)
	if err != nil {
		log.Info(fmt.Sprintf("Tekton Pipelines may not be installed"))
		return err
	}

	err = c.Watch(&source.Kind{Type: &pipelinev1alpha1.Task{}}, t_h, t_pred)
	if err != nil {
		log.Info(fmt.Sprintf("Tekton Pipelines may not be installed"))
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
	indexResolver func(kabanerov1alpha1.RepositoryConfig) (*CollectionV1Index, error)
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
	if failedAssets(instance.Status) && (rr.Requeue == false) {
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
	var err error = nil
	maxVersion, _ := semver.Make("0.0.0")
	curVersion, _ := semver.Make("0.0.0")

	for i, _ := range collections {

		switch {
		//v1
		case collections[i].collection.Manifest.Version != "":
			curVersion, err = semver.ParseTolerant(collections[i].collection.Manifest.Version)
		//v2
		case collections[i].collectionv2.Version != "":
			curVersion, err = semver.ParseTolerant(collections[i].collectionv2.Version)
		}

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

func (r *ReconcileCollection) ensureCollectionHasOwner(c *kabanerov1alpha1.Collection, k *kabanerov1alpha1.Kabanero) error {
	foundKabanero := false
	ownerReferences := c.GetOwnerReferences()
	if ownerReferences != nil {
		for _, ownerRef := range ownerReferences {
			if ownerRef.Kind == "Kabanero" {
				if ownerRef.UID == k.ObjectMeta.UID {
					foundKabanero = true
				}
			}
		}
	}
	if !foundKabanero {
		// Get kabanero instance. Input one does not have APIVersion or Kind.
		ownerIsController := true
		kInstance := &kabanerov1alpha1.Kabanero{}
		name := types.NamespacedName{
			Name:      k.ObjectMeta.Name,
			Namespace: c.GetNamespace(),
		}
		err := r.client.Get(context.Background(), name, kInstance)
		if err != nil {
			return err
		}

		// Make kabanero the owner of the collection
		ownerRef := metav1.OwnerReference{
			APIVersion: kInstance.TypeMeta.APIVersion,
			Kind:       kInstance.TypeMeta.Kind,
			Name:       kInstance.ObjectMeta.Name,
			UID:        kInstance.ObjectMeta.UID,
			Controller: &ownerIsController,
		}
		c.SetOwnerReferences(append(c.GetOwnerReferences(), ownerRef))
		err = r.client.Update(context.Background(), c)
		if err != nil {
			return err
		}
		log.Info("Updated collection owner")
	}
	return nil
}

// Used internally by ReconcileCollection to store matching collections
type resolvedCollection struct {
	//v1
	collection    CollectionV1
	repositoryUrl string
	//v2
	collectionv2 IndexedCollectionV2
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

	// A collection created by the CLI might not have kabanero as the owner.
	// In that case we want to make kabanero the owner
	err := r.ensureCollectionHasOwner(c, k)
	if err != nil {
		r_log.Error(err, "Could not make kabanero the owner of the collection")
	}

	// Retreive all matching collection names from all remote indexes.  If none were specified,
	// build and log an error and return.
	var matchingCollections []resolvedCollection
	repositories := k.Spec.Collections.Repositories
	if len(repositories) == 0 {
		err = me.New(fmt.Sprintf("No repositories were configured in the Kabanero instance"))
		r_log.Error(err, "Could not continue without any repositories")

		// Update the status message so the user knows that something needs to be done.
		c.Status.StatusMessage = "Could not find any configured repositories."
		return reconcile.Result{Requeue: false, RequeueAfter: 60 * time.Second}, err
	}

	for _, repo := range repositories {
		index, err := r.indexResolver(repo)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
		}
		// Handle Index Collection version
		switch apiVersion := index.ApiVersion; apiVersion {
		case "v1":
			// Search for all versions of the collection in this repository index.
			_collections, err := SearchCollection(repo, collectionName, index)
			if err != nil {
				r_log.Error(err, "Could not search the provided index")
			}

			// Build out the list of all collections across all repository indexes
			for _, collection := range _collections {
				matchingCollections = append(matchingCollections, resolvedCollection{collection: collection, repositoryUrl: repo.Url})
			}
		case "v2":
			// Search for all versions of the collection in this repository index.
			_collections, err := SearchCollectionV2(collectionName, index)
			if err != nil {
				r_log.Error(err, "Could not search the provided index")
			}

			// Build out the list of all collections across all repository indexes
			for _, collection := range _collections {
				matchingCollections = append(matchingCollections, resolvedCollection{collectionv2: collection, repositoryUrl: repo.Url})
			}

		default:
			fmt.Sprintf("Index is unsupported version: %s", apiVersion)
		}
	}

	// We have a list of all collections that match the name.  We'll use this list to see
	// if we have one at the requested version, as well as find the one at the highest level
	// to inform the user if an upgrade is available.
	if len(matchingCollections) > 0 {
		specVersion, semverErr := semver.ParseTolerant(c.Spec.Version)
		if semverErr == nil {
			// Search for the highest version.  Update the Status field if one is found that
			// is higher than the requested version.  This will only work if the Spec.Version adheres
			// to the semver standard.
			upgradeCollection := findMaxVersionCollection(matchingCollections)
			if upgradeCollection != nil {
				// The upgrade collection semver is valid, we tested it in findMaxVersionCollection
				upgradeVersion, _ := semver.Make("0.0.0")
				switch {
				//v1
				case upgradeCollection.collection.Manifest.Version != "":
					upgradeVersion, _ = semver.ParseTolerant(upgradeCollection.collection.Manifest.Version)
					if upgradeVersion.Compare(specVersion) > 0 {
						c.Status.AvailableVersion = upgradeCollection.collection.Manifest.Version
						c.Status.AvailableLocation = upgradeCollection.repositoryUrl
					} else {
						c.Status.AvailableVersion = ""
						c.Status.AvailableLocation = ""
					}
				//v2
				case upgradeCollection.collectionv2.Version != "":
					upgradeVersion, _ = semver.ParseTolerant(upgradeCollection.collectionv2.Version)
					if upgradeVersion.Compare(specVersion) > 0 {
						c.Status.AvailableVersion = upgradeCollection.collectionv2.Version
						c.Status.AvailableLocation = upgradeCollection.repositoryUrl
					} else {
						c.Status.AvailableVersion = ""
						c.Status.AvailableLocation = ""
					}
				}

			} else {
				// None of the collections versions adher to semver standards
				c.Status.AvailableVersion = ""
				c.Status.AvailableLocation = ""
			}
		} else {
			r_log.Error(semverErr, "Could not determine upgrade availability for collection "+collectionName)
		}

		// Search for the correct version.  The list may have duplicates, we're just searching for
		// the first match.
		for _, matchingCollection := range matchingCollections {
			switch {
			//v1
			case matchingCollection.collection.Manifest.Version == c.Spec.Version:
				err := activate(c, &matchingCollection.collection, r.client)
				if err != nil {
					return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
				}
				c.Status.ActiveLocation = matchingCollection.repositoryUrl
				return reconcile.Result{}, nil
			//v2
			case matchingCollection.collectionv2.Version == c.Spec.Version:
				//need a v2 activate
				err := activatev2(c, &matchingCollection.collectionv2, r.client)
				if err != nil {
					return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
				}
				c.Status.ActiveLocation = matchingCollection.repositoryUrl
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

// Check if the asset read from a manifest is equal to the asset in the status
// object.  They are equal if they have the same name, or if the name is
// nil, if the URLs are equal.
func assetMatchv2(assetStatus kabanerov1alpha1.RepositoryAssetStatus, asset IndexedPipelinesV2) bool {
	if len(asset.Id) == 0 {
		return asset.Url == assetStatus.Url
	}

	return asset.Id == assetStatus.Name
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

func updateAssetStatusv2(status *kabanerov1alpha1.CollectionStatus, asset IndexedPipelinesV2, message string) {
	log.Info(fmt.Sprintf("Updating status for asset %v", asset.Id))

	// Assume that if there is a message, it's an error condition
	assetStatusMessage := message
	assetStatus := asset_active_status
	if len(assetStatusMessage) > 0 {
		assetStatus = asset_failed_status
	}

	// First find the asset in the Collection status.
	for index, curAssetStatus := range status.ActiveAssets {
		if assetMatchv2(curAssetStatus, asset) {
			// We found it - update the digest and asset status.
			status.ActiveAssets[index].Digest = asset.Sha256
			status.ActiveAssets[index].Status = assetStatus
			status.ActiveAssets[index].StatusMessage = assetStatusMessage
			return
		}
	}

	// If the asset was not found, create a status for it
	status.ActiveAssets = append(status.ActiveAssets, kabanerov1alpha1.RepositoryAssetStatus{asset.Id, asset.Url, asset.Sha256, assetStatus, assetStatusMessage})
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
			m        mf.Manifest
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

			transforms := []mf.Transformer{
				mf.InjectNamespace(collectionResource.GetNamespace()),
			}

			err = m.Transform(transforms...)
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
					if assetMatch(assetStatus, asset) && (assetStatus.Digest == asset.Digest) && (assetStatus.Status == asset_active_status) {
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

				err = m.Transform(transforms...)
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
	collectionResource.Status.ActiveVersion = manifest.Version

	return nil
}

// Create a Manifest from unstructured, rather than path/url
//type Manifest struct {
//	mf.Manifest
//	Resources []unstructured.Unstructured
//	client    client.Client
//}

//func NewManifest(resources []unstructured.Unstructured, client client.Client) (Manifest, error) {
//	return Manifest{Resources: resources, client: client}, nil
//}

func activatev2(collectionResource *kabanerov1alpha1.Collection, collection *IndexedCollectionV2, c client.Client) error {
	//The context which will be used when rendering remote yaml
	renderingContext := map[string]interface{}{
		"CollectionId":   collection.Id,
		"CollectionName": collection.Name,
	}

	// Detect if the version is changing from the active version.  If it is, we need to clean up the
	// assets from the previous version.
	if (collectionResource.Status.ActiveVersion != "") && (collectionResource.Status.ActiveVersion != collection.Version) {
		// Our version change strategy is going to be as follows:
		// 1) Attempt to load all of the known artifacts.  Any failure, status message and punt.
		// 2) Delete the artifacts.  If something goes wrong here, the state of the collection is unknown.

		type transformedRemoteManifest struct {
			m        mf.Manifest
			assetUrl string
		}

		var transformedManifests []transformedRemoteManifest
		errorMessage := "Error during version change from " + collectionResource.Status.ActiveVersion + " to " + collection.Version

		for _, asset := range collectionResource.Status.ActiveAssets {
			log.Info(fmt.Sprintf("Preparing to delete asset %v", asset.Url))

			// Retrieve manifests as unstructured
			manifests, err := GetManifests(asset.Url, renderingContext)
			if err != nil {
				log.Error(err, errorMessage, "resource", asset.Url)
				collectionResource.Status.StatusMessage = errorMessage + ": " + err.Error()
				return nil // Forces status to be updated
			}

			// Construct dummy Manifest and client, due to client being private struct field
			m, err := mf.NewManifest("usr/local/bin/dummy.yaml", false, c)
			if err != nil {
				log.Error(err, errorMessage, "resource", asset.Url)
				collectionResource.Status.StatusMessage = errorMessage + ": " + err.Error()
				return nil // Forces status to be updated
			}

			// Assign the real manifests
			m.Resources = manifests

			log.Info(fmt.Sprintf("Resources: %v", m.Resources))

			transforms := []mf.Transformer{
				mf.InjectNamespace(collectionResource.GetNamespace()),
			}

			err = m.Transform(transforms...)
			if err != nil {
				log.Error(err, errorMessage, "resource", asset.Url)
				collectionResource.Status.StatusMessage = errorMessage + ": " + err.Error()
				return nil // Forces status to be updated
			}

			// Queue the resolved manifest to be deleted in the next step.
			transformedManifests = append(transformedManifests, transformedRemoteManifest{m, asset.Url})
		}

		// Indicate there is not currently an active version of this collection.
		collectionResource.Status.ActiveVersion = ""
		collectionResource.Status.ActiveAssets = nil
		collectionResource.Status.Images = nil
	}

	// Now apply the new version
	for _, asset := range collection.Pipelines {
		log.Info(fmt.Sprintf("Applying asset %v", asset.Url))

		// Retrieve manifests as unstructured
		manifests, err := GetManifests(asset.Url, renderingContext)
		if err != nil {
			return err
		}

		// Construct dummy Manifest and client, due to client being private struct field
		m, err := mf.NewManifest("usr/local/bin/dummy.yaml", false, c)
		if err != nil {
			return err
		}

		// Assign the real manifests
		m.Resources = manifests

		log.Info(fmt.Sprintf("Resources: %v", m.Resources))

		transforms := []mf.Transformer{
			mf.InjectOwner(collectionResource),
			mf.InjectNamespace(collectionResource.GetNamespace()),
		}

		err = m.Transform(transforms...)
		if err != nil {
			return err
		}

		for _, spec := range m.Resources {
			log.Info(fmt.Sprintf("Applying resource: %v", spec))

			err := m.Apply(&spec)
			if err != nil {
				// Update the asset status with the error message
				log.Error(err, "Error installing the resource", "resource", asset.Url)
				updateAssetStatusv2(&collectionResource.Status, asset, err.Error())
			} else {
				// Update the digest for this asset in the status
				updateAssetStatusv2(&collectionResource.Status, asset, "")
			}
		}
	}

	// Update the status of the Collection object to reflect the version we applied.
	collectionResource.Status.ActiveVersion = collection.Version

	// Update the status of the Collection object to reflect the images used
	var statusImages []kabanerov1alpha1.Image
	for _, image := range collection.Images {
		var statusImage kabanerov1alpha1.Image
		statusImage.Id = image.Id
		statusImage.Image = image.Image
		statusImages = append(statusImages, statusImage)
	}
	collectionResource.Status.Images = statusImages

	return nil
}
