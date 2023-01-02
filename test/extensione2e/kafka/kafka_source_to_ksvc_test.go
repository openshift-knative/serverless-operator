package knativekafkae2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/openshift-knative/serverless-operator/test/eventinge2e"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/eventing-kafka/test/e2e/helpers"
	"knative.dev/eventing/pkg/utils"
	"knative.dev/eventing/test/lib"

	kafkabindingv1beta1 "knative.dev/eventing-kafka/pkg/apis/bindings/v1beta1"
	kafkasourcev1beta1 "knative.dev/eventing-kafka/pkg/apis/sources/v1beta1"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/test"
)

const (
	kafkaSourceName    = "smoke-ks"
	kafkaTopicName     = "smoke-topic"
	kafkaConsumerGroup = "smoke-cg"
	helloWorldService  = "helloworld-go"
	ksvcAPIVersion     = "serving.knative.dev/v1"
	ksvcKind           = "Service"
	clusterName        = "my-cluster" // there should be a way to get this from test setup
	cronJobName        = "smoke-cronjob"
)

var (
	baseURI              = "-kafka-bootstrap.kafka:"
	plainBootstrapServer = clusterName + baseURI + "9092"
	tlsBootstrapServer   = clusterName + baseURI + "9093"
	saslBootstrapServer  = clusterName + baseURI + "9094"
	tlsSecret            = "my-tls-secret"
	saslSecret           = "my-sasl-secret"
	kafkaGVR             = schema.GroupVersionResource{Group: "kafka.strimzi.io", Version: "v1beta1", Resource: "kafkatopics"}
)

func createCronJobObjV1Beta1(name, topic, server string) *batchv1beta1.CronJob {
	return &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: test.Namespace,
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
									Command: []string{"sh", "-c", fmt.Sprintf(`echo '%s' | bin/kafka-console-producer.sh --broker-list %s --topic %s`, eventinge2e.PingSourceData, server, topic)},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}
}

func createCronJobObjV1(name, topic, server string) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: test.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "* * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    "kafka-message-test",
									Image:   "strimzi/kafka:0.16.2-kafka-2.4.0",
									Command: []string{"sh", "-c", fmt.Sprintf(`echo '%s' | bin/kafka-console-producer.sh --broker-list %s --topic %s`, eventinge2e.PingSourceData, server, topic)},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}
}

func createKafkaSourceObj(sourceName, sinkName, topicName string, auth kafkabindingv1beta1.KafkaAuthSpec) kafkasourcev1beta1.KafkaSource {
	return kafkasourcev1beta1.KafkaSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceName,
			Namespace: test.Namespace,
		},
		Spec: kafkasourcev1beta1.KafkaSourceSpec{
			KafkaAuthSpec: auth,
			Topics:        []string{topicName},
			ConsumerGroup: kafkaConsumerGroup + "-" + sourceName,
			SourceSpec: duckv1.SourceSpec{
				Sink: duckv1.Destination{
					Ref: &duckv1.KReference{
						APIVersion: ksvcAPIVersion,
						Kind:       ksvcKind,
						Name:       sinkName,
					},
				},
			},
		},
	}
}

