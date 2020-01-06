package kabaneroplatform

import (
	"context"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/collection"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func reconcileFeaturedCollections(ctx context.Context, k *kabanerov1alpha1.Kabanero, cl client.Client) error {
	// Resolve the collections which are currently featured across the various indexes.
	collectionMap, err := featuredCollections(k)
	if err != nil {
		return err
	}

	// Each key is a collection id.  Get that Collection CR instance and see if the versions are set correctly.
	for key, value := range collectionMap {
		updateCollection := utils.Update
		name := types.NamespacedName{
			Name:      key,
			Namespace: k.GetNamespace(),
		}

		collectionResource := &kabanerov1alpha1.Collection{}
		err := cl.Get(ctx, name, collectionResource)
		if err != nil {
			if errors.IsNotFound(err) {
				// Not found. Need to create it.
				updateCollection = utils.Create
				ownerIsController := true
				collectionResource = &kabanerov1alpha1.Collection{
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
					Spec: kabanerov1alpha1.CollectionSpec{
						Name:         key,
					},
				}
			} else {
				return err
			}
		} else {
			// Handle the case where the collection existed before the versions array was added to the Collection CRD.
			// If the versions array is empty, sync it up.
			if (len(collectionResource.Spec.Versions) == 0) && (len(collectionResource.Spec.Version) != 0) {
				collectionResource.Spec.Versions = []kabanerov1alpha1.CollectionVersion{{RepositoryUrl: collectionResource.Spec.RepositoryUrl, Version: collectionResource.Spec.Version, DesiredState: collectionResource.Spec.DesiredState}}
			}
		}

		// Add each version to the versions array if it's not already there.  If it's already there, just
		// update the repository URL, don't touch the desired state.
		for _, collection := range value {
			foundVersion := false
			for _, collectionVersion := range collectionResource.Spec.Versions {
				if collectionVersion.Version == collection.Version {
					foundVersion = true
					collectionVersion.RepositoryUrl = collection.RepositoryUrl
				}
			}

			if foundVersion == false {
				collectionResource.Spec.Versions = append(collectionResource.Spec.Versions, collection)
			}
		}

		// Update the CR instance with the new version information.
		err = updateCollection(cl, ctx, collectionResource)
		if err != nil {
			return err
		}
	}

	return nil
}

// Holds collection related data.
type collectionData struct {
	Collections      []*collection.Collection
	repositoryConfig kabanerov1alpha1.RepositoryConfig
}

// Resolves all collections for the given Kabanero instance
func featuredCollections(k *kabanerov1alpha1.Kabanero) (map[string][]kabanerov1alpha1.CollectionVersion, error) {
	collectionMap := make(map[string][]kabanerov1alpha1.CollectionVersion)
	for _, r := range k.Spec.Collections.Repositories {
		index, err := collection.ResolveIndex(r)
		if err != nil {
			return nil, err
		}

		desiredState := kabanerov1alpha1.CollectionDesiredStateActive
		if r.ActivateDefaultCollections == false {
			desiredState = kabanerov1alpha1.CollectionDesiredStateInactive
		}
		
		for _, c := range index.Collections {
			collectionMap[c.Id] = append(collectionMap[c.Id], kabanerov1alpha1.CollectionVersion{RepositoryUrl: r.Url, Version: c.Version, DesiredState: desiredState})
		}		
	}

	return collectionMap, nil
}
