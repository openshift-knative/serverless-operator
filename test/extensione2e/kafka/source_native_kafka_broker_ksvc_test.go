package knativekafkae2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	sourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
)

const (
	nativeKafkaBrokerName = "smoke-test-native-kafka-broker"
	kafkaTriggerName      = "smoke-test-kafka-trigger"
	kafkaTriggerKsvcName  = helloWorldService + "-" + kafkaTriggerName
)

var (
	nativeKafkaBroker = &eventingv1.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nativeKafkaBrokerName,
			Namespace:   testNamespace,
			Annotations: map[string]string{"eventing.knative.dev/broker.class": "Kafka"},
		},
		Spec: eventingv1.BrokerSpec{
			Config: &duckv1.KReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "kafka-broker-config",
				Namespace:  "knative-eventing",
			},
		},
	}

	triggerForNativeBroker = &eventingv1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaTriggerName,
			Namespace: testNamespace,
		},
		Spec: eventingv1.TriggerSpec{
			Broker: nativeKafkaBrokerName,
			Subscriber: duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: ksvcAPIVersion,
					Kind:       ksvcKind,
					Name:       helloWorldService + "-native-kafka-channel-broker",
				},
			},
		},
	}

	nativeKafkaBrokerPingSource = &sourcesv1.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pingSourceName,
			Namespace: testNamespace,
		},
		Spec: sourcesv1.PingSourceSpec{
			Data: helloWorldText,
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: brokerAPIVersion,
						Kind:       brokerKind,
						Name:       nativeKafkaBrokerName,
					},
				},
			},
		},
	}
)

func TestSourceToNativeKafkaBasedBrokerToKnativeService(t *testing.T) {
	ctx := context.Background()
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)
		ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
		defer cancel()

		if err := client.Clients.Eventing.EventingV1().Brokers(testNamespace).Delete(ctx, nativeKafkaBrokerName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			br, _ := client.Clients.Eventing.EventingV1().Brokers(testNamespace).Get(ctx, nativeKafkaBrokerName, metav1.GetOptions{})
			brStr, _ := json.Marshal(br)
			t.Errorf("failed to delete broker %s/%s: %v\n%s\n", testNamespace, nativeKafkaBrokerName, err, string(brStr))
		}
		if err := client.Clients.Eventing.SourcesV1().PingSources(testNamespace).Delete(ctx, pingSourceName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("failed to delete pingsource %s/%s: %v", testNamespace, pingSourceName, err)
		}
		if err := client.Clients.Eventing.EventingV1().Triggers(testNamespace).Delete(ctx, kafkaTriggerName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			tr, _ := client.Clients.Eventing.EventingV1().Triggers(testNamespace).Get(ctx, kafkaTriggerName, metav1.GetOptions{})
			trStr, _ := json.Marshal(tr)
			t.Errorf("failed to delete trigger %s/%s: %v\n%s\n", testNamespace, kafkaTriggerName, err, string(trStr))
		}
		if err := client.Clients.Serving.ServingV1().Services(testNamespace).Delete(ctx, kafkaTriggerKsvcName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("failed to delete ksvc %s/%s: %v", testNamespace, kafkaTriggerKsvcName, err)
		}
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	ksvc, err := test.WithServiceReady(client, kafkaTriggerKsvcName, testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Create the (native) Kafka Broker
	_, err = client.Clients.Eventing.EventingV1().Brokers(testNamespace).Create(context.Background(), nativeKafkaBroker, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create Kafka Backed Broker: ", err)
	}

	// Create the Trigger
	_, err = client.Clients.Eventing.EventingV1().Triggers(testNamespace).Create(context.Background(), triggerForNativeBroker, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create trigger: ", err)
	}

	// Create the source
	_, err = client.Clients.Eventing.SourcesV1().PingSources(testNamespace).Create(context.Background(), nativeKafkaBrokerPingSource, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create pingsource: ", err)
	}

	// Wait for text in kservice
	servinge2e.WaitForRouteServingText(t, client, ksvc.Status.URL.URL(), helloWorldText)
}
