package kabaneroplatform

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	knev1alpha1 "github.com/openshift-knative/knative-eventing-operator/pkg/apis/eventing/v1alpha1"	
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Retrieves the current knative eventing instance status.
func getKnativeEventingStatus(k *kabanerov1alpha1.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	k.Status.KnativeEventing.ErrorMessage = ""
	k.Status.KnativeEventing.Ready = "False"

	// Hack get unstructured and then feed it into the knative-eventing object as JSON.
	// The controller-runtime client is a caching client for reads and I can't figure out
	// how to get it to cache arbitrary objects in another namespace.  Unstructured reads
	// are not cached.
	kneInstance := &unstructured.Unstructured{}
	kneInstance.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "KnativeEventing",
		Group:   "eventing.knative.dev",
		Version: "v1alpha1",
	})

	kneInstNamespace, kneInstName := "knative-eventing", "knative-eventing"
	err := c.Get(context.TODO(), client.ObjectKey{
		Name:      kneInstName,
		Namespace: kneInstNamespace}, kneInstance)

	if err != nil {
		if apierrors.IsNotFound(err) {
			k.Status.KnativeEventing.ErrorMessage = "Knative eventing instance with the name of " + kneInstName + " under the namespace of " + kneInstNamespace + " could not be found."
		} else {
			k.Status.KnativeEventing.ErrorMessage = "Error retrieving KnativeEventing instance: " + err.Error()
		}

		reqLogger.Error(err, k.Status.KnativeEventing.ErrorMessage)
		return false, err
	}
	
	data, err := kneInstance.MarshalJSON()
	if err != nil {
		k.Status.KnativeEventing.ErrorMessage = err.Error()
		reqLogger.Error(err, "Error marshalling unstructured KnativeEventing data")
		return false, err
	}

	kne := &knev1alpha1.KnativeEventing{}
	err = json.Unmarshal(data, kne)
	if err != nil {
		k.Status.KnativeEventing.ErrorMessage = err.Error()
		reqLogger.Error(err, "Error unmarshalling unstructured KnativeEventing data")
		return false, err
	}
	
	// Find the ready type condition. A status can be either True, False, or Unknown.
	// An Unknown status value is treated the same as a value of False.
	statusReadyType := "ready"
	ready := false
	k.Status.KnativeEventing.Version = kne.Status.Version

	kneConditions := kne.Status.Conditions
	for _, condition := range kneConditions {
		if strings.ToLower(string(condition.Type)) == statusReadyType {
			status := string(condition.Status)
			k.Status.KnativeEventing.Ready = status

			if strings.ToLower(status) == "true" {
				ready = true
			} else {
				k.Status.KnativeEventing.ErrorMessage = condition.Message
			}

			break
		}
	}

	return ready, err
}
