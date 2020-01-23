package collection

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"

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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_collection")

const (
	// Finalizer.
	collectionFinalizerName = "kabanero.io/collection-controller"
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

	// The indexResolver which will be used during reconciliation
	indexResolver func(kabanerov1alpha1.RepositoryConfig, []Pipelines, []Trigger, string) (*Index, error)
}

// Reconcile processes collection resource instances.
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
// NOTE:
// Collection resources are deprecated as of Kabanero Operator version 0.6.0 and are replaced with Stack resources.
// Therefore, the function of this controller is to handle collection migration to stacks and collection deletion.
func (r *ReconcileCollection) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Collection")

	// Fetch the Collection instance
	instance := &kabanerov1alpha1.Collection{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Resolve the kabanero instance
	var k *kabanerov1alpha1.Kabanero
	l := kabanerov1alpha1.KabaneroList{}
	err = r.client.List(context.Background(), &l, client.InNamespace(instance.GetNamespace()))
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, _k := range l.Items {
		k = &_k
	}
	reqLogger.Info("Resolved Kabanero", "kabanero", k)

	// If the collection is being deleted, and our finalizer is set, process it.
	beingDeleted, err := processDeletion(ctx, instance, r.client, reqLogger)
	if err != nil {
		return reconcile.Result{}, err
	}

	if beingDeleted {
		return reconcile.Result{}, nil
	}

	// The collection is not being deleted. Migrate the collection to a stack.
	collectionName := instance.ObjectMeta.Name
	collectionNamespace := instance.ObjectMeta.Namespace

	// Before we convert the collection instance to a stack instance, see if
	// the stack with that same name in the same namespace is already available.
	stack := kabanerov1alpha2.Stack{}
	name := types.NamespacedName{
		Name:      collectionName,
		Namespace: collectionNamespace,
	}

	err = r.client.Get(context.TODO(), name, &stack)
	if err != nil {
		if errors.IsNotFound(err) {
			stackInstance, err := r.convertCollectionToStack(k, instance)

			if err != nil {
				reqLogger.Error(err, fmt.Sprintf("Unable convert collection with the name of %v to stack.", collectionName))
				return reconcile.Result{}, err
			}

			err = r.client.Create(context.TODO(), stackInstance)
			if err != nil {
				reqLogger.Error(err, fmt.Sprintf("Unable create a stack from collection with the name of %v.", collectionName))
				return reconcile.Result{}, err
			}
		} else {
			return reconcile.Result{}, err
		}
	}

	// Clean up existing collection assets.
	err = deleteCollection(r.client, instance, reqLogger)
	if err != nil {
		reqLogger.Error(err, fmt.Sprintf("Unable delete collection with the name of %v.", collectionName))
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// Converts a collection to a stack.
func (r *ReconcileCollection) convertCollectionToStack(k *kabanerov1alpha1.Kabanero, collection *kabanerov1alpha1.Collection) (*kabanerov1alpha2.Stack, error) {
	ownerIsController := true
	stackInstance := &kabanerov1alpha2.Stack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      collection.ObjectMeta.Name,
			Namespace: collection.ObjectMeta.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: k.TypeMeta.APIVersion,
					Kind:       k.TypeMeta.Kind,
					Name:       k.ObjectMeta.Name,
					UID:        k.ObjectMeta.UID,
					Controller: &ownerIsController,
				},
			},
		},
		Spec: kabanerov1alpha2.StackSpec{
			Name: collection.Spec.Name,
		},
	}

	// Add the versions field.
	cCopy := collection.DeepCopy()
	if len(collection.Spec.Versions) == 0 && len(collection.Spec.Version) != 0 {
		cCopy.Spec.Versions = []kabanerov1alpha1.CollectionVersion{{
			RepositoryUrl:        collection.Spec.RepositoryUrl,
			Version:              collection.Spec.Version,
			DesiredState:         collection.Spec.DesiredState,
			SkipCertVerification: collection.Spec.SkipCertVerification,
		}}
	}
	err := r.processVersionsField(k, stackInstance, cCopy)
	if err != nil {
		return nil, err
	}

	return stackInstance, nil
}

