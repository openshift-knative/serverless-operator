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

func KnativeEventing(name, namespace string) *operatorv1beta1.KnativeEventing {
	return &operatorv1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func WithKnativeEventingReady(ctx *test.Context, name, namespace string) (*operatorv1beta1.KnativeEventing, error) {
	eventing, err := CreateKnativeEventing(ctx, name, namespace)
	if err != nil {
		return nil, err
	}
	if _, err = WaitForKnativeEventingState(ctx, eventing.Name, eventing.Namespace, IsKnativeEventingReady); err != nil {
		return nil, err
	}
	return eventing, nil
}

func CreateKnativeEventing(ctx *test.Context, name, namespace string) (*operatorv1beta1.KnativeEventing, error) {
	eventing, err := ctx.Clients.Operator.KnativeEventings(namespace).Create(context.Background(), KnativeEventing(name, namespace), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up KnativeEventing '%s/%s'", eventing.Namespace, eventing.Name)
		return DeleteKnativeEventing(ctx, name, namespace)
	})
	return eventing, nil
}

func DeleteKnativeEventing(ctx *test.Context, name, namespace string) error {
	if err := ctx.Clients.Operator.KnativeEventings(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	// Wait until the KnativeEventing got removed.
	_, err := WaitForKnativeEventingState(ctx, name, namespace,
		func(s *operatorv1beta1.KnativeEventing, err error) (bool, error) {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		})
	return err
}

func WaitForKnativeEventingState(ctx *test.Context, name, namespace string, inState EventingInStateFunc) (*operatorv1beta1.KnativeEventing, error) {
	var (
		lastState *operatorv1beta1.KnativeEventing
		err       error
	)
	waitErr := wait.PollImmediate(test.Interval, test.Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.Operator.KnativeEventings(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("knativeeventing %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

func IsKnativeEventingReady(e *operatorv1beta1.KnativeEventing, err error) (bool, error) {
	return e.Status.IsReady(), err
}

type EventingInStateFunc func(e *operatorv1beta1.KnativeEventing, err error) (bool, error)

func IsKnativeEventingWithVersionReady(version string) EventingInStateFunc {
	return func(e *operatorv1beta1.KnativeEventing, err error) (bool, error) {
		return e.Status.Version == version && e.Status.IsReady(), err
	}
}
