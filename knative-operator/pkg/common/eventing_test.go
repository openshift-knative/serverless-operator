package common_test

import (
	"knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	eventingv1alpha1 "knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	configv1.AddToScheme(scheme.Scheme)
}

func TestMutateEventing(t *testing.T) {
	const (
		image = "docker.io/foo:tag"
	)
	client := fake.NewFakeClient()
	ke := &eventingv1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "knative-eventing",
			Namespace: "default",
		},
	}
	// Setup image override
	os.Setenv("IMAGE_foo", image)
	// Mutate for OpenShift
	if err := common.MutateEventing(ke, client); err != nil {
		t.Error(err)
	}
	verifyImageOverride(t, (*v1alpha1.Registry)(&ke.Spec.Registry), "foo", image)
}
