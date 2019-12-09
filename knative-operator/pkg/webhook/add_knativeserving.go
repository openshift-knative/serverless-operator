package webhook

import (
	ks "github.com/openshift-knative/knative-serving-openshift/pkg/webhook/knativeserving"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, ks.MutatingWebhook, ks.ValidatingWebhook)
}
