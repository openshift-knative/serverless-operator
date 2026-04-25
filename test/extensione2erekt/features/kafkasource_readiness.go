package features

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"

	sourcesv1 "knative.dev/eventing-kafka-broker/control-plane/pkg/apis/sources/v1"
	kafkaclientset "knative.dev/eventing-kafka-broker/control-plane/pkg/client/injection/client"
)

// WaitForKafkaSourceConsuming polls until the KafkaSource has active consumers
// with partition assignments. The duck-typed IsReady only checks control-plane
// conditions; this additionally verifies the data-plane fields
// (status.consumers, status.placements) that prove the dispatcher joined the
// consumer group. On failure it dumps the full status for diagnosis.
func WaitForKafkaSourceConsuming(name string) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		interval, timeout := environment.PollTimingsFromContext(ctx)
		ns := environment.FromContext(ctx).Namespace()
		client := kafkaclientset.Get(ctx).SourcesV1().KafkaSources(ns)

		var last *sourcesv1.KafkaSource
		err := wait.PollUntilContextTimeout(ctx, interval, timeout, true, func(ctx2 context.Context) (bool, error) {
			ks, err := client.Get(ctx2, name, metav1.GetOptions{})
			if err != nil {
				return false, fmt.Errorf("failed to get KafkaSource %s/%s: %w", ns, name, err)
			}
			last = ks

			if !ks.Status.IsReady() {
				return false, nil
			}

			if ks.Status.Consumers < 1 {
				return false, nil
			}

			totalVReplicas := int32(0)
			for _, p := range ks.Status.Placements {
				totalVReplicas += p.VReplicas
			}
			return totalVReplicas >= 1, nil
		})

		if err != nil {
			bytes, _ := json.MarshalIndent(last.Status, "", "  ")
			t.Fatalf("KafkaSource %s did not reach consuming state: %v\nStatus:\n%s", name, err, string(bytes))
		}
	}
}
