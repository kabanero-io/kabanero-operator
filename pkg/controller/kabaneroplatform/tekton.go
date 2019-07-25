package kabaneroplatform

import (
	"context"
	mf "github.com/jcrossley3/manifestival"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func reconcile_tekton(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	filename := "config/reconciler/tekton.yaml"
	m, err := mf.NewManifest(filename, true, c)
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
	}

	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	if !k.Spec.Tekton.Disabled {
		err = m.ApplyAll()

		k.Status.Tekton.Status = "created"
	} else {
		m.DeleteAll(nil)
	}

	return err
}
