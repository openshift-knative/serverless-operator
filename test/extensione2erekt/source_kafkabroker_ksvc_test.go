package extensione2erekt

import (
	"context"
	"testing"
	"time"

	kafkafeatures "github.com/openshift-knative/serverless-operator/test/extensione2erekt/features"
	"github.com/openshift-knative/serverless-operator/test/monitoringe2e"
	"knative.dev/eventing-kafka-broker/control-plane/pkg/kafka"
	"knative.dev/eventing/pkg/apis/eventing"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
)

// Source (Eventshub) -> KafkaBroker -> Trigger -> Ksvc -> Sink (Eventshub)
func TestSourceKafkaBrokerKsvc(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	since := time.Now()

	env.Test(ctx, t, kafkafeatures.BrokerSmokeTest(kafka.BrokerClass))
	env.Test(ctx, t, VerifyMetricsKafkaBroker())

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, kafkafeatures.VerifyEncryptedTrafficForKafkaBroker(env.References(), since))
	}
}

// Source (Eventshub) -> NamespacedKafkaBroker -> Trigger -> Ksvc -> Sink (Eventshub)
func TestSourceNamespacedKafkaBrokerKsvc(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	since := time.Now()

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		// With Istio this issue happens often.
		t.Skip("https://issues.redhat.com/browse/SRVKE-1424")
	}

	env.Test(ctx, t, BrokerSmokeTest(kafka.NamespacedBrokerClass))
	env.Test(ctx, t, VerifyMetricsNamespacedKafkaBroker(environment.FromContext(ctx).Namespace()))

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, kafkafeatures.VerifyEncryptedTrafficForNamespacedKafkaBroker(env.References(), since))
	}
}

// Source (Eventshub) -> MTChannelBased-KafkaBroker -> Trigger -> Ksvc -> Sink (Eventshub)
func TestSourceChannelBasedKafkaBrokerKsvc(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	since := time.Now()

	env.Test(ctx, t, BrokerSmokeTest(eventing.MTChannelBrokerClassValue))

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, kafkafeatures.VerifyEncryptedTrafficForChannelBasedKafkaBroker(env.References(), since))
	}
}

func VerifyMetricsKafkaBroker() *feature.Feature {
	f := feature.NewFeature()

	f.Stable("kafka broker").
		Must("has metrics", func(ctx context.Context, t feature.T) {
			if err := monitoringe2e.VerifyMetrics(ctx, monitoringe2e.KafkaBrokerDataPlaneQueries); err != nil {
				t.Fatal("Failed to verify that Kafka Broker data plane metrics work correctly", err)
			}
		})

	return f
}

func VerifyMetricsNamespacedKafkaBroker(namespace string) *feature.Feature {
	f := feature.NewFeature()

	f.Stable("namespaced kafka broker").
		Must("has metrics", func(ctx context.Context, t feature.T) {
			if err := monitoringe2e.VerifyMetrics(ctx, monitoringe2e.NamespacedKafkaBrokerDataPlaneQueries(namespace)); err != nil {
				t.Fatal("Failed to verify that Kafka Broker data plane metrics work correctly", err)
			}
		})

	return f
}
