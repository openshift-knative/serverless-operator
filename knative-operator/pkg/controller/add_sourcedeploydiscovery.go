package controller

import (
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/monitoring/sources"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, sources.Add)
}
