package common_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestBuildImageOverrideMapFromEnviron(t *testing.T) {
	cases := []struct {
		name     string
		envMap   map[string]string
		prefix   string
		expected map[string]string
	}{
		{
			name: "Simple container name",
			envMap: map[string]string{
				"IMAGE_foo": "quay.io/myimage",
			},
			prefix: "IMAGE_",
			expected: map[string]string{
				"foo": "quay.io/myimage",
			},
		},
		{
			name: "Simple env var",
			envMap: map[string]string{
				"IMAGE_CRONJOB_RA_IMAGE": "quay.io/myimage",
			},
			prefix: "IMAGE_",
			expected: map[string]string{
				"CRONJOB_RA_IMAGE": "quay.io/myimage",
			},
		},
		{
			name: "Simple env var with deployment name",
			envMap: map[string]string{
				"IMAGE_eventing-controller__CRONJOB_RA_IMAGE": "quay.io/myimage",
			},
			prefix: "IMAGE_",
			expected: map[string]string{
				"eventing-controller/CRONJOB_RA_IMAGE": "quay.io/myimage",
			},
		},
		{
			name: "Deployment+container name",
			envMap: map[string]string{
				"IMAGE_foo__bar": "quay.io/myimage",
			},
			prefix: "IMAGE_",
			expected: map[string]string{
				"foo/bar": "quay.io/myimage",
			},
		},
		{
			name: "Deployment+container and container name",
			envMap: map[string]string{
				"IMAGE_foo__bar": "quay.io/myimage1",
				"IMAGE_bar":      "quay.io/myimage2",
			},
			prefix: "IMAGE_",
			expected: map[string]string{
				"foo/bar": "quay.io/myimage1",
				"bar":     "quay.io/myimage2",
			},
		},
		{
			name: "Nothing in prefix",
			envMap: map[string]string{
				"IMAGE_foo__bar": "quay.io/myimage1",
				"IMAGE_bar":      "quay.io/myimage2",
			},
			prefix:   "HELLO",
			expected: map[string]string{},
		},
		{
			name: "Ignore overrides not in the prefix",
			envMap: map[string]string{
				"IMAGE_foo": "quay.io/myimage1",
				"HELLO_bar": "quay.io/myimage2",
			},
			prefix: "IMAGE_",
			expected: map[string]string{
				"foo": "quay.io/myimage1",
			},
		},
		{
			name: "No env var value",
			envMap: map[string]string{
				"IMAGE_foo": "",
			},
			prefix:   "IMAGE_",
			expected: map[string]string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			environ := environFromMap(tc.envMap)
			overrideMap := common.BuildImageOverrideMapFromEnviron(environ, tc.prefix)

			if !cmp.Equal(overrideMap, tc.expected) {
				t.Errorf("Image override map not as expected, diff: %s", cmp.Diff(tc.expected, overrideMap))
			}
		})
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := &unstructured.Unstructured{}
			u.SetAnnotations(tc.existing)

			if err := common.SetAnnotations(tc.toSet)(u); err != nil {
				t.Errorf("Error when setting annotations. Case name: %q. Error: %s", tc.name, err.Error())
			}

			if !cmp.Equal(u.GetAnnotations(), tc.expected) {
				t.Errorf("Annotations not as expected, diff: %s", cmp.Diff(tc.expected, u.GetAnnotations()))
			}
		})
	}
}
