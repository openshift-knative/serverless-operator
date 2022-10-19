package test

import (
	"context"
	"fmt"
	"time"

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
