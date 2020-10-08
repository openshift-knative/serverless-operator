package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func TestConfigure(t *testing.T) {
	cases := []struct {
		name     string
		in       v1alpha1.ConfigMapData
		expected v1alpha1.ConfigMapData
	}{{
		name: "all nil",
		expected: v1alpha1.ConfigMapData{
			"foo": map[string]string{
				"bar": "baz",
			},
		},
	}, {
		name: "first level already set",
		in: v1alpha1.ConfigMapData{
			"foo": map[string]string{},
		},
		expected: v1alpha1.ConfigMapData{
			"foo": map[string]string{
				"bar": "baz",
			},
		},
	}, {
		name: "override",
		in: v1alpha1.ConfigMapData{
			"foo": map[string]string{
				"bar": "nope",
			},
		},
		expected: v1alpha1.ConfigMapData{
			"foo": map[string]string{
				"bar": "baz",
			},
		},
	}, {
		name: "unrelated values",
		in: v1alpha1.ConfigMapData{
			"foo": map[string]string{
				"bar2": "baz2",
			},
			"foo2": map[string]string{
				"bar": "baz",
			},
		},
		expected: v1alpha1.ConfigMapData{
			"foo": map[string]string{
				"bar":  "baz",
				"bar2": "baz2",
			},
			"foo2": map[string]string{
				"bar": "baz",
			},
		},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &v1alpha1.CommonSpec{Config: c.in}
			Configure(s, "foo", "bar", "baz")

			if !cmp.Equal(s.Config, c.expected) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", s.Config, c.expected, cmp.Diff(s.Config, c.expected))
			}
		})
	}
}
