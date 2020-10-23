package knativekafkae2e

import (
	"fmt"
	"net/url"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kafkabindingv1beta1 "knative.dev/eventing-contrib/kafka/source/pkg/apis/bindings/v1beta1"
	kafkasourcev1beta1 "knative.dev/eventing-contrib/kafka/source/pkg/apis/sources/v1beta1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	pkgTest "knative.dev/pkg/test"

	"github.com/openshift-knative/serverless-operator/test"
)

const (
	kafkaSourceName     = "smoke-ks"
	kafkaTopicName      = "smoke-topic"
	kafkaConsumerGroup  = "smoke-cg"
	testNamespace       = "serverless-tests"
	image               = "gcr.io/knative-samples/helloworld-go"
	helloWorldService   = "helloworld-go"
	ksvcAPIVersion      = "serving.knative.dev/v1"
	ksvcKind            = "Service"
	kafkaTopicKind      = "KafkaTopic"
	kafkaAPIVersion     = "kafka.strimzi.io/v1beta1"
	clusterName         = "my-cluster" // there should be a way to get this from test setup
	strimziClusterLabel = "strimzi.io/cluster"
	cronJobName         = "smoke-cronjob"
)

var (
	bootstrapServer = clusterName + "-kafka-bootstrap.kafka:9092"
	kafkaGVR        = schema.GroupVersionResource{Group: "kafka.strimzi.io", Version: "v1beta1", Resource: "kafkatopics"}

	// We use unstructured to avoid having a hard dep on any specific kafka implementation
	kafkaTopicObj = unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": kafkaAPIVersion,
			"kind":       kafkaTopicKind,
			"metadata": map[string]interface{}{
				"name":      kafkaTopicName,
				"namespace": testNamespace,
				"labels": map[string]interface{}{
					strimziClusterLabel: clusterName,
				},
			},
			//Taken from https://github.com/strimzi/strimzi-kafka-operator/blob/0.19.0/examples/topic/kafka-topic.yaml
			"spec": map[string]interface{}{
				"partitions": 1,
				"replicas":   1,
			},
		},
	}

	kafkaSource = kafkasourcev1beta1.KafkaSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaSourceName,
			Namespace: testNamespace,
		},
		Spec: kafkasourcev1beta1.KafkaSourceSpec{
			KafkaAuthSpec: kafkabindingv1beta1.KafkaAuthSpec{
				BootstrapServers: []string{bootstrapServer},
			},
			Topics:        []string{kafkaTopicName},
			ConsumerGroup: kafkaConsumerGroup,
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: ksvcAPIVersion,
						Kind:       ksvcKind,
						Name:       helloWorldService,
					},
				},
			},
		},
	}

	cj = &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronJobName,
			Namespace: testNamespace,
		},
		Spec: batchv1beta1.CronJobSpec{
			Schedule: "* * * * *",
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    "kafka-message-test",
									Image:   "strimzi/kafka:0.16.2-kafka-2.4.0",
									Command: []string{"sh", "-c", fmt.Sprintf(`echo "%s" | bin/kafka-console-producer.sh --broker-list %s --topic %s`, helloWorldText, bootstrapServer, kafkaTopicName)},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}
)

func TestKafkaSourceToKnativeService(t *testing.T) {
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)
		client.Clients.Dynamic.Resource(kafkaGVR).Namespace(testNamespace).Delete(kafkaTopicName, &metav1.DeleteOptions{})
		client.Clients.KafkaSource.SourcesV1beta1().KafkaSources(testNamespace).Delete(kafkaSourceName, &metav1.DeleteOptions{})
		client.Clients.Kube.BatchV1beta1().CronJobs(testNamespace).Delete(cronJobName, &metav1.DeleteOptions{})
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer test.CleanupAll(t, client)
	defer cleanup()

	// Setup a knative service
	ksvc, err := test.WithServiceReady(client, helloWorldService, testNamespace, image)
	if err != nil {
		t.Fatal("Knative Service not ready", err)
	}

	// Create kafkatopic
	_, err = client.Clients.Dynamic.Resource(kafkaGVR).Namespace(testNamespace).Create(&kafkaTopicObj, metav1.CreateOptions{})
	if err != nil {
		t.Fatal("Unable to create KafkaTopic: ", err)
	}

	// create kafka source
	_, err = client.Clients.KafkaSource.SourcesV1beta1().KafkaSources(testNamespace).Create(&kafkaSource)
	if err != nil {
		t.Fatal("Unable to create kafkaSource: ", err)
	}

	// send event to kafka topic
	_, err = client.Clients.Kube.BatchV1beta1().CronJobs(testNamespace).Create(cj)
	if err != nil {
		t.Fatal("Unable to create batch cronjob: ", err)
	}

	waitForRouteServingText(t, client, ksvc.Status.URL.URL(), helloWorldText)
	// cleanup if everything ends smoothly
	cleanup()
}

// This should probably move to an exported function from servinge2e
func waitForRouteServingText(t *testing.T, client *test.Context, routeURL *url.URL, expectedText string) {
	t.Helper()
	if _, err := pkgTest.WaitForEndpointState(
		&pkgTest.KubeClient{Kube: client.Clients.Kube},
		t.Logf,
		routeURL,
		pkgTest.EventuallyMatchesBody(expectedText),
		"WaitForRouteToServeText",
		true); err != nil {
		t.Fatalf("The Route at domain %s didn't serve the expected text \"%s\": %v", routeURL, expectedText, err)
	}

}
