package test

import (
	"context"
	"testing"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/reconciler-test/pkg/feature"
)

type AllowedOperations struct {
	Get    bool
	List   bool
	Create bool
	Delete bool
}

var (
	AllowAll = AllowedOperations{
		Get:    true,
		List:   true,
		Create: true,
		Delete: true,
	}
	AllowViewOnly = AllowedOperations{
		Get:  true,
		List: true,
	}
)

type UserPermissionTest struct {
	Name              string
	UserContext       *Context
	AllowedOperations map[schema.GroupVersionResource]AllowedOperations
}

func RunUserPermissionTests(t *testing.T, objects map[schema.GroupVersionResource]*unstructured.Unstructured, tests ...UserPermissionTest) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			for gvr, allowed := range test.AllowedOperations {
				client := test.UserContext.Clients.Dynamic.Resource(gvr).Namespace("serverless-tests")

				obj := objects[gvr].DeepCopy()
				obj.SetName(feature.MakeRandomK8sName("test-" + gvr.Resource))

				_, err := client.Create(context.Background(), obj, metav1.CreateOptions{})
				if (allowed.Create && err != nil) || (!allowed.Create && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error creating %s, allowed = %v, err = %v", gvr.String(), allowed.Create, err)
				}

				err = client.Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
				if (allowed.Delete && err != nil) || (!allowed.Delete && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error deleting %s, allowed = %v, err = %v", gvr.String(), allowed.Delete, err)
				}

				if allowed.Delete {
					// If we've been able to delete the object we can assume we're able to get it as well.
					// Some objects take a while to be deleted, so we retry a few times.
					if err := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
						_, err = client.Get(context.Background(), obj.GetName(), metav1.GetOptions{})
						if apierrs.IsNotFound(err) {
							return true, nil
						}
						return false, err
					}); err != nil {
						t.Fatalf("Unexpected error waiting for %s to be deleted, err = %v", gvr.String(), err)
					}
				}

				_, err = client.Get(context.Background(), obj.GetName(), metav1.GetOptions{})
				// Ignore IsNotFound errors as "Forbidden" would overrule it anyway.
				if (allowed.Get && err != nil && !apierrs.IsNotFound(err)) || (!allowed.Get && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error getting %s, allowed = %v, err = %v", gvr.String(), allowed.Get, err)
				}

				_, err = client.List(context.Background(), metav1.ListOptions{})
				if (allowed.List && err != nil) || (!allowed.List && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error listing %s, allowed = %v, err = %v", gvr.String(), allowed.List, err)
				}
			}
		})
	}
}
