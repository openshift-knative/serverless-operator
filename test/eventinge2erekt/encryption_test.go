package eventinge2erekt

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test/eventinge2erekt/features"
)

func TestCertManagerCertificatesReady(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	env.Test(ctx, t, features.VerifyCertManagerCertificatesReady())
}
