package v1alpha1

import (
	"context"
	"fmt"

	"github.com/openshift-knative/serverless-operator/test"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	servingoperatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func KnativeServing(name, namespace string) *servingoperatorv1alpha1.KnativeServing {
	return &servingoperatorv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func WithKnativeServingReady(ctx *test.Context, name, namespace string) (*servingoperatorv1alpha1.KnativeServing, error) {
	serving, err := CreateKnativeServing(ctx, name, namespace)
	if err != nil {
		return nil, err
	}
	if _, err = WaitForKnativeServingState(ctx, serving.Name, serving.Namespace, IsKnativeServingReady); err != nil {
		return nil, err
	}
	return serving, nil
}

func CreateKnativeServing(ctx *test.Context, name, namespace string) (*servingoperatorv1alpha1.KnativeServing, error) {
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
		func(s *servingoperatorv1alpha1.KnativeServing, err error) (bool, error) {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		})
	return err
}

func WaitForKnativeServingState(ctx *test.Context, name, namespace string, inState func(s *servingoperatorv1alpha1.KnativeServing, err error) (bool, error)) (*servingoperatorv1alpha1.KnativeServing, error) {
	var (
		lastState *servingoperatorv1alpha1.KnativeServing
		err       error
	)
	waitErr := wait.PollImmediate(test.Interval, test.Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.Operator.KnativeServings(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("knativeserving %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

func IsKnativeServingReady(s *servingoperatorv1alpha1.KnativeServing, err error) (bool, error) {
	return s.Status.IsReady(), err
}
