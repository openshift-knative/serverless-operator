package common

import (
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
)

// UpgradePodDisruptionBudget upgrade the API version to policy/v1
func UpgradePodDisruptionBudget() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "PodDisruptionBudget" {
			return nil
		}
		if u.GetAPIVersion() != "policy/v1beta1" {
			return nil
		}
		u.SetAPIVersion("policy/v1")
		return nil
	}
}

// UpgradeHorizontalPodAutoscaler upgrade the API version to autoscaling/v2
func UpgradeHorizontalPodAutoscaler() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "HorizontalPodAutoscaler" {
			return nil
		}
		if u.GetAPIVersion() != "autoscaling/v2beta2" {
			return nil
		}
		u.SetAPIVersion("autoscaling/v2")
		return nil
	}
}

// CheckMinimumVersion checks if the version in the arg meets the requirement or not.
// It is similar logic with CheckMinimumVersion() in knative.dev/pkg/version.
func CheckMinimumVersion(versioner discovery.ServerVersionInterface, version string) error {
	v, err := versioner.ServerVersion()
	if err != nil {
		return err
	}
	currentVersion, err := semver.Make(normalizeVersion(v.GitVersion))
	if err != nil {
		return err
	}

	minimumVersion, err := semver.Make(normalizeVersion(version))
	if err != nil {
		return err
	}

	// If no specific pre-release requirement is set, we default to "-0" to always allow
	// pre-release versions of the same Major.Minor.Patch version.
	if len(minimumVersion.Pre) == 0 {
		minimumVersion.Pre = []semver.PRVersion{{VersionNum: 0, IsNum: true}}
	}

	if currentVersion.LT(minimumVersion) {
		return fmt.Errorf("kubernetes version %q is not compatible, need at least %q",
			currentVersion, minimumVersion)
	}
	return nil
}

func DeprecatedAPIsTranformers(d discovery.DiscoveryInterface) []mf.Transformer {
	transformers := []mf.Transformer{}
	// Enforce the new version, try to upgrade existing resources.
	// The policy/v1beta1 API version of PodDisruptionBudget will no longer be served in v1.25.
	// The autoscaling/v2beta2 API version of HorizontalPodAutoscaler will no longer be served in v1.26
	// TODO: When we move away from releases that bring v1beta1 we can remove this part
	if err := CheckMinimumVersion(d, "1.25.0"); err == nil {
		transformers = append(transformers, UpgradePodDisruptionBudget(), UpgradeHorizontalPodAutoscaler())
	}
	return transformers
}

func normalizeVersion(v string) string {
	if strings.HasPrefix(v, "v") {
		// No need to account for unicode widths.
		return v[1:]
	}
	return v
}
