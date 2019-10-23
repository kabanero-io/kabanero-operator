package kabaneroplatform

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	knsv1alpha1 "github.com/knative/serving-operator/pkg/apis/serving/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Retrieves the knative serving instance status.
func getKnativeServingStatus(k *kabanerov1alpha1.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	k.Status.KnativeServing.ErrorMessage = ""
	k.Status.KnativeServing.Ready = "False"

	// Hack get unstructured and then feed it into the knative-serving object as JSON.
	// The controller-runtime client is a caching client for reads and I can't figure out
	// how to get it to cache arbitrary objects in another namespace.  Unstructured reads
	// are not cached.
	knsInstance := &unstructured.Unstructured{}
	knsInstance.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "KnativeServing",
		Group:   "serving.knative.dev",
		Version: "v1alpha1",
	})

	knsInstNamespace, knsInstName := "knative-serving", "knative-serving"
	err := c.Get(context.TODO(), client.ObjectKey{
		Name:      knsInstName,
		Namespace: knsInstNamespace}, knsInstance)

	if err != nil {
		if apierrors.IsNotFound(err) {
			k.Status.KnativeServing.ErrorMessage = "Knative serving instance with the name of " + knsInstName + " under the namespace of " + knsInstNamespace + " could not be found."
		} else {
			k.Status.KnativeServing.ErrorMessage = "Error retrieving KnativeServing instance: " + err.Error()
		}

		reqLogger.Error(err, k.Status.KnativeServing.ErrorMessage)
		return false, err
	}
	
	data, err := knsInstance.MarshalJSON()
	if err != nil {
		k.Status.KnativeServing.ErrorMessage = err.Error()
		reqLogger.Error(err, "Error marshalling unstructured KnativeServing data")
		return false, err
	}

	kns := &knsv1alpha1.KnativeServing{}
	err = json.Unmarshal(data, kns)
	if err != nil {
		k.Status.KnativeServing.ErrorMessage = err.Error()
		reqLogger.Error(err, "Error unmarshalling unstructured KnativeServing data")
		return false, err
	}

	// Find the ready type condition. A status can be either True, False, or Unknown.
	// An Unknown status value is treated the same as a value of False.
	statusReadyType := "ready"
	ready := false
	k.Status.KnativeServing.Version = kns.Status.Version

	knsConditions := kns.Status.Conditions
	for _, condition := range knsConditions {
		if strings.ToLower(string(condition.Type)) == statusReadyType {
			status := string(condition.Status)
			k.Status.KnativeServing.Ready = status

			if strings.ToLower(status) == "true" {
				ready = true
			} else {
				k.Status.KnativeServing.ErrorMessage = condition.Message
			}

			break
		}
	}

	return ready, err
}
