package webhook

import (
	"github.com/jcrossley3/knative-serving-openshift/pkg/webhook/knativeserving"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, knativeserving.Add)
}
