package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	mf "github.com/kabanero-io/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rlog "sigs.k8s.io/controller-runtime/pkg/log"
)

var cclog = rlog.Log.WithName("collection-controller-install")

const (
	ccVersionSoftCompName   = "collection-controller"
	ccOrchestrationFileName = "collection-controller.yaml"

	ccDeploymentResourceName = "kabanero-operator-collection-controller"
)

// Installs the Kabanero collection controller.
func reconcileCollectionController(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	logger := chelog.WithValues("Kabanero instance namespace", k.Namespace, "Kabanero instance Name", k.Name)
	logger.Info("Reconciling Kabanero collection controller installation.")

	// Deploy the Kabanero collection operator.
	rev, err := resolveSoftwareRevision(k, ccVersionSoftCompName, k.Spec.CollectionController.Version)
	if err != nil {
		logger.Error(err, "Kabanero collection controller deloyment failed. Unable to resolve software revision.")
		return err
	}

	templateCtx := rev.Identifiers
	image, err := imageUriWithOverrides(k.Spec.CollectionController.Repository, k.Spec.CollectionController.Tag, k.Spec.CollectionController.Image, rev)
	if err != nil {
		logger.Error(err, "Kabanero collection controller deloyment failed. Unable to process image overrides.")
		return err
	}
	templateCtx["image"] = image

	f, err := rev.OpenOrchestration(ccOrchestrationFileName)
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateCtx)
	if err != nil {
		return err
	}

	m, err := mf.FromReader(strings.NewReader(s), c)
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
	}

	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.ApplyAll()
	if err != nil {
		return err
	}

	// Create a RoleBinding in the tekton-pipelines namespace that will allow
	// the collection controller to create triggerbinding and triggertemplate
	// objects in the tekton-pipelines namespace.
	templateCtx["name"] = "kabanero-" + k.GetNamespace() + "-trigger-rolebinding"
	templateCtx["kabaneroNamespace"] = k.GetNamespace()

	f, err = rev.OpenOrchestration("collection-controller-tekton.yaml")
	if err != nil {
		return err
	}

	s, err = renderOrchestration(f, templateCtx)
	if err != nil {
		return err
	}

	m, err = mf.FromReader(strings.NewReader(s), c)
	if err != nil {
		return err
	}

	err = m.ApplyAll()
	if err != nil {
		return err
	}
	
	return nil
}

// Removes the cross-namespace objects created during the collection controller
// deployment.
func cleanupCollectionController(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	logger := chelog.WithValues("Kabanero instance namespace", k.Namespace, "Kabanero instance Name", k.Name)
	logger.Info("Removing Kabanero collection controller installation.")

	// First, we need to delete all of the collections that we own.  We must do this first, to let the
	// collection controller run its finalizer for all of the collections, before deleting the
	// collection controller pods etc.
	collectionList := &kabanerov1alpha1.CollectionList{}
	err := c.List(ctx, collectionList, client.InNamespace(k.GetNamespace()))
	if err != nil {
		return fmt.Errorf("Unable to list collections in finalizer: %v", err.Error())
	}

	collectionCount := 0
	for _, collection := range collectionList.Items {
		for _, ownerRef := range collection.OwnerReferences {
			if ownerRef.UID == k.UID {
				collectionCount = collectionCount + 1
				if collection.DeletionTimestamp.IsZero() {
					err = c.Delete(ctx, &collection)
					if err != nil {
						// Just log the error... but continue on to the next object.
						logger.Error(err, "Unable to delete collection %v", collection.Name)
					}
				}
			}
		}
	}

	// If there are still some collections left, need to come back and try again later...
	if collectionCount > 0 {
		return fmt.Errorf("Deletion blocked waiting for %v owned Collections to be deleted", collectionCount)
	}

	// Now that the collections have all been deleted, proceed with the cross-namespace objects.
	// Objects in this namespace will be deleted implicitly when the Kabanero CR instance is
	// deleted, because of the OwnerReference in those objects.
	rev, err := resolveSoftwareRevision(k, ccVersionSoftCompName, k.Spec.CollectionController.Version)
	if err != nil {
		logger.Error(err, "Unable to resolve software revision.")
		return err
	}

	templateCtx := rev.Identifiers
	templateCtx["name"] = "kabanero-" + k.GetNamespace() + "-trigger-rolebinding"
	templateCtx["kabaneroNamespace"] = k.GetNamespace()

	f, err := rev.OpenOrchestration("collection-controller-tekton.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateCtx)
	if err != nil {
		return err
	}

	m, err := mf.FromReader(strings.NewReader(s), c)
	if err != nil {
		return err
	}

	err = m.DeleteAll()
	if err != nil {
		return err
	}

	return nil
}

// Returns the readiness status of the Kabanero collection controller installation.
func getCollectionControllerStatus(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	k.Status.CollectionController.ErrorMessage = ""
	k.Status.CollectionController.Ready = "False"

	// Retrieve the Kabanero collection controller version.
	rev, err := resolveSoftwareRevision(k, ccVersionSoftCompName, k.Spec.CollectionController.Version)
	if err != nil {
		message := "Unable to retrieve the collection controller version."
		kanlog.Error(err, message)
		k.Status.CollectionController.ErrorMessage = message + ": " + err.Error()
		return false, err
	}
	k.Status.CollectionController.Version = rev.Version

	// Base the status on the Kabanero collection controller's deployment resource.
	ccdeployment := &appsv1.Deployment{}
	err = c.Get(ctx, client.ObjectKey{
		Name:      ccDeploymentResourceName,
		Namespace: k.ObjectMeta.Namespace}, ccdeployment)

	if err != nil {
		message := "Unable to retrieve the Kabanero collection controller deployment object."
		kanlog.Error(err, message)
		k.Status.CollectionController.ErrorMessage = message + ": " + err.Error()
		return false, err
	}

	conditions := ccdeployment.Status.Conditions
	ready := false
	for _, condition := range conditions {
		if strings.ToLower(string(condition.Type)) == "available" {
			if strings.ToLower(string(condition.Status)) == "true" {
				ready = true
				k.Status.CollectionController.Ready = "True"
			} else {
				k.Status.CollectionController.ErrorMessage = condition.Message
			}

			break
		}
	}

	return ready, err
}
