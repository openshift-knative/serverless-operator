package test

import (
	"github.com/pkg/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	servingoperatorv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
)

func KnativeServing(name, namespace string) *servingoperatorv1alpha1.KnativeServing {
	return &servingoperatorv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func WithKnativeServingReady(ctx *Context, name, namespace string) (*servingoperatorv1alpha1.KnativeServing, error) {
	serving, err := CreateKnativeServing(ctx, name, namespace)
	if err != nil {
		return nil, err
	}
	_, err = WaitForKnativeServingState(ctx, serving.Name, serving.Namespace, IsKnativeServingReady)
	if err != nil {
		return nil, err
	}
	return serving, nil
}

func CreateKnativeServing(ctx *Context, name, namespace string) (*servingoperatorv1alpha1.KnativeServing, error) {
	serving, err := ctx.Clients.ServingOperator.KnativeServings(namespace).Create(KnativeServing(name, namespace))
	if err != nil {
		return nil, err
	}
	ctx.AddToCleanup(func() error {
		return DeleteKnativeServing(ctx, name, namespace)
	})
	return serving, nil
}

func DeleteKnativeServing(ctx *Context, name, namespace string) error {
	if err := ctx.Clients.ServingOperator.KnativeServings(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		return err
	}

	// Wait until the KnativeServing got removed.
	_, err := WaitForKnativeServingState(ctx, name, namespace,
		func(s *servingoperatorv1alpha1.KnativeServing, err error) (bool, error) {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		})
	return err
}

func WaitForKnativeServingState(ctx *Context, name, namespace string, inState func(s *servingoperatorv1alpha1.KnativeServing, err error) (bool, error)) (*servingoperatorv1alpha1.KnativeServing, error) {
	var lastState *servingoperatorv1alpha1.KnativeServing
	var err error
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.ServingOperator.KnativeServings(namespace).Get(name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "knativeserving %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

func IsKnativeServingReady(s *servingoperatorv1alpha1.KnativeServing, err error) (bool, error) {
	return s.Status.IsReady(), err
}
