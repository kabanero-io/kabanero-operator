// +build integration

package kabaneroplatform

import (
	"context"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
)

func destroyCollection(ctx context.Context, cl client.Client, name string, namespace string) error {
	//Cleanup any prior test
	collectionResource := &kabanerov1alpha1.Collection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := cl.Delete(ctx, collectionResource)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}

func TestReconcileFeaturedCollections(t *testing.T) {
	ctx := context.Background()

	scheme, _ := kabanerov1alpha1.SchemeBuilder.Build()
	cl, err := client.New(config.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal("Could not create a client", err)
	}

	//Cleanup any prior run
	err = destroyCollection(ctx, cl, "java-microprofile", "default")
	if err != nil {
		t.Fatal(err)
	}
	//Cleanup after run
	defer destroyCollection(ctx, cl, "java-microprofile", "default")

	collection_index_url := "https://raw.githubusercontent.com/kabanero-io/kabanero-collection/master/experimental/index.yaml"

	k := &kabanerov1alpha1.Kabanero{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kabanero.io/v1alpha1",
			Kind:       "Kabanero",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "kabanero",
			UID:       "12345",
		},
		Spec: kabanerov1alpha1.KabaneroSpec{
			Collections: kabanerov1alpha1.InstanceCollectionConfig{
				Repositories: []kabanerov1alpha1.RepositoryConfig{
					kabanerov1alpha1.RepositoryConfig{
						Name: "default",
						Url:  collection_index_url,
						ActivateDefaultCollections: true,
					},
				},
			},
		},
	}

	err = reconcileFeaturedCollections(context.Background(), k, cl)
	if err != nil {
		t.Fatal(err)
	}

	//Verify the collection was created
	collectionResource := &kabanerov1alpha1.Collection{}
	err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile", Namespace: "default"}, collectionResource)
	if err != nil {
		t.Fatal("Could not resolve the automatically created collection", err)
	}
}

// specify  ActivateDefaultCollections: true
func TestReconcileFeaturedCollectionsActivateDefaultCollectionsTrue(t *testing.T) {
        ctx := context.Background()

        scheme, _ := kabanerov1alpha1.SchemeBuilder.Build()
        cl, err := client.New(config.GetConfigOrDie(), client.Options{Scheme: scheme})
        if err != nil {
                t.Fatal("Could not create a client", err)
        }

        //Cleanup any prior run
        err = destroyCollection(ctx, cl, "java-microprofile", "default")
        if err != nil {
                t.Fatal(err)
        }
        //Cleanup after run
        defer destroyCollection(ctx, cl, "java-microprofile", "default")

        collection_index_url := "https://raw.githubusercontent.com/kabanero-io/kabanero-collection/master/experimental/index.yaml"

        k := &kabanerov1alpha1.Kabanero{
		TypeMeta:   metav1.TypeMeta{
			Kind:       "Kabanero",
			APIVersion: "v1",
		},
                ObjectMeta: metav1.ObjectMeta{
                        Namespace: "default",
			Name:      "k",
			UID:       "1",
                },
                Spec: kabanerov1alpha1.KabaneroSpec{
                        Collections: kabanerov1alpha1.InstanceCollectionConfig{
                                Repositories: []kabanerov1alpha1.RepositoryConfig{
                                        kabanerov1alpha1.RepositoryConfig{
                                                Name: "default",
                                                Url:  collection_index_url,
						ActivateDefaultCollections: true,
                                        },
                                },
                        },
                },
        }

        err = reconcileFeaturedCollections(context.Background(), k, cl)
        if err != nil {
                t.Fatal(err)
        }

        //Verify the collection was created
        collectionResource := &kabanerov1alpha1.Collection{}
        err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile", Namespace: "default"}, collectionResource)
        if err != nil {
                t.Fatal("Could not resolve the automatically created collection", err)
        }
}

// specify ActivateDefaultCollections: false
func TestReconcileFeaturedCollectionsActivateDefaultCollectionsFalse(t *testing.T) {
        ctx := context.Background()

        scheme, _ := kabanerov1alpha1.SchemeBuilder.Build()
        cl, err := client.New(config.GetConfigOrDie(), client.Options{Scheme: scheme})
        if err != nil {
                t.Fatal("Could not create a client", err)
        }

        //Cleanup any prior run
        err = destroyCollection(ctx, cl, "java-microprofile", "default")
        if err != nil {
                t.Fatal(err)
        }
        //Cleanup after run
        defer destroyCollection(ctx, cl, "java-microprofile", "default")

        collection_index_url := "https://raw.githubusercontent.com/kabanero-io/kabanero-collection/master/experimental/index.yaml"

        k := &kabanerov1alpha1.Kabanero{
                ObjectMeta: metav1.ObjectMeta{
                        Namespace: "default",
                },
                Spec: kabanerov1alpha1.KabaneroSpec{
                        Collections: kabanerov1alpha1.InstanceCollectionConfig{
                                Repositories: []kabanerov1alpha1.RepositoryConfig{
                                        kabanerov1alpha1.RepositoryConfig{
                                                Name: "default",
                                                Url:  collection_index_url,
                                                ActivateDefaultCollections: false,
                                        },
                                },
                        },
                },
        }

        err = reconcileFeaturedCollections(context.Background(), k, cl)
        if err != nil {
                t.Fatal(err)
        }

        //Verify the collection was created
        collectionResource := &kabanerov1alpha1.Collection{}
        err = cl.Get(ctx, types.NamespacedName{Name: "java-microprofile", Namespace: "default"}, collectionResource)
        if err == nil {
                t.Fatal("Collection created when it should not have been created", err)
        }
}

// Attempts to resolve the featured collections from the default repository
// Note that this test is fragile since it relies on connectivity to the central example index
// and the presence of specific collections
func TestResolveFeaturedCollections(t *testing.T) {
	collection_index_url := "https://raw.githubusercontent.com/kabanero-io/kabanero-collection/master/experimental/index.yaml"

	k := &kabanerov1alpha1.Kabanero{
		Spec: kabanerov1alpha1.KabaneroSpec{
			Collections: kabanerov1alpha1.InstanceCollectionConfig{
				Repositories: []kabanerov1alpha1.RepositoryConfig{
					kabanerov1alpha1.RepositoryConfig{
						Name: "default",
						Url:  collection_index_url,
						ActivateDefaultCollections: true,
					},
				},
			},
		},
	}

	collections, err := featuredCollections(k)
	if err != nil {
		t.Fatal("Could not resolve the featured collections from the default index", err)
	}

	if len(collections) < 1 {
		t.Fatal("Was expecting at least one collection to be found in the default repository: ", collection_index_url)
	}
}
