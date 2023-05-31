package features

import (
	"context"
	"fmt"
	"time"

	eventingfeatures "github.com/openshift-knative/serverless-operator/test/eventinge2erekt/features"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
)

func VerifyEncryptedTrafficToKafkaSink(sinkName string, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	f.Stable("path to kafka sink").
		Must("has encrypted traffic", verifyEncryptedTrafficToKafkaSink(sinkName, since))

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

func VerifyEncryptedTrafficToKafkaBroker(sinkName string, since time.Time) *feature.Feature {
	f := feature.NewFeature()

	f.Stable("path to kafka broker").
		Must("has encrypted traffic", verifyEncryptedTrafficToKafkaBroker(sinkName, since))

	return f
}

func verifyEncryptedTrafficToKafkaBroker(sinkName string, since time.Time) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		// source -> kafka-broker-receiver
		sinkPath := fmt.Sprintf("/%s/%s", environment.FromContext(ctx).Namespace(), sinkName)
		logFilter := eventingfeatures.LogFilter{
			PodNamespace:  "knative-eventing",
			PodSelector:   metav1.ListOptions{LabelSelector: "app=kafka-broker-receiver"},
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
