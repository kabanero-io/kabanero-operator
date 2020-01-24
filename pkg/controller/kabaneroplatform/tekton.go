package kabaneroplatform

import (
	"context"
	"fmt"
	"strings"

	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	tektoncdv1alpha1 "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Retrieves the Tekton instance status.
func getTektonStatus(k *kabanerov1alpha2.Kabanero, c client.Client) (bool, error) {
	k.Status.Tekton.ErrorMessage = ""
	k.Status.Tekton.Ready = "False"

	// Get the tekton instance.
	tektonInstName := "cluster"

	tekton := &tektoncdv1alpha1.Config{}
	err := c.Get(context.TODO(), client.ObjectKey{
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
