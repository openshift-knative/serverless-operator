package eventinge2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	eventingsourcesv1 "knative.dev/eventing/pkg/apis/sources/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const (
	brokerName       = "smoke-test-broker"
	triggerName      = "smoke-test-trigger"
	cmName           = "smoke-test-br-cm"
	brokerAPIVersion = "eventing.knative.dev/v1"
	brokerKind       = "Broker"
)

func TestKnativeSourceBrokerTriggerKnativeService(t *testing.T) {
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)
		client.Clients.Eventing.EventingV1().Brokers(testNamespace).Delete(context.Background(), brokerName, metav1.DeleteOptions{})
		client.Clients.Eventing.EventingV1().Triggers(testNamespace).Delete(context.Background(), triggerName, metav1.DeleteOptions{})
		client.Clients.Eventing.SourcesV1().PingSources(testNamespace).Delete(context.Background(), pingSourceName, metav1.DeleteOptions{})
		client.Clients.Kube.CoreV1().ConfigMaps(testNamespace).Delete(context.Background(), cmName, metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	// Setup a knative service
	ksvc, err := test.WithServiceReady(client, helloWorldService, testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: cmName,
		},
		Data: map[string]string{
			"channelTemplateSpec": fmt.Sprintf(`
apiVersion: %q
kind: %q`, channelAPIVersion, channelKind),
		},
	}
	configMap, err := client.Clients.Kube.CoreV1().ConfigMaps(testNamespace).Create(context.Background(), cm, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create ConfigMap: ", err)
	}
	br := &eventingv1.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      brokerName,
			Namespace: testNamespace,
		},
		Spec: eventingv1.BrokerSpec{
			Config: &duckv1.KReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       configMap.Name,
			},
		},
	}
	broker, err := client.Clients.Eventing.EventingV1().Brokers(testNamespace).Create(context.Background(), br, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create broker: ", err)
	}
	tr := &eventingv1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      triggerName,
			Namespace: testNamespace,
		},
		Spec: eventingv1.TriggerSpec{
			Broker: broker.Name,
			Subscriber: duckv1.Destination{
				Ref: &duckv1.KReference{
					APIVersion: ksvcAPIVersion,
					Kind:       ksvcKind,
					Name:       helloWorldService,
				},
			},
		},
	}
	_, err = client.Clients.Eventing.EventingV1().Triggers(testNamespace).Create(context.Background(), tr, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create trigger: ", err)
	}

	ps := &eventingsourcesv1.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pingSourceName,
			Namespace: testNamespace,
		},
		Spec: eventingsourcesv1.PingSourceSpec{
			Data: helloWorldText,
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: brokerAPIVersion,
						Kind:       brokerKind,
						Name:       broker.Name,
					},
				},
			},
		},
	}
	_, err = client.Clients.Eventing.SourcesV1().PingSources(testNamespace).Create(context.Background(), ps, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Knative PingSource not created: %+V", err)
	}
	servinge2e.WaitForRouteServingText(t, client, ksvc.Status.URL.URL(), helloWorldText)

}
