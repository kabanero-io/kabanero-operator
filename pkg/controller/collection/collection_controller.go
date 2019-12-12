package collection

import (
	"context"
	me "errors"
	"fmt"
	"strings"
	"time"

	"github.com/blang/semver"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/transforms"
	mf "github.com/kabanero-io/manifestival"

	//	corev1 "k8s.io/api/core/v1"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
)

var log = logf.Log.WithName("controller_collection")

const (
	// Asset status.
	assetStatusActive  = "active"
	assetStatusFailed  = "failed"
	assetStatusUnknown = "unknown"
)

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
	cPred := predicate.Funcs{
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
	err = c.Watch(&source.Kind{Type: &kabanerov1alpha1.Collection{}}, &handler.EnqueueRequestForObject{}, cPred)
	if err != nil {
		return err
	}

	// Create a handler for handling Tekton Pipeline & Task events
	tH := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kabanerov1alpha1.Collection{},
	}

	// Create Tekton predicate
	tPred := predicate.Funcs{
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
	err = c.Watch(&source.Kind{Type: &pipelinev1alpha1.Pipeline{}}, tH, tPred)
	if err != nil {
		log.Info(fmt.Sprintf("Tekton Pipelines may not be installed"))
		return err
	}

	err = c.Watch(&source.Kind{Type: &pipelinev1alpha1.Task{}}, tH, tPred)
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
	indexResolver func(kabanerov1alpha1.RepositoryConfig) (*Index, error)
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

	// Update the status
	r.client.Status().Update(ctx, instance)

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
	for _, pipelineStatus := range status.ActivePipelines {
		for _, assetStatus := range pipelineStatus.ActiveAssets {
			if assetStatus.Status == assetStatusFailed {
				return true
			}
		}
	}

	return false
}

