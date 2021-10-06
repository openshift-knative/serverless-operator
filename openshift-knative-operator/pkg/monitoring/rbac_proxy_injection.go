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
)

const (
	rbacContainerName    = "kube-rbac-proxy"
	rbacProxyImageEnvVar = "IMAGE_KUBE_RBAC_PROXY"
)

func InjectRbacProxyContainerToDeployments(deployments sets.String) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Deployment" && deployments.Has(u.GetName()) {
			var dep = &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, dep, nil); err != nil {
				return fmt.Errorf("failed to transform Unstructred into Deployment: %w", err)
			}

			// Make sure we export metrics only locally.
			firstContainer := &dep.Spec.Template.Spec.Containers[0]
			firstContainer.Env = append(firstContainer.Env, corev1.EnvVar{
				Name:  "METRICS_PROMETHEUS_HOST",
				Value: "127.0.0.1",
			})

			// Add an RBAC proxy to the deployment as it's second container.
			// Order is important here as there is an assumption elsewhere about the first container being the component one.
			volumeName := fmt.Sprintf("secret-%s-sm-service-tls", dep.Name)
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
					fmt.Sprintf("--upstream=http://127.0.0.1:%s/", getDefaultMetricsPort(dep.Name)),
					"--tls-cert-file=" + filepath.Join(mountPath, "tls.crt"),
					"--tls-private-key-file=" + filepath.Join(mountPath, "tls.key"),
					"--logtostderr=true",
					"--v=10",
				},
			}
			dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, rbacProxyContainer)
			dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: fmt.Sprintf("%s-sm-service-tls", dep.Name),
					},
				},
			})
			return scheme.Scheme.Convert(dep, u, nil)
		}
		return nil
	}
}
