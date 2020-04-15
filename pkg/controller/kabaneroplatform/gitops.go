package kabaneroplatform

import (
	"context"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	cutils "github.com/kabanero-io/kabanero-operator/pkg/controller/utils"
	"github.com/go-logr/logr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Activates the Gitops pipelines
func reconcileGitopsPipelines(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {
	reqLogger.Info("Reconciling Gitops pipelines.")

	// Gather the known asset (*-tasks, *-pipeline) substitution data.  (none presently)
	renderingContext := make(map[string]interface{})

	// Identify the owner of the pipeline resources
	ownerIsController := false
	assetOwner := metav1.OwnerReference{
		APIVersion: k.TypeMeta.APIVersion,
		Kind:       k.TypeMeta.Kind,
		Name:       k.ObjectMeta.Name,
		UID:        k.ObjectMeta.UID,
		Controller: &ownerIsController,
	}

	// Activate the pipelines used by the gitops repository
	assetUseMap, err := cutils.ActivatePipelines(k.Spec.Gitops, k.Status.Gitops, k.GetNamespace(), renderingContext, assetOwner, c, reqLogger)

	if err != nil {
		return err
	}
	
	// Now update the GitopsStatus to reflect the current state of things.
	newGitopsStatus := kabanerov1alpha2.GitopsStatus{Ready: "True"}
	for _, pipeline := range k.Spec.Gitops.Pipelines {
		key := cutils.PipelineUseMapKey{Url: pipeline.Https.Url, GitRelease: pipeline.GitRelease, Digest: pipeline.Sha256}
		value := assetUseMap[key]
		if value == nil {
			// TODO: ???
		} else {
			newStatus := kabanerov1alpha2.PipelineStatus{}
			value.DeepCopyInto(&newStatus)
			newStatus.Name = pipeline.Id
			newGitopsStatus.Pipelines = append(newGitopsStatus.Pipelines, newStatus)
			// If we had a problem loading the pipeline manifests, say so.
			if value.ManifestError != nil {
				newGitopsStatus.Message = value.ManifestError.Error()
			}
		}
	}

	// Troll thru the pipeline assets, if any are not active then update the status.
	for _, pipeline := range newGitopsStatus.Pipelines {
		for _, asset := range pipeline.ActiveAssets {
			if asset.Status != "active" {
				newGitopsStatus.Ready = "False"
			}
		}
	}

	if len(newGitopsStatus.Message) != 0 {
		newGitopsStatus.Ready = "False"
	}
	
	k.Status.Gitops = newGitopsStatus

	return nil
}

// Removes the cross-namespace objects created during the gitops pipelines deployment
func cleanupGitopsPipelines(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {
	reqLogger.Info("Removing Gitops pipelines.")

	ownerIsController := false
	assetOwner := metav1.OwnerReference{
		APIVersion: k.APIVersion,
		Kind:       k.Kind,
		Name:       k.Name,
		UID:        k.UID,
		Controller: &ownerIsController,
	}

	// Run thru the status and delete everything.... we're just going to try once since it's unlikely
	// that anything that goes wrong here would be rectified by a retry.
	for _, pipeline := range k.Status.Gitops.Pipelines {
		for _, asset := range pipeline.ActiveAssets {
			// Old assets may not have a namespace set - correct that now.
			if len(asset.Namespace) == 0 {
				asset.Namespace = k.GetNamespace()
			}
			
			cutils.DeleteAsset(c, asset, assetOwner, reqLogger)
		}
	}

	return nil
}

// Returns the readiness status of the Gitops pipelines.  Presently the status is determined
// when the pipelines are activated.  We are just reporting that status here.
func getGitopsStatus(k *kabanerov1alpha2.Kabanero) (bool, error) {
	return k.Status.Gitops.Ready == "True", nil
}
