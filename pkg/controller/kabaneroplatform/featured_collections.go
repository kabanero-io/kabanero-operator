package kabaneroplatform

import (
	"context"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform/utils"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/stack"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func reconcileFeaturedStacks(ctx context.Context, k *kabanerov1alpha2.Kabanero, cl client.Client) error {
	// Resolve the stacks which are currently featured across the various indexes.
	stackMap, err := featuredStacks(k)
	if err != nil {
		return err
	}

	// Each key is a stack id.  Get that Stack CR instance and see if the versions are set correctly.
	for key, value := range stackMap {
		updateStack := utils.Update
		name := types.NamespacedName{
			Name:      key,
			Namespace: k.GetNamespace(),
		}

		stackResource := &kabanerov1alpha2.Stack{}
		err := cl.Get(ctx, name, stackResource)
		if err != nil {
			if errors.IsNotFound(err) {
				// Not found. Need to create it.
				updateStack = utils.Create
				ownerIsController := true
				stackResource = &kabanerov1alpha2.Stack{
					ObjectMeta: metav1.ObjectMeta{
						Name:      key,
						Namespace: k.GetNamespace(),
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
						Name: key,
					},
				}
			} else {
				return err
			}
		}

		// Add each version to the versions array if it's not already there.  If it's already there, just
		// update the repository URL, don't touch the desired state.
		for _, stack := range value {
			foundVersion := false
			for j, stackVersion := range stackResource.Spec.Versions {
				if stackVersion.Version == stack.Version {
					foundVersion = true
					stackVersion.Pipelines = stack.Pipelines
					stackVersion.SkipCertVerification = stack.SkipCertVerification
					stackVersion.Images = stack.Images
					stackResource.Spec.Versions[j] = stackVersion
				}
			}

			if foundVersion == false {
				stackResource.Spec.Versions = append(stackResource.Spec.Versions, stack)
			}
		}

		// Update the CR instance with the new version information.
		err = updateStack(cl, ctx, stackResource)
		if err != nil {
			return err
		}
	}

	return nil
}

// Resolves all stacks for the given Kabanero instance
func featuredStacks(k *kabanerov1alpha2.Kabanero) (map[string][]kabanerov1alpha2.StackVersion, error) {

	stackMap := make(map[string][]kabanerov1alpha2.StackVersion)
	for _, r := range k.Spec.Stacks.Repositories {
		// Figure out what set of pipelines to use.  The Kabanero instance defines a default
		// set, but this can be over-ridden by the specific repository.
		pipelines := r.Pipelines
		if len(pipelines) == 0 {
			pipelines = k.Spec.Stacks.Pipelines
		}

		indexPipelines := []stack.Pipelines{}
		for _, pipeline := range pipelines {
			indexPipelines = append(indexPipelines, stack.Pipelines{Id: pipeline.Id, Sha256: pipeline.Sha256, Url: pipeline.Https.Url, SkipCertVerification: pipeline.Https.SkipCertVerification})
		}

		index, err := stack.ResolveIndex(r, indexPipelines, []stack.Trigger{}, "")
		if err != nil {
			return nil, err
		}

		// Create the stack versions
		for _, c := range index.Stacks {
			// The pipeline information will be in the stack, either because this is a legacy hub and the information was already there, or
			// because we provided it at the time we read the appsody stack index (in ResolveIndex).
			pipelines := []kabanerov1alpha2.PipelineSpec{}
			for _, pipeline := range c.Pipelines {
				pipelineUrl := kabanerov1alpha2.HttpsProtocolFile{Url: pipeline.Url, SkipCertVerification: pipeline.SkipCertVerification}
				pipelines = append(pipelines, kabanerov1alpha2.PipelineSpec{Id: pipeline.Id, Sha256: pipeline.Sha256, Https: pipelineUrl})
			}
			// The image information will be in the stack.  Today we just support reading the legacy field from the collection hub.
			images := []kabanerov1alpha2.Image{}
			for _, image := range c.Images {
				images = append(images, kabanerov1alpha2.Image{Id: image.Id, Image: image.Image})
			}

			stackMap[c.Id] = append(stackMap[c.Id], kabanerov1alpha2.StackVersion{Pipelines: pipelines, Version: c.Version, Images: images})
		}
	}

	return stackMap, nil
}
