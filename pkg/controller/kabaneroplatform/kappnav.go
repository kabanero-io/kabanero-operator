package kabaneroplatform

import (
	"context"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rlog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var kanlog = rlog.Log.WithName("kabanero-kappnav")

const currentVersion = "0.1.0"

// Retrieves the Kubernetes Application Navigator deployment status.
func getKappnavStatus(k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	// KNavApp is optional.  We're basically just reporting if we found it.
	status := kabanerov1alpha1.KappnavStatus{}
	k.Status.Kappnav = &status
	k.Status.Kappnav.ErrorMessage = ""
	k.Status.Kappnav.Ready = "False"

	// We're looking in the default location for KAppNav.
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "KAppNav",
		Group:   "charts.helm.k8s.io",
		Version: "v1alpha1",
	})
	err := c.Get(context.Background(), client.ObjectKey{
		Namespace: "kappnav",
		Name:      "instance",
	}, u)
	
	if err == nil {
		k.Status.Kappnav.Ready = "True"
		return true, nil
	}

	if errors.IsNotFound(err) {
		k.Status.Kappnav.ErrorMessage = "The default deployment of KAppNav was not found.  KAppNav is an optional component."
		return true, nil // Don't fail Kabanero status for this.
	}

	k.Status.Kappnav.ErrorMessage = "Could not detect the status of KAppNav: " + err.Error()
	return true, nil // Don't fail Kabanero status for this.
}
