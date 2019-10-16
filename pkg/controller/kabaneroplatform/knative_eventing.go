package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	knev1alpha1 "github.com/openshift-knative/knative-eventing-operator/pkg/apis/eventing/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Retrieves the current knative eventing instance status.
func getKnativeEventingStatus(k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	k.Status.KnativeEventing.ErrorMessage = ""
	k.Status.KnativeEventing.Ready = "False"

	// Get the knative eventing installation instance.
	kneInstNamespace, kneInstName := "knative-eventing", "knative-eventing"
	kne := &knev1alpha1.KnativeEventing{}
	err := c.Get(context.TODO(), client.ObjectKey{
		Namespace: kneInstNamespace,
		Name:      kneInstName}, kne)
	if err != nil {
		message := "Knative eventing instance with the name of " + kneInstName + " under the namespace of " + kneInstNamespace + " could not be found."
		k.Status.KnativeEventing.Ready = "False"
		k.Status.KnativeEventing.ErrorMessage = message
		fmt.Println("Error while assessing Knative eventing readiness. "+message, err)
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
