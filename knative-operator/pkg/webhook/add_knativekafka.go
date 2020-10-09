package webhook

import (
	kk "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativekafka"
)

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, kk.ValidatingWebhook)
}
