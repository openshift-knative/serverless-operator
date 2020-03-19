package util

import (
	"context"
	"fmt"
	"github.com/coreos/go-semver/semver"
	configv1 "github.com/openshift/api/config/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidateOpenShiftVersion validates the current openshift version against the minimum
// version specified in MIN_OPENSHIFT_VERSION env var
func ValidateOpenShiftVersion(ctx context.Context, cl client.Client) (bool, string, error) {
	version, present := os.LookupEnv("MIN_OPENSHIFT_VERSION")
	if !present {
		return true, "", nil
	}
	minVersion, err := semver.NewVersion(version)
	if err != nil {
		return false, "Unable to validate version; check MIN_OPENSHIFT_VERSION env var", nil
	}

	clusterVersion := &configv1.ClusterVersion{}
	if err := cl.Get(ctx, client.ObjectKey{Name: "version"}, clusterVersion); err != nil {
		return false, "Unable to get ClusterVersion", err
	}

	current, err := semver.NewVersion(clusterVersion.Status.Desired.Version)
	if err != nil {
		return false, "Could not parse version string", err
	}

	if current.Major == 0 && current.Minor == 0 {
		return true, "CI build detected, bypassing version check", nil
	}

	if current.LessThan(*minVersion) {
		msg := fmt.Sprintf("Version constraint not fulfilled: minimum version: %s, current version: %s", minVersion.String(), current.String())
		return false, msg, nil
	}
	return true, "", nil
}
