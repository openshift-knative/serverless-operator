package main

import (
	// This defines the shared main for injected controllers.
	"knative.dev/pkg/injection/sharedmain"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/reconciler/ingress"
)

func main() {
	sharedmain.Main("openshift-ingress-controller", ingress.NewController)
}
