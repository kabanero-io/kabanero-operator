package client

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type DeleteOptions struct {
	Namespace string
}

type ApplyOptions struct {
	//When present, sets a controller reference on the provided objects
	OwningController runtime.Object

	// When set, the provided function will be executed
	// prior to saving the object on the API server
	Transformation func(runtime.Object) runtime.Object

	//Overrides the namespace in any namespace aware objects
	Namespace string
}

// Called by the Apply function to set the controller refrence on the provided object
func (o *ApplyOptions) setControllerReference(obj runtime.Object, c *Config) (runtime.Object, error) {
	if o.OwningController != nil {
		rowner := o.OwningController.(metav1.Object)
		robj := obj.(metav1.Object)
		if err := controllerutil.SetControllerReference(rowner, robj, c.Scheme); err != nil {
			return nil, err
		}
	}

	return obj, nil
}
