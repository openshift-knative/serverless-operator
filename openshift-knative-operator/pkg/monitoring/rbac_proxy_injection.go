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
	"knative.dev/pkg/ptr"
)

const (
	rbacContainerName    = "kube-rbac-proxy"
	rbacProxyImageEnvVar = "IMAGE_KUBE_RBAC_PROXY"
)

func InjectRbacProxyContainer(deployments sets.String) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		var podSpec *corev1.PodSpec
		var convert func(spec *corev1.PodSpec) error

		if u.GetKind() == "StatefulSet" && deployments.Has(u.GetName()) {
			ss := &appsv1.StatefulSet{}
			if err := scheme.Scheme.Convert(u, ss, nil); err != nil {
				return fmt.Errorf("failed to transform Unstructred into Deployment: %w", err)
			}
			podSpec = &ss.Spec.Template.Spec
			convert = func(spec *corev1.PodSpec) error {
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
			convert = func(spec *corev1.PodSpec) error {
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
				Name:  rbacContainerName,
				Image: os.Getenv(rbacProxyImageEnvVar),
				VolumeMounts: []corev1.VolumeMount{{
					Name:      volumeName,
					MountPath: mountPath,
				}},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						"memory": resource.MustParse("20Mi"),
						"cpu":    resource.MustParse("10m"),
					}},
				Args: []string{
					"--secure-listen-address=0.0.0.0:8444",
					fmt.Sprintf("--upstream=http://127.0.0.1:%s/", getDefaultMetricsPort(u.GetName())),
					"--tls-cert-file=" + filepath.Join(mountPath, "tls.crt"),
					"--tls-private-key-file=" + filepath.Join(mountPath, "tls.key"),
					"--logtostderr=true",
					"--v=10",
				},
			}
			rbacProxyContainer.SecurityContext = &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.Bool(false),
				ReadOnlyRootFilesystem:   ptr.Bool(true),
				RunAsNonRoot:             ptr.Bool(true),
				Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
				SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
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
