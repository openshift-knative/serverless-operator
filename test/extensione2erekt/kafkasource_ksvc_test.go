package extensione2erekt

import (
	"testing"
	"time"

	"knative.dev/eventing-kafka-broker/test/rekt/features"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"

	kafkafeatures "github.com/openshift-knative/serverless-operator/test/extensione2erekt/features"
)


func TestKafkaSourceBinaryEvent(t *testing.T) {
	t.Parallel()

	ctx, env := defaultEnvironment(t)

	env.Test(ctx, t, features.KafkaSourceBinaryEvent())
}

func TestKafkaSourceTLS(t *testing.T) {

	t.Parallel()

	ctx, env := defaultEnvironment(t)

	kafkaSource := feature.MakeRandomK8sName("kafkaSource")
	kafkaSink := feature.MakeRandomK8sName("kafkaSink")
	topic := feature.MakeRandomK8sName("topic")

	since := time.Now()

	f := features.KafkaSourceTLS(kafkaSource, kafkaSink, topic)
	// Gate event send until the dispatcher has active consumers with
	// partition assignments — IsReady alone only checks control-plane
	// conditions, not data-plane readiness.
	f.Requirement("kafka source has active consumers",
		kafkafeatures.WaitForKafkaSourceConsuming(kafkaSource))
	env.Test(ctx, t, f)

	if ic := environment.GetIstioConfig(ctx); ic.Enabled {
		env.Test(ctx, t, kafkafeatures.VerifyEncryptedTrafficForKafkaSource(kafkaSink, since))
	}
}

func TestKafkaSourceSASL(t *testing.T) {

	t.Parallel()

	ctx, env := defaultEnvironment(t)

	env.Test(ctx, t, features.KafkaSourceSASL())
}