// Adds collection information to the versions field of the input stack object.
func (r *ReconcileCollection) processVersionsField(k *kabanerov1alpha1.Kabanero, stackInstance *kabanerov1alpha2.Stack, collection *kabanerov1alpha1.Collection) error {
	stackInstance.Spec.Versions = make([]kabanerov1alpha2.StackVersion, len(collection.Spec.Versions))

	for i, cv := range collection.Spec.Versions {
		stackInstance.Spec.Versions[i] = kabanerov1alpha2.StackVersion{
			Version:              cv.Version,
			DesiredState:         cv.DesiredState,
			SkipCertVerification: cv.SkipCertVerification,
		}

		// Add pipelines and images entries to the stack's versions field.
		// This information is obtained from the collection's status fields. However, if there is reason to believe
		// that the data reported in the collection's status fields is not accurate or it does not exists, try to
		// get the information from the index specified in the kabanero instance.
		var images []kabanerov1alpha1.Image = []kabanerov1alpha1.Image{}
		var pipelines []kabanerov1alpha1.PipelineStatus = []kabanerov1alpha1.PipelineStatus{}
		for _, scvData := range collection.Status.Versions {
			if cv.Version == scvData.Version {
				images = scvData.Images
				pipelines = scvData.Pipelines
				break
			}
		}

		if collection.Status.Status != kabanerov1alpha1.CollectionDesiredStateActive || len(images) == 0 || len(pipelines) == 0 {
			collections, err := r.readCollectionsFromIndex(k)
			if err != nil {
				return err
			}

			cByNameMap, ok := collections[collection.Spec.Name]
			if ok {
				indexCollection, ok := cByNameMap[cv.Version]
				if ok {
					stackInstance.Spec.Versions[i].Images = make([]kabanerov1alpha2.Image, len(indexCollection.Images))
					for j, m := range indexCollection.Images {
						stackInstance.Spec.Versions[i].Images[j].Image = m.Image
						stackInstance.Spec.Versions[i].Images[j].Id = m.Id
					}

					stackInstance.Spec.Versions[i].Pipelines = make([]kabanerov1alpha2.PipelineSpec, len(indexCollection.Pipelines))
					for k, p := range indexCollection.Pipelines {
						stackInstance.Spec.Versions[i].Pipelines[k].Id = p.Id
						stackInstance.Spec.Versions[i].Pipelines[k].Sha256 = p.Sha256
						stackInstance.Spec.Versions[i].Pipelines[k].Url = p.Url
					}

					continue
				}
			}

			err = fmt.Errorf(fmt.Sprintf("Collection with the name of %v and version of %v was not found in kabanero configured indexes.", collection.Name, cv.Version))
			return err
		}

		// Copy data from the status section
		stackInstance.Spec.Versions[i].Images = make([]kabanerov1alpha2.Image, len(images))
		for j, m := range images {
			stackInstance.Spec.Versions[i].Images[j].Image = m.Image
			stackInstance.Spec.Versions[i].Images[j].Id = m.Id
		}
		stackInstance.Spec.Versions[i].Pipelines = make([]kabanerov1alpha2.PipelineSpec, len(pipelines))
		for k, p := range pipelines {
			stackInstance.Spec.Versions[i].Pipelines[k].Id = p.Name
			stackInstance.Spec.Versions[i].Pipelines[k].Sha256 = p.Digest
			stackInstance.Spec.Versions[i].Pipelines[k].Url = p.Url
		}
	}

	return nil
}

// Retrieves all collection objects associated with the kabanero specified indexes.
func (r *ReconcileCollection) readCollectionsFromIndex(k *kabanerov1alpha1.Kabanero) (map[string]map[string]Collection, error) {
	collectionMap := make(map[string]map[string]Collection)
	for _, repo := range k.Spec.Collections.Repositories {
		index, err := r.indexResolver(repo, []Pipelines{}, []Trigger{}, "")
		if err != nil {
			return nil, err
		}
		versionMap := make(map[string]Collection)
		for i, c := range index.Collections {
			versionMap[index.Collections[i].Version] = c
			collectionMap[c.Id] = versionMap
		}
	}

	return collectionMap, nil
}

