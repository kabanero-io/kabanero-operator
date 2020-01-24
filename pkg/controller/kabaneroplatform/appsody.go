package kabaneroplatform

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	kutils "github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform/utils"
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
	appsodySubscriptionName    = "appsody-operator-certified"
)

// Retrieves the Appsody deployment status.
func getAppsodyStatus(k *kabanerov1alpha1.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	ready := false

	// Get the appsody version.
	csvVersion, err := getAppsodyOperatorVersion(k, c)
	if err != nil {
		message := "Unable to retrieve the version of installed Appsody operator"
		k.Status.Appsody.Ready = "False"
		k.Status.Appsody.ErrorMessage = message + ". Error: " + err.Error()
		reqLogger.Error(err, message)
		return false, err
	}
	k.Status.Appsody.Version = csvVersion

	// Get the Appsody operator deployment.
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    appsodyDeploymentKind,
		Group:   appsodyDeploymentGroup,
		Version: appsodyDeploymentVersion,
	})

	err = c.Get(context.Background(), client.ObjectKey{
		Namespace: appsodyDeploymentNamespace,
		Name:      appsodyDeploymentName,
	}, u)

	if err != nil {
		message := "Unable to retrieve Appsody deployment object"
		k.Status.Appsody.Ready = "False"
		k.Status.Appsody.ErrorMessage = message + ". Error: " + err.Error()
		reqLogger.Error(err, message+". Name: "+appsodyDeploymentName+". Namespace: "+appsodyDeploymentNamespace)
		return false, err
	}

	// Get the status.conditions section.
	conditions, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if err != nil {
		message := "Unable to retrieve Appsody deployment status.condition field"
		k.Status.Appsody.Ready = "False"
		k.Status.Appsody.ErrorMessage = message + ". Error: " + err.Error()
		reqLogger.Error(err, message)
		return false, err
	}
	if !found {
		message := "The Appsody deployment entry of status.condition was not found."
		err = fmt.Errorf(message)
		k.Status.Appsody.ErrorMessage = err.Error()
		reqLogger.Error(err, "")
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
		k.Status.Appsody.Ready = "False"
		k.Status.Appsody.ErrorMessage = message + ". Error: " + err.Error()
		reqLogger.Error(err, message)
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

// Returns the installed Appsody operator version.
func getAppsodyOperatorVersion(k *kabanerov1alpha1.Kabanero, c client.Client) (string, error) {
	cok := client.ObjectKey{
		Namespace: appsodyDeploymentNamespace,
		Name:      appsodySubscriptionName}

	installedCSVName, err := kutils.GetInstalledCSVName(c, cok)
	if err != nil {
		return "", err
	}

	cok = client.ObjectKey{
		Namespace: appsodyDeploymentNamespace,
		Name:      installedCSVName}

	csvVersion, err := kutils.GetCSVSpecVersion(c, cok)
	if err != nil {
		return "", err
	}

	return csvVersion, nil
}
