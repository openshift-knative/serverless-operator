package common_test

import (
	"fmt"
	"testing"

	"reflect"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	servingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestBuildImageOverrideMapFromEnviron(t *testing.T) {
	cases := []struct {
		name     string
		envMap   map[string]string
		scope    string
		expected map[string]string
	}{
		{
			name: "Simple container name",
			envMap: map[string]string{
				"IMAGE_EVENTING_foo": "quay.io/myimage",
			},
			scope: "EVENTING",
			expected: map[string]string{
				"foo": "quay.io/myimage",
			},
		},
		{
			name: "Simple env var",
			envMap: map[string]string{
				"IMAGE_EVENTING_CRONJOB_RA_IMAGE": "quay.io/myimage",
			},
			scope: "EVENTING",
			expected: map[string]string{
				"CRONJOB_RA_IMAGE": "quay.io/myimage",
			},
		},
		{
			name: "Simple env var with deployment name",
			envMap: map[string]string{
				"IMAGE_EVENTING_eventing-controller__CRONJOB_RA_IMAGE": "quay.io/myimage",
			},
			scope: "EVENTING",
			expected: map[string]string{
				"eventing-controller/CRONJOB_RA_IMAGE": "quay.io/myimage",
			},
		},
		{
			name: "Deployment+container name",
			envMap: map[string]string{
				"IMAGE_EVENTING_foo__bar": "quay.io/myimage",
			},
			scope: "EVENTING",
			expected: map[string]string{
				"foo/bar": "quay.io/myimage",
			},
		},
		{
			name: "Deployment+container and container name",
			envMap: map[string]string{
				"IMAGE_EVENTING_foo__bar": "quay.io/myimage1",
				"IMAGE_EVENTING_bar":      "quay.io/myimage2",
			},
			scope: "EVENTING",
			expected: map[string]string{
				"foo/bar": "quay.io/myimage1",
				"bar":     "quay.io/myimage2",
			},
		},
		{
			name: "Empty scope",
			envMap: map[string]string{
				"IMAGE_foo__bar": "quay.io/myimage1",
				"IMAGE_bar":      "quay.io/myimage2",
			},
			scope:    "EVENTING",
			expected: map[string]string{},
		},
		{
			name: "Ignore overrides not in the scope",
			envMap: map[string]string{
				"IMAGE_EVENTING_foo": "quay.io/myimage1",
				"IMAGE_SERVING_bar":  "quay.io/myimage2",
			},
			scope: "EVENTING",
			expected: map[string]string{
				"foo": "quay.io/myimage1",
			},
		},
		{
			name: "Different prefix",
			envMap: map[string]string{
				"X_EVENTING_foo": "quay.io/myimage",
			},
			scope:    "EVENTING",
			expected: map[string]string{},
		},
		{
			name: "No env var value",
			envMap: map[string]string{
				"IMAGE_EVENTING_foo": "",
			},
			scope:    "EVENTING",
			expected: map[string]string{},
		},
	}

	for i := range cases {
		tc := cases[i]
		environ := environFromMap(tc.envMap)
		overrideMap := common.BuildImageOverrideMapFromEnviron(environ, tc.scope)

		if !reflect.DeepEqual(overrideMap, tc.expected) {
			t.Errorf("Image override map is not equal. Case name: %q. Expected: %v, actual: %v", tc.name, tc.expected, overrideMap)
		}

	}
}

func verifyImageOverride(t *testing.T, registry *servingv1alpha1.Registry, imageName string, expected string) {
	if registry.Override[imageName] != expected {
		t.Errorf("Missing queue image. Expected a map with following override in it : %v=%v, actual: %v", imageName, expected, registry.Override)
	}
}

func environFromMap(envMap map[string]string) []string {
	e := []string{}

	for k, v := range envMap {
		e = append(e, fmt.Sprintf("%s=%s", k, v))
	}

	return e
}

func TestSetAnnotations(t *testing.T) {
	cases := []struct {
		name     string
		existing map[string]string
		toSet    map[string]string
		expected map[string]string
	}{
		{
			name:     "No existing annotations",
			existing: map[string]string{},
			toSet: map[string]string{
				"foo": "bar",
			},
			expected: map[string]string{
				"foo": "bar",
			},
		},
		{
			name: "Overwrite existing",
			existing: map[string]string{
				"foo":   "bar",
				"hello": "there",
			},
			toSet: map[string]string{
				"foo": "OVERRIDDEN",
				"baz": "NEW",
			},
			expected: map[string]string{
				"foo":   "OVERRIDDEN",
				"baz":   "NEW",
				"hello": "there",
			},
		},
		{
			name: "Do not do anything",
			existing: map[string]string{
				"foo":   "bar",
				"hello": "there",
			},
			toSet: map[string]string{},
			expected: map[string]string{
				"foo":   "bar",
				"hello": "there",
			},
		},
	}

	for i := range cases {
		tc := cases[i]

		u := &unstructured.Unstructured{}
		u.SetAnnotations(tc.existing)

		if err := common.SetAnnotations(tc.toSet)(u); err != nil {
			t.Errorf("Error when setting annotations. Case name: %q. Error: %s", tc.name, err.Error())
		}

		if !reflect.DeepEqual(u.GetAnnotations(), tc.expected) {
			t.Errorf("Annotations are not equal. Case name: %q. Expected: %v, actual: %v", tc.name, tc.expected, u.GetAnnotations())
		}
	}
}
