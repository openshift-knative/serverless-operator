package knativekafka_test

import (
	"context"
	"os"
	"testing"

	operatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	. "github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/knativekafka"
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
	operatorv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)
	eventingv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)
}

var defaultCR = &operatorv1alpha1.KnativeKafka{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "defaultCR",
		Namespace: "knative-eventing",
	},
	Spec: operatorv1alpha1.KnativeKafkaSpec{
		Source: operatorv1alpha1.Source{
			Enabled: false,
		},
		Channel: operatorv1alpha1.Channel{
			Enabled: false,
		},
	},
}
var duplicateCR = &operatorv1alpha1.KnativeKafka{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "duplicateCR",
		Namespace: "knative-eventing",
	},
	Spec: operatorv1alpha1.KnativeKafkaSpec{
		Source: operatorv1alpha1.Source{
			Enabled: false,
		},
		Channel: operatorv1alpha1.Channel{
			Enabled: false,
		},
	},
}

var invalidNamespaceCR = &operatorv1alpha1.KnativeKafka{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "invalidNamespaceCR",
		Namespace: "FOO",
	},
	Spec: operatorv1alpha1.KnativeKafkaSpec{
		Source: operatorv1alpha1.Source{
			Enabled: false,
		},
		Channel: operatorv1alpha1.Channel{
			Enabled: false,
		},
	},
}
var invalidShapeCR = &operatorv1alpha1.KnativeKafka{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "invalidShapeCR",
		Namespace: "knative-eventing",
	},
	Spec: operatorv1alpha1.KnativeKafkaSpec{
		Source: operatorv1alpha1.Source{
			Enabled: false,
		},
		Channel: operatorv1alpha1.Channel{
			Enabled: true,
			// need to have bootstrapServers defined here!
		},
	},
}

var validKnativeEventingCR = &eventingv1alpha1.KnativeEventing{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "validKnativeEventing",
		Namespace: "knative-eventing",
	},
}

func TestHappy(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")
	validator := Validator{}
	validator.InjectDecoder(&mockDecoder{defaultCR})
	validator.InjectClient(fake.NewFakeClient(validKnativeEventingCR))
	result := validator.Handle(context.TODO(), types.Request{})
	if !result.Response.Allowed {
		t.Error("The request is not allowed but should be")
	}
}

func TestInvalidNamespace(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")
	validator := Validator{}
	validator.InjectDecoder(&mockDecoder{invalidNamespaceCR})
	validator.InjectClient(fake.NewFakeClient(validKnativeEventingCR))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The required namespace is wrong, but the request is allowed")
	}
}

func TestLoneliness(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")
	validator := Validator{}
	validator.InjectDecoder(&mockDecoder{defaultCR})
	validator.InjectClient(fake.NewFakeClient(duplicateCR, validKnativeEventingCR))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Errorf("Too many KnativeKafkas: %v", result.Response)
	}
}

func TestInvalidShape(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")
	validator := Validator{}
	validator.InjectDecoder(&mockDecoder{invalidShapeCR})
	validator.InjectClient(fake.NewFakeClient(validKnativeEventingCR))
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("The shape is invalid, but the request is allowed")
	}
}

func TestValidateDeps(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")
	validator := Validator{}
	validator.InjectDecoder(&mockDecoder{defaultCR})
	validator.InjectClient(fake.NewFakeClient())
	result := validator.Handle(context.TODO(), types.Request{})
	if result.Response.Allowed {
		t.Error("No KnativeEventing instance install, but request allowed")
	}
}

type mockDecoder struct {
	ke *operatorv1alpha1.KnativeKafka
}

var _ types.Decoder = (*mockDecoder)(nil)

func (mock *mockDecoder) Decode(_ types.Request, obj runtime.Object) error {
	if p, ok := obj.(*operatorv1alpha1.KnativeKafka); ok {
		*p = *mock.ke
	}
	return nil
}
