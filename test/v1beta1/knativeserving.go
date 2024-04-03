package v1beta1

import (
	"context"
	"fmt"

	"github.com/openshift-knative/serverless-operator/test"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

func KnativeServing(name, namespace string) *operatorv1beta1.KnativeServing {
	return &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func WithKnativeServingReady(ctx *test.Context, name, namespace string) (*operatorv1beta1.KnativeServing, error) {
	serving, err := CreateKnativeServing(ctx, name, namespace)
	if err != nil {
		return nil, err
	}
	if _, err = WaitForKnativeServingState(ctx, serving.Name, serving.Namespace, IsKnativeServingReady); err != nil {
		return nil, err
	}
	return serving, nil
}

func CreateKnativeServing(ctx *test.Context, name, namespace string) (*operatorv1beta1.KnativeServing, error) {
	serving, err := ctx.Clients.Operator.KnativeServings(namespace).Create(context.Background(), KnativeServing(name, namespace), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up KnativeServing '%s/%s'", serving.Namespace, serving.Name)
		return DeleteKnativeServing(ctx, name, namespace)
	})
	return serving, nil
}

func DeleteKnativeServing(ctx *test.Context, name, namespace string) error {
	if err := ctx.Clients.Operator.KnativeServings(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	// Wait until the KnativeServing got removed.
	_, err := WaitForKnativeServingState(ctx, name, namespace,
		func(_ *operatorv1beta1.KnativeServing, err error) (bool, error) {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		})
	return err
}

func WaitForKnativeServingState(ctx *test.Context, name, namespace string, inState ServingInStateFunc) (*operatorv1beta1.KnativeServing, error) {
	var (
		lastState *operatorv1beta1.KnativeServing
		err       error
	)
	waitErr := wait.PollUntilContextTimeout(context.Background(), test.Interval, test.Timeout, true, func(_ context.Context) (bool, error) {
		lastState, err = ctx.Clients.Operator.KnativeServings(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("knativeserving %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

func UpdateServingExpectedScale(ctx *test.Context, name, namespace string, deployments []test.Deployment, defaultScale *int32) error {
	serving, err := ctx.Clients.Operator.KnativeServings(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for i := range deployments {
		for _, w := range serving.Spec.Workloads {
			if deployments[i].Name == w.Name {
				deployments[i].ExpectedScale = w.Replicas
			}
		}
		if deployments[i].ExpectedScale == nil {
			if serving.Spec.HighAvailability != nil && serving.Spec.HighAvailability.Replicas != nil {
				deployments[i].ExpectedScale = serving.Spec.HighAvailability.Replicas
			} else {
				deployments[i].ExpectedScale = defaultScale
			}
		}
	}
	return nil
}

func IsKnativeServingReady(s *operatorv1beta1.KnativeServing, err error) (bool, error) {
	return s.Status.IsReady(), err
}

type ServingInStateFunc func(s *operatorv1beta1.KnativeServing, err error) (bool, error)

func IsKnativeServingWithVersionReady(version string) ServingInStateFunc {
	return func(s *operatorv1beta1.KnativeServing, err error) (bool, error) {
		return s.Status.Version == version && s.Status.IsReady(), err
	}
}
