package serving

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	networking "knative.dev/networking/pkg/config"
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
			Namespace:   "knative-serving",
			Name:        "test",
			Annotations: map[string]string{"serverless.openshift.io/default-enable-http2": "true"},
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

func TestKourierBootstrap(t *testing.T) {
	ks := &operatorv1beta1.KnativeServing{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "knative-serving",
			Name:      "test",
		},
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kourier-bootstrap",
			Labels: map[string]string{providerLabel: "kourier"},
		},
		Data: map[string]string{"envoy-bootstrap.yaml": bootstrapData("net-kourier-controller.knative-serving")},
	}

	expected := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kourier-bootstrap",
			Labels: map[string]string{providerLabel: "kourier"},
		},
		Data: map[string]string{"envoy-bootstrap.yaml": bootstrapData("net-kourier-controller.knative-serving-ingress.svc.cluster.local.")},
	}

	got := &unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(cm, got, nil); err != nil {
		t.Fatal("Failed to convert configmap to unstructured", err)
	}

	overrideKourierBootstrap(ks)(got)

	want := &unstructured.Unstructured{}
	if err := scheme.Scheme.Convert(expected, want, nil); err != nil {
		t.Fatal("Failed to convert configmap to unstructured", err)
	}

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
						networking.SystemInternalTLSKey: "true",
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
						networking.SystemInternalTLSKey: "true",
						IngressDefaultCertificateKey:    "custom-cert",
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

func bootstrapData(address string) string {
	return fmt.Sprintf(testData, address)
}

const testData = `
    dynamic_resources:
      ads_config:
        transport_api_version: V3
        api_type: GRPC
        rate_limit_settings: {}
        grpc_services:
        - envoy_grpc: {cluster_name: xds_cluster}
      cds_config:
        resource_api_version: V3
        ads: {}
      lds_config:
        resource_api_version: V3
        ads: {}
    node:
      cluster: kourier-knative
      id: 3scale-kourier-gateway
    static_resources:
      listeners:
        - name: stats_listener
          address:
            socket_address:
              address: 0.0.0.0
              port_value: 9000
          filter_chains:
            - filters:
                - name: envoy.filters.network.http_connection_manager
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                    stat_prefix: stats_server
                    http_filters:
                      - name: envoy.filters.http.router
                        typed_config:
                          "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
                    route_config:
                      virtual_hosts:
                        - name: admin_interface
                          domains:
                            - "*"
                          routes:
                            - match:
                                safe_regex:
                                  regex: '/(certs|stats(/prometheus)?|server_info|clusters|listeners|ready)?'
                                headers:
                                  - name: ':method'
                                    string_match:
                                      exact: GET
                              route:
                                cluster: service_stats
      clusters:
        - name: service_stats
          connect_timeout: 0.250s
          type: static
          load_assignment:
            cluster_name: service_stats
            endpoints:
              lb_endpoints:
                endpoint:
                  address:
                    pipe:
                      path: /tmp/envoy.admin
        - name: xds_cluster
          # This keepalive is recommended by envoy docs.
          # https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol
          typed_extension_protocol_options:
            envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
              "@type": type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
              explicit_http_config:
                http2_protocol_options:
                  connection_keepalive:
                    interval: 30s
                    timeout: 5s
          connect_timeout: 1s
          load_assignment:
            cluster_name: xds_cluster
            endpoints:
              lb_endpoints:
                endpoint:
                  address:
                    socket_address:
                      address: %q
                      port_value: 18000
          type: STRICT_DNS
    admin:
      access_log:
      - name: envoy.access_loggers.stdout
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
      address:
        pipe:
          path: /tmp/envoy.admin
    layered_runtime:
      layers:
        - name: static-layer
          static_layer:
            envoy.reloadable_features.override_request_timeout_by_gateway_timeout: false
`
