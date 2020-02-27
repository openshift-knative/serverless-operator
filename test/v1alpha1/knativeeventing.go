package v1alpha1

import (
	"github.com/openshift-knative/serverless-operator/test"
	"github.com/pkg/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	eventingoperatorv1alpha1 "knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
)

func KnativeEventing(name, namespace string) *eventingoperatorv1alpha1.KnativeEventing {
	return &eventingoperatorv1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func WithKnativeEventingReady(ctx *test.Context, name, namespace string) (*eventingoperatorv1alpha1.KnativeEventing, error) {
	eventing, err := CreateKnativeEventing(ctx, name, namespace)
	if err != nil {
		return nil, err
	}
	_, err = WaitForKnativeEventingState(ctx, eventing.Name, eventing.Namespace, IsKnativeEventingReady)
	if err != nil {
		return nil, err
	}
	return eventing, nil
}

func CreateKnativeEventing(ctx *test.Context, name, namespace string) (*eventingoperatorv1alpha1.KnativeEventing, error) {
	eventing, err := ctx.Clients.EventingOperator.KnativeEventings(namespace).Create(KnativeEventing(name, namespace))
	if err != nil {
		return nil, err
	}
	ctx.AddToCleanup(func() error {
		return DeleteKnativeEventing(ctx, name, namespace)
	})
	return eventing, nil
}

func DeleteKnativeEventing(ctx *test.Context, name, namespace string) error {
	if err := ctx.Clients.EventingOperator.KnativeEventings(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		return err
	}

	// Wait until the KnativeEventing got removed.
	_, err := WaitForKnativeEventingState(ctx, name, namespace,
		func(s *eventingoperatorv1alpha1.KnativeEventing, err error) (bool, error) {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		})
	return err
}

func WaitForKnativeEventingState(ctx *test.Context, name, namespace string, inState func(s *eventingoperatorv1alpha1.KnativeEventing, err error) (bool, error)) (*eventingoperatorv1alpha1.KnativeEventing, error) {
	var lastState *eventingoperatorv1alpha1.KnativeEventing
	var err error
	waitErr := wait.PollImmediate(test.Interval, test.Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.EventingOperator.KnativeEventings(namespace).Get(name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "knativeeventing %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

func IsKnativeEventingReady(s *eventingoperatorv1alpha1.KnativeEventing, err error) (bool, error) {
	return s.Status.IsReady(), err
}
