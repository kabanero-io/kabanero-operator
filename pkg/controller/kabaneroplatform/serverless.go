package kabaneroplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	knsv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Constants
const (
	serverlessSubscriptionName      = "serverless-operator"
	serverlessSubscriptionNamespace = "openshift-operators"
)

// Returns the OpenShift serverless status. The status is based on the availability of the components
// that make up OpenShift serverless.
func getServerlessStatus(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	// Find the installed CSV name.
	installedCSVName, err := getInstalledCSVName(k, c, reqLogger)
	if err != nil {
		message := "Unable to retrieve the name of the installed CSV from the serverless subscription"
		k.Status.Serverless.Ready = "False"
		k.Status.Serverless.Message = message + ". Error: " + err.Error()
		reqLogger.Error(err, message)
		return false, err
	}

	// Find and set the serverless version.
	csvVersion, err := getServerlessCSVVersion(k, c, installedCSVName, reqLogger)
	if err != nil {
		message := "Unable to retrieve the version of installed serverless CSV with the name of " + installedCSVName
		k.Status.Serverless.Ready = "False"
		k.Status.Serverless.Message = message + ". Error: " + err.Error()
		reqLogger.Error(err, message)
		return false, err
	}
	k.Status.Serverless.Version = csvVersion

	// Set the status. The serverless status is based on the status of the components that are part of
	// the serverless operator.
	ready, _ := getKnativeServingStatus(k, c, reqLogger)

	if !ready {
		k.Status.Serverless.Ready = "False"
		return ready, nil
	}

	k.Status.Serverless.Ready = "True"
	k.Status.Serverless.Message = ""

	return ready, err
}

// Returns the name of the installed serverless CSV.
func getInstalledCSVName(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) (string, error) {
	sList := &unstructured.UnstructuredList{}
	sList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   olmapiv1alpha1.GroupName,
		Version: olmapiv1alpha1.GroupVersion,
		Kind:    olmapiv1alpha1.SubscriptionKind,
	})

	listOptions := []client.ListOption{client.InNamespace(serverlessSubscriptionNamespace)}

	err := c.List(context.TODO(), sList, listOptions...)
	if err != nil {
		return "", err
	}

	for _, curSub := range sList.Items {
		subscriptionPackageName, found, err := unstructured.NestedString(curSub.Object, "spec", "name")
		if err != nil {
			return "", err
		}
		if !found {
			continue
		}
		if subscriptionPackageName == serverlessSubscriptionName {
			installedCSVName, found, err := unstructured.NestedString(curSub.Object, "status", "installedCSV")
			if err != nil {
				return "", err
			}
			if !found {
				err = fmt.Errorf("The value of the installedCSV entry in the serverless subscription entry was not found")
				return "", err
			}

			return installedCSVName, nil
		}
	}

	return "", fmt.Errorf("The subscription for serverless-operator could not be found")
}

// Returns the version of the serverless CSV associated with the input CSV name.
func getServerlessCSVVersion(k *kabanerov1alpha2.Kabanero, c client.Client, csvName string, reqLogger logr.Logger) (string, error) {
	csvInstance := &unstructured.Unstructured{}
	csvInstance.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   olmapiv1alpha1.GroupName,
		Version: olmapiv1alpha1.GroupVersion,
		Kind:    olmapiv1alpha1.ClusterServiceVersionKind,
	})

	err := c.Get(context.TODO(), client.ObjectKey{
		Namespace: serverlessSubscriptionNamespace,
		Name:      csvName}, csvInstance)

	if err != nil {
		return "", err
	}

	csvVersion, found, err := unstructured.NestedString(csvInstance.Object, "spec", "version")
	if err != nil {
		return "", err
	}
	if !found {
		err = fmt.Errorf("The value of the spec.version entry in the serverless CSV was not found")
		return "", err
	}

	return csvVersion, nil
}

// Retrieves the knative serving instance status.
func getKnativeServingStatus(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	k.Status.Serverless.KnativeServing.Message = ""
	k.Status.Serverless.KnativeServing.Ready = "False"

	// Hack get unstructured and then feed it into the knative-serving object as JSON.
	// The controller-runtime client is a caching client for reads and I can't figure out
	// how to get it to cache arbitrary objects in another namespace.  Unstructured reads
	// are not cached.
	knsInstance := &unstructured.Unstructured{}
	knsInstance.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "KnativeServing",
		Group:   "operator.knative.dev",
		Version: "v1alpha1",
	})

	knsInstNamespace, knsInstName := "knative-serving", "knative-serving"
	err := c.Get(context.TODO(), client.ObjectKey{
		Name:      knsInstName,
		Namespace: knsInstNamespace}, knsInstance)

	if err != nil {
		if apierrors.IsNotFound(err) {
			k.Status.Serverless.KnativeServing.Message = "Knative serving instance with the name of " + knsInstName + " under the namespace of " + knsInstNamespace + " could not be found."
		} else {
			k.Status.Serverless.KnativeServing.Message = "Error retrieving KnativeServing instance: " + err.Error()
		}

		reqLogger.Error(err, k.Status.Serverless.KnativeServing.Message)
		return false, err
	}

	data, err := knsInstance.MarshalJSON()
	if err != nil {
		k.Status.Serverless.KnativeServing.Message = err.Error()
		reqLogger.Error(err, "Error marshalling unstructured KnativeServing data")
		return false, err
	}

	kns := &knsv1alpha1.KnativeServing{}
	err = json.Unmarshal(data, kns)
	if err != nil {
		k.Status.Serverless.KnativeServing.Message = err.Error()
		reqLogger.Error(err, "Error unmarshalling unstructured KnativeServing data")
		return false, err
	}

	// Find the ready type condition. A status can be either True, False, or Unknown.
	// An Unknown status value is treated the same as a value of False.
	statusReadyType := "ready"
	ready := false
	k.Status.Serverless.KnativeServing.Version = kns.Status.Version

	knsConditions := kns.Status.Conditions
	for _, condition := range knsConditions {
		if strings.ToLower(string(condition.Type)) == statusReadyType {
			status := string(condition.Status)
			k.Status.Serverless.KnativeServing.Ready = status

			if strings.ToLower(status) == "true" {
				ready = true
			} else {
				k.Status.Serverless.KnativeServing.Message = condition.Message
			}

			break
		}
	}

	return ready, err
}