func TestKafkaSourceToKnativeService(t *testing.T) {
	client := test.SetupClusterAdmin(t)
	cleanup := func() {
		test.CleanupAll(t, client)

		_ = deleteKafkaSource(client, test.Namespace, kafkaSourceName+"-plain")
		_ = deleteKafkaSource(client, test.Namespace, kafkaSourceName+"-sasl")
		_ = deleteKafkaSource(client, test.Namespace, kafkaSourceName+"-tls")

		// Delete topics
		client.Clients.Dynamic.Resource(kafkaGVR).Namespace("kafka").Delete(context.Background(), kafkaTopicName+"-plain", metav1.DeleteOptions{})
		client.Clients.Dynamic.Resource(kafkaGVR).Namespace("kafka").Delete(context.Background(), kafkaTopicName+"-tls", metav1.DeleteOptions{})
		client.Clients.Dynamic.Resource(kafkaGVR).Namespace("kafka").Delete(context.Background(), kafkaTopicName+"-sasl", metav1.DeleteOptions{})

		// Jobs and Pods are sometimes left in the namespace.
		// Ref: https://github.com/kubernetes/kubernetes/issues/74741
		if err := common.CheckMinimumKubeVersion(client.Clients.Kube.Discovery(), common.MinimumK8sAPIDeprecationVersion); err == nil {
			client.Clients.Kube.BatchV1().CronJobs(test.Namespace).Delete(context.Background(), cronJobName+"-plain", metav1.DeleteOptions{})
			client.Clients.Kube.BatchV1().CronJobs(test.Namespace).Delete(context.Background(), cronJobName+"-tls", metav1.DeleteOptions{})
			client.Clients.Kube.BatchV1().CronJobs(test.Namespace).Delete(context.Background(), cronJobName+"-sasl", metav1.DeleteOptions{})
			deleteJobs(t, client, test.Namespace, cronJobName)
		} else {
			client.Clients.Kube.BatchV1beta1().CronJobs(test.Namespace).Delete(context.Background(), cronJobName+"-plain", metav1.DeleteOptions{})
			client.Clients.Kube.BatchV1beta1().CronJobs(test.Namespace).Delete(context.Background(), cronJobName+"-tls", metav1.DeleteOptions{})
			client.Clients.Kube.BatchV1beta1().CronJobs(test.Namespace).Delete(context.Background(), cronJobName+"-sasl", metav1.DeleteOptions{})
			deleteJobsV1Beta1(t, client, test.Namespace, cronJobName)
		}
		deletePods(t, client, test.Namespace, cronJobName)
		client.Clients.Kube.CoreV1().Secrets(test.Namespace).Delete(context.Background(), tlsSecret, metav1.DeleteOptions{})
		client.Clients.Kube.CoreV1().Secrets(test.Namespace).Delete(context.Background(), saslSecret, metav1.DeleteOptions{})
		removePullSecretFromSA(t, client, test.Namespace, serviceAccount, tlsSecret)
		removePullSecretFromSA(t, client, test.Namespace, serviceAccount, saslSecret)
	}
	test.CleanupOnInterrupt(t, cleanup)
	defer cleanup()

	// Get Secret Name -> AuthSecretName
	_, err := utils.CopySecret(client.Clients.Kube.CoreV1(), "default", tlsSecret, test.Namespace, serviceAccount)
	if err != nil {
		t.Fatalf("Could not copy Secret: %s to test namespace: %s: %v", tlsSecret, test.Namespace, err)
	}

	_, err = utils.CopySecret(client.Clients.Kube.CoreV1(), "default", saslSecret, test.Namespace, serviceAccount)
	if err != nil {
		t.Fatalf("Could not copy Secret: %s to test namespace: %s: %v", saslSecret, test.Namespace, err)
	}

	tests := map[string]kafkabindingv1beta1.KafkaAuthSpec{
		"plain": {
			BootstrapServers: []string{plainBootstrapServer},
		},
		"tls": {
			BootstrapServers: []string{tlsBootstrapServer},
			Net: kafkabindingv1beta1.KafkaNetSpec{
				TLS: kafkabindingv1beta1.KafkaTLSSpec{
					Enable: true,
					Cert: kafkabindingv1beta1.SecretValueFromSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: tlsSecret,
							},
							Key: "user.crt",
						},
					},
					Key: kafkabindingv1beta1.SecretValueFromSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: tlsSecret,
							},
							Key: "user.key",
						},
					},
					CACert: kafkabindingv1beta1.SecretValueFromSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: tlsSecret,
							},
							Key: "ca.crt",
						},
					},
				},
			},
		},
		"sasl": {
			BootstrapServers: []string{saslBootstrapServer},
			Net: kafkabindingv1beta1.KafkaNetSpec{
				TLS: kafkabindingv1beta1.KafkaTLSSpec{
					Enable: true,
					CACert: kafkabindingv1beta1.SecretValueFromSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: tlsSecret,
							},
							Key: "ca.crt",
						},
					},
				},
				SASL: kafkabindingv1beta1.KafkaSASLSpec{
					Enable: true,
					User: kafkabindingv1beta1.SecretValueFromSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: saslSecret,
							},
							Key: "user",
						},
					},
					Password: kafkabindingv1beta1.SecretValueFromSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: saslSecret,
							},
							Key: "password",
						},
					},
					Type: kafkabindingv1beta1.SecretValueFromSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: saslSecret,
							},
							Key: "saslType",
						},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		name := name

		// Setup a knative service
		eventStore, ksvc := eventinge2e.DeployKsvcWithEventInfoStoreOrFail(client, t, test.Namespace, helloWorldService+"-"+name)

		t.Logf("Knative service %s/%s is ready: %#v", ksvc.GetNamespace(), ksvc.GetName(), ksvc.Status)

		topicName := kafkaTopicName + "-" + name
		c := &lib.Client{
			Kube:          client.Clients.Kube,
			Eventing:      client.Clients.Eventing,
			Dynamic:       client.Clients.Dynamic,
			Config:        nil,
			EventListener: nil,
			Namespace:     test.Namespace,
			T:             t,
			Tracker:       lib.NewTracker(t, client.Clients.Dynamic),
			TracingCfg:    "",
		}
		helpers.MustCreateTopic(c, clusterName, "kafka", topicName, 1)

		kafkaSource := createKafkaSourceObj(kafkaSourceName+"-"+name, helloWorldService+"-"+name, topicName, tc)
		_, err = client.Clients.Kafka.SourcesV1beta1().KafkaSources(test.Namespace).Create(context.Background(), &kafkaSource, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Unable to create kafkaSource(%s): %v", kafkaSource.GetName(), err)
		}

		var last *kafkasourcev1beta1.KafkaSource
		err = wait.Poll(time.Second, time.Minute, func() (done bool, err error) {
			ks, err := client.Clients.Kafka.
				SourcesV1beta1().
				KafkaSources(test.Namespace).
				Get(context.Background(), kafkaSource.GetName(), metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			last = ks
			return ks.Status.IsReady(), nil
		})
		if err != nil {
			t.Fatalf("failed while waiting for KafkaSource to become ready: %v\n%#v", err, last)
		}

		// send event to kafka topic
		if err := common.CheckMinimumKubeVersion(client.Clients.Kube.Discovery(), common.MinimumK8sAPIDeprecationVersion); err == nil {
			cj := createCronJobObjV1(cronJobName+"-"+name, topicName, kafkaSource.Spec.BootstrapServers[0])
			_, err = client.Clients.Kube.BatchV1().CronJobs(test.Namespace).Create(context.Background(), cj, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("Unable to create batch cronjob(%s): %v", cj.GetName(), err)
			}
		} else {
			cj := createCronJobObjV1Beta1(cronJobName+"-"+name, topicName, kafkaSource.Spec.BootstrapServers[0])
			_, err = client.Clients.Kube.BatchV1beta1().CronJobs(test.Namespace).Create(context.Background(), cj, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("Unable to create batch cronjob(%s): %v", cj.GetName(), err)
			}
		}

		eventinge2e.AssertPingSourceDataReceivedAtLeastOnce(eventStore)
	}
}

