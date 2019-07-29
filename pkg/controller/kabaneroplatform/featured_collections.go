package kabaneroplatform

import (
	"context"
	_ "fmt"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"github.com/kabanero-io/kabanero-operator/pkg/controller/collection"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func reconcileFeaturedCollections(ctx context.Context, k *kabanerov1alpha1.Kabanero, cl client.Client) error {
	//Resolve the collections which are currently featured across the various indexes
	featured, err := featuredCollections(k)
	if err != nil {
		return err
	}

	for _, c := range featured {
		//For each collection, assure that a corresponding resource exists
		name := types.NamespacedName{
			Name:      c.Manifest.Name,
			Namespace: k.GetNamespace(),
		}

		collectionResource := &kabanerov1alpha1.Collection{}
		err := cl.Get(ctx, name, collectionResource)
		if errors.IsNotFound(err) {
			//Not found, so create
			collectionResource = &kabanerov1alpha1.Collection{
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.Manifest.Name,
					Namespace: k.GetNamespace(),
				},
				Spec: kabanerov1alpha1.CollectionSpec{
					Name:    c.Manifest.Name,
					Version: c.Manifest.Version,
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

	if k.Spec.Collections.EnableFeatured {
		for _, r := range k.Spec.Collections.Repositories {
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
