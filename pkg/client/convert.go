package client

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func AsRuntimeObjects(objs []metav1.Object) []runtime.Object {
	//then drop this conversion
	_objs := make([]runtime.Object, len(objs))
	for i, obj := range objs {
		_objs[i] = obj.(runtime.Object)
	}

	return _objs
}
