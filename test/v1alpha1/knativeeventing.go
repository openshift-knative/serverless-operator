package v1alpha1

import (
	"context"
	"fmt"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

type EventingInStateFunc func(e *operatorv1alpha1.KnativeEventing, err error) (bool, error)

func WaitForKnativeEventingState(ctx *test.Context, name, namespace string, inState EventingInStateFunc) (*operatorv1alpha1.KnativeEventing, error) {
	var (
		lastState *operatorv1alpha1.KnativeEventing
		err       error
	)
	waitErr := wait.PollImmediate(test.Interval, test.Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.OperatorAlpha.KnativeEventings(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("knativeeventing %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

func IsKnativeEventingWithVersionReady(version string) EventingInStateFunc {
	return func(e *operatorv1alpha1.KnativeEventing, err error) (bool, error) {
		return e.Status.Version == version && e.Status.IsReady(), err
	}
}
