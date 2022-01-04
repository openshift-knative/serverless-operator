package test

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func WithDeploymentReady(ctx *Context, name string, namespace string) (*appsv1.Deployment, error) {
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

	if waitErr != nil {
		return nil, fmt.Errorf("deployment %s in namespace %s not ready in time: %w", name, namespace, waitErr)
	}
	return deployment, nil
}
