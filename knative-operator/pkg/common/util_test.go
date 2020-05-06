package common_test

import (
	"fmt"
	"testing"

	"reflect"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"

	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
)

func TestBuildImageOverrideMapFromEnviron(t *testing.T) {
	cases := []struct {
		name     string
		envMap   map[string]string
		expected map[string]string
	}{
		{
			name: "Simple container name",
			envMap: map[string]string{
				"IMAGE_foo": "quay.io/myimage",
			},
			expected: map[string]string{
				"foo": "quay.io/myimage",
			},
		},
		{
			name: "Deployment+container name",
			envMap: map[string]string{
				"IMAGE_foo_bar": "quay.io/myimage",
			},
			expected: map[string]string{
				"foo/bar": "quay.io/myimage",
			},
		},
		{
			name: "Deployment+container and container name",
			envMap: map[string]string{
				"IMAGE_foo_bar": "quay.io/myimage1",
				"IMAGE_bar":     "quay.io/myimage2",
			},
			expected: map[string]string{
				"foo/bar": "quay.io/myimage1",
				"bar":     "quay.io/myimage2",
			},
		},
		{
			name: "Different prefix",
			envMap: map[string]string{
				"X_foo": "quay.io/myimage",
			},
			expected: map[string]string{},
		},
		{
			name: "No env var value",
			envMap: map[string]string{
				"IMAGE_foo": "",
			},
			expected: map[string]string{},
		},
	}

	for i := range cases {
		tc := cases[i]
		environ := environFromMap(tc.envMap)
		overrideMap := common.BuildImageOverrideMapFromEnviron(environ)

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
