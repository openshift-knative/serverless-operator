package monitoring

import (
	"fmt"
	"os"
	"path/filepath"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	operator "knative.dev/operator/pkg/reconciler/common"
)

const (
	RBACContainerName    = "kube-rbac-proxy"
	rbacProxyImageEnvVar = "IMAGE_KUBE_RBAC_PROXY"
)

var defaultKubeRBACProxyRequests = corev1.ResourceList{
	"memory": resource.MustParse("20Mi"),
	"cpu":    resource.MustParse("10m"),
}

func InjectRbacProxyContainer(deployments sets.Set[string], cfg base.ConfigMapData) mf.Transformer {
	resources := corev1.ResourceRequirements{
		Requests: defaultKubeRBACProxyRequests,
		Limits:   corev1.ResourceList{},
	}
	if cfg != nil && cfg["deployment"] != nil {
		if cpuRequest, ok := cfg["deployment"]["kube-rbac-proxy-cpu-request"]; ok {
			resources.Requests["cpu"] = resource.MustParse(cpuRequest)
		}
		if memRequest, ok := cfg["deployment"]["kube-rbac-proxy-memory-request"]; ok {
			resources.Requests["memory"] = resource.MustParse(memRequest)
		}
		if cpuLimit, ok := cfg["deployment"]["kube-rbac-proxy-cpu-limit"]; ok {
			resources.Limits["cpu"] = resource.MustParse(cpuLimit)
		}
		if memLimit, ok := cfg["deployment"]["kube-rbac-proxy-memory-limit"]; ok {
			resources.Limits["memory"] = resource.MustParse(memLimit)
		}
	}
	return func(u *unstructured.Unstructured) error {
		var podSpec *corev1.PodSpec
		var convert func(spec *corev1.PodSpec) error

		if u.GetKind() == "StatefulSet" && deployments.Has(u.GetName()) {
			ss := &appsv1.StatefulSet{}
			if err := scheme.Scheme.Convert(u, ss, nil); err != nil {
				return fmt.Errorf("failed to transform Unstructred into Deployment: %w", err)
			}
			podSpec = &ss.Spec.Template.Spec
			convert = func(_ *corev1.PodSpec) error {
				ss.Spec.Template.Spec = *podSpec
				return scheme.Scheme.Convert(ss, u, nil)
			}
		}

		if u.GetKind() == "Deployment" && deployments.Has(u.GetName()) {
			var dep = &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, dep, nil); err != nil {
				return fmt.Errorf("failed to transform Unstructred into Deployment: %w", err)
			}
			podSpec = &dep.Spec.Template.Spec
			convert = func(_ *corev1.PodSpec) error {
				dep.Spec.Template.Spec = *podSpec
				return scheme.Scheme.Convert(dep, u, nil)
			}
		}
		if podSpec != nil {

			// Make sure we export metrics only locally.
			firstContainer := &podSpec.Containers[0]
			firstContainer.Env = append(firstContainer.Env, corev1.EnvVar{
				Name:  "METRICS_PROMETHEUS_HOST",
				Value: "127.0.0.1",
			})

			// Add an RBAC proxy to the deployment as it's second container.
			// Order is important here as there is an assumption elsewhere about the first container being the component one.
			volumeName := fmt.Sprintf("secret-%s-sm-service-tls", u.GetName())
			mountPath := "/etc/tls/private"
			rbacProxyContainer := corev1.Container{
				Name:  RBACContainerName,
				Image: os.Getenv(rbacProxyImageEnvVar),
				VolumeMounts: []corev1.VolumeMount{{
					Name:      volumeName,
					MountPath: mountPath,
				}},
				Resources: resources,
				Args: []string{
					"--secure-listen-address=0.0.0.0:8444",
					fmt.Sprintf("--upstream=http://127.0.0.1:%s/", getDefaultMetricsPort(u.GetName())),
					"--tls-cert-file=" + filepath.Join(mountPath, "tls.crt"),
					"--tls-private-key-file=" + filepath.Join(mountPath, "tls.key"),
					"--logtostderr=true",
					"--http2-disable",
					"--v=10",
				},
			}
			podSpec.Containers = append(podSpec.Containers, rbacProxyContainer)
			podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: fmt.Sprintf("%s-sm-service-tls", u.GetName()),
					},
				},
			})
			return convert(podSpec)
		}
		return nil
	}
}

// ExtensionDeploymentOverrides allows to override deployment attributes when extension transforms are applied.
// Normally the knative operator applies deployment overrides before extension transforms are applied.
// For example, we inject the kube-rbac-proxy container at the extension side and this allows to configure the container
// for each deployment it appears in using regular deployment overrides.
func ExtensionDeploymentOverrides(overrides []base.WorkloadOverride, deployments sets.Set[string]) mf.Transformer {
	var ovs []base.WorkloadOverride
	for _, override := range overrides {
		if deployments.Has(override.Name) {
			ovs = append(ovs, override)
		}
	}
	return operator.OverridesTransform(ovs, nil)
}
