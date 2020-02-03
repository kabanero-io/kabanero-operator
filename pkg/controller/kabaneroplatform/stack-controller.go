package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	mf "github.com/kabanero-io/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rlog "sigs.k8s.io/controller-runtime/pkg/log"
)

var sclog = rlog.Log.WithName("stack-controller-install")

const (
	scVersionSoftCompName   = "stack-controller"
	scOrchestrationFileName = "stack-controller.yaml"

	scDeploymentResourceName = "kabanero-operator-stack-controller"
)

// Installs the Kabanero stack controller.
func reconcileStackController(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client) error {
	logger := sclog.WithValues("Kabanero instance namespace", k.Namespace, "Kabanero instance Name", k.Name)
	logger.Info("Reconciling Kabanero stack controller installation.")

	// Deploy the Kabanero stack operator.
	rev, err := resolveSoftwareRevision(k, scVersionSoftCompName, k.Spec.StackController.Version)
	if err != nil {
		logger.Error(err, "Kabanero stack controller deployment failed. Unable to resolve software revision.")
		return err
	}

	templateCtx := rev.Identifiers
	image, err := imageUriWithOverrides(k.Spec.StackController.Repository, k.Spec.StackController.Tag, k.Spec.StackController.Image, rev)
	if err != nil {
		logger.Error(err, "Kabanero stack controller deployment failed. Unable to process image overrides.")
		return err
	}
	templateCtx["image"] = image

	f, err := rev.OpenOrchestration(scOrchestrationFileName)
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
	// the stack controller to create triggerbinding and triggertemplate
	// objects in the tekton-pipelines namespace.
	templateCtx["name"] = "kabanero-" + k.GetNamespace() + "-trigger-rolebinding"
	templateCtx["kabaneroNamespace"] = k.GetNamespace()

	f, err = rev.OpenOrchestration("stack-controller-tekton.yaml")
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

// Removes the cross-namespace objects created during the stack controller
// deployment.
func cleanupStackController(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client) error {
	logger := sclog.WithValues("Kabanero instance namespace", k.Namespace, "Kabanero instance Name", k.Name)
	logger.Info("Removing Kabanero stack controller installation.")

	// First, we need to delete all of the stacks that we own.  We must do this first, to let the
	// stack controller run its finalizer for all of the stacks, before deleting the
	// stack controller pods etc.
	stackList := &kabanerov1alpha2.StackList{}
	err := c.List(ctx, stackList, client.InNamespace(k.GetNamespace()))
	if err != nil {
		return fmt.Errorf("Unable to list stacks in finalizer: %v", err.Error())
	}

	stackCount := 0
	for _, stack := range stackList.Items {
		for _, ownerRef := range stack.OwnerReferences {
			if ownerRef.UID == k.UID {
				stackCount = stackCount + 1
				if stack.DeletionTimestamp.IsZero() {
					err = c.Delete(ctx, &stack)
					if err != nil {
						// Just log the error... but continue on to the next object.
						logger.Error(err, "Unable to delete stack %v", stack.Name)
					}
				}
			}
		}
	}

	// If there are still some stacks left, need to come back and try again later...
	if stackCount > 0 {
		return fmt.Errorf("Deletion blocked waiting for %v owned Stacks to be deleted", stackCount)
	}

	// Now that the stacks have all been deleted, proceed with the cross-namespace objects.
	// Objects in this namespace will be deleted implicitly when the Kabanero CR instance is
	// deleted, because of the OwnerReference in those objects.
	rev, err := resolveSoftwareRevision(k, scVersionSoftCompName, k.Spec.StackController.Version)
	if err != nil {
		logger.Error(err, "Unable to resolve software revision.")
		return err
	}

	templateCtx := rev.Identifiers
	templateCtx["name"] = "kabanero-" + k.GetNamespace() + "-trigger-rolebinding"
	templateCtx["kabaneroNamespace"] = k.GetNamespace()

	f, err := rev.OpenOrchestration("stack-controller-tekton.yaml")
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

// Returns the readiness status of the Kabanero stack controller installation.
func getStackControllerStatus(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client) (bool, error) {
	k.Status.StackController.ErrorMessage = ""
	k.Status.StackController.Ready = "False"

	// Retrieve the Kabanero stack controller version.
	rev, err := resolveSoftwareRevision(k, scVersionSoftCompName, k.Spec.StackController.Version)
	if err != nil {
		message := "Unable to retrieve the stack controller version."
		sclog.Error(err, message)
		k.Status.StackController.ErrorMessage = message + ": " + err.Error()
		return false, err
	}
	k.Status.StackController.Version = rev.Version

	// Base the status on the Kabanero stack controller's deployment resource.
	scdeployment := &appsv1.Deployment{}
	err = c.Get(ctx, client.ObjectKey{
		Name:      scDeploymentResourceName,
		Namespace: k.ObjectMeta.Namespace}, scdeployment)

	if err != nil {
		message := "Unable to retrieve the Kabanero stack controller deployment object."
		sclog.Error(err, message)
		k.Status.StackController.ErrorMessage = message + ": " + err.Error()
		return false, err
	}

	conditions := scdeployment.Status.Conditions
	ready := false
	for _, condition := range conditions {
		if strings.ToLower(string(condition.Type)) == "available" {
			if strings.ToLower(string(condition.Status)) == "true" {
				ready = true
				k.Status.StackController.Ready = "True"
			} else {
				k.Status.StackController.ErrorMessage = condition.Message
			}

			break
		}
	}

	return ready, err
}