// Finds the collection with the highest semver.  Caller has validated that resolvedCollection
// contains at least one collection.
func findMaxVersionCollection(collections []resolvedCollection) *resolvedCollection {
	log.Info(fmt.Sprintf("findMaxVersionCollection: processing %v collections", len(collections)))

	var maxCollection *resolvedCollection
	var err error
	maxVersion, _ := semver.Make("0.0.0")
	curVersion, _ := semver.Make("0.0.0")

	for _, collection := range collections {
		switch {
		case collection.collection.Version != "":
			curVersion, err = semver.ParseTolerant(collection.collection.Version)

			if err != nil {
				log.Info(fmt.Sprintf("findMaxVersionCollection: Invalid semver " + collection.collection.Version))
			}
		}

		if err == nil {
			if curVersion.Compare(maxVersion) > 0 {
				maxCollection = &collection
				maxVersion = curVersion
			}
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
	repositoryURL string
	collection    Collection
}

// ReconcileCollection activates or deactivates the input collection.
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

	// Process deactivates regardless of whether the collection is available in the remote
	// repository.
	if strings.EqualFold(c.Spec.DesiredState, kabanerov1alpha1.CollectionDesiredStateInactive) {
		err := reconcileActiveVersions(c, nil, r.client)
		if err != nil {
			return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
		}
		
		c.Status.Status = kabanerov1alpha1.CollectionDesiredStateInactive
		return reconcile.Result{}, nil
	}
	
	// Retreive all matching collection names from all remote indexes.  If none were specified,
	// build and log an error and return.
	// TODO: Start using the URL in the Collection object so we don't need to reference the
	//       parent Kabanero anymore.
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
		switch apiVersion := index.APIVersion; apiVersion {
		case "v2":
			// Search for all versions of the collection in this repository index.
			_collections, err := SearchCollection(collectionName, index)
			if err != nil {
				r_log.Error(err, "Could not search the provided index")
			}

			// Build out the list of all collections across all repository indexes
			for _, collection := range _collections {
				matchingCollections = append(matchingCollections, resolvedCollection{collection: collection, repositoryURL: repo.Url})
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
				case upgradeCollection.collection.Version != "":
					upgradeVersion, _ = semver.ParseTolerant(upgradeCollection.collection.Version)
					if upgradeVersion.Compare(specVersion) > 0 {
						c.Status.AvailableVersion = upgradeCollection.collection.Version
						c.Status.AvailableLocation = upgradeCollection.repositoryURL
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
			case matchingCollection.collection.Version == c.Spec.Version:
				// Activate or deactivate the collection based on the collection's current desiredState.
				// The activateDefaultCollections setting in the CR instance's collection repository entry has
				// no influence here as that only sets a collection's initial state.
				if !strings.EqualFold(c.Spec.DesiredState, kabanerov1alpha1.CollectionDesiredStateActive) {
					c.Status.StatusMessage = "An invalid desiredState value of " + c.Spec.DesiredState + " was specified. The collection is activated by default."
				}

				// Activate the collection.
				err := reconcileActiveVersions(c, &matchingCollection.collection, r.client)
				if err != nil {
					return reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}, err
				}
				c.Status.ActiveLocation = matchingCollection.repositoryURL
				c.Status.Status = kabanerov1alpha1.CollectionDesiredStateActive
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

// A key to the pipeline use count map
type pipelineUseMapKey struct {
	url string
	digest string
}

// The value in the pipeline use count map
type pipelineUseMapValue struct {
	kabanerov1alpha1.PipelineStatus
	useCount int64
	manifests []CollectionAsset
	manifestError error
}

// A specific version of a pipeline zip in a specific version of a collection
type pipelineVersion struct {
	pipelineUseMapKey
	version string
}

func reconcileActiveVersions(collectionResource *kabanerov1alpha1.Collection, collection *Collection, c client.Client) error {
	// In practice right now there is one active version and one status.  But in preparation for the future we're going to
	// pretend there can be more than one.
	specList := []kabanerov1alpha1.CollectionSpec{collectionResource.Spec}
	statusList := []kabanerov1alpha1.CollectionStatus{collectionResource.Status}
	collectionList := []*Collection{}
	if collection != nil {
		collectionList = append(collectionList, collection)
	}
	
	ownerIsController := false
	newOwner := metav1.OwnerReference{
		APIVersion: collectionResource.TypeMeta.APIVersion,
		Kind:       collectionResource.TypeMeta.Kind,
		Name:       collectionResource.ObjectMeta.Name,
		UID:        collectionResource.ObjectMeta.UID,
		Controller: &ownerIsController,
	}

	newStatusList, err := reconcileActiveVersionsInternal(collectionResource.Namespace, newOwner, specList, statusList, collectionList, c)

	// In practice there will only be one status, until the Collection CRD is updated to support a list.
	if err == nil {
		collectionResource.Status = newStatusList[0]
	}

	// Fix up the version if it's inactive.  The multiple versions support tries to tell us which version is inactive
	// but the caller expects all versions to be inactive.
	if collectionResource.Status.Status == kabanerov1alpha1.CollectionDesiredStateInactive {
		collectionResource.Status.ActiveVersion = ""
	}
	
	return err
}

func reconcileActiveVersionsInternal(namespace string, assetOwner metav1.OwnerReference, specs []kabanerov1alpha1.CollectionSpec, statuses []kabanerov1alpha1.CollectionStatus, collections []*Collection, c client.Client) ([]kabanerov1alpha1.CollectionStatus, error) {
	renderingContext := make(map[string]interface{})
	if len(collections) > 0 {
		renderingContext["CollectionId"] = collections[0].Id
		renderingContext["CollectionName"] = collections[0].Name
	}
	
	// Multiple versions of the same collection, could be using the same pipeline zip.  Count how many
	// times each pipeline has been used.
	assetUseMap := make(map[pipelineUseMapKey]*pipelineUseMapValue)
	for _, curStatus := range statuses {
		for _, pipeline := range curStatus.ActivePipelines {
			key := pipelineUseMapKey{url: pipeline.Url, digest: pipeline.Digest}
			value := assetUseMap[key]
			if value == nil {
				value = &pipelineUseMapValue{}
				pipeline.DeepCopyInto(&(value.PipelineStatus))
				assetUseMap[key] = value
			}
			value.useCount++
		}
	}

	// Reconcile the version changes.  Make a set of versions being removed, and versions being added.  Be
	// sure to take into consideration the digest on the individual pipeline zips.
	assetsToDecrement := make(map[pipelineVersion]bool)
	assetsToIncrement := make(map[pipelineVersion]bool)
	for _, curStatus := range statuses {
		for _, pipeline := range curStatus.ActivePipelines {
			cur := pipelineVersion{pipelineUseMapKey: pipelineUseMapKey{url: pipeline.Url, digest: pipeline.Digest}, version: curStatus.ActiveVersion}
			assetsToDecrement[cur] = true
		}
	}

	for _, curSpec := range specs {
		if !strings.EqualFold(curSpec.DesiredState, kabanerov1alpha1.CollectionDesiredStateInactive) {
			collection := getCollectionForSpecVersion(curSpec, collections)
			if collection == nil {
				// This version of the collection was not found in the collection hub.  See if it's currently active.  
				// If it is, we should continue to use what is active.
				//activeMatch := false
				for _, curStatus := range statuses {
					if curStatus.ActiveVersion == curSpec.Version {
						//activeMatch = true
						for _, pipeline := range curStatus.ActivePipelines {
							cur := pipelineVersion{pipelineUseMapKey: pipelineUseMapKey{url: pipeline.Url, digest: pipeline.Digest}, version: curStatus.ActiveVersion}
							if assetsToDecrement[cur] == true {
								delete(assetsToDecrement, cur)
							} else {
								assetsToIncrement[cur] = true
							}
						}
						break
					}
				}
			} else {
				for _, pipeline := range collection.Pipelines {
					cur := pipelineVersion{pipelineUseMapKey: pipelineUseMapKey{url: pipeline.Url, digest: pipeline.Sha256}, version: curSpec.Version}
					if assetsToDecrement[cur] == true {
						delete(assetsToDecrement, cur)
					} else {
						assetsToIncrement[cur] = true
					}
				}
			}
		}
	}
		
	// Now go thru the maps and update the use counts
	for cur, _ := range assetsToDecrement {
		value := assetUseMap[cur.pipelineUseMapKey]
		if value == nil {
			return nil, fmt.Errorf("Pipeline version not found in use map: %v", cur)
		}

		value.useCount--
	}

	for cur, _ := range assetsToIncrement {
		value := assetUseMap[cur.pipelineUseMapKey]
		if value == nil {
			// Need to add a new entry for this pipeline.
			value = &pipelineUseMapValue{PipelineStatus: kabanerov1alpha1.PipelineStatus{Url: cur.url, Digest: cur.digest}}
			assetUseMap[cur.pipelineUseMapKey] = value
		}

		value.useCount++
	}

	// Now iterate thru the asset use map and delete any assets with a use count of 0,
	// and create any assets with a positive use count.
	for _, value := range assetUseMap {
		if value.useCount <= 0 {
			log.Info(fmt.Sprintf("Deleting assets with use count %v: %v", value.useCount, value))

			for _, asset := range value.ActiveAssets {
				u := &unstructured.Unstructured{}
				u.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   asset.Group,
					Version: asset.Version,
					Kind:    asset.Kind,
				})

				err := c.Get(context.Background(), client.ObjectKey{
					Namespace: namespace,
					Name:      asset.Name,
				}, u)

				if err != nil {
					if errors.IsNotFound(err) == false {
						log.Error(err, fmt.Sprintf("Unable to check asset name %v", asset.Name))
						// TODO: Report in collection status somewhere?
					}
				} else {
					// Get the owner references.  See if we're the last one.
					ownerRefs := u.GetOwnerReferences()
					newOwnerRefs := []metav1.OwnerReference{}
					for _, ownerRef := range ownerRefs {
						if ownerRef.UID != assetOwner.UID {
							newOwnerRefs = append(newOwnerRefs, ownerRef)
						}
					}

					if len(newOwnerRefs) == 0 {
						err = c.Delete(context.TODO(), u)
						if err != nil {
							log.Error(err, fmt.Sprintf("Unable to delete asset name %v", asset.Name))
							// TODO: Report in collection status somewhere?
						}
					} else {
						u.SetOwnerReferences(newOwnerRefs)
						err = c.Update(context.TODO(), u)
						if err != nil {
							log.Error(err, fmt.Sprintf("Unable to delete owner reference from %v", asset.Name))
							// TODO: Report in collection status somewhere?
						}
					}
				}
			}
		}
	}

	for _, value := range assetUseMap {
		if value.useCount > 0 {
			log.Info(fmt.Sprintf("Creating assets with use count %v: %v", value.useCount, value))

			// Check to see if there is already an asset list.  If not, read the manifests and
			// create one.
			if len(value.ActiveAssets) == 0 {
				// Add the Digest to the rendering context.
				renderingContext["Digest"] = value.Digest[0:8]

				// Retrieve manifests as unstructured.  If we could not get them, skip.
				manifests, err := GetManifests(value.Url, value.Digest, renderingContext, log)
				if err != nil {
					log.Error(err, fmt.Sprintf("Error retrieving manifests at %v", value.Url))
					value.manifestError = err
					continue
				}

				// Save the manifests for later.
				value.manifests = manifests
				
				// Create the asset status slice, but don't apply anything yet.
				for _, asset := range manifests {
					value.ActiveAssets = append(value.ActiveAssets, kabanerov1alpha1.RepositoryAssetStatus{
						Name: asset.Name,
						Group: asset.Group,
					  Version: asset.Version,
					  Kind: asset.Kind,
						Digest: asset.Sha256,
						Status: assetStatusUnknown,
						StatusMessage: "Asset has not been applied yet.",
					})
				}
			}

			// Now go thru the asset list and see if the objects are there.  If not, create them.
			for index, asset := range value.ActiveAssets {
				u := &unstructured.Unstructured{}
				u.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   asset.Group,
					Version: asset.Version,
					Kind:    asset.Kind,
				})

				err := c.Get(context.Background(), client.ObjectKey{
					Namespace: namespace,
					Name:      asset.Name,
				}, u)

				if err != nil {
					if errors.IsNotFound(err) == false {
						log.Error(err, fmt.Sprintf("Unable to check asset name %v", asset.Name))
						// TODO: Report in collection status somewhere?
					} else {
						// Make sure the manifests are loaded.
						if len(value.manifests) == 0 {
							// Add the Digest to the rendering context.
							renderingContext["Digest"] = value.Digest[0:8]

							// Retrieve manifests as unstructured
							manifests, err := GetManifests(value.Url, value.Digest, renderingContext, log)
							if err != nil {
								log.Error(err, fmt.Sprintf("Object %v not found and manifests not available at %v", asset.Name, value.Url))
								value.ActiveAssets[index].Status = assetStatusFailed
								value.ActiveAssets[index].StatusMessage = "Manifests are no longer available at specified URL"
							} else {
								// Save the manifests for later.
								value.manifests = manifests
							}
						}

						// Now find the correct manifest and create the object
						for _, manifest := range value.manifests {
							if asset.Name == manifest.Name {
								resources := []unstructured.Unstructured{manifest.Yaml}
								m, err := mf.FromResources(resources, c)

								log.Info(fmt.Sprintf("Resources: %v", m.Resources))

								transforms := []mf.Transformer{
									transforms.InjectOwnerReference(assetOwner), 
									mf.InjectNamespace(namespace),
								}

								err = m.Transform(transforms...)
								if err != nil {
									log.Error(err, fmt.Sprintf("Error transforming manifests for %v", asset.Name))
									value.ActiveAssets[index].Status = assetStatusFailed
									value.ActiveAssets[index].Status = err.Error()
								} else {
									log.Info(fmt.Sprintf("Applying resources: %v", m.Resources))
									err = m.ApplyAll()
									if err != nil {
										// Update the asset status with the error message
										log.Error(err, "Error installing the resource", "resource", asset.Name)
										value.ActiveAssets[index].Status = assetStatusFailed
										value.ActiveAssets[index].StatusMessage = err.Error()
									} else {
										value.ActiveAssets[index].Status = assetStatusActive
										value.ActiveAssets[index].StatusMessage = ""
									}
								}
							}
						}
					}
				} else {
					// Add owner reference
					ownerRefs := u.GetOwnerReferences()
					foundOurselves := false
					for _, ownerRef := range ownerRefs {
						if ownerRef.UID == assetOwner.UID {
							foundOurselves = true
						}
					}

					if foundOurselves == false {

						// There can only be one 'controller' reference, so additional references should not
						// be controller references.  It's not clear what Kubernetes does with this field.
						ownerRefs = append(ownerRefs, assetOwner)
						u.SetOwnerReferences(ownerRefs)

						err = c.Update(context.TODO(), u)
						if err != nil {
							log.Error(err, fmt.Sprintf("Unable to add owner reference to %v", asset.Name))
							// TODO: Report in collection status somewhere?
						}
					}
					
					value.ActiveAssets[index].Status = assetStatusActive
					value.ActiveAssets[index].StatusMessage = ""
				}
			}
		}
	}
	
	// Now update the CollectionStatus to reflect the current state of things.
	newCollectionStatusList := []kabanerov1alpha1.CollectionStatus{}
	for _, curSpec := range specs {
		newCollectionStatus := kabanerov1alpha1.CollectionStatus{ActiveVersion: curSpec.Version}
		if !strings.EqualFold(curSpec.DesiredState, kabanerov1alpha1.CollectionDesiredStateInactive) {
			if !strings.EqualFold(curSpec.DesiredState, kabanerov1alpha1.CollectionDesiredStateActive) {
				newCollectionStatus.StatusMessage = "An invalid desiredState value of " + curSpec.DesiredState + " was specified. The collection is activated by default."
			}
			newCollectionStatus.Status = kabanerov1alpha1.CollectionDesiredStateActive
			collection := getCollectionForSpecVersion(curSpec, collections)
			if collection != nil {
				for _, pipeline := range collection.Pipelines {
					key := pipelineUseMapKey{url: pipeline.Url, digest: pipeline.Sha256}
					value := assetUseMap[key]
					if value == nil {
						// TODO: ???
					} else {
						newStatus := kabanerov1alpha1.PipelineStatus{}
						value.DeepCopyInto(&newStatus)
						newStatus.Name = pipeline.Id // This may vary by collection version
						newCollectionStatus.ActivePipelines = append(newCollectionStatus.ActivePipelines, newStatus)
						// If we had a problem loading the pipeline manifests, say so.
						if value.manifestError != nil {
							newCollectionStatus.StatusMessage = value.manifestError.Error()
						}
					}
				}

				// Update the status of the Collection object to reflect the images used
				for _, image := range collection.Images {
					newCollectionStatus.Images = append(newCollectionStatus.Images,
						kabanerov1alpha1.Image{Id: image.Id, Image: image.Image})
				}
			} else {
				// Collection was not available, need to get pipeline information from previous status.
				foundPrevStatus := false
				for _, curStatus := range statuses {
					if curStatus.ActiveVersion == curSpec.Version {
						foundPrevStatus = true
						for _, pipeline := range curStatus.ActivePipelines {
							key := pipelineUseMapKey{url: pipeline.Url, digest: pipeline.Digest}
							value := assetUseMap[key]
							if value == nil {
								// TODO: ???
							} else {
								newStatus := kabanerov1alpha1.PipelineStatus{}
								value.DeepCopyInto(&newStatus)
								newStatus.Name = pipeline.Name
								newCollectionStatus.ActivePipelines = append(newCollectionStatus.ActivePipelines, newStatus)
								// If we had a problem loading the pipeline manifests, say so.
								if value.manifestError != nil {
									newCollectionStatus.StatusMessage = value.manifestError.Error()
								}
							}
						}
						newCollectionStatus.Status = curStatus.Status
						newCollectionStatus.Images = curStatus.Images
					}
				}

				// If there was no previous status, then the collection is inactive.
				if foundPrevStatus == false {
					newCollectionStatus.Status = kabanerov1alpha1.CollectionDesiredStateInactive
				}
				
				// Tell the user that the collection was not in the hub, if no other errors
				if newCollectionStatus.StatusMessage == "" {
						newCollectionStatus.StatusMessage = fmt.Sprintf("The requested version of the collection (%v) is not available at %v", curSpec.Version, curSpec.RepositoryUrl)
				}
			}
		} else {
			newCollectionStatus.Status = kabanerov1alpha1.CollectionDesiredStateInactive
			newCollectionStatus.StatusMessage = "The collection has been deactivated."
		}

		log.Info(fmt.Sprintf("Updated collection status: %#v", newCollectionStatus))
		newCollectionStatusList = append(newCollectionStatusList, newCollectionStatus)
	}
	
	return newCollectionStatusList, nil
}

func getCollectionForSpecVersion(spec kabanerov1alpha1.CollectionSpec, collections []*Collection) *Collection {
	for _, collection := range collections {
		if collection.Version == spec.Version {
			return collection
		}
	}
	return nil
}
