package test

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var clusterExtensionGVR = schema.GroupVersionResource{
	Group:    "olm.operatorframework.io",
	Version:  "v1",
	Resource: "clusterextensions",
}

const ClusterExtensionName = "serverless-operator"

func PatchClusterExtensionVersion(ctx *Context, name, version string) error {
	patch := []byte(fmt.Sprintf(`{"spec":{"source":{"catalog":{"version":"%s"}}}}`, version))
	_, err := ctx.Clients.Dynamic.Resource(clusterExtensionGVR).Patch(
		context.Background(), name, types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}

func PatchClusterExtensionVersionWithPolicy(ctx *Context, name, version, policy string) error {
	patch := []byte(fmt.Sprintf(
		`{"spec":{"source":{"catalog":{"version":"%s","upgradeConstraintPolicy":"%s"}}}}`,
		version, policy))
	_, err := ctx.Clients.Dynamic.Resource(clusterExtensionGVR).Patch(
		context.Background(), name, types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}

func WaitForClusterExtensionReady(ctx *Context, name, version string, timeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(context.Background(), Interval, timeout, true, func(_ context.Context) (bool, error) {
		ce, err := ctx.Clients.Dynamic.Resource(clusterExtensionGVR).Get(
			context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		// Check installed version matches requested version.
		installed, found, err := unstructured.NestedString(ce.Object, "status", "install", "bundle", "version")
		if err != nil || !found || installed != version {
			return false, nil
		}

		// Check Progressing condition indicates success.
		conditions, found, err := unstructured.NestedSlice(ce.Object, "status", "conditions")
		if err != nil || !found {
			return false, nil
		}

		for _, c := range conditions {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			condType, _, _ := unstructured.NestedString(cond, "type")
			reason, _, _ := unstructured.NestedString(cond, "reason")
			if condType == "Progressing" && reason == "Succeeded" {
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("ClusterExtension %s did not reach version %s: %w", name, version, err)
	}
	return nil
}

func GetClusterExtensionInstalledVersion(ctx *Context, name string) (string, error) {
	ce, err := ctx.Clients.Dynamic.Resource(clusterExtensionGVR).Get(
		context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	version, found, err := unstructured.NestedString(ce.Object, "status", "install", "bundle", "version")
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("version not found in ClusterExtension %s status", name)
	}
	return version, nil
}
