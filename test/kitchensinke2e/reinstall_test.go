//go:build reinstall
// +build reinstall

package kitchensinke2e

import (
	"context"
	"testing"
	"time"

	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/features"
	"github.com/openshift-knative/serverless-operator/test/kitchensinke2e/reinstall"
	"knative.dev/pkg/system"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
)

// Using custom env for re-install tests because of the need of the "resource stack" in the context
func testUninstalledFeatureSet(t *testing.T, fss ...feature.FeatureSet) {
	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithPollTimings(4*time.Second, 10*time.Minute),
		environment.Managed(t),
		func(ctx context.Context, env environment.Environment) (context.Context, error) {
			return reinstall.ContextWithResourceStack(ctx, new(reinstall.ResourceStack)), nil
		},
	)

	reinstall.TestUninstalledFeatureSet(ctx, env, t, fss...)
}

func TestServerlessReinstallWithBrokerFeatures(t *testing.T) {
	t.Skip("https://issues.redhat.com/browse/SRVKE-1808 known issues with Broker and reinstall")

	// Split the big Broker featuresets
	for _, fs := range split(features.BrokerFeatureSetWithBrokerDLS(), groupSize) {
		testUninstalledFeatureSet(t, fs)
	}

	for _, fs := range split(features.BrokerFeatureSetWithTriggerDLS(), groupSize) {
		testUninstalledFeatureSet(t, fs)
	}
}

func TestServerlessReinstallWithChannelFeatures(t *testing.T) {
	testUninstalledFeatureSet(t, features.ChannelFeatureSet())
}

func TestServerlessReinstallWithSequenceFeatures(t *testing.T) {
	// join the small sequence feature sets
	testUninstalledFeatureSet(t, features.SequenceNoReplyFeatureSet(),
		features.ParallelNoReplyFeatureSet(),
		features.SequenceGlobalReplyFeatureSet(),
		features.ParallelGlobalReplyFeatureSet())
}

func TestServerlessReinstallWithSourceFeatures(t *testing.T) {
	testUninstalledFeatureSet(t, features.SourceFeatureSet())
}

func TestServerlessReinstallWithEventTransformFeatures(t *testing.T) {
	testUninstalledFeatureSet(t, features.EventTransformFeatureSet())
}
