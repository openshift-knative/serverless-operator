package controller

import (
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/dashboard/health"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, health.Add)
}
