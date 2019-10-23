package webhook

import (
	"github.com/openshift-knative/knative-serving-openshift/pkg/webhook/knativeserving"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, knativeserving.Add)
}
