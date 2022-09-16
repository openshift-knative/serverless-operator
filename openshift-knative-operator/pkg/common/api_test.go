package common

import (
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/version"
)

type testVersioner struct {
	version string
	err     error
}

func (t *testVersioner) ServerVersion() (*version.Info, error) {
	return &version.Info{GitVersion: t.version}, t.err
}

func TestVersionCheck(t *testing.T) {
	tests := []struct {
		name          string
		actualVersion *testVersioner
		wantError     bool
	}{{
		name:          "greater version (patch)",
		actualVersion: &testVersioner{version: "v1.20.0"},
	}, {
		name:          "greater version (patch), no v",
		actualVersion: &testVersioner{version: "1.20.0"},
	}, {
		name:          "greater version (patch), pre-release",
		actualVersion: &testVersioner{version: "1.20.2-kpn-065dce"},
	}, {
		name:          "greater version (patch), pre-release with build",
		actualVersion: &testVersioner{version: "1.20.0-1095+9689d22dc3121e-dirty"},
	}, {
		name:          "greater version (minor)",
		actualVersion: &testVersioner{version: "v1.20.0"},
	}, {
		name:          "same version",
		actualVersion: &testVersioner{version: "v1.20.0"},
	}, {
		name:          "same version with build",
		actualVersion: &testVersioner{version: "v1.20.0+k3s.1"},
	}, {
		name:          "same version with pre-release",
		actualVersion: &testVersioner{version: "v1.20.0-k3s.1"},
	}, {
		name:          "smaller version",
		actualVersion: &testVersioner{version: "v1.19.3"},
		wantError:     true,
	}, {
		name:          "error while fetching",
		actualVersion: &testVersioner{err: errors.New("random error")},
		wantError:     true,
	}, {
		name:          "unparseable actual version",
		actualVersion: &testVersioner{version: "v1.19.foo"},
		wantError:     true,
	}}

	minVersion := "1.20.0"

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CheckMinimumVersion(test.actualVersion, minVersion)
			if err == nil && test.wantError {
				t.Errorf("Expected an error for minimum: %q, actual: %v", minVersion, test.actualVersion)
			}

			if err != nil && !test.wantError {
				t.Errorf("Expected no error but got %v for minimum: %q, actual: %v", err, minVersion, test.actualVersion)
			}
		})
	}
}
