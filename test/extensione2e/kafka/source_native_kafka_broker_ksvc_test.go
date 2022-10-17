package knativekafkae2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	pkgTest "knative.dev/pkg/test"

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
			Namespace:   test.Namespace,
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
			Namespace: test.Namespace,
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
			Namespace: test.Namespace,
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

		if err := client.Clients.Eventing.EventingV1().Brokers(test.Namespace).Delete(ctx, nativeKafkaBrokerName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			br, _ := client.Clients.Eventing.EventingV1().Brokers(test.Namespace).Get(ctx, nativeKafkaBrokerName, metav1.GetOptions{})
			brStr, _ := json.Marshal(br)
			t.Errorf("failed to delete broker %s/%s: %v\n%s\n", test.Namespace, nativeKafkaBrokerName, err, string(brStr))
		}
		if err := client.Clients.Eventing.SourcesV1().PingSources(test.Namespace).Delete(ctx, pingSourceName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("failed to delete pingsource %s/%s: %v", test.Namespace, pingSourceName, err)
		}
		if err := client.Clients.Eventing.EventingV1().Triggers(test.Namespace).Delete(ctx, kafkaTriggerName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			tr, _ := client.Clients.Eventing.EventingV1().Triggers(test.Namespace).Get(ctx, kafkaTriggerName, metav1.GetOptions{})
			trStr, _ := json.Marshal(tr)
			t.Errorf("failed to delete trigger %s/%s: %v\n%s\n", test.Namespace, kafkaTriggerName, err, string(trStr))
		}
		if err := client.Clients.Serving.ServingV1().Services(test.Namespace).Delete(ctx, kafkaTriggerKsvcName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("failed to delete ksvc %s/%s: %v", test.Namespace, kafkaTriggerKsvcName, err)
		}

		err := wait.PollImmediateUntil(2*time.Second, waitForBrokerDeletion(ctx, client, t), ctx.Done())
		if err != nil {
			t.Fatal(err)
		}

		cmName := nativeKafkaBroker.Spec.Config.Name
		cmNamepace := nativeKafkaBroker.Spec.Config.Namespace
		cm, err := client.Clients.Kube.
			CoreV1().
			ConfigMaps(cmNamepace).
			Get(ctx, cmName, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get ConfigMap")
		}
		for _, f := range cm.GetFinalizers() {
			if strings.Contains(f, nativeKafkaBrokerName) && strings.Contains(f, test.Namespace) {
				cmBytes, _ := json.MarshalIndent(cm, "", " ")
				t.Fatalf("ConfigMap still contains the finalizer %s\n%s\n", f, string(cmBytes))
			}
		}
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	ksvc, err := test.WithServiceReady(client, kafkaTriggerKsvcName, test.Namespace, pkgTest.ImagePath(test.HelloworldGoImg))
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Create the (native) Kafka Broker
	_, err = client.Clients.Eventing.EventingV1().Brokers(test.Namespace).Create(context.Background(), nativeKafkaBroker, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create Kafka Backed Broker: ", err)
	}

	// Create the Trigger
	_, err = client.Clients.Eventing.EventingV1().Triggers(test.Namespace).Create(context.Background(), triggerForNativeBroker, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create trigger: ", err)
	}

	// Create the source
	_, err = client.Clients.Eventing.SourcesV1().PingSources(test.Namespace).Create(context.Background(), nativeKafkaBrokerPingSource, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create pingsource: ", err)
	}

	// Wait for text in kservice
	servinge2e.WaitForRouteServingText(t, client, ksvc.Status.URL.URL(), helloWorldText)
}

func waitForBrokerDeletion(ctx context.Context, client *test.Context, t *testing.T) wait.ConditionFunc {
	return func() (bool, error) {
		br, err := client.
			Clients.
			Eventing.
			EventingV1().
			Brokers(test.Namespace).
			Get(ctx, nativeKafkaBrokerName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, fmt.Errorf("failed to get broker %s/%s: %w", test.Namespace, nativeKafkaBrokerName, err)
		}

		brBytes, _ := json.MarshalIndent(br, "", " ")
		t.Logf("Broker still present\n%s\n", string(brBytes))

		return false, nil
	}
}
