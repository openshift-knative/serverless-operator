package eventinge2erekt

import (
	"testing"

	"knative.dev/eventing/test/rekt/features/jobsink"
)

func TestJobSinkSuccess(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	env.Test(ctx, t, jobsink.Success())
}
