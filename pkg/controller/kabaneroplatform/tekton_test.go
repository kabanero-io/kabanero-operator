// +build integration

package kabaneroplatform

import (
	"context"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"github.com/kabanero-io/kabanero-operator/pkg/client"
	"testing"
)

func TestReconcileTekton(t *testing.T) {
	k := &kabanerov1alpha1.Kabanero{}
	c := client.DefaultClient
	err := reconcile_tekton(context.Background(), k, c)
	if err != nil {
		t.Fatal(err)
	}
}
