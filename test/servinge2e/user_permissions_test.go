package servinge2e

import (
	"testing"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	networkingv1alpha1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
	autoscalingv1alpha1 "knative.dev/serving/pkg/apis/autoscaling/v1alpha1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	testNamespace = "serverless-tests"
)

func init() {
	servingv1.AddToScheme(scheme.Scheme)
	networkingv1alpha1.AddToScheme(scheme.Scheme)
	autoscalingv1alpha1.AddToScheme(scheme.Scheme)
}

func TestServingUserPermissions(t *testing.T) {
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

	tests := []test.UserPermissionTest{{
		Name:        "project admin user",
		UserContext: paCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			serviceGVR: test.AllowAll,
			ingressGVR: test.AllowViewOnly,
			paGVR:      test.AllowViewOnly,
		},
	}, {
		Name:        "edit user",
		UserContext: editCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			serviceGVR: test.AllowAll,
			ingressGVR: test.AllowViewOnly,
			paGVR:      test.AllowViewOnly,
		},
	}, {
		Name:        "view user",
		UserContext: viewCtx,
		AllowedOperations: map[schema.GroupVersionResource]test.AllowedOperations{
			serviceGVR: test.AllowViewOnly,
			ingressGVR: test.AllowViewOnly,
			paGVR:      test.AllowViewOnly,
		},
	}}

	test.RunUserPermissionTests(t, objects, tests...)
}
