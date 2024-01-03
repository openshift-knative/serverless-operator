package v1alpha1

import (
	"context"
	"fmt"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	kafkav1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/test"
)

func KnativeKafka(name, namespace string) *kafkav1alpha1.KnativeKafka {
	return &kafkav1alpha1.KnativeKafka{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KnativeKafka",
			APIVersion: "operator.serverless.openshift.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: kafkav1alpha1.KnativeKafkaSpec{
			Broker: kafkav1alpha1.Broker{
				Enabled: true,
				DefaultConfig: kafkav1alpha1.BrokerDefaultConfig{
					BootstrapServers: "my-cluster-kafka-bootstrap.kafka:9092",
				},
			},
			Source: kafkav1alpha1.Source{
				Enabled: true,
			},
			Channel: kafkav1alpha1.Channel{
				Enabled:          true,
				BootstrapServers: "my-cluster-kafka-bootstrap.kafka:9092",
			},
		},
	}
}

func WithKnativeKafkaReady(ctx *test.Context, name, namespace string) (*kafkav1alpha1.KnativeKafka, error) {
	kafka, err := CreateKnativeKafka(ctx, name, namespace)
	if err != nil {
		return nil, err
	}
	if _, err = WaitForKnativeKafkaState(ctx, kafka.Name, kafka.Namespace, IsKnativeKafkaReady); err != nil {
		return nil, err
	}
	return kafka, nil
}

func CreateKnativeKafka(ctx *test.Context, name, namespace string) (*kafkav1alpha1.KnativeKafka, error) {
	uo, err := runtime.DefaultUnstructuredConverter.ToUnstructured(KnativeKafka(name, namespace))
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{Object: uo}
	ru, err := ctx.Clients.Dynamic.Resource(kafkav1alpha1.SchemeGroupVersion.WithResource("knativekafkas")).Namespace(namespace).Create(context.Background(), u, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	kafka := &kafkav1alpha1.KnativeKafka{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(ru.Object, kafka)
	if err != nil {
		return nil, err
	}
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up KnativeKafka '%s/%s'", kafka.Namespace, kafka.Name)
		return DeleteKnativeKafka(ctx, name, namespace)
	})
	return kafka, nil
}

func DeleteKnativeKafka(ctx *test.Context, name, namespace string) error {
	if err := ctx.Clients.Dynamic.Resource(kafkav1alpha1.SchemeGroupVersion.WithResource("knativekafkas")).Namespace(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	// Wait until the KnativeKafka got removed.
	_, err := WaitForKnativeKafkaState(ctx, name, namespace,
		func(s *kafkav1alpha1.KnativeKafka, err error) (bool, error) {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		})
	return err
}

func WaitForKnativeKafkaState(ctx *test.Context, name, namespace string, inState KafkaInStateFunc) (*kafkav1alpha1.KnativeKafka, error) {
	var (
		lastState *kafkav1alpha1.KnativeKafka
		err       error
	)
	waitErr := wait.PollImmediate(test.Interval, 3*test.Timeout, func() (bool, error) {
		lastState = &kafkav1alpha1.KnativeKafka{}
		var u *unstructured.Unstructured
		u, err = ctx.Clients.Dynamic.Resource(kafkav1alpha1.SchemeGroupVersion.WithResource("knativekafkas")).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return inState(nil, err)
		}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, lastState)
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("knativekafka %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

func UpdateKnativeKafkaExpectedScale(ctx *test.Context, name, namespace string, deployments []test.Deployment, defaultScale *int32) error {
	knativeKafka := &kafkav1alpha1.KnativeKafka{}
	var u *unstructured.Unstructured
	u, err := ctx.Clients.Dynamic.Resource(kafkav1alpha1.SchemeGroupVersion.WithResource("knativekafkas")).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, knativeKafka)

	for i := range deployments {
		for _, w := range knativeKafka.Spec.Workloads {
			if deployments[i].Name == w.Name {
				deployments[i].ExpectedScale = w.Replicas
			}
		}
		if deployments[i].ExpectedScale == nil {
			if knativeKafka.Spec.HighAvailability != nil && knativeKafka.Spec.HighAvailability.Replicas != nil {
				deployments[i].ExpectedScale = knativeKafka.Spec.HighAvailability.Replicas
			} else {
				deployments[i].ExpectedScale = defaultScale
			}
		}
	}
	return nil
}

func IsKnativeKafkaReady(s *kafkav1alpha1.KnativeKafka, err error) (bool, error) {
	return s.Status.IsReady(), err
}

type KafkaInStateFunc func(k *kafkav1alpha1.KnativeKafka, err error) (bool, error)

func IsKnativeKafkaWithVersionReady(version string) KafkaInStateFunc {
	return func(k *kafkav1alpha1.KnativeKafka, err error) (bool, error) {
		return k.Status.Version == version && k.Status.IsReady(), err
	}
}
