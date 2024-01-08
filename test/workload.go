package test

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Deployment struct {
	Name          string
	ExpectedScale *int32
}

func WithWorkloadReady(ctx *Context, name string, namespace string) error {
	waitErr := withDeploymentReady(ctx, name, namespace)
	if apierrors.IsNotFound(waitErr) {
		waitErr = withStatefulSetReady(ctx, name, namespace)
	}

	if waitErr != nil {
		return fmt.Errorf("deployment %s in namespace %s not ready in time: %w", name, namespace, waitErr)
	}

	return nil
}

func withDeploymentReady(ctx *Context, name string, namespace string) error {
	var deployment *appsv1.Deployment
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		var err error
		deployment, err = ctx.Clients.Kube.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
			return false, nil
		}
		return true, nil
	})
	return waitErr
}

func withStatefulSetReady(ctx *Context, name string, namespace string) error {
	var ss *appsv1.StatefulSet
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		var err error
		ss, err = ctx.Clients.Kube.AppsV1().StatefulSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if ss.Status.ReadyReplicas < *ss.Spec.Replicas {
			return false, nil
		}
		return true, nil
	})
	return waitErr
}
