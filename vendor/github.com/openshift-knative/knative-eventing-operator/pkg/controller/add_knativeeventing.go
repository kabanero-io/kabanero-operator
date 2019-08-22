package controller

import (
	"github.com/openshift-knative/knative-eventing-operator/pkg/controller/knativeeventing"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, knativeeventing.Add)
}
