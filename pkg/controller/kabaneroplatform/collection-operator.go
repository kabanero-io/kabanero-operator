package kabaneroplatform

import (
	"context"
	"strings"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	mf "github.com/kabanero-io/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rlog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var cclog = rlog.Log.WithName("collection-operator-install")

const (
	ccVersionSoftCompName   = "collection-operator"
	ccOrchestrationFileName = "collection-operator.yaml"

	ccDeploymentResourceName = "kabanero-collection-operator"
)

// Installs the Kabanero collection operator.
func reconcileCollectionOperator(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	logger := chelog.WithValues("Kabanero instance namespace", k.Namespace, "Kabanero instance Name", k.Name)
	logger.Info("Reconciling Kabanero collection operator installation.")

	// Deploy the Kabanero collection operator.
	rev, err := resolveSoftwareRevision(k, ccVersionSoftCompName, k.Spec.CollectionOperator.Version)
	if err != nil {
		logger.Error(err, "Kabanero collection operator deloyment failed. Unable to resolve software revision.")
		return err
	}

	templateCtx := rev.Identifiers
	image, err := imageUriWithOverrides(k.Spec.CollectionOperator.Repository, k.Spec.CollectionOperator.Tag, k.Spec.CollectionOperator.Image, rev)
	if err != nil {
		logger.Error(err, "Kabanero collection operator deloyment failed. Unable to process image overrides.")
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

	return nil
}

// Returns the readiness status of the Kabanero collection operator installation.
func getCollectionControllerStatus(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	k.Status.CollectionOperator.ErrorMessage = ""
	k.Status.CollectionOperator.Ready = "False"

	// Retrieve the Kabanero collection operator version.
	rev, err := resolveSoftwareRevision(k, ccVersionSoftCompName, k.Spec.CollectionOperator.Version)
	if err != nil {
		message := "Unable to retrieve the collection controller version."
		kanlog.Error(err, message)
		k.Status.CollectionOperator.ErrorMessage = message + ": " + err.Error()
		return false, err
	}
	k.Status.CollectionOperator.Version = rev.Version

	// Base the status on the Kabanero collection operator's deployment resource.
	ccdeployment := &appsv1.Deployment{}
	err = c.Get(ctx, client.ObjectKey{
		Name:      ccDeploymentResourceName,
		Namespace: k.ObjectMeta.Namespace}, ccdeployment)

	if err != nil {
		message := "Unable to retrieve the Kabanero collection operator deployment object."
		kanlog.Error(err, message)
		k.Status.CollectionOperator.ErrorMessage = message + ": " + err.Error()
		return false, err
	}

	conditions := ccdeployment.Status.Conditions
	ready := false
	for _, condition := range conditions {
		if strings.ToLower(string(condition.Type)) == "available" {
			if strings.ToLower(string(condition.Status)) == "true" {
				ready = true
				k.Status.CollectionOperator.Ready = "True"
			} else {
				k.Status.CollectionOperator.ErrorMessage = condition.Message
			}

			break
		}
	}

	return ready, err
}
