package main

import (
	// This defines the shared main for injected controllers.
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/reconciler/ingress"
)

var ctors = []injection.ControllerConstructor{
	ingress.NewIstioController,
	ingress.NewKourierController,
}

func main() {
	ctx := signals.NewContext()

	// Disable leader election to allow both ingress controllers to do their job.
	// TODO: Fix the respective clash in Knative's reconciler framework.
	ctx = sharedmain.WithHADisabled(ctx)

	sharedmain.MainWithContext(ctx, "openshift-ingress-controller", ctors...)
}
