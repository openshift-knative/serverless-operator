package test

import (
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func WithDeploymentReady(ctx *Context, name string, namespace string) (*appsv1.Deployment, error) {
	var deployment *appsv1.Deployment
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		var err error
		deployment, err = ctx.Clients.Kube.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
			return false, nil
		}
		return true, nil
	})

	if waitErr != nil {
		return nil, errors.Wrapf(waitErr, "Deployment %s in namespace %s not ready in time.", name, namespace)
	}
	return deployment, nil
}

func WithDeploymentGone(ctx *Context, name string, namespace string) error {
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		_, err := ctx.Clients.Kube.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if err != nil && apierrs.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, err
		} else {
			return false, nil
		}
	})

	if waitErr != nil {
		return errors.Wrapf(waitErr, "Deployment %s in namespace %s not gone in time.", name, namespace)
	}
	return nil
}

func WithDeploymentCount(ctx *Context, namespace string, count int) error {
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		deploymentList, err := ctx.Clients.Kube.AppsV1().Deployments(namespace).List(metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		if len(deploymentList.Items) != count {
			return false, nil
		}
		return true, nil
	})

	if waitErr != nil {
		return errors.Wrapf(waitErr, "Deployment count in namespace %s did not reach the expected count %d in time", namespace, count)
	}
	return nil
}
