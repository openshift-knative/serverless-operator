package features

import (
	"context"
	"fmt"
	"time"

	eventingfeatures "github.com/openshift-knative/serverless-operator/test/eventinge2erekt/features"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/eventing-kafka/test/rekt/resources/kafkachannel"
	"knative.dev/eventing/test/rekt/resources/broker"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
)

func VerifyEncryptedTrafficForKafkaSource(refs []corev1.ObjectReference, sinkName string, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	f.Stable("kafka source path").
		Must("has encrypted traffic to kafka sink", verifyEncryptedTrafficToKafkaSink(sinkName, since)).
		Must("has encrypted traffic from kafka source to activator", eventingfeatures.VerifyEncryptedTrafficToActivator(refs, since)).
		Must("has encrypted traffic to app", eventingfeatures.VerifyEncryptedTrafficToApp(refs, since))

	return f
}

func verifyEncryptedTrafficToKafkaSink(sinkName string, since time.Time) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		// source -> kafka-sink-receiver
		sinkPath := fmt.Sprintf("/%s/%s", environment.FromContext(ctx).Namespace(), sinkName)
		logFilter := eventingfeatures.LogFilter{
			PodNamespace:  "knative-eventing",
			PodSelector:   metav1.ListOptions{LabelSelector: "app=kafka-sink-receiver"},
			PodLogOptions: &corev1.PodLogOptions{Container: "istio-proxy", SinceTime: &metav1.Time{Time: since}},
			JSONLogFilter: func(m map[string]interface{}) bool {
				return eventingfeatures.GetMapValueAsString(m, "path") == sinkPath &&
					eventingfeatures.GetMapValueAsString(m, "authority") == "kafka-sink-ingress.knative-eventing.svc.cluster.local"
			}}

		err := eventingfeatures.VerifyPodLogsEncryptedRequestToHost(ctx, logFilter)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func VerifyEncryptedTrafficForKafkaBroker(refs []corev1.ObjectReference, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	f.Stable("broker path").
		Must("has encrypted traffic to broker", verifyEncryptedTrafficToKafkaBroker(refs, false /*namespaced*/, since)).
		Must("has encrypted traffic to activator", eventingfeatures.VerifyEncryptedTrafficToActivator(refs, since)).
		Must("has encrypted traffic to app", eventingfeatures.VerifyEncryptedTrafficToApp(refs, since))

	return f
}

func VerifyEncryptedTrafficForNamespacedKafkaBroker(refs []corev1.ObjectReference, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	f.Stable("broker path").
		Must("has encrypted traffic to broker", verifyEncryptedTrafficToKafkaBroker(refs, true /*namespaced*/, since)).
		Must("has encrypted traffic to activator", eventingfeatures.VerifyEncryptedTrafficToActivator(refs, since)).
		Must("has encrypted traffic to app", eventingfeatures.VerifyEncryptedTrafficToApp(refs, since))

	return f
}

func verifyEncryptedTrafficToKafkaBroker(refs []corev1.ObjectReference, namespacedBroker bool, since time.Time) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		brokerName, err := getBrokerName(refs)
		if err != nil {
			t.Fatalf("Unable to get Broker name: %v", err)
		}
		// source -> kafka-broker-receiver
		brokerPath := fmt.Sprintf("/%s/%s", environment.FromContext(ctx).Namespace(), brokerName)
		brokerReceiverNamespace := "knative-eventing"
		if namespacedBroker {
			brokerReceiverNamespace = environment.FromContext(ctx).Namespace()
		}
		authority := fmt.Sprintf("kafka-broker-ingress.%s.svc.cluster.local", brokerReceiverNamespace)

		logFilter := eventingfeatures.LogFilter{
			PodNamespace:  brokerReceiverNamespace,
			PodSelector:   metav1.ListOptions{LabelSelector: "app=kafka-broker-receiver"},
			PodLogOptions: &corev1.PodLogOptions{Container: "istio-proxy", SinceTime: &metav1.Time{Time: since}},
			JSONLogFilter: func(m map[string]interface{}) bool {
				return eventingfeatures.GetMapValueAsString(m, "path") == brokerPath &&
					eventingfeatures.GetMapValueAsString(m, "authority") == authority
			}}

		err = eventingfeatures.VerifyPodLogsEncryptedRequestToHost(ctx, logFilter)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func VerifyEncryptedTrafficForChannelBasedKafkaBroker(refs []corev1.ObjectReference, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	f.Stable("broker path").
		Must("has encrypted traffic to broker", verifyEncryptedTrafficToChannelBasedKafkaBroker(refs, since)).
		Must("has encrypted traffic to activator", eventingfeatures.VerifyEncryptedTrafficToActivator(refs, since)).
		Must("has encrypted traffic to app", eventingfeatures.VerifyEncryptedTrafficToApp(refs, since))

	return f
}

