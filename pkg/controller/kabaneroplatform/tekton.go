package kabaneroplatform

import (
	"context"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"github.com/kabanero-io/kabanero-operator/pkg/assets/config"
	"github.com/kabanero-io/kabanero-operator/pkg/client"
)

func reconcile_tekton(ctx context.Context, k *kabanerov1alpha1.Kabanero, c client.Client) error {
	f, _ := config.Open("reconciler/tekton.yaml")
	defer f.Close()

	if !k.Spec.Tekton.Disabled {
		log.Info("Tekton is currently enabled")

		options := &client.ApplyOptions{
			OwningController: k,
			Namespace:        k.GetNamespace(),
		}

		_, err := c.ApplyText(f, options)
		if err != nil {
			return err
		}

		k.Status.Tekton.Status = "created"
	} else {
		log.Info("Tekton is disabled")

		objs, err := c.Unmarshal(f, "yaml")
		if err != nil {
			return err
		}

		//TODO: align metav1 vs runtime.Objects across this package
		_objs := client.AsRuntimeObjects(objs)

		err = c.DeleteAll(ctx, _objs, &client.DeleteOptions{Namespace: k.GetNamespace()})
		if err != nil {
			return err
		}
	}

	return nil
}
