package serving

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

func TestOverrideKourierNamespace(t *testing.T) {
	kourierLabels := map[string]string{
		providerLabel: "kourier",
	}

	withKourier := &unstructured.Unstructured{}
	withKourier.SetNamespace("foo")
	withKourier.SetLabels(kourierLabels)
	withKourier.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: "v1",
		Kind:       "Foo",
		Name:       "bar",
	}})

	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "knative-serving",
			Name:      "test",
		},
	}

	want := withKourier.DeepCopy()
	want.SetNamespace("knative-serving-ingress")
	want.SetLabels(map[string]string{
		providerLabel:                  "kourier",
		socommon.ServingOwnerNamespace: ks.Namespace,
		socommon.ServingOwnerName:      ks.Name,
	})
	want.SetOwnerReferences(nil)

	overrideKourierNamespace(ks)(withKourier)

	if !cmp.Equal(withKourier, want) {
		t.Errorf("Resource was not as expected:\n%s", cmp.Diff(withKourier, want))
	}
}

func TestKourierServiceAppProtocol(t *testing.T) {
	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "knative-serving",
			Name:      "test",
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kourier",
			Labels: map[string]string{providerLabel: "kourier"},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name: "http2",
			}},
		},
	}

	appProtocolName := "h2c"
	expected := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kourier",
			Labels: map[string]string{providerLabel: "kourier"},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:        "http2",
				AppProtocol: &appProtocolName,
			}},
		},
	}

	got := &unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(svc, got, nil); err != nil {
		t.Fatal("Failed to convert service to unstructured", err)
	}

	want := &unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(expected, want, nil); err != nil {
		t.Fatal("Failed to convert service to unstructured", err)
	}

	addKourierAppProtocol(ks)(got)

	if !cmp.Equal(got, want) {
		t.Errorf("Resource was not as expected:\n%s", cmp.Diff(got, want))
	}
}

func TestKourierEnvValue(t *testing.T) {
	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "knative-serving",
			Name:      "test",
		},
		Spec: operatorv1beta1.KnativeServingSpec{
			CommonSpec: base.CommonSpec{
				Config: base.ConfigMapData{
					"network": map[string]string{
						InternalEncryptionKey: "true",
					},
				},
			},
		},
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "net-kourier-controller",
			Labels: map[string]string{providerLabel: "kourier"},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "controller",
						Env:  []corev1.EnvVar{{Name: "a", Value: "b"}},
					}},
				},
			},
		},
	}

	expected := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "net-kourier-controller",
			Labels: map[string]string{providerLabel: "kourier"},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "controller",
						Env: []corev1.EnvVar{
							{Name: "a", Value: "b"},
							{Name: "KOURIER_HTTPOPTION_DISABLED", Value: "true"},
							{Name: "SERVING_NAMESPACE", Value: "knative-serving"},
							{Name: "CERTS_SECRET_NAMESPACE", Value: ingressDefaultCertificateNameSpace},
							{Name: "CERTS_SECRET_NAME", Value: ingressDefaultCertificateName},
						},
					}},
				},
			},
		},
	}

	got := &unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(deploy, got, nil); err != nil {
		t.Fatal("Failed to convert deployment to unstructured", err)
	}

	want := &unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(expected, want, nil); err != nil {
		t.Fatal("Failed to convert deployment to unstructured", err)
	}

	addKourierEnvValues(ks)(got)

	if !cmp.Equal(got, want) {
		t.Errorf("Resource was not as expected:\n%s", cmp.Diff(got, want))
	}
}

func TestKourierInternalEncryptionOverrideCertName(t *testing.T) {
	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "knative-serving",
			Name:      "test",
		},
		Spec: operatorv1beta1.KnativeServingSpec{
			CommonSpec: base.CommonSpec{
				Config: base.ConfigMapData{
					"network": map[string]string{
						InternalEncryptionKey:        "true",
						IngressDefaultCertificateKey: "custom-cert",
					},
				},
			},
		},
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "net-kourier-controller",
			Labels: map[string]string{providerLabel: "kourier"},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "controller",
						Env:  []corev1.EnvVar{{Name: "a", Value: "b"}},
					}},
				},
			},
		},
	}

	expected := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "net-kourier-controller",
			Labels: map[string]string{providerLabel: "kourier"},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "controller",
						Env: []corev1.EnvVar{
							{Name: "a", Value: "b"},
							{Name: "KOURIER_HTTPOPTION_DISABLED", Value: "true"},
							{Name: "SERVING_NAMESPACE", Value: "knative-serving"},
							{Name: "CERTS_SECRET_NAMESPACE", Value: ingressDefaultCertificateNameSpace},
							{Name: "CERTS_SECRET_NAME", Value: "custom-cert"},
						},
					}},
				},
			},
		},
	}

	got := &unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(deploy, got, nil); err != nil {
		t.Fatal("Failed to convert deployment to unstructured", err)
	}

	want := &unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(expected, want, nil); err != nil {
		t.Fatal("Failed to convert deployment to unstructured", err)
	}

	addKourierEnvValues(ks)(got)

	if !cmp.Equal(got, want) {
		t.Errorf("Resource was not as expected:\n%s", cmp.Diff(got, want))
	}
}

func TestOverrideKourierNamespaceOther(t *testing.T) {
	otherLabels := map[string]string{
		providerLabel: "foo",
	}

	other := &unstructured.Unstructured{}
	other.SetNamespace("foo")
	other.SetLabels(otherLabels)
	other.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: "v1",
		Kind:       "Foo",
		Name:       "bar",
	}})
	want := other.DeepCopy()

	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "knative-serving",
			Name:      "test",
		},
	}

	overrideKourierNamespace(ks)(other)

	if !cmp.Equal(other, want) {
		t.Errorf("Resource was not as expected:\n%s", cmp.Diff(other, want))
	}
}
