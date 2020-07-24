package common_test

import (
	"context"
	"os"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	configv1.AddToScheme(scheme.Scheme)
	servingv1alpha1.AddToScheme(scheme.Scheme)
}

type env map[string]string
type fixtures []runtime.Object

func TestValidation(t *testing.T) {
	tests := []struct {
		input    v1alpha1.KComponent
		env      env
		fixtures fixtures
		allowed  bool
	}{{
		input:   knativeServing("no problem"),
		allowed: true,
	}, {
		input:   knativeEventing("no problem"),
		allowed: true,
	}, {
		input:   knativeServing("invalid serving namespace"),
		env:     env{"REQUIRED_SERVING_NAMESPACE": "knative-serving"},
		allowed: false,
	}, {
		input:   knativeEventing("invalid eventing namespace"),
		env:     env{"REQUIRED_EVENTING_NAMESPACE": "knative-eventing"},
		allowed: false,
	}, {
		input:    knativeEventing("invalid openshift version"),
		env:      env{"MIN_OPENSHIFT_VERSION": "4.1.13"},
		fixtures: fixtures{clusterVersion("3.2.0")},
		allowed:  false,
	}, {
		input:    knativeServing("valid"),
		env:      env{"MIN_OPENSHIFT_VERSION": "4.3.0-0"},
		fixtures: fixtures{clusterVersion("4.3.0")},
		allowed:  true,
	}, {
		input:    knativeServing("valid"),
		env:      env{"MIN_OPENSHIFT_VERSION": "4.3.0-0"},
		fixtures: fixtures{clusterVersion("4.4.0")},
		allowed:  true,
	}, {
		input:    knativeServing("valid"),
		env:      env{"MIN_OPENSHIFT_VERSION": "4.3.0-0"},
		fixtures: fixtures{clusterVersion("4.3.5")},
		allowed:  true,
	}, {
		input:    knativeServing("valid"),
		env:      env{"MIN_OPENSHIFT_VERSION": "4.3.0-0"},
		fixtures: fixtures{clusterVersion("4.3.0-alpha")},
		allowed:  true,
	}, {
		input:    knativeServing("valid"),
		env:      env{"MIN_OPENSHIFT_VERSION": "4.3.0-0"},
		fixtures: fixtures{clusterVersion("4.3.0-0.ci-2020-03-11-221411")},
		allowed:  true,
	}, {
		input:    knativeServing("valid"),
		env:      env{"MIN_OPENSHIFT_VERSION": "4.3.0-0"},
		fixtures: fixtures{clusterVersion("4.3.0+build")},
		allowed:  true,
	}, {
		input:    knativeServing("invalid"),
		env:      env{"MIN_OPENSHIFT_VERSION": "4.1.13"},
		fixtures: fixtures{clusterVersion("4.0.0")},
		allowed:  false,
	}, {
		input:    knativeServing("invalid"),
		env:      env{"MIN_OPENSHIFT_VERSION": "4.1.13"},
		fixtures: fixtures{clusterVersion("4.1.12")},
		allowed:  false,
	}, {
		input:    knativeServing("invalid"),
		env:      env{"MIN_OPENSHIFT_VERSION": "4.1.13"},
		fixtures: fixtures{clusterVersion("4.1.13-alpha")},
		allowed:  false,
	}, {
		input:    knativeServing("invalid"),
		env:      env{"MIN_OPENSHIFT_VERSION": "4.1.13"},
		fixtures: fixtures{clusterVersion("4.1.13-0.ci-2020-03-11-221411")},
		allowed:  false,
	}, {
		input:    knativeServing("too many"),
		fixtures: fixtures{knativeServing("I was here first")},
		allowed:  false,
	}}
	for _, test := range tests {
		t.Run(test.input.GetName(), func(t *testing.T) {
			setenv(test.env)
			client := fake.NewFakeClient(test.fixtures...)
			allowed, reason, err := common.Validate(context.TODO(), client, test.input)
			if allowed != test.allowed {
				t.Errorf("Failed expectation: expected=%t, allowed=%t, reason=%s, err=%v", test.allowed, allowed, reason, err)
			}
		})
	}
}

func knativeServing(name string) *v1alpha1.KnativeServing {
	return &servingv1alpha1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func knativeEventing(name string) *v1alpha1.KnativeEventing {
	return &servingv1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func clusterVersion(version string) *configv1.ClusterVersion {
	return &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Status: configv1.ClusterVersionStatus{
			Desired: configv1.Update{
				Version: version,
			},
		},
	}
}

func setenv(env map[string]string) {
	os.Clearenv()
	for k, v := range env {
		os.Setenv(k, v)
	}
}
