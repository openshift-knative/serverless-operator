package knativeeventing_test

import (
	"context"
	"os"
	"testing"

	. "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativeeventing"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
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
	validator := Validator{}
	validator.InjectDecoder(&mockDecoder{ke1})
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The required namespace is wrong, but the request is allowed")
	}
}

func TestLoneliness(t *testing.T) {
	os.Clearenv()
	validator := Validator{}
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
