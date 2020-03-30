package knativeserving_test

import (
	"context"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/testutil"
	"os"
	"testing"

	. "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativeserving"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
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

func TestInvalidVersion(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.1.13")
	validator := KnativeServingValidator{}
	validator.InjectDecoder(&mockDecoder{ks1})
	validator.InjectClient(fake.NewFakeClient(testutil.MockClusterVersion("3.2.0")))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The version is too low, but the request is allowed")
	}
}

func TestPreReleaseVersionConstraint(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.3.0-0")

	for _, version := range []string{"4.3.0", "4.4.0", "4.3.5", "4.3.0-alpha", "4.3.0-0.ci-2020-03-11-221411", "4.3.0+build"} {
		validator := KnativeServingValidator{}
		validator.InjectDecoder(&mockDecoder{ks1})
		validator.InjectClient(fake.NewFakeClient(testutil.MockClusterVersion(version)))
		result := validator.Handle(context.TODO(), types.Request{})
		if !result.Response.Allowed {
			t.Errorf("Version %q was supposed to pass but didn't: %v", version, result.Response)
		}
	}
}

func TestInvalidVersionConstraint(t *testing.T) {
	os.Clearenv()
	os.Setenv("MIN_OPENSHIFT_VERSION", "4.1.13")

	for _, version := range []string{"4.0.0", "4.1.12", "4.1.13-alpha", "4.1.13-0.ci-2020-03-11-221411"} {
		validator := KnativeServingValidator{}
		validator.InjectDecoder(&mockDecoder{ks1})
		validator.InjectClient(fake.NewFakeClient(testutil.MockClusterVersion(version)))
		result := validator.Handle(context.TODO(), types.Request{})
		if result.Response.Allowed {
			t.Errorf("Version %q was NOT supposed to pass but it did: %v", version, result.Response)
		}
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
