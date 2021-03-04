package monitoring

import (
	"fmt"
	"os"
	"strings"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	rbacContainerName = "kube-rbac-proxy"
	fallbackImage     = "registry.ci.openshift.org/origin/4.7:kube-rbac-proxy"
)

func InjectRbacProxyContainerToDeployments() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		kind := strings.ToLower(u.GetKind())
		// Only touch the related deployments
		joinedComp := strings.Join(servingComponents, ",")
		if kind == "deployment" && strings.Contains(joinedComp, u.GetName()) {
			var dep = &appsv1.Deployment{}
			if err := scheme.Scheme.Convert(u, dep, nil); err != nil {
				return err
			}
			depName := u.GetName()
			firstContainer := &dep.Spec.Template.Spec.Containers[0]
			// Make sure we export metrics only locally
			firstContainer.Env = append(firstContainer.Env, corev1.EnvVar{Name: "METRICS_PROMETHEUS_HOST", Value: "127.0.0.1"})
			// Order is important here as there is an assumption elsewhere about the first container being the component one
			dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, makeRbacProxyContainer(depName))
			dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, []corev1.Volume{{
				Name: fmt.Sprintf("secret-%s-sm-service-tls", depName),
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: fmt.Sprintf("%s-sm-service-tls", depName),
					},
				},
			}}...)
			return scheme.Scheme.Convert(dep, u, nil)
		}
		return nil
	}
}

func makeRbacProxyContainer(depName string) corev1.Container {
	c := corev1.Container{}
	c.Name = rbacContainerName
	c.Image = getRbacProxyImage(depName)
	c.VolumeMounts = []corev1.VolumeMount{{
		Name:      fmt.Sprintf("secret-%s-sm-service-tls", depName),
		MountPath: "/etc/tls/private",
	}}
	c.Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			"memory": resource.MustParse("20Mi"),
			"cpu":    resource.MustParse("10m"),
		}}
	c.Args = []string{
		"--secure-listen-address=0.0.0.0:8444",
		"--upstream=http://127.0.0.1:9090/",
		"--tls-cert-file=/etc/tls/private/tls.crt",
		"--tls-private-key-file=/etc/tls/private/tls.key",
		"--logtostderr=true",
		"--v=10",
	}
	return c
}

func getRbacProxyImage(depName string) string {
	image := os.Getenv(fmt.Sprintf("IMAGE_%s__kube-rbac-proxy", depName))
	if image == "" {
		return fallbackImage
	}
	return image
}
