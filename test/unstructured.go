package test

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
)

func CreateUnstructured(ctx *Context, schema schema.GroupVersionResource, unstructured *unstructured.Unstructured) *unstructured.Unstructured {
	ret, err := ctx.Clients.Dynamic.Resource(schema).Namespace(unstructured.GetNamespace()).Create(context.Background(), unstructured, metav1.CreateOptions{})
	if err != nil {
		ctx.T.Fatalf("Error creating %s %s: %v", schema.GroupResource(), unstructured.GetName(), err)
	}

	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up %s %s", schema.GroupResource(), ret.GetName())
		return ctx.Clients.Dynamic.Resource(schema).Namespace(ret.GetNamespace()).Delete(context.Background(), ret.GetName(), metav1.DeleteOptions{})
	})

	return ret
}

func DoesUnstructuredNotExist(_ *unstructured.Unstructured, err error) (bool, error) {
	return errors.IsNotFound(err), nil
}

func IsUnstructuredReady(u *unstructured.Unstructured, err error) (bool, error) {
	if err != nil {
		return false, err
	}

	conditions, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if err != nil || !found {
		return false, err
	}

	for _, condition := range conditions {
		conditionMap, ok := condition.(map[string]interface{})

		if ok {
			_type, _, _ := unstructured.NestedString(conditionMap, "type")
			if _type == "Ready" {
				status, _, _ := unstructured.NestedString(conditionMap, "status")
				if status == "True" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func WaitForUnstructuredState(ctx *Context, schema schema.GroupVersionResource, name, namespace string, inState func(u *unstructured.Unstructured, err error) (bool, error)) (*unstructured.Unstructured, error) {
	var (
		lastState *unstructured.Unstructured
		err       error
	)
	waitErr := wait.PollUntilContextTimeout(context.Background(), Interval, 10*time.Minute, true, func(_ context.Context) (bool, error) {
		lastState, err = ctx.Clients.Dynamic.Resource(schema).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("%s %s is not in desired state, got: %+v: %w", schema.Resource, name, lastState, waitErr)
	}
	return lastState, nil
}
