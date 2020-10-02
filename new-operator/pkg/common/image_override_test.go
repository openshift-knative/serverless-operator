package common

import (
	"fmt"
	"reflect"
	"testing"
)

func TestImageMapFromEnvironment(t *testing.T) {
	cases := []struct {
		name     string
		envMap   map[string]string
		expected map[string]string
	}{{
		name: "Simple container name",
		envMap: map[string]string{
			"IMAGE_foo": "quay.io/myimage",
		},
		expected: map[string]string{
			"foo": "quay.io/myimage",
		},
	}, {
		name: "Simple env var",
		envMap: map[string]string{
			"IMAGE_CRONJOB_RA_IMAGE": "quay.io/myimage",
		},
		expected: map[string]string{
			"CRONJOB_RA_IMAGE": "quay.io/myimage",
		},
	}, {
		name: "Simple env var with deployment name",
		envMap: map[string]string{
			"IMAGE_eventing-controller__CRONJOB_RA_IMAGE": "quay.io/myimage",
		},
		expected: map[string]string{
			"eventing-controller/CRONJOB_RA_IMAGE": "quay.io/myimage",
		},
	}, {
		name: "Deployment+container name",
		envMap: map[string]string{
			"IMAGE_foo__bar": "quay.io/myimage",
		},
		expected: map[string]string{
			"foo/bar": "quay.io/myimage",
		},
	}, {
		name: "Deployment+container and container name",
		envMap: map[string]string{
			"IMAGE_foo__bar": "quay.io/myimage1",
			"IMAGE_bar":      "quay.io/myimage2",
		},
		expected: map[string]string{
			"foo/bar": "quay.io/myimage1",
			"bar":     "quay.io/myimage2",
		},
	}, {
		name: "Different prefix",
		envMap: map[string]string{
			"X_foo": "quay.io/myimage",
		},
		expected: map[string]string{},
	}, {
		name: "No env var value",
		envMap: map[string]string{
			"IMAGE_foo": "",
		},
		expected: map[string]string{},
	}}

	for i := range cases {
		tc := cases[i]
		environ := environFromMap(tc.envMap)
		overrideMap := ImageMapFromEnvironment(environ)

		if !reflect.DeepEqual(overrideMap, tc.expected) {
			t.Errorf("Image override map is not equal. Case name: %q. Expected: %v, actual: %v", tc.name, tc.expected, overrideMap)
		}

	}
}

func environFromMap(envMap map[string]string) []string {
	e := make([]string, 0, len(envMap))
	for k, v := range envMap {
		e = append(e, fmt.Sprintf("%s=%s", k, v))
	}

	return e
}
