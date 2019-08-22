package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	knepis "github.com/openshift-knative/knative-eventing-operator/pkg/apis"
	knev1alpha1 "github.com/openshift-knative/knative-eventing-operator/pkg/apis/eventing/v1alpha1"
)

// Retrieves the current knative eventing instance status. 
func getKnativeEventingStatus(k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	k.Status.KnativeEventing.ErrorMessage = ""
	k.Status.KnativeEventing.Ready = "False"

	// Get the knative eventing installation instance.
	kneInstNamespace, kneInstName := "knative-eventing", "knative-eventing"
	config, err := clientcmd.BuildConfigFromFlags("", "")
	myScheme := runtime.NewScheme()
	cl, _ := client.New(config, client.Options{Scheme: myScheme})
	knepis.AddToScheme(myScheme)
	kne := &knev1alpha1.KnativeEventing{}
	err = cl.Get(context.TODO(), client.ObjectKey{
		Namespace: kneInstNamespace,
		Name: kneInstName}, kne)
	if err != nil {
		message := "Knative eventing instance with the name of " + kneInstName + " under the namespace of " + kneInstNamespace + " could not be found."
		k.Status.KnativeEventing.Ready = "False"
		k.Status.KnativeEventing.ErrorMessage = message
		fmt.Println("Error while assessing Knative eventing readiness. " + message, err)
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

			break;
		}
	}

	return ready, err
}
