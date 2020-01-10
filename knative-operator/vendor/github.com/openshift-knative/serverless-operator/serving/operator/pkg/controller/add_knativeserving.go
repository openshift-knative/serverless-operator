package controller

import (
	"github.com/openshift-knative/serverless-operator/serving/operator/pkg/controller/knativeserving"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, knativeserving.Add)
}
