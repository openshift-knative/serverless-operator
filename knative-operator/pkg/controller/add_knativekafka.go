package controller

import (
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/knativekafka"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, knativekafka.Add)
}
