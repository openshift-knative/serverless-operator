package features

import (
	"strconv"
	"strings"
)

// filterByVersion returns only components available in the given version.
// If version is empty, all components are returned (no filtering).
func filterByVersion(components []component, version string) []component {
	if version == "" {
		return components
	}
	var filtered []component
	for _, c := range components {
		if sv, ok := c.(sinceVersionable); ok {
			since := sv.SinceVersion()
			if since != "" && compareVersions(version, since) < 0 {
				continue
			}
		}
		filtered = append(filtered, c)
	}
	return filtered
}

// compareVersions compares two semver-like version strings (e.g. "1.36.0").
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	aParts := parseVersion(a)
	bParts := parseVersion(b)

	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		result[i], _ = strconv.Atoi(parts[i])
	}
	return result
}
