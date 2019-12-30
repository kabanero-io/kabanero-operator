package utils

import (
	"context"
	"fmt"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetInstalledCSVName retrieves the name of the installed CSV from the subscription.
func GetInstalledCSVName(c client.Client, cok client.ObjectKey) (string, error) {
	sList := &unstructured.UnstructuredList{}
	sList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   olmapiv1alpha1.GroupName,
		Version: olmapiv1alpha1.GroupVersion,
		Kind:    olmapiv1alpha1.SubscriptionKind,
	})

  err := c.List(context.TODO(), sList, client.InNamespace(cok.Namespace))
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
		if subscriptionPackageName == cok.Name {
			installedCSVName, found, err := unstructured.NestedString(curSub.Object, "status", "installedCSV")
			if err != nil {
				return "", err
			}
			if !found {
				err = fmt.Errorf("The value of the installedCSV entry in the subscription entry was not found")
				return "", err
			}

			return installedCSVName, nil
		}
	}

	return "", fmt.Errorf("The subscription %v could not be found", cok.Name)
}

// GetCSVSpecVersion retrieves the version of the serverless CSV associated with the input CSV name.
func GetCSVSpecVersion(c client.Client, cok client.ObjectKey) (string, error) {
	csvInstance := &unstructured.Unstructured{}
	csvInstance.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   olmapiv1alpha1.GroupName,
		Version: olmapiv1alpha1.GroupVersion,
		Kind:    olmapiv1alpha1.ClusterServiceVersionKind,
	})

	err := c.Get(context.TODO(), cok, csvInstance)

	if err != nil {
		return "", err
	}

	csvVersion, found, err := unstructured.NestedString(csvInstance.Object, "spec", "version")
	if err != nil {
		return "", err
	}
	if !found {
		err = fmt.Errorf("The value of the spec.version entry in the CSV was not found")
		return "", err
	}

	return csvVersion, nil
}
