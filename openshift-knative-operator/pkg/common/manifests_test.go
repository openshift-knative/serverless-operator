package common

import (
	"testing"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
)

// TestSanitizeManifestSpecClearsUserManifests verifies the unsupported spec.manifests and
// spec.additionalManifests fields are cleared. The operator installs only its own bundled manifests.
func TestSanitizeManifestSpecClearsUserManifests(t *testing.T) {
	t.Parallel()
	spec := &base.CommonSpec{
		Manifests: []base.Manifest{
			{Url: "https://example.com/manifests.yaml"},
		},
		AdditionalManifests: []base.Manifest{
			{Url: "https://example.com/additional-manifests.yaml"},
		},
	}

	if got := SanitizeManifestSpec(spec); !got {
		t.Errorf("SanitizeManifestSpec() = %v, want true", got)
	}
	if spec.Manifests != nil {
		t.Errorf("spec.Manifests = %v, want nil", spec.Manifests)
	}
	if spec.AdditionalManifests != nil {
		t.Errorf("spec.AdditionalManifests = %v, want nil", spec.AdditionalManifests)
	}
}

// TestSanitizeManifestSpecClearsOnlySetFields verifies each field is handled independently:
// only the field that was actually set is cleared and counted.
func TestSanitizeManifestSpecClearsOnlySetFields(t *testing.T) {
	t.Parallel()
	t.Run("only spec.manifests set", func(t *testing.T) {
		t.Parallel()
		spec := &base.CommonSpec{
			Manifests: []base.Manifest{
				{Url: "https://example.com/manifests.yaml"},
			},
		}
		if got := SanitizeManifestSpec(spec); !got {
			t.Errorf("SanitizeManifestSpec() = %v, want true", got)
		}
		if spec.Manifests != nil {
			t.Errorf("spec.Manifests = %v, want nil", spec.Manifests)
		}
	})

	t.Run("only spec.additionalManifests set", func(t *testing.T) {
		t.Parallel()
		spec := &base.CommonSpec{
			AdditionalManifests: []base.Manifest{
				{Url: "https://example.com/additional-manifests.yaml"},
			},
		}
		if got := SanitizeManifestSpec(spec); !got {
			t.Errorf("SanitizeManifestSpec() = %v, want true", got)
		}
		if spec.AdditionalManifests != nil {
			t.Errorf("spec.AdditionalManifests = %v, want nil", spec.AdditionalManifests)
		}
	})

	t.Run("neither set", func(t *testing.T) {
		t.Parallel()
		spec := &base.CommonSpec{}
		if got := SanitizeManifestSpec(spec); got {
			t.Errorf("SanitizeManifestSpec() = %v, want false", got)
		}
	})
}

// TestSanitizeManifestStatusStripsRemoteURLs verifies URL entries are dropped from status.manifests
// while local filesystem paths are kept.
func TestSanitizeManifestStatusStripsRemoteURLs(t *testing.T) {
	t.Parallel()
	status := &v1beta1.KnativeServingStatus{}
	status.SetManifests([]string{
		"/var/run/ko/knative-serving",        // local path -> kept
		"https://example.com/manifests.yaml", // URL -> dropped
		"http://127.0.0.1:1/manifests.yaml",  // URL -> dropped
	})

	if got := SanitizeManifestStatus(status); !got {
		t.Errorf("SanitizeManifestStatus() = %v, want true", got)
	}
	got := status.GetManifests()
	if len(got) != 1 || got[0] != "/var/run/ko/knative-serving" {
		t.Errorf("status.manifests = %v, want [/var/run/ko/knative-serving]", got)
	}
}

// TestSanitizeManifestStatusKeepsLocalPaths verifies that a status made up entirely of local
// paths is left untouched and reports that nothing was dropped.
func TestSanitizeManifestStatusKeepsLocalPaths(t *testing.T) {
	t.Parallel()
	local := []string{
		"/var/run/ko/knative-serving/1.19.0",
		"/var/run/ko/knative-serving/ingress",
	}
	status := &v1beta1.KnativeServingStatus{}
	status.SetManifests(local)

	if got := SanitizeManifestStatus(status); got {
		t.Errorf("SanitizeManifestStatus() = %v, want false", got)
	}
	got := status.GetManifests()
	if len(got) != len(local) || got[0] != local[0] || got[1] != local[1] {
		t.Errorf("status.manifests = %v, want %v", got, local)
	}
}

// TestSanitizeManifestStatusDropsAllRemote verifies that a status made up entirely of URLs is
// emptied and reports that something was dropped.
func TestSanitizeManifestStatusDropsAllRemote(t *testing.T) {
	t.Parallel()
	status := &v1beta1.KnativeServingStatus{}
	status.SetManifests([]string{
		"https://example.com/a.yaml",
		"https://example.com/b.yaml",
	})

	if got := SanitizeManifestStatus(status); !got {
		t.Errorf("SanitizeManifestStatus() = %v, want true", got)
	}
	if got := status.GetManifests(); len(got) != 0 {
		t.Errorf("status.manifests = %v, want empty", got)
	}
}

// TestIsRemoteURL verifies that only http/https URLs are treated as remote, while local
// filesystem paths (existing or not) are treated as local.
func TestIsRemoteURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"https URL", "https://example.com/manifests.yaml", true},
		{"http URL", "http://127.0.0.1:1/manifests.yaml", true},
		{"comma-joined URLs", "https://a.example/x.yaml,https://b.example/y.yaml", true},
		{"absolute local path", "/var/run/ko/knative-serving/1.19.0", false},
		{"nonexistent local path", "/no/such/path/on/disk-xyz", false},
		{"relative local path", "knative-serving/1.19.0", false},
		{"file scheme", "file:///tmp/manifests.yaml", false},
		{"empty string", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isRemoteURL(tt.path); got != tt.want {
				t.Errorf("isRemoteURL(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
