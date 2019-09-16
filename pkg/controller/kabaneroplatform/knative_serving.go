package kabaneroplatform

import (
	"context"
	"fmt"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	knsapis "github.com/knative/serving-operator/pkg/apis"
	knsv1alpha1 "github.com/knative/serving-operator/pkg/apis/serving/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// Retrieves the knative serving instance status.
func getKnativeServingStatus(k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	k.Status.KnativeServing.ErrorMessage = ""
	k.Status.KnativeServing.Ready = "False"

	// Get the knative serving installation instance.
	knsInstNamespace, knsInstName := "knative-serving", "knative-serving"
	config, err := clientcmd.BuildConfigFromFlags("", "")
	myScheme := runtime.NewScheme()
	cl, _ := client.New(config, client.Options{Scheme: myScheme})
	knsapis.AddToScheme(myScheme)
	kns := &knsv1alpha1.KnativeServing{}
	err = cl.Get(context.TODO(), client.ObjectKey{
		Namespace: knsInstNamespace,
		Name:      knsInstName}, kns)

	if err != nil {
		message := "Knative serving instance with the name of " + knsInstName + " under the namespace of " + knsInstNamespace + " could not be found."
		k.Status.KnativeServing.Ready = "False"
		k.Status.KnativeServing.ErrorMessage = message
		fmt.Println("Error while assessing Knative serving readiness. "+message, err)
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
