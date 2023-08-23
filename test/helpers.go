package test

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/eventing/pkg/utils"
)

func IsServiceMeshInstalled(ctx *Context) bool {
	_, err := ctx.Clients.Dynamic.Resource(schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}).Get(context.Background(), "servicemeshcontrolplanes.maistra.io", metav1.GetOptions{})

	if err == nil {
		return true
	}

	if !errors.IsNotFound(err) {
		ctx.T.Fatalf("Error checking if servicemeshcontrolplanes.maistra.io CRD exists: %v", err)
	}

	return false
}

func IsInternalEncryption(ctx *Context) bool {
	knativeServing := "knative-serving"
	serving, err := ctx.Clients.Operator.KnativeServings(knativeServing).Get(context.Background(), knativeServing, metav1.GetOptions{})
	if err != nil {
		ctx.T.Fatalf("Error getting KnativeServing: %v", err)
	}

	networkConfig, ok := serving.Spec.Config["network"]

	return ok && networkConfig["internal-encryption"] == "true"
}

func LinkGlobalPullSecretToNamespace(ctx *Context, ns string) error {
	// Wait for the default ServiceAccount to exist.
	if err := wait.PollImmediate(1*time.Second, 2*time.Minute, func() (bool, error) {
		sas := ctx.Clients.Kube.CoreV1().ServiceAccounts(ns)
		if _, err := sas.Get(context.Background(), "default", metav1.GetOptions{}); err == nil {
			return true, nil
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("default ServiceAccount was not created for namespace %s: %w", ns, err)
	}
	// Link global pull secrets for accessing private registries, see https://issues.redhat.com/browse/SRVKS-833
	_, err := utils.CopySecret(ctx.Clients.Kube.CoreV1(),
		"openshift-config", "pull-secret", ns, "default")
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error copying secret into ns %s: %w", ns, err)
	}
	return nil
}

func DeleteNamespace(ctx *Context, name string) error {
	if err := ctx.Clients.Kube.CoreV1().Namespaces().Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	if err := wait.PollImmediate(1*time.Second, 2*time.Minute, func() (bool, error) {
		if _, err := ctx.Clients.Kube.CoreV1().Namespaces().Get(context.Background(),
			name, metav1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("timed out deleting namespace %s: %w", name, err)
	}
	return nil
}

func CreateNamespace(ctx *Context, name string) (*corev1.Namespace, error) {
	ns, err := ctx.Clients.Kube.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("failed to create namespace %s: %w", name, err)
	}
	return ns, nil
}
