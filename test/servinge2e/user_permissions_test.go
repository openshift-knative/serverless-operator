package servinge2e

import (
	"context"
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	networkingv1alpha1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
	autoscalingv1alpha1 "knative.dev/serving/pkg/apis/autoscaling/v1alpha1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	testNamespace         = "serverless-tests"
	testNamespace2        = "serverless-tests2"
	image                 = "gcr.io/knative-samples/helloworld-go"
	helloworldService     = "helloworld-go"
	helloworldService2    = "helloworld-go2"
	kubeHelloworldService = "kube-helloworld-go"
	helloworldText        = "Hello World!"
)

type allowedOperations struct {
	get    bool
	list   bool
	create bool
	delete bool
}

func init() {
	servingv1.AddToScheme(scheme.Scheme)
	networkingv1alpha1.AddToScheme(scheme.Scheme)
	autoscalingv1alpha1.AddToScheme(scheme.Scheme)
}

func TestUserPermissions(t *testing.T) {
	paCtx := test.SetupProjectAdmin(t)
	editCtx := test.SetupEdit(t)
	viewCtx := test.SetupView(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, paCtx, editCtx, viewCtx) })
	defer test.CleanupAll(t, paCtx, editCtx, viewCtx)

	serviceGVR := servingv1.SchemeGroupVersion.WithResource("services")
	ingressGVR := networkingv1alpha1.SchemeGroupVersion.WithResource("ingresses")
	paGVR := autoscalingv1alpha1.SchemeGroupVersion.WithResource("podautoscalers")

	service := &servingv1.Service{
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Image: "some-image",
							}},
						},
					},
				},
			},
		},
	}
	ingress := &networkingv1alpha1.Ingress{}
	pa := &autoscalingv1alpha1.PodAutoscaler{}
	objects := map[schema.GroupVersionResource]*unstructured.Unstructured{
		serviceGVR: {},
		ingressGVR: {},
		paGVR:      {},
	}
	if err := scheme.Scheme.Convert(service, objects[serviceGVR], nil); err != nil {
		t.Fatalf("Failed to convert Service: %v", err)
	}
	if err := scheme.Scheme.Convert(ingress, objects[ingressGVR], nil); err != nil {
		t.Fatalf("Failed to convert Ingress: %v", err)
	}
	if err := scheme.Scheme.Convert(pa, objects[paGVR], nil); err != nil {
		t.Fatalf("Failed to convert PodAutoscaler: %v", err)
	}

	allowAll := allowedOperations{
		get:    true,
		list:   true,
		create: true,
		delete: true,
	}
	allowViewOnly := allowedOperations{
		get:  true,
		list: true,
	}

	tests := []struct {
		name        string
		userContext *test.Context
		allowed     map[schema.GroupVersionResource]allowedOperations
	}{{
		name:        "project admin user",
		userContext: paCtx,
		allowed: map[schema.GroupVersionResource]allowedOperations{
			serviceGVR: allowAll,
			ingressGVR: allowViewOnly,
			paGVR:      allowViewOnly,
		},
	}, {
		name:        "edit user",
		userContext: editCtx,
		allowed: map[schema.GroupVersionResource]allowedOperations{
			serviceGVR: allowAll,
			ingressGVR: allowViewOnly,
			paGVR:      allowViewOnly,
		},
	}, {
		name:        "view user",
		userContext: viewCtx,
		allowed: map[schema.GroupVersionResource]allowedOperations{
			serviceGVR: allowViewOnly,
			ingressGVR: allowViewOnly,
			paGVR:      allowViewOnly,
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for gvr, allowed := range test.allowed {
				client := test.userContext.Clients.Dynamic.Resource(gvr).Namespace(testNamespace)

				obj := objects[gvr].DeepCopy()
				obj.SetName("test-" + gvr.Resource)

				_, err := client.Create(context.Background(), obj, metav1.CreateOptions{})
				if (allowed.create && err != nil) || (!allowed.create && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error creating %s, allowed = %v, err = %v", gvr.String(), allowed.create, err)
				}

				err = client.Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
				if (allowed.delete && err != nil) || (!allowed.delete && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error deleting %s, allowed = %v, err = %v", gvr.String(), allowed.delete, err)
				}

				_, err = client.Get(context.Background(), obj.GetName(), metav1.GetOptions{})
				// Ignore IsNotFound errors as "Forbidden" would overrule it anyway.
				if (allowed.get && err != nil && !apierrs.IsNotFound(err)) || (!allowed.get && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error getting %s, allowed = %v, err = %v", gvr.String(), allowed.get, err)
				}

				_, err = client.List(context.Background(), metav1.ListOptions{})
				if (allowed.list && err != nil) || (!allowed.list && !apierrs.IsForbidden(err)) {
					t.Errorf("Unexpected error listing %s, allowed = %v, err = %v", gvr.String(), allowed.list, err)
				}
			}
		})
	}
}
