package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"knative.dev/operator/pkg/apis/operator/base"
)

func TestConfigure(t *testing.T) {
	cases := []struct {
		name     string
		in       base.ConfigMapData
		expected base.ConfigMapData
	}{{
		name: "all nil",
		expected: base.ConfigMapData{
			"foo": map[string]string{
				"bar": "baz",
			},
		},
	}, {
		name: "first level already set",
		in: base.ConfigMapData{
			"foo": map[string]string{},
		},
		expected: base.ConfigMapData{
			"foo": map[string]string{
				"bar": "baz",
			},
		},
	}, {
		name: "override",
		in: base.ConfigMapData{
			"foo": map[string]string{
				"bar": "nope",
			},
		},
		expected: base.ConfigMapData{
			"foo": map[string]string{
				"bar": "baz",
			},
		},
	}, {
		name: "unrelated values",
		in: base.ConfigMapData{
			"foo": map[string]string{
				"bar2": "baz2",
			},
			"foo2": map[string]string{
				"bar": "baz",
			},
		},
		expected: base.ConfigMapData{
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
			s := &base.CommonSpec{Config: c.in}
			Configure(s, "foo", "bar", "baz")

			if !cmp.Equal(s.Config, c.expected) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", s.Config, c.expected, cmp.Diff(s.Config, c.expected))
			}
		})
	}
}

func TestConfigureIfUnsetAny(t *testing.T) {
	cases := []struct {
		name     string
		in       base.ConfigMapData
		expected base.ConfigMapData
	}{{
		name: "all nil",
		expected: base.ConfigMapData{
			"foo": map[string]string{
				"bar": "baz",
			},
		},
	}, {
		name: "first level already set but empty configuration",
		in: base.ConfigMapData{
			"foo": map[string]string{},
		},
		expected: base.ConfigMapData{
			"foo": map[string]string{
				"bar": "baz",
			},
		},
	}, {
		name: "override",
		in: base.ConfigMapData{
			"foo": map[string]string{
				"bar": "nope",
			},
		},
		expected: base.ConfigMapData{
			"foo": map[string]string{
				"bar": "nope",
			},
		},
	}, {
		name: "unrelated values",
		in: base.ConfigMapData{
			"foo": map[string]string{
				"bar2": "baz2",
			},
			"foo2": map[string]string{
				"bar": "baz",
			},
		},
		expected: base.ConfigMapData{
			"foo": map[string]string{
				"bar2": "baz2",
			},
			"foo2": map[string]string{
				"bar": "baz",
			},
		},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &base.CommonSpec{Config: c.in}
			ConfigureIfUnsetAny(s, "foo", "bar", "baz")

			if !cmp.Equal(s.Config, c.expected) {
				t.Errorf("Got = %v, want: %v, diff:\n%s", s.Config, c.expected, cmp.Diff(s.Config, c.expected))
			}
		})
	}
}
