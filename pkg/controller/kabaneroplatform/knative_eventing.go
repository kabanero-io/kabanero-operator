package kabaneroplatform

import (
	"context"
	mf "github.com/jcrossley3/manifestival"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KNativeEventingReconciler struct {
	client client.Client
	config mf.Manifest
}

func NewKNativeEventingReconciler(c client.Client) (*KNativeEventingReconciler, error) {
	filename := "config/reconciler/knative-eventing"
	m, err := mf.NewManifest(filename, true, c)
	if err != nil {
		return nil, err
	}

	r := &KNativeEventingReconciler{
		client: c,
		config: m,
	}
	return r, nil
}

func (r *KNativeEventingReconciler) Reconcile(ctx context.Context, k *kabanerov1alpha1.Kabanero) error {
	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
	}

	r.config.Transform(transforms...)

	err := r.config.ApplyAll()

	return err
}
