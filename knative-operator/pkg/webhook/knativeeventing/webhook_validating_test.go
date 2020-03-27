package knativeeventing_test

import (
	"context"
	. "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativeeventing"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/testutil"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	eventingv1alpha1 "knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
	"testing"
)

func init() {
	configv1.AddToScheme(scheme.Scheme)
	eventingv1alpha1.AddToScheme(scheme.Scheme)
}

var ke1 = &eventingv1alpha1.KnativeEventing{
	ObjectMeta: metav1.ObjectMeta{
		Name: "ke1",
	},
}
var ke2 = &eventingv1alpha1.KnativeEventing{
	ObjectMeta: metav1.ObjectMeta{
		Name: "ke2",
	},
}

func TestInvalidNamespace(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_EVENTING_NAMESPACE", "knative-eventing")
	validator := KnativeEventingValidator{}
	validator.InjectDecoder(&mockDecoder{ke1})
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The required namespace is wrong, but the request is allowed")
	}
}

func TestInvalidVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.1.13")
	validator := KnativeEventingValidator{}
	validator.InjectDecoder(&mockDecoder{ke1})
	validator.InjectClient(fake.NewFakeClient(testutil.MockClusterVersion("3.2.0")))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The version is too low, but the request is allowed")
	}
}

func TestLoneliness(t *testing.T) {
	os.Clearenv()
	validator := KnativeEventingValidator{}
	validator.InjectDecoder(&mockDecoder{ke1})
	validator.InjectClient(fake.NewFakeClient(ke2))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Errorf("Too many KnativeEventings: %v", result.Response)
	}
}

type mockDecoder struct {
	ke *eventingv1alpha1.KnativeEventing
}

var _ types.Decoder = (*mockDecoder)(nil)

func (mock *mockDecoder) Decode(_ types.Request, obj runtime.Object) error {
	if p, ok := obj.(*eventingv1alpha1.KnativeEventing); ok {
		*p = *mock.ke
	}
	return nil
}
