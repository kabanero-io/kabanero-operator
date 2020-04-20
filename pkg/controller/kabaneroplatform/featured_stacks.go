package kabaneroplatform

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform/utils"
	cutils "github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform/utils"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/stack"
	sutils "github.com/kabanero-io/kabanero-operator/pkg/controller/stack/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func reconcileFeaturedStacks(ctx context.Context, k *kabanerov1alpha2.Kabanero, cl client.Client, reqLogger logr.Logger) error {
	// Before we attempt to read the stacks, validate that the stack policy, if defined, is supported.
	valid, reason, err := cutils.ValidateGovernanceStackPolicy(k)
	if !valid {
		return fmt.Errorf(reason)
	}

	// Resolve the stacks which are currently featured across the various indexes.
	stackMap, err := featuredStacks(k, cl, reqLogger)
	if err != nil {
		return err
	}

	// Clean existing stacks based on the stacks read from the repository index(es).
	err = preProcessCurrentStacks(ctx, k, cl, stackMap)
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

		alreadyDeployed := true
		stackResource := &kabanerov1alpha2.Stack{}
		err := cl.Get(ctx, name, stackResource)

		pipelinesNamespace := pipelinesNamespace(k)

		if err == nil {
			// Ensure the featured stack pipelinesNamespace = Kabanero pipelinesNamespace
			stackResource.Spec.PipelinesNamespace = pipelinesNamespace
		} else {
			if errors.IsNotFound(err) {
				alreadyDeployed = false
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
						PipelinesNamespace: pipelinesNamespace,
					},
				}
			} else {
				return err
			}
		}

		// Add each version to the versions array if it's not already there.  If it's already there, just
		// update the repository URL, don't touch the desired state.
		for i, stack := range value {
			// Remove the tag portion of all images associated with the input stack version.
			err := sutils.RemoveTagFromStackImages(&stack, key)
			if err != nil {
				return err
			}
			value[i].Images = stack.Images

			foundVersion := false
			for j, stackVersion := range stackResource.Spec.Versions {
				if stackVersion.Version == stack.Version {
					foundVersion = true
					// Per the new defintion of desired state, do not update any existing stacks if the desired state is set
					// to an allowed value.
					if !(alreadyDeployed && len(stackVersion.DesiredState) > 0) {
						stackVersion.Pipelines = stack.Pipelines
						stackVersion.SkipCertVerification = stack.SkipCertVerification
						stackVersion.SkipRegistryCertVerification = stack.SkipRegistryCertVerification
						stackVersion.Images = stack.Images
						stackResource.Spec.Versions[j] = stackVersion
					}
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
func featuredStacks(k *kabanerov1alpha2.Kabanero, cl client.Client, reqLogger logr.Logger) (map[string][]kabanerov1alpha2.StackVersion, error) {
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
			indexPipelines = append(indexPipelines, stack.Pipelines{Id: pipeline.Id, Sha256: pipeline.Sha256, Url: pipeline.Https.Url, GitRelease: pipeline.GitRelease, SkipCertVerification: pipeline.Https.SkipCertVerification})
		}

		index, err := stack.ResolveIndex(cl, r, k.Namespace, indexPipelines, []stack.Trigger{}, "", reqLogger)
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
				pipelines = append(pipelines, kabanerov1alpha2.PipelineSpec{Id: pipeline.Id, Sha256: pipeline.Sha256, Https: pipelineUrl, GitRelease: pipeline.GitRelease})
			}

			// The image information will be in the stack.  Today we just support reading the legacy field from the collection hub.
			images := []kabanerov1alpha2.Image{}
			for _, image := range c.Images {
				images = append(images, kabanerov1alpha2.Image{Id: image.Id, Image: image.Image})
			}

			stackMap[c.Id] = append(stackMap[c.Id], kabanerov1alpha2.StackVersion{Pipelines: pipelines, Version: c.Version, Images: images, SkipRegistryCertVerification: k.Spec.Stacks.SkipRegistryCertVerification})
		}
	}

	return stackMap, nil
}

// Cleans up currently deployed stacks based on desired state. Stack versions with an non-empty state must be preserved and not modified.
func preProcessCurrentStacks(ctx context.Context, k *kabanerov1alpha2.Kabanero, cl client.Client, indexStackMap map[string][]kabanerov1alpha2.StackVersion) error {
	deployedStacks := &kabanerov1alpha2.StackList{}
	err := cl.List(ctx, deployedStacks, client.InNamespace(k.GetNamespace()))
	if err != nil {
		return err
	}

	// Only keep the FeaturedStack if the Kabanero pipelinesNamespace did not change, otherwise delete & recreate
	pipelinesNamespace := pipelinesNamespace(k)
	for _, deployedStack := range deployedStacks.Items {
		if deployedStack.Spec.PipelinesNamespace != pipelinesNamespace {
			err := cl.Delete(ctx, &deployedStack)
			if err != nil {
				return err
			}
		}
	}

	deployedStacks = &kabanerov1alpha2.StackList{}
	err = cl.List(ctx, deployedStacks, client.InNamespace(k.GetNamespace()))
	if err != nil {
		return err
	}

	// Compare the list of currently deployed stacks and the stacks in the index.
	for _, deployedStack := range deployedStacks.Items {

		iStackList, _ := indexStackMap[deployedStack.GetName()]
		newStackVersions := []kabanerov1alpha2.StackVersion{}
		for _, dStackVersion := range deployedStack.Spec.Versions {
			deployedStackVersionMatchIndex := false
			// Keep the stacks with matching versions. The caller will do the updates if necessary.
			for _, iStack := range iStackList {
				if dStackVersion.Version == iStack.Version {
					deployedStackVersionMatchIndex = true
					newStackVersions = append(newStackVersions, dStackVersion)
					break
				}
			}

			// Keep any stack versions that have a desired state that is not empty.
			if !deployedStackVersionMatchIndex && len(dStackVersion.DesiredState) > 0 {
				newStackVersions = append(newStackVersions, dStackVersion)
				continue
			}
		}

		// If there were no indications that the stack should be kept around, delete it.
		if len(newStackVersions) == 0 {
			err := cl.Delete(ctx, &deployedStack)
			if err != nil {
				return err
			}
			break
		}

		// If there were differences between the deployed list of versions and the list of deployed versions that need to be kept,
		// update the current stack.
		if len(deployedStack.Spec.Versions) != len(newStackVersions) {
			deployedStack.Spec.Versions = newStackVersions
			cl.Update(ctx, &deployedStack)
		}
	}

	return nil
}