func verifyEncryptedTrafficToChannelBasedKafkaBroker(refs []corev1.ObjectReference, since time.Time) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		brokerName, err := getBrokerName(refs)
		if err != nil {
			t.Fatalf("Unable to get Broker name: %v", err)
		}
		// source -> kafka-channel-receiver
		authority := fmt.Sprintf("%s-kne-trigger-kn-channel.%s.svc.cluster.local", brokerName,
			environment.FromContext(ctx).Namespace())

		logFilter := eventingfeatures.LogFilter{
			PodNamespace:  "knative-eventing",
			PodSelector:   metav1.ListOptions{LabelSelector: "app=kafka-channel-receiver"},
			PodLogOptions: &corev1.PodLogOptions{Container: "istio-proxy", SinceTime: &metav1.Time{Time: since}},
			JSONLogFilter: func(m map[string]interface{}) bool {
				return eventingfeatures.GetMapValueAsString(m, "path") == "/" &&
					eventingfeatures.GetMapValueAsString(m, "authority") == authority
			}}

		err = eventingfeatures.VerifyPodLogsEncryptedRequestToHost(ctx, logFilter)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func VerifyEncryptedTrafficForKafkaChannel(refs []corev1.ObjectReference, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	f.Stable("channel path").
		Must("has encrypted traffic to channel", verifyEncryptedTrafficToKafkaChannel(refs, since)).
		Must("has encrypted traffic to activator", eventingfeatures.VerifyEncryptedTrafficToActivator(refs, since)).
		Must("has encrypted traffic to app", eventingfeatures.VerifyEncryptedTrafficToApp(refs, since))

	return f
}

func verifyEncryptedTrafficToKafkaChannel(refs []corev1.ObjectReference, since time.Time) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		channelName, err := getChannelName(refs)
		if err != nil {
			t.Fatalf("Unable to get Channel name: %v", err)
		}
		// source -> kafka-channel-receiver
		authority := fmt.Sprintf("%s-kn-channel.%s.svc.cluster.local", channelName,
			environment.FromContext(ctx).Namespace())

		logFilter := eventingfeatures.LogFilter{
			PodNamespace:  "knative-eventing",
			PodSelector:   metav1.ListOptions{LabelSelector: "app=kafka-channel-receiver"},
			PodLogOptions: &corev1.PodLogOptions{Container: "istio-proxy", SinceTime: &metav1.Time{Time: since}},
			JSONLogFilter: func(m map[string]interface{}) bool {
				return eventingfeatures.GetMapValueAsString(m, "path") == "/" &&
					eventingfeatures.GetMapValueAsString(m, "authority") == authority
			}}

		err = eventingfeatures.VerifyPodLogsEncryptedRequestToHost(ctx, logFilter)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func getBrokerName(refs []corev1.ObjectReference) (string, error) {
	var (
		brokerName string
		numBrokers int
	)
	for _, ref := range refs {
		if ref.GroupVersionKind() == broker.GVR().GroupVersion().WithKind("Broker") {
			// Make sure we verify traffic for the right Broker.
			// This is for safety and to guarantee the feature invariance.
			if numBrokers != 0 {
				return "", fmt.Errorf("found more than one Broker: %s, %s", brokerName, ref.Name)
			}
			brokerName = ref.Name
			numBrokers++
		}
	}

	return brokerName, nil
}

func getChannelName(refs []corev1.ObjectReference) (string, error) {
	var (
		channelName string
		numChannels int
	)
	for _, ref := range refs {
		if ref.GroupVersionKind() == kafkachannel.GVR().GroupVersion().WithKind("KafkaChannel") {
			// Make sure we verify traffic for the right Channel.
			// This is for safety and to guarantee the feature invariance.
			if numChannels != 0 {
				return "", fmt.Errorf("found more than one Kafka Channel: %s, %s", channelName, ref.Name)
			}
			channelName = ref.Name
			numChannels++
		}
	}

	return channelName, nil
}
