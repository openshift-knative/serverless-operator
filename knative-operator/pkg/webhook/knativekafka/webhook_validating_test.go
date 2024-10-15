package knativekafka

import (
	"context"
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	serverlessoperatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/webhook/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	defaultCR = &serverlessoperatorv1alpha1.KnativeKafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "defaultCR",
			Namespace: "knative-eventing",
		},
		Spec: serverlessoperatorv1alpha1.KnativeKafkaSpec{
			Source: serverlessoperatorv1alpha1.Source{
				Enabled: false,
			},
			Channel: serverlessoperatorv1alpha1.Channel{
				Enabled: false,
			},
		},
	}
	duplicateCR = &serverlessoperatorv1alpha1.KnativeKafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "duplicateCR",
			Namespace: "knative-eventing",
		},
		Spec: serverlessoperatorv1alpha1.KnativeKafkaSpec{
			Source: serverlessoperatorv1alpha1.Source{
				Enabled: false,
			},
			Channel: serverlessoperatorv1alpha1.Channel{
				Enabled: false,
			},
		},
	}
	invalidNamespaceCR = &serverlessoperatorv1alpha1.KnativeKafka{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalidNamespaceCR",
			Namespace: "FOO",
		},
		Spec: serverlessoperatorv1alpha1.KnativeKafkaSpec{
			Source: serverlessoperatorv1alpha1.Source{
				Enabled: false,
			},
			Channel: serverlessoperatorv1alpha1.Channel{
				Enabled: false,
			},
		},
	}
	invalidShapeCRs = []serverlessoperatorv1alpha1.KnativeKafka{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalidShapeCR-1",
				Namespace: "knative-eventing",
			},
			Spec: serverlessoperatorv1alpha1.KnativeKafkaSpec{
				Source: serverlessoperatorv1alpha1.Source{
					Enabled: false,
				},
				Channel: serverlessoperatorv1alpha1.Channel{
					Enabled: true,
					// need to have bootstrapServers defined here!
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalidShapeCR-2",
				Namespace: "knative-eventing",
			},
			Spec: serverlessoperatorv1alpha1.KnativeKafkaSpec{
				Source: serverlessoperatorv1alpha1.Source{
					Enabled: false,
				},
				Channel: serverlessoperatorv1alpha1.Channel{
					Enabled:             true,
					BootstrapServers:    "foo.example.com",
					AuthSecretNamespace: "my-ns",
					// need to have AuthSecretName here
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "invalidShapeCR-3",
				Namespace: "knative-eventing",
			},
			Spec: serverlessoperatorv1alpha1.KnativeKafkaSpec{
				Source: serverlessoperatorv1alpha1.Source{
					Enabled: false,
				},
				Channel: serverlessoperatorv1alpha1.Channel{
					Enabled:          true,
					BootstrapServers: "foo.example.com",
					AuthSecretName:   "my-secret",
					// need to have AuthSecretNamespace here
				},
			},
		},
	}
	validKnativeEventingCR = &operatorv1beta1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "validKnativeEventing",
			Namespace: "knative-eventing",
		},
	}

	decoder admission.Decoder
)

func init() {
	apis.AddToScheme(scheme.Scheme)
	decoder = admission.NewDecoder(scheme.Scheme)
}

func TestHappy(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")

	validator := NewValidator(
		fake.NewClientBuilder().WithObjects(validKnativeEventingCR).Build(),
		decoder)

	req, err := testutil.RequestFor(defaultCR)
	if err != nil {
		t.Fatalf("Failed to generate a request for %v: %v", defaultCR, err)
	}

	result := validator.Handle(context.Background(), req)
	if !result.Allowed {
		t.Error("The request is not allowed but should be")
	}
}

func TestInvalidNamespace(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")

	validator := NewValidator(
		fake.NewClientBuilder().WithObjects(validKnativeEventingCR).Build(),
		decoder)

	req, err := testutil.RequestFor(invalidNamespaceCR)
	if err != nil {
		t.Fatalf("Failed to generate a request for %v: %v", invalidNamespaceCR, err)
	}

	result := validator.Handle(context.Background(), req)
	if result.Allowed {
		t.Error("The required namespace is wrong, but the request is allowed")
	}
}

func TestLoneliness(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")

	validator := NewValidator(
		fake.NewClientBuilder().WithObjects(duplicateCR, validKnativeEventingCR).Build(),
		decoder)

	req, err := testutil.RequestFor(defaultCR)
	if err != nil {
		t.Fatalf("Failed to generate a request for %v: %v", defaultCR, err)
	}

	result := validator.Handle(context.Background(), req)
	if result.Allowed {
		t.Errorf("Too many KnativeKafkas: %v", result.AdmissionResponse)
	}
}

func TestInvalidShape(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")

	validator := NewValidator(
		fake.NewClientBuilder().WithObjects(duplicateCR, validKnativeEventingCR).Build(),
		decoder)

	for _, cr := range invalidShapeCRs {
		req, err := testutil.RequestFor(&cr)
		if err != nil {
			t.Fatalf("Failed to generate a request for %v: %v", cr, err)
		}

		result := validator.Handle(context.Background(), req)
		if result.Allowed {
			t.Error("The shape is invalid, but the request is allowed")
		}
	}
}

func TestValidateDeps(t *testing.T) {
	os.Clearenv()
	os.Setenv("REQUIRED_KAFKA_NAMESPACE", "knative-eventing")

	validator := NewValidator(fake.NewClientBuilder().Build(), decoder)

	req, err := testutil.RequestFor(defaultCR)
	if err != nil {
		t.Fatalf("Failed to generate a request for %v: %v", defaultCR, err)
	}

	result := validator.Handle(context.Background(), req)
	if result.Allowed {
		t.Error("No KnativeEventing instance install, but request allowed")
	}
}
