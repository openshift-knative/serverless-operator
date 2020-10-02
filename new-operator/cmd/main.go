package main

import (
	"knative.dev/operator/pkg/reconciler/knativeeventing"
	"knative.dev/operator/pkg/reconciler/knativeserving"
	"knative.dev/pkg/injection/sharedmain"

	"github.com/openshift-knative/serverless-operator/new-operator/pkg/eventing"
	"github.com/openshift-knative/serverless-operator/new-operator/pkg/serving"
)

func main() {
	sharedmain.Main("knative-operator",
		knativeeventing.NewExtendedController(eventing.NewExtension),
		knativeserving.NewExtendedController(serving.NewExtension),
	)
}
