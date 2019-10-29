package utils

import (
	"context"
	"fmt"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetInstalledCSVName retrieves the name of the installed serverless CSV.
func GetInstalledCSVName(c client.Client, cok client.ObjectKey) (string, error) {
	sInstance := &unstructured.Unstructured{}
	sInstance.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   olmapiv1alpha1.GroupName,
		Version: olmapiv1alpha1.GroupVersion,
		Kind:    olmapiv1alpha1.SubscriptionKind,
	})

	err := c.Get(context.TODO(), cok, sInstance)

	if err != nil {
		return "", err
	}

	installedCSVName, found, err := unstructured.NestedString(sInstance.Object, "status", "installedCSV")
	if err != nil {
		return "", err
	}
	if !found {
		err = fmt.Errorf("The value of the installedCSV entry in the subscription was not found")
		return "", err
	}

	return installedCSVName, nil
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
