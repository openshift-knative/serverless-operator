package eventinge2e

import (
	"fmt"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	"github.com/openshift-knative/serverless-operator/test/servinge2e"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1beta1 "knative.dev/eventing/pkg/apis/eventing/v1beta1"
	eventingsourcesv1alpha2 "knative.dev/eventing/pkg/apis/sources/v1alpha2"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const (
	brokerName        = "smoke-test-broker"
	triggerName       = "smoke-test-trigger"
	cmName            = "smoke-test-br-cm"
	brokerAPIVersion  = "eventing.knative.dev/v1beta1"
	brokerKind        = "Broker"
	triggerAPIVersion = "eventing.knative.dev/v1beta1"
	triggerKind       = "trigger"
)

func TestKnativeSourceBrokerTriggerKnativeService(t *testing.T) {
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)
		client.Clients.Eventing.EventingV1beta1().Brokers(testNamespace).Delete(brokerName, &metav1.DeleteOptions{})
		client.Clients.Eventing.EventingV1beta1().Triggers(testNamespace).Delete(triggerName, &metav1.DeleteOptions{})
		client.Clients.Eventing.SourcesV1alpha2().PingSources(testNamespace).Delete(pingSourceName, &metav1.DeleteOptions{})
		client.Clients.Kube.CoreV1().ConfigMaps(testNamespace).Delete(cmName, &metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer test.CleanupAll(t, client)
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
	configMap, err := client.Clients.Kube.CoreV1().ConfigMaps(testNamespace).Create(cm)
	if err != nil {
		t.Fatal("Unable to create ConfigMap: ", err)
	}
	br := &eventingv1beta1.Broker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      brokerName,
			Namespace: testNamespace,
		},
		Spec: eventingv1beta1.BrokerSpec{
			Config: &duckv1.KReference{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       configMap.Name,
			},
		},
	}
	broker, err := client.Clients.Eventing.EventingV1beta1().Brokers(testNamespace).Create(br)
	if err != nil {
		t.Fatal("Unable to create broker: ", err)
	}
	tr := &eventingv1beta1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      triggerName,
			Namespace: testNamespace,
		},
		Spec: eventingv1beta1.TriggerSpec{
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
	_, err = client.Clients.Eventing.EventingV1beta1().Triggers(testNamespace).Create(tr)
	if err != nil {
		t.Fatal("Unable to create trigger: ", err)
	}

	ps := &eventingsourcesv1alpha2.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pingSourceName,
			Namespace: testNamespace,
		},
		Spec: eventingsourcesv1alpha2.PingSourceSpec{
			JsonData: helloWorldText,
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
	_, err = client.Clients.Eventing.SourcesV1alpha2().PingSources(testNamespace).Create(ps)
	if err != nil {
		t.Fatal("Knative PingSource not created: %+V", err)
	}
	servinge2e.WaitForRouteServingText(t, client, ksvc.Status.URL.URL(), helloWorldText)

}
