package common

import (
	"fmt"
	"os"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/blang/semver/v4"
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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

// SetSecurityContextForAdmissionController set the required pod security context to avoid issues on K8s 1.25+.
// For more check:  https://connect.redhat.com/en/blog/important-openshift-changes-pod-security-standards
func SetSecurityContextForAdmissionController() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		switch u.GetKind() {
		case "Deployment":
			deployment := &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, deployment, nil); err != nil {
				return fmt.Errorf("failed to convert Unstructured to Deployment: %w", err)
			}
			obj := deployment
			podSpec := &deployment.Spec.Template.Spec
			containers := podSpec.Containers
			for i := range containers {
				setPodSecurityContext(&containers[i])
			}
			if err := scheme.Scheme.Convert(obj, u, nil); err != nil {
				return err
			}
		case "StatefulSet":
			sset := &appsv1.StatefulSet{}
			if err := scheme.Scheme.Convert(u, sset, nil); err != nil {
				return fmt.Errorf("failed to convert Unstructured to StatefulSet: %w", err)
			}
			obj := sset
			podSpec := &sset.Spec.Template.Spec
			containers := podSpec.Containers
			for i := range containers {
				setPodSecurityContext(&containers[i])
			}
			if err := scheme.Scheme.Convert(obj, u, nil); err != nil {
				return err
			}
		case "Job":
			job := &batchv1.Job{}
			if err := scheme.Scheme.Convert(u, job, nil); err != nil {
				return fmt.Errorf("failed to convert Unstructured to Job: %w", err)
			}
			obj := job
			podSpec := &job.Spec.Template.Spec
			containers := podSpec.Containers
			for i := range containers {
				setPodSecurityContext(&containers[i])
			}
			if err := scheme.Scheme.Convert(obj, u, nil); err != nil {
				return err
			}
		}
		return nil
	}
}

func setPodSecurityContext(container *corev1.Container) {
	if container.SecurityContext == nil {
		container.SecurityContext = &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.Bool(false),
			ReadOnlyRootFilesystem:   ptr.Bool(true),
			RunAsNonRoot:             ptr.Bool(true),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		}
	} else {
		if container.SecurityContext.RunAsNonRoot == nil {
			container.SecurityContext.RunAsNonRoot = ptr.Bool(true)
		}
		if container.SecurityContext.ReadOnlyRootFilesystem == nil {
			container.SecurityContext.ReadOnlyRootFilesystem = ptr.Bool(true)
		}
		if container.SecurityContext.AllowPrivilegeEscalation == nil {
			container.SecurityContext.AllowPrivilegeEscalation = ptr.Bool(false)
		}
		container.SecurityContext.Capabilities = &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}
		if container.SecurityContext.SeccompProfile == nil {
			container.SecurityContext.SeccompProfile = &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}
		}
	}
}

// CheckMinimumKubeVersion checks if current K8s version we are on is higher than the one passed.
// If an error is returned then we
func CheckMinimumKubeVersion(versioner discovery.ServerVersionInterface, version string) error {
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

// DeprecatedAPIsTranformersFromConfig check if we are on the right K8s version and return
// the related transformers. Meant to be used by the knative-openshift operator which uses controller runtime
// and for which we need to construct the discovery value.
func DeprecatedAPIsTranformersFromConfig() []mf.Transformer {
	if v := os.Getenv("TEST_DEPRECATED_APIS_K8S_VERSION"); v != "" {
		return FakeDeprecatedAPIsTranformers(v)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	clset := kubernetes.NewForConfigOrDie(cfg)
	return DeprecatedAPIsTranformers(clset.Discovery())
}

// DeprecatedAPIsTranformers check if we are on the right K8s version and return
// the related transformers.
func DeprecatedAPIsTranformers(d discovery.DiscoveryInterface) []mf.Transformer {
	transformers := []mf.Transformer{}
	// Enforce the new version, try to upgrade existing resources for 4.11 to also avoid warnings.
	// The policy/v1beta1 API version of PodDisruptionBudget will no longer be served in v1.25.
	// The autoscaling/v2beta2 API version of HorizontalPodAutoscaler will no longer be served in v1.26
	// TODO: When we move away from releases that bring v1beta1 we can remove this part
	if err := CheckMinimumKubeVersion(d, "1.24.0"); err == nil {
		transformers = append(transformers, UpgradePodDisruptionBudget(), UpgradeHorizontalPodAutoscaler(), SetSecurityContextForAdmissionController())
	}
	return transformers
}

// FakeDeprecatedAPIsTranformers check if we are on the right K8s version and return
// the related transformers.
func FakeDeprecatedAPIsTranformers(version string) []mf.Transformer {
	transformers := []mf.Transformer{}
	// Enforce the new version, try to upgrade existing resources for 4.11 to also avoid warnings.
	// The policy/v1beta1 API version of PodDisruptionBudget will no longer be served in v1.25.
	// The autoscaling/v2beta2 API version of HorizontalPodAutoscaler will no longer be served in v1.26
	// TODO: When we move away from releases that bring v1beta1 we can remove this part
	if err := CheckMinimumKubeVersion(&dummyVersioner{version: version}, "1.24.0"); err == nil {
		transformers = append(transformers, UpgradePodDisruptionBudget(), UpgradeHorizontalPodAutoscaler(), SetSecurityContextForAdmissionController())
	}
	return transformers
}

type dummyVersioner struct {
	version string
	err     error
}

func (t *dummyVersioner) ServerVersion() (*version.Info, error) {
	return &version.Info{GitVersion: t.version}, t.err
}

func normalizeVersion(v string) string {
	if strings.HasPrefix(v, "v") {
		// No need to account for unicode widths.
		return v[1:]
	}
	return v
}
