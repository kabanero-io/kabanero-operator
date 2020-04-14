package utils

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/transforms"
	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Asset status.
	assetStatusActive  = "active"
	assetStatusFailed  = "failed"
	assetStatusUnknown = "unknown"
)

// A key to the pipeline use count map
type PipelineUseMapKey struct {
	Url        string
	GitRelease kabanerov1alpha2.GitReleaseSpec
	Digest     string
}

// The value in the pipeline use count map
type PipelineUseMapValue struct {
	kabanerov1alpha2.PipelineStatus
	useCount      int64
	manifests     []StackAsset
	ManifestError error
}

type PipelineUseMap map[PipelineUseMapKey]*PipelineUseMapValue

// A specific version of a pipeline zip in a specific version of a stack
type pipelineVersion struct {
	PipelineUseMapKey
	version string
}


func ActivatePipelines(spec kabanerov1alpha2.ComponentSpec, status kabanerov1alpha2.ComponentStatus, targetNamespace string, renderingContext map[string]interface{}, assetOwner metav1.OwnerReference, c client.Client, logger logr.Logger) (PipelineUseMap, error) {

	// Multiple versions of the same stack, could be using the same pipeline zip.  Count how many
	// times each pipeline has been used.
	assetUseMap := make(PipelineUseMap)
	for _, curStatus := range status.GetVersions() {
		for _, pipeline := range curStatus.GetPipelines() {
			key := PipelineUseMapKey{Url: pipeline.Url, GitRelease: pipeline.GitRelease, Digest: pipeline.Digest}
			value := assetUseMap[key]
			if value == nil {
				value = &PipelineUseMapValue{}
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
	for _, curStatus := range status.GetVersions() {
		for _, pipeline := range curStatus.GetPipelines() {
			cur := pipelineVersion{PipelineUseMapKey: PipelineUseMapKey{Url: pipeline.Url, GitRelease: pipeline.GitRelease, Digest: pipeline.Digest}, version: curStatus.GetVersion()}
			assetsToDecrement[cur] = true
		}
	}

	for _, curSpec := range spec.GetVersions() {
		for _, pipeline := range curSpec.GetPipelines() {
			cur := pipelineVersion{PipelineUseMapKey: PipelineUseMapKey{Url: pipeline.Https.Url, GitRelease: pipeline.GitRelease, Digest: pipeline.Sha256}, version: curSpec.GetVersion()}
			if assetsToDecrement[cur] == true {
				delete(assetsToDecrement, cur)
			} else {
				assetsToIncrement[cur] = true
			}
		}
	}

	// Now go thru the maps and update the use counts
	for cur, _ := range assetsToDecrement {
		value := assetUseMap[cur.PipelineUseMapKey]
		if value == nil {
			return nil, fmt.Errorf("Pipeline version not found in use map: %v", cur)
		}

		value.useCount--
	}

	for cur, _ := range assetsToIncrement {
		value := assetUseMap[cur.PipelineUseMapKey]
		if value == nil {
			// Need to add a new entry for this pipeline.
			value = &PipelineUseMapValue{PipelineStatus: kabanerov1alpha2.PipelineStatus{Url: cur.Url, GitRelease: cur.GitRelease, Digest: cur.Digest}}
			assetUseMap[cur.PipelineUseMapKey] = value
		}

		value.useCount++
	}

	// Now iterate thru the asset use map and delete any assets with a use count of 0,
	// and create any assets with a positive use count.
	for _, value := range assetUseMap {
		if value.useCount <= 0 {
			logger.Info(fmt.Sprintf("Deleting assets with use count %v: %v", value.useCount, value))

			for _, asset := range value.ActiveAssets {
				// Old assets may not have a namespace set - correct that now.
				if len(asset.Namespace) == 0 {
					asset.Namespace = targetNamespace
				}

				DeleteAsset(c, asset, assetOwner, logger)
			}
		}
	}

	for _, value := range assetUseMap {
		if value.useCount > 0 {
			logger.Info(fmt.Sprintf("Creating assets with use count %v: %v", value.useCount, value))

			// Check to see if there is already an asset list.  If not, read the manifests and
			// create one.
			if len(value.ActiveAssets) == 0 {
				// Add the Digest to the rendering context. No need to validate if the digest was tampered
				// with here. Later one and before we do anything with this, we will have validated the specified
				// digest against the generated digest from the archive.
				if len(value.Digest) >= 8 {
					renderingContext["Digest"] = value.Digest[0:8]
				} else {
					renderingContext["Digest"] = "nodigest"
				}

				// Retrieve manifests as unstructured.  If we could not get them, skip.
				manifests, err := GetManifests(c, targetNamespace, value.PipelineStatus, renderingContext, logger)
				if err != nil {
					logger.Error(err, fmt.Sprintf("Error retrieving archive manifests: %v", value))
					value.ManifestError = err
					continue
				}

				// Save the manifests for later.
				value.manifests = manifests

				// Create the asset status slice, but don't apply anything yet.
				for _, asset := range manifests {
					// Figure out what namespace we should create the object in.
					value.ActiveAssets = append(value.ActiveAssets, kabanerov1alpha2.RepositoryAssetStatus{
						Name:          asset.Name,
						Namespace:     getNamespaceForObject(&asset.Yaml, targetNamespace),
						Group:         asset.Group,
						Version:       asset.Version,
						Kind:          asset.Kind,
						Digest:        asset.Sha256,
						Status:        assetStatusUnknown,
						StatusMessage: "Asset has not been applied yet.",
					})
				}
			}

			// Now go thru the asset list and see if the objects are there.  If not, create them.
			for index, asset := range value.ActiveAssets {
				// Old assets may not have a namespace set - correct that now.
				if len(asset.Namespace) == 0 {
					asset.Namespace = targetNamespace
					value.ActiveAssets[index].Namespace = asset.Namespace
				}

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
						logger.Error(err, fmt.Sprintf("Unable to check asset name %v", asset.Name))
						value.ActiveAssets[index].Status = assetStatusUnknown
						value.ActiveAssets[index].StatusMessage = "Unable to check asset: " + err.Error()
					} else {
						// Make sure the manifests are loaded.
						if len(value.manifests) == 0 {
							// Add the Digest to the rendering context.
							if len(value.Digest) >= 8 {
								renderingContext["Digest"] = value.Digest[0:8]
							} else {
								renderingContext["Digest"] = "nodigest"
							}

							// Retrieve manifests as unstructured
							manifests, err := GetManifests(c, targetNamespace, value.PipelineStatus, renderingContext, logger)
							if err != nil {
								logger.Error(err, fmt.Sprintf("Object %v not found and manifests not available: %v", asset.Name, value))
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

								// Only allow Group: tekton.dev
								allowed := true
								for _, resource := range resources {
									if resource.GroupVersionKind().Group != "tekton.dev" {
										value.ActiveAssets[index].Status = assetStatusFailed
										value.ActiveAssets[index].StatusMessage = "Manifest rejected: contains a Group not equal to tekton.dev"
										allowed = false
									}
								}

								if allowed == true {
									mOrig, err := mf.ManifestFrom(mf.Slice(resources), mf.UseClient(mfc.NewClient(c)), mf.UseLogger(logger.WithName("manifestival")))

									logger.Info(fmt.Sprintf("Resources: %v", mOrig.Resources()))

									transforms := []mf.Transformer{
										transforms.InjectOwnerReference(assetOwner),
										mf.InjectNamespace(asset.Namespace),
									}

									m, err := mOrig.Transform(transforms...)
									if err != nil {
										logger.Error(err, fmt.Sprintf("Error transforming manifests for %v", asset.Name))
										value.ActiveAssets[index].Status = assetStatusFailed
										value.ActiveAssets[index].Status = err.Error()
									} else {
										logger.Info(fmt.Sprintf("Applying resources: %v", m.Resources()))
										err = m.Apply()
										if err != nil {
											// Update the asset status with the error message
											logger.Error(err, "Error installing the resource", "resource", asset.Name)
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
							logger.Error(err, fmt.Sprintf("Unable to add owner reference to %v", asset.Name))
						}
					}

					value.ActiveAssets[index].Status = assetStatusActive
					value.ActiveAssets[index].StatusMessage = ""
				}
			}
		}
	}

	return assetUseMap, nil
}

// Deletes an asset.  This can mean removing an object owner, or completely deleting it.
func DeleteAsset(c client.Client, asset kabanerov1alpha2.RepositoryAssetStatus, assetOwner metav1.OwnerReference, logger logr.Logger) error {
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
			logger.Error(err, fmt.Sprintf("Unable to check asset name %v", asset.Name))
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
				logger.Error(err, fmt.Sprintf("Unable to delete asset name %v", asset.Name))
				return err
			}
		} else {
			u.SetOwnerReferences(newOwnerRefs)
			err = c.Update(context.TODO(), u)
			if err != nil {
				logger.Error(err, fmt.Sprintf("Unable to delete owner reference from %v", asset.Name))
				return err
			}
		}
	}

	return nil
}

// Some objects need to get created in a specific namespace.  Try and figure out what that is.
func getNamespaceForObject(u *unstructured.Unstructured, defaultNamespace string) string {
	kind := u.GetKind()

	// Presently, TriggerBinding, TriggerTemplate and EventListener objects are created
	// in the tekton-pipelines namespace.
	//
	// TODO (future): Kabanero Eventing will likely want to create these objects in the
	//                kabanero namespace, since they are not interacting with the Tekton
	//                Webhook Extension anymore.  We'll need to make this selectable
	//                somehow, perhaps just using the namespace in the yaml.
	if (kind == "TriggerBinding") || (kind == "TriggerTemplate") || (kind == "EventListener") {
		return "tekton-pipelines"
	}

	return defaultNamespace
}