// Deletes the specified collection.
func deleteCollection(c client.Client, collection *kabanerov1alpha1.Collection, reqLogger logr.Logger) error {
	// Clean up existing collection assets.
	err := cleanupAssets(context.TODO(), collection, c, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Error during cleanup processing.")
		return err
	}

	// Remove the finalizer entry from the instance.
	err = removeCollectionFinalizer(context.TODO(), collection, c, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Error while attempting to remove the finalizer.")
		return err
	}

	// Delete the collection.
	err = c.Delete(context.TODO(), collection)
	if err != nil {
		reqLogger.Error(err, "Error while attempting to remove the finalizer.")
		return err
	}

	return nil
}

// Drives collection instance deletion processing. This includes creating a finalizer, handling
// collection instance cleanup logic, and finalizer removal.
func processDeletion(ctx context.Context, collection *kabanerov1alpha1.Collection, c client.Client, reqLogger logr.Logger) (bool, error) {
	// The collection instance is not deleted. Create a finalizer if it was not created already.
	foundFinalizer := false
	for _, finalizer := range collection.Finalizers {
		if finalizer == collectionFinalizerName {
			foundFinalizer = true
		}
	}

	beingDeleted := !collection.DeletionTimestamp.IsZero()
	if !beingDeleted {
		if !foundFinalizer {
			collection.Finalizers = append(collection.Finalizers, collectionFinalizerName)
			err := c.Update(ctx, collection)
			if err != nil {
				reqLogger.Error(err, "Unable to set the collection controller finalizer.")
				return beingDeleted, err
			}
		}

		return beingDeleted, nil
	}

	// The instance is being deleted.
	if foundFinalizer {
		// Drive collection cleanup processing.
		err := cleanupAssets(ctx, collection, c, reqLogger)
		if err != nil {
			reqLogger.Error(err, "Error during cleanup processing.")
			return beingDeleted, err
		}

		// Remove the finalizer entry from the instance.
		err = removeCollectionFinalizer(ctx, collection, c, reqLogger)
		if err != nil {
			reqLogger.Error(err, "Error while attempting to remove the finalizer.")
			return beingDeleted, err
		}
	}

	return beingDeleted, nil
}

// Removes the collection finalizer.
func removeCollectionFinalizer(ctx context.Context, collection *kabanerov1alpha1.Collection, c client.Client, reqLogger logr.Logger) error {
	var newFinalizerList []string
	for _, finalizer := range collection.Finalizers {
		if finalizer == collectionFinalizerName {
			continue
		}
		newFinalizerList = append(newFinalizerList, finalizer)
	}

	collection.Finalizers = newFinalizerList
	err := c.Update(ctx, collection)

	return err
}

// Handles the finalizer cleanup logic for the Collection instance.
func cleanupAssets(ctx context.Context, collection *kabanerov1alpha1.Collection, c client.Client, reqLogger logr.Logger) error {
	ownerIsController := false
	assetOwner := metav1.OwnerReference{
		APIVersion: collection.APIVersion,
		Kind:       collection.Kind,
		Name:       collection.Name,
		UID:        collection.UID,
		Controller: &ownerIsController,
	}

	// Run thru the status and delete everything.... we're just going to try once since it's unlikely
	// that anything that goes wrong here would be rectified by a retry.
	for _, version := range collection.Status.Versions {
		for _, pipeline := range version.Pipelines {
			for _, asset := range pipeline.ActiveAssets {
				// Old assets may not have a namespace set - correct that now.
				if len(asset.Namespace) == 0 {
					asset.Namespace = collection.GetNamespace()
				}

				deleteAsset(c, asset, assetOwner)
			}
		}
	}

	return nil
}

// Deletes an asset.  This can mean removing an object owner, or completely deleting it.
func deleteAsset(c client.Client, asset kabanerov1alpha1.RepositoryAssetStatus, assetOwner metav1.OwnerReference) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   asset.Group,
		Version: asset.Version,
		Kind:    asset.Kind,
	})

	err := c.Get(context.Background(), client.ObjectKey{
		Namespace: asset.Namespace,
		Name:      asset.Name,
	}, u)

	if err != nil {
		if errors.IsNotFound(err) == false {
			log.Error(err, fmt.Sprintf("Unable to check asset name %v", asset.Name))
			return err
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
				return err
			}
		} else {
			u.SetOwnerReferences(newOwnerRefs)
			err = c.Update(context.TODO(), u)
			if err != nil {
				log.Error(err, fmt.Sprintf("Unable to delete owner reference from %v", asset.Name))
				return err
			}
		}
	}

	return nil
}
