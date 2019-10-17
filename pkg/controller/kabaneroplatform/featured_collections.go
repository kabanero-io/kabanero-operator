package kabaneroplatform

import (
	"context"

	"github.com/blang/semver"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/collection"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Finds the highest version (semver) of the collection with the given name, in the provided
// list of collections.  The caller has verified that the list contains at least one collection
// with the given name.
func findMaxVersionCollectionWithName(collections []*collection.CollectionV1, name string) string {

	highestVersion, _ := semver.Make("0.0.0")

	for _, candidate := range collections {
		if candidate.Manifest.Name == name {
			candidateVersion, err := semver.ParseTolerant(candidate.Manifest.Version)
			if err == nil { // TODO: log error?
				if candidateVersion.Compare(highestVersion) > 0 {
					highestVersion = candidateVersion
				}
			}
		}
	}

	return highestVersion.String()
}

// Finds the highest version (semver) of the V2 collection with the given id, in the provided
// list of collections.  The caller has verified that the list contains at least one collection
// with the given id.
func findMaxVersionCollectionWithIdV2(collections []*collection.IndexedCollectionV2, id string) string {

	highestVersion, _ := semver.Make("0.0.0")

	for _, candidate := range collections {
		if candidate.Id == id {
			candidateVersion, err := semver.ParseTolerant(candidate.Version)
			if err == nil { // TODO: log error?
				if candidateVersion.Compare(highestVersion) > 0 {
					highestVersion = candidateVersion
				}
			}
		}
	}
	return highestVersion.String()
}

func reconcileFeaturedCollections(ctx context.Context, k *kabanerov1alpha1.Kabanero, cl client.Client) error {
	//Resolve the collections which are currently featured across the various indexes
	featuredCollectionData, err := featuredCollections(k)
	if err != nil {
		return err
	}

	for _, data := range featuredCollectionData {
		ownerIsController := true
		featured := data.indexedCollections
		for _, c := range featured {
			//For each collection, assure that a corresponding resource exists
			name := types.NamespacedName{
				Name:      c.Manifest.Name,
				Namespace: k.GetNamespace(),
			}

			collectionResource := &kabanerov1alpha1.Collection{}
			err := cl.Get(ctx, name, collectionResource)
			if err != nil {
				if errors.IsNotFound(err) {
					// Not found. Need to create it at the highest supported version found
					// in the repositories. At the same time, set the collection's desiredState
					// based on the value of the activateDefaultCollections setting specified in the
					// collection repo section of the kabanero CR instance.
					desiredState := kabanerov1alpha1.CollectionDesiredStateActive
					if !data.repositoryConfig.ActivateDefaultCollections {
						desiredState = kabanerov1alpha1.CollectionDesiredStateInactive
					}

					collectionResource = &kabanerov1alpha1.Collection{
						ObjectMeta: metav1.ObjectMeta{
							Name:      c.Manifest.Name,
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
							Name:         c.Manifest.Name,
							Version:      findMaxVersionCollectionWithName(featured, c.Manifest.Name),
							DesiredState: desiredState,
						},
					}

					err := cl.Create(ctx, collectionResource)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}
		}
	}

	return nil
}

func reconcileFeaturedCollectionsV2(ctx context.Context, k *kabanerov1alpha1.Kabanero, cl client.Client) error {
	// Resolve the collections which are currently featured across the various indexes.
	featuredCollectionData, err := featuredCollectionsV2(k)
	if err != nil {
		return err
	}

	for _, data := range featuredCollectionData {
		ownerIsController := true

		featured := data.indexedCollections
		for _, c := range featured {
			// For each collection, assure that a corresponding resource exists and it is at
			// the highest level found among the repositories.
			updateCollection := cl.Update
			name := types.NamespacedName{
				Name:      c.Id,
				Namespace: k.GetNamespace(),
			}

			collectionResource := &kabanerov1alpha1.Collection{}
			err := cl.Get(ctx, name, collectionResource)
			if err != nil {
				if errors.IsNotFound(err) {
					// Not found. Need to create it at the highest supported version found
					// in the repositories. At the same time, set the collection's desiredState
					// based on the value of the activateDefaultCollections setting specified in the
					// collection repo section of the kabanero CR instance.
					desiredState := kabanerov1alpha1.CollectionDesiredStateActive
					if !data.repositoryConfig.ActivateDefaultCollections {
						desiredState = kabanerov1alpha1.CollectionDesiredStateInactive
					}

					updateCollection = cl.Create
					collectionResource = &kabanerov1alpha1.Collection{
						ObjectMeta: metav1.ObjectMeta{
							Name:      c.Id,
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
							Name:         c.Id,
							DesiredState: desiredState,
						},
					}
				} else {
					return err
				}
			}

			collectionResource.Spec.Version = findMaxVersionCollectionWithIdV2(featured, c.Id)
			err = updateCollection(ctx, collectionResource)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Holds V1 collection related data.
type collectionDataV1 struct {
	indexedCollections []*collection.CollectionV1
	repositoryConfig   kabanerov1alpha1.RepositoryConfig
}

// Resolves all featured collections for the given Kabanero instance
func featuredCollections(k *kabanerov1alpha1.Kabanero) ([]*collectionDataV1, error) {
	var collections []*collection.CollectionV1
	var collectionData []*collectionDataV1

	for _, r := range k.Spec.Collections.Repositories {
		data := collectionDataV1{}
		index, err := collection.ResolveIndex(r)
		if err != nil {
			return nil, err
		}

		for _, c := range index.ListCollections() {
			c, err := collection.ResolveCollection(r, c.CollectionUrls...)
			if err != nil {
				return nil, err
			}

			collections = append(collections, c)
		}

		data.repositoryConfig = r
		data.indexedCollections = collections
		collectionData = append(collectionData, &data)
	}

	return collectionData, nil
}

// Holds V2 collection related data.
type collectionDataV2 struct {
	indexedCollections []*collection.IndexedCollectionV2
	repositoryConfig   kabanerov1alpha1.RepositoryConfig
}

// Resolves all V2 featured collections for the given Kabanero instance
func featuredCollectionsV2(k *kabanerov1alpha1.Kabanero) ([]*collectionDataV2, error) {

	var collectionData []*collectionDataV2
	var collections []*collection.IndexedCollectionV2

	for _, r := range k.Spec.Collections.Repositories {
		index, err := collection.ResolveIndex(r)
		if err != nil {
			return nil, err
		}

		for _, c := range index.CollectionsV2 {
			//forced to re-declare the variable on the stack, thereby giving it a new unique memory address.
			var col collection.IndexedCollectionV2
			col = c
			collections = append(collections, &col)
		}

		collectionData = append(collectionData, &collectionDataV2{repositoryConfig: r, indexedCollections: collections})
	}

	return collectionData, nil
}
