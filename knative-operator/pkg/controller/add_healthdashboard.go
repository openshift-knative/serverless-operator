package controller

import (
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/healthdashboard"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, healthdashboard.Add)
}
