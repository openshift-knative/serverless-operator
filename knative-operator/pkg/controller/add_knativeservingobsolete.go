package controller

import (
	"github.com/openshift-knative/knative-serving-openshift/pkg/controller/knativeservingobsolete"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, knativeservingobsolete.Add)
}
