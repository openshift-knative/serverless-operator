package eventinge2erekt

import (
	"testing"

	"knative.dev/eventing/test/rekt/features/eventtransform"
)

func TestEventTransform(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	env.Test(ctx, t, eventtransform.JsonataDirect())
	env.Test(ctx, t, eventtransform.JsonataSink())
	env.Test(ctx, t, eventtransform.JsonataSinkReplyTransform())
}
