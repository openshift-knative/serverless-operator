package webhook

import (
	ks "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativeserving"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, ks.ValidatingWebhook)
}