func deleteKafkaSource(client *test.Context, namespace string, name string) error {
	ctx := context.Background()
	pp := metav1.DeletePropagationForeground
	err := client.Clients.Kafka.SourcesV1beta1().KafkaSources(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &pp,
	})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	err = wait.Poll(time.Second, 2*test.Timeout, func() (done bool, err error) {
		_, err = client.Clients.Kafka.SourcesV1beta1().KafkaSources(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		client.T.Errorf("Failed to delete KafkaSource %s/%s: %v", namespace, name, err)
	}
	return err
}

func removePullSecretFromSA(t *testing.T, ctx *test.Context, namespace, serviceAccount, secretName string) {
	t.Helper()
	sa, err := ctx.Clients.Kube.CoreV1().ServiceAccounts(namespace).
		Get(context.Background(), serviceAccount, metav1.GetOptions{})
	if err != nil {
		t.Error("Unable to get ServiceAccount", serviceAccount)
	}
	for i, secret := range sa.ImagePullSecrets {
		if secret.Name == secretName {
			patch := []byte(fmt.Sprintf(`[{"op": "remove", "path": "/imagePullSecrets/%d"}]`, i))
			_, err = ctx.Clients.Kube.CoreV1().ServiceAccounts(namespace).
				Patch(context.Background(), serviceAccount, types.JSONPatchType, patch, metav1.PatchOptions{})
			if err != nil {
				t.Errorf("Patch failed on NS/SA (%s/%s): %s", namespace, serviceAccount, err)
			}
		}
	}
}

func deleteJobs(t *testing.T, ctx *test.Context, namespace, name string) {
	t.Helper()
	jobList, err := ctx.Clients.Kube.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Error("Unable to list jobs in namespace:", namespace)
	}
	for _, job := range jobList.Items {
		if strings.Contains(job.Name, name) {
			ctx.Clients.Kube.BatchV1().Jobs(namespace).
				Delete(context.Background(), job.Name, metav1.DeleteOptions{})
		}
	}
}

func deleteJobsV1Beta1(t *testing.T, ctx *test.Context, namespace, name string) {
	t.Helper()
	jobList, err := ctx.Clients.Kube.BatchV1beta1().CronJobs(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Error("Unable to list jobs in namespace:", namespace)
	}
	for _, job := range jobList.Items {
		if strings.Contains(job.Name, name) {
			ctx.Clients.Kube.BatchV1beta1().CronJobs(namespace).
				Delete(context.Background(), job.Name, metav1.DeleteOptions{})
		}
	}
}

func deletePods(t *testing.T, ctx *test.Context, namespace, name string) {
	t.Helper()
	podList, err := ctx.Clients.Kube.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Error("Unable to list pods in namespace:", namespace)
	}
	for _, pod := range podList.Items {
		if strings.Contains(pod.Name, name) {
			ctx.Clients.Kube.CoreV1().Pods(namespace).
				Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		}
	}
}
