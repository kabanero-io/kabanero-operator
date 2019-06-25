package controller

import (
	"github.com/kabanero-io/kabanero-operator/pkg/controller/kabaneroplatform"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, kabaneroplatform.Add)
}
