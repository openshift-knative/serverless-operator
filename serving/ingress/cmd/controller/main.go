package main

import (
	// This defines the shared main for injected controllers.
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/reconciler/ingress"
)

var ctors = []injection.ControllerConstructor{
	ingrerss.NewIstioController,
	ingrerss.NewKourierController,
}

func main() {
	sharedmain.Main("openshift-ingress-controller", ctors...)
}
