package eventinge2erekt

import (
	"testing"

	"knative.dev/eventing/test/rekt/features/eventtransform"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

func TestEventTransform(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	env.Test(ctx, t, eventtransform.JsonataDirect())
	env.Test(ctx, t, eventtransform.JsonataSink())
	env.Test(ctx, t, eventtransform.JsonataSinkReplyTransform())
}
