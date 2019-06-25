// +build integration

package kabaneroplatform

import (
	"context"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
)

func TestReconcileKNativeEventing(t *testing.T) {
	ctx := context.Background()

	k := &kabanerov1alpha1.Kabanero{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kabanero.io/v1alpha1",
			Kind:       "Kabanero",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-kabanero",
			Namespace: "default",
			UID:       "generated",
		},
	}
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		t.Fatal(err)
	}

	r, err := NewKNativeEventingReconciler(c)
	if err != nil {
		t.Fatal(err)
	}

	err = r.Reconcile(ctx, k)
	if err != nil {
		t.Fatal(err)
	}
}
