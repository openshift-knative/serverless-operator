package main

import (
	"knative.dev/operator/pkg/reconciler/knativeeventing"
	"knative.dev/operator/pkg/reconciler/knativeserving"
	"knative.dev/pkg/injection/sharedmain"
)

func main() {
	sharedmain.Main("knative-operator",
		knativeeventing.NewController,
		knativeserving.NewController,
	)
}
