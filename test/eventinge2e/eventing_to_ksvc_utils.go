package eventinge2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/eventing/test/lib"
	"knative.dev/eventing/test/lib/recordevents"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/helpers"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	pingSourceName = "smoke-test-ping"
	pingSourceData = "{\"message\":\"Hello, world!\"}"

	triggerName       = "smoke-test-trigger"
	cmName            = "smoke-test-br-cm"
	ksvcAPIVersion    = "serving.knative.dev/v1"
	ksvcKind          = "Service"
	helloWorldService = "helloworld-go"
	brokerAPIVersion  = "eventing.knative.dev/v1"
	brokerKind        = "Broker"

	subscriptionName = "smoke-test-subscription"
)

// DeployKsvcWithEventInfoStoreOrFail deploys a wathola-forwarder ksvc forwarding events to a recordevents receiver
func DeployKsvcWithEventInfoStoreOrFail(ctx *test.Context, t *testing.T, namespace string, name string) (*recordevents.EventInfoStore, *servingv1.Service) {
	libclient, err := lib.NewClient(namespace, t)
	if err != nil {
		t.Fatal("error creating testlib client", err)
	}

	lib.CreateRBACPodsGetEventsAll(libclient, namespace)
	lib.CreateRBACPodsEventsGetListWatch(libclient, namespace)

	// have a random suffix for the recordevents name,
	// so we can safely re-run the same tests and not get conflicts with Event names
	recordeventsname := helpers.AppendRandomString(name + "-re")

	eventStore, _ := recordevents.StartEventRecordOrFail(context.Background(), libclient, recordeventsname)

	ctx.AddToCleanup(func() error {
		return libclient.Tracker.Clean(true)
	})

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-config",
		},
		Data: map[string]string{
			"config.toml": "[forwarder]\ntarget = \"http://" + recordeventsname + "\"\n",
		},
	}
	_, err = ctx.Clients.Kube.CoreV1().ConfigMaps(namespace).Create(context.Background(), cm, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Error creating ConfigMap", cm.Name, err)
	}
	ctx.AddToCleanup(func() error {
		return ctx.Clients.Kube.CoreV1().ConfigMaps(namespace).Delete(context.Background(), name+"-config", metav1.DeleteOptions{})
	})

	// Setup a knative service for the wathola-forwarder
	ksvc, err := test.WithServiceReady(ctx, name, namespace, pkgTest.ImagePath(test.WatholaForwarderImg), func(service *servingv1.Service) {
		service.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: "config", VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name + "-config",
					},
				},
			}},
		}

		service.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{Name: "config", ReadOnly: true, MountPath: "/.config/wathola/"},
		}
	})

	if err != nil {
		t.Fatalf("%s does not become Ready: %v", name, err)
	}

	return eventStore, ksvc
}

func AssertPingSourceDataReceivedAtLeastOnce(eventStore *recordevents.EventInfoStore) {
	eventStore.AssertAtLeast(1, func(info recordevents.EventInfo) error {
		if info.Event == nil {
			return fmt.Errorf("event body nil")
		}
		if string(info.Event.Data()) != pingSourceData {
			return fmt.Errorf("event body %q does not match %q", info.Event.Data(), pingSourceData)
		}
		return nil
	})
}
