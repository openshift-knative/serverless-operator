package v1alpha1

import (
	"context"
	"fmt"

	"github.com/openshift-knative/serverless-operator/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

type ServingInStateFunc func(s *operatorv1alpha1.KnativeServing, err error) (bool, error)

func WaitForKnativeServingState(ctx *test.Context, name, namespace string, inState ServingInStateFunc) (*operatorv1alpha1.KnativeServing, error) {
	var (
		lastState *operatorv1alpha1.KnativeServing
		err       error
	)
	waitErr := wait.PollImmediate(test.Interval, test.Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.OperatorAlpha.KnativeServings(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("knativeserving %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

func IsKnativeServingWithVersionReady(version string) ServingInStateFunc {
	return func(s *operatorv1alpha1.KnativeServing, err error) (bool, error) {
		return s.Status.Version == version && s.Status.IsReady(), err
	}
}
