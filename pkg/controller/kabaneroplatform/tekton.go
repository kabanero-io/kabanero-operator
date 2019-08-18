package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	tektonv1alpha1 "github.com/openshift/tektoncd-pipeline-operator/pkg/apis/operator/v1alpha1"
)

// Add tekton type to scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	SchemeGroupVersion := schema.GroupVersion{Group: "operator.tekton.dev", Version: "v1alpha1"}
      scheme.AddKnownTypes(SchemeGroupVersion,
             &tektonv1alpha1.Config{},
             &tektonv1alpha1.ConfigList{},
      )
      metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
      return nil
}

// Retrieves the Tekton instance status.
func getTektonStatus(k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	k.Status.Tekton.ErrorMessage = ""
	k.Status.Tekton.Ready = "False"

	// Get the tekton instance.
	tektonInstName := "cluster"
	tconfig, err := clientcmd.BuildConfigFromFlags("", "")

	myScheme := runtime.NewScheme()
	SchemeBuilder := runtime.NewSchemeBuilder(addKnownTypes)
	if err := SchemeBuilder.AddToScheme(myScheme); err != nil {
		message := "Unable to process scheme."
		k.Status.Tekton.ErrorMessage = message
		fmt.Println("Error while assessing Tekton readiness. " + message, err)
		return false, err
	}

	cl, _ := client.New(tconfig, client.Options{Scheme: myScheme})
	tekton := &tektonv1alpha1.Config{}
	err = cl.Get(context.TODO(), client.ObjectKey{
                Name: tektonInstName}, tekton)

	if err != nil {
		message := "Tekton instance with the name of " + tektonInstName + " could not be found."
		k.Status.Tekton.ErrorMessage = message
		fmt.Println("Error while assessing Tekton readiness. Unable to add tekton scheme.", err)
		return false, err
	}

	// Starting with version 0.5.*, the first condition in the list is the one that matters.
	// The state of an installation can be: installing, installed, or error.
	ready := false
	readyCondition := tekton.Status.Conditions[0]
	k.Status.Tekton.Version = readyCondition.Version
	code := strings.ToLower(string(readyCondition.Code))	
	if code == "error" {
		k.Status.Tekton.ErrorMessage = readyCondition.Details
	} else if code == "installed" {
		ready = true
		k.Status.Tekton.Ready = "True"
	}

        return ready, err
}
