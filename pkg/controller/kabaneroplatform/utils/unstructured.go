package utils

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Helps retrieve objects outside of the operator's namespace using a controller-runtime client.
func UnstructuredGet(c client.Client, gvk schema.GroupVersionKind, key client.ObjectKey, obj runtime.Object, reqLogger logr.Logger) error {

	objectInstance := &unstructured.Unstructured{}
	objectInstance.SetGroupVersionKind(gvk)
	err := c.Get(context.TODO(), key, objectInstance)

	if err != nil {
		reqLogger.Error(err, "Could not retrieve unstructured object: ")
		return err
	}
	
	data, err := objectInstance.MarshalJSON()
	if err != nil {
		reqLogger.Error(err, "Error marshalling unstructured data: ")
		return err
	}

	err = json.Unmarshal(data, obj)
	if err != nil {
		reqLogger.Error(err, "Error unmarshalling unstructured data: ")
		return err
	}

	return nil
}
