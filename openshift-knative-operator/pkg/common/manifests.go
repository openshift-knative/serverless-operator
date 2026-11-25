package common

import (
	"net/url"

	"knative.dev/operator/pkg/apis/operator/base"
)

// manifestPathStatus is the minimal component-status view needed to sanitize status.manifests.
type manifestPathStatus interface {
	GetManifests() []string
	SetManifests(manifests []string)
}

// SanitizeManifestSpec clears the unsupported spec.manifests and spec.additionalManifests fields,
// reporting whether anything was cleared. The operator installs only its own bundled manifests.
func SanitizeManifestSpec(spec *base.CommonSpec) bool {
	cleared := false
	if len(spec.Manifests) > 0 {
		spec.Manifests = nil
		cleared = true
	}
	if len(spec.AdditionalManifests) > 0 {
		spec.AdditionalManifests = nil
		cleared = true
	}
	return cleared
}

// SanitizeManifestStatus keeps only local paths in status.manifests, dropping any URL entry,
// reporting whether anything was dropped. The operator records only local (koData) paths there itself.
func SanitizeManifestStatus(status manifestPathStatus) bool {
	paths := status.GetManifests()
	kept := make([]string, 0, len(paths))
	for _, p := range paths {
		if isRemoteURL(p) {
			continue
		}
		kept = append(kept, p)
	}
	if len(kept) == len(paths) {
		return false
	}
	status.SetManifests(kept)
	return true
}

// isRemoteURL reports whether p is an http(s) URL rather than a local filesystem path.
// Local (koData) paths never carry an http/https scheme, so the scheme check alone is
// sufficient to distinguish them.
func isRemoteURL(p string) bool {
	u, err := url.Parse(p)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https")
}
