package knativekafka

import (
	"context"
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	operatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

var (
	defaultCR = &operatorv1alpha1.KnativeKafka{
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
	duplicateCR = &operatorv1alpha1.KnativeKafka{
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
	invalidNamespaceCR = &operatorv1alpha1.KnativeKafka{
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
	invalidShapeCR = &operatorv1alpha1.KnativeKafka{
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
	validKnativeEventingCR = &eventingv1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "validKnativeEventing",
			Namespace: "knative-eventing",
		},
	}

	decoder types.Decoder
)

func init() {
	apis.AddToScheme(scheme.Scheme)
	decoder, _ = admission.NewDecoder(scheme.Scheme)
}

func TestHappy(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")

	validator := Validator{}
	validator.InjectDecoder(decoder)
	validator.InjectClient(fake.NewFakeClient(validKnativeEventingCR))

	req, err := testutil.RequestFor(defaultCR)
	if err != nil {
		t.Fatalf("Failed to generate a request for %v: %v", defaultCR, err)
	}

	result := validator.Handle(context.Background(), req)
	if !result.Response.Allowed {
		t.Error("The request is not allowed but should be")
	}
}

func TestInvalidNamespace(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")

	validator := Validator{}
	validator.InjectDecoder(decoder)
	validator.InjectClient(fake.NewFakeClient(validKnativeEventingCR))

	req, err := testutil.RequestFor(invalidNamespaceCR)
	if err != nil {
		t.Fatalf("Failed to generate a request for %v: %v", invalidNamespaceCR, err)
	}

	result := validator.Handle(context.Background(), req)
	if result.Response.Allowed {
		t.Error("The required namespace is wrong, but the request is allowed")
	}
}

func TestLoneliness(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")

	validator := Validator{}
	validator.InjectDecoder(decoder)
	validator.InjectClient(fake.NewFakeClient(duplicateCR, validKnativeEventingCR))

	req, err := testutil.RequestFor(defaultCR)
	if err != nil {
		t.Fatalf("Failed to generate a request for %v: %v", defaultCR, err)
	}

	result := validator.Handle(context.Background(), req)
	if result.Response.Allowed {
		t.Errorf("Too many KnativeKafkas: %v", result.Response)
	}
}

func TestInvalidShape(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")

	validator := Validator{}
	validator.InjectDecoder(decoder)
	validator.InjectClient(fake.NewFakeClient(validKnativeEventingCR))

	req, err := testutil.RequestFor(invalidShapeCR)
	if err != nil {
		t.Fatalf("Failed to generate a request for %v: %v", invalidShapeCR, err)
	}

	result := validator.Handle(context.Background(), req)
	if result.Response.Allowed {
		t.Error("The shape is invalid, but the request is allowed")
	}
}

func TestValidateDeps(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")

	validator := Validator{}
	validator.InjectDecoder(decoder)
	validator.InjectClient(fake.NewFakeClient())

	req, err := testutil.RequestFor(defaultCR)
	if err != nil {
		t.Fatalf("Failed to generate a request for %v: %v", defaultCR, err)
	}

	result := validator.Handle(context.Background(), req)
	if result.Response.Allowed {
		t.Error("No KnativeEventing instance install, but request allowed")
	}
}
