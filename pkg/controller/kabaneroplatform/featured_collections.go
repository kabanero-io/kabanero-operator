package kabaneroplatform

import (
	"context"
	_ "fmt"

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


func reconcileFeaturedCollections(ctx context.Context, k *kabanerov1alpha1.Kabanero, cl client.Client) error {
	//Resolve the collections which are currently featured across the various indexes
	featured, err := featuredCollections(k)
	if err != nil {
		return err
	}
	ownerIsController := true
	for _, c := range featured {
		//For each collection, assure that a corresponding resource exists
		name := types.NamespacedName{
			Name:      c.Manifest.Name,
			Namespace: k.GetNamespace(),
		}

		collectionResource := &kabanerov1alpha1.Collection{}
		err := cl.Get(ctx, name, collectionResource)
		if errors.IsNotFound(err) {
			// Not found, so create.  Need to create at the highest supported
			// version found in the repositories.
			collectionResource = &kabanerov1alpha1.Collection{
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.Manifest.Name,
					Namespace: k.GetNamespace(),
					OwnerReferences: []metav1.OwnerReference{
						metav1.OwnerReference{
							APIVersion:           k.TypeMeta.APIVersion,
							Kind:                 k.TypeMeta.Kind,
							Name:                 k.ObjectMeta.Name,
							UID:                  k.ObjectMeta.UID,
							Controller:           &ownerIsController,
						},
					},
				},
				Spec: kabanerov1alpha1.CollectionSpec{
					Name:    c.Manifest.Name,
					Version: findMaxVersionCollectionWithName(featured, c.Manifest.Name),
				},
			}
			err := cl.Create(ctx, collectionResource)
			return err
		} else {
			return err
		}
	}

	return nil
}

// Resolves all featured collections for the given Kabanero instance
func featuredCollections(k *kabanerov1alpha1.Kabanero) ([]*collection.CollectionV1, error) {
	var collections []*collection.CollectionV1

	for _, r := range k.Spec.Collections.Repositories {
		if r.ActivateDefaultCollections {
			index, err := collection.ResolveIndex(r.Url)
			if err != nil {
				return nil, err
			}

			for _, c := range index.ListCollections() {
				c, err := collection.ResolveCollection(c.CollectionUrls...)
				if err != nil {
					return nil, err
				}

				collections = append(collections, c)
			}
		}
	}

	return collections, nil
}
