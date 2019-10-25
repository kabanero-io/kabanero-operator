package kabaneroplatform

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	appsodyDeploymentName      = "appsody-operator"
	appsodyDeploymentNamespace = "openshift-operators"
	appsodyDeploymentKind      = "Deployment"
	appsodyDeploymentGroup     = "apps"
	appsodyDeploymentVersion   = "v1"
)

// Retrieves the Appsody deployment status.
func getAppsodyStatus(k *kabanerov1alpha1.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	ready := false

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    appsodyDeploymentKind,
		Group:   appsodyDeploymentGroup,
		Version: appsodyDeploymentVersion,
	})

	// Get the Appsody operator deployment.
	err := c.Get(context.Background(), client.ObjectKey{
		Namespace: appsodyDeploymentNamespace,
		Name:      appsodyDeploymentName,
	}, u)

	if err != nil {
		message := "Unable to retrieve Appsody deployment object"
		reqLogger.Error(err, message+". Name: "+appsodyDeploymentName+". Namespace: "+appsodyDeploymentNamespace)
		k.Status.Appsody.Ready = "False"
		k.Status.Appsody.ErrorMessage = message + ": " + err.Error()
		return false, err
	}

	// Get the status.conditions section.
	conditions, ok, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if err != nil || !ok {
		message := "Unable to retrieve Appsody deployment status.condition field"
		reqLogger.Error(err, message+".")
		k.Status.Appsody.Ready = "False"
		k.Status.Appsody.ErrorMessage = message + ": " + err.Error()
		return false, err
	}

	// Validate that the deployment is available.
	for _, condition := range conditions {
		typeValue, err := getNestedString(condition, "type")
		if err != nil {
			break
		}

		if typeValue == "Available" {
			statusValue, err := getNestedString(condition, "status")
			if err != nil {
				break
			}

			if statusValue == "True" {
				ready = true
				k.Status.Appsody.Ready = "True"
				k.Status.Appsody.ErrorMessage = ""
			} else {
				k.Status.Appsody.Ready = "False"
				message, err := getNestedString(condition, "message")
				if err != nil {
					break
				}

				k.Status.Appsody.ErrorMessage = message
			}

			break
		}
	}

	if err != nil {
		message := "Unable to retrieve Appsody operator deployment status"
		reqLogger.Error(err, message+".")
		k.Status.Appsody.Ready = "False"
		k.Status.Appsody.ErrorMessage = message + ": " + err.Error()
		return false, err
	}
	return ready, nil
}

// Wraps an unstructured NestedString call.
func getNestedString(genObject interface{}, key string) (string, error) {
	var value string
	genObjectMap, ok := genObject.(map[string]interface{})
	if !ok {
		return value, fmt.Errorf("Unable to retrieve value for key %v because object %v is not a map", key, genObject)
	}

	value, found, err := unstructured.NestedString(genObjectMap, key)
	if err != nil {
		return value, err
	}
	if !found {
		return value, fmt.Errorf("Unable to retrieve value for key %v. Value not found", key)
	}

	return value, nil
}
