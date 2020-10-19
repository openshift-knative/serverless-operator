package knativeserving_test

import (
	"context"
	"os"
	"testing"

	. "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativeserving"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

func init() {
	configv1.AddToScheme(scheme.Scheme)
	servingv1alpha1.AddToScheme(scheme.Scheme)
}

var ks1 = &servingv1alpha1.KnativeServing{
	ObjectMeta: metav1.ObjectMeta{
		Name: "ks1",
	},
}
var ks2 = &servingv1alpha1.KnativeServing{
	ObjectMeta: metav1.ObjectMeta{
		Name: "ks2",
	},
}

func TestInvalidNamespace(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_SERVING_NAMESPACE", "knative-serving")
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{ks1})
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The required namespace is wrong, but the request is allowed")
	}
}

func TestLoneliness(t *testing.T) {
	os.Clearenv()
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{ks1})
	validator.InjectClient(fake.NewFakeClient(ks2))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Errorf("Too many KnativeServings: %v", result.Response)
	}
}

type mockDecoder struct {
	ks *servingv1alpha1.KnativeServing
}

var _ types.Decoder = (*mockDecoder)(nil)

func (mock *mockDecoder) Decode(_ types.Request, obj runtime.Object) error {
	if p, ok := obj.(*servingv1alpha1.KnativeServing); ok {
		*p = *mock.ks
	}
	return nil
}
