package monitoring

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/ptr"
)

func TestInjectRbacProxyContainerToDeployments(t *testing.T) {
	rbacImage := "registry.ci.openshift.org/origin/4.7:kube-rbac-proxy"
	os.Setenv(rbacProxyImageEnvVar, rbacImage)

	in := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "activator",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: "testVolume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "testSecret",
							},
						},
					}},
					Containers: []corev1.Container{{
						Name:  "test",
						Image: "testimage",
						Env: []corev1.EnvVar{{
							Name:  "testEnv",
							Value: "testValue",
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "testVolume",
							MountPath: "/foo/bar",
						}},
					}},
				},
			},
		},
	}

	inU := unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(in, &inU, nil); err != nil {
		t.Fatalf("Failed to convert Deployment to Unstructured: %s", err)
	}
	manifest, err := mf.ManifestFrom(mf.Slice{inU})
	if err != nil {
		t.Fatalf("Failed to construct manifest: %s", err)
	}

	if manifest, err = manifest.Transform(InjectRbacProxyContainer(sets.NewString(in.Name))); err != nil {
		t.Fatalf("Unable to transform test manifest: %s", err)
	}

	got := &appsv1.Deployment{}
	if err := scheme.Scheme.Convert(&manifest.Resources()[0], got, nil); err != nil {
		t.Fatalf("Unable to convert Unstructured to Deployment: %s", err)
	}

	want := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "activator",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: "testVolume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "testSecret",
							},
						},
					}, {
						Name: "secret-activator-sm-service-tls",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "activator-sm-service-tls",
							},
						},
					}},
					Containers: []corev1.Container{{
						Name:  "test",
						Image: "testimage",
						Env: []corev1.EnvVar{{
							Name:  "testEnv",
							Value: "testValue",
						}, {
							Name:  "METRICS_PROMETHEUS_HOST",
							Value: "127.0.0.1",
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "testVolume",
							MountPath: "/foo/bar",
						}},
					}, {
						Name:  rbacContainerName,
						Image: os.Getenv(rbacProxyImageEnvVar),
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "secret-activator-sm-service-tls",
							MountPath: "/etc/tls/private",
						}},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								"memory": resource.MustParse("20Mi"),
								"cpu":    resource.MustParse("10m"),
							}},
						Args: []string{
							"--secure-listen-address=0.0.0.0:8444",
							"--upstream=http://127.0.0.1:9090/",
							"--tls-cert-file=/etc/tls/private/tls.crt",
							"--tls-private-key-file=/etc/tls/private/tls.key",
							"--logtostderr=true",
							"--v=10",
						},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: ptr.Bool(false),
							ReadOnlyRootFilesystem:   ptr.Bool(true),
							RunAsNonRoot:             ptr.Bool(true),
							Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
							SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
						},
					}},
				},
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Error("Unexpected Deployment diff (-want +got): ", diff)
	}
}
