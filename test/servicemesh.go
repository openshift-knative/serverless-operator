package test

import (
	"context"
	"fmt"

	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

// IstioGateway creates an Istio Gateway for HTTP traffic via istio-ingressgateway
func IstioGateway(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1alpha3",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"istio": "ingressgateway",
				},
				"servers": []map[string]interface{}{
					{
						"port": map[string]interface{}{
							"number":   80,
							"name":     "http",
							"protocol": "HTTP",
						},
						"hosts": []string{
							"*",
						},
					},
				},
			},
		},
	}
}

// IstioGatewayWithTLS creates an Istio Gateway for HTTPS traffic via istio-ingressgateway
// for a specific host with a custom domain and certificates.
// The certificate/privateKey must be already mounted on the istio-ingressgateway on the given paths
func IstioGatewayWithTLS(name, namespace string, host, privateKeyPath, serverCertificatePath string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1alpha3",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"istio": "ingressgateway",
				},
				"servers": []map[string]interface{}{
					{
						"port": map[string]interface{}{
							"number":   80,
							"name":     "http",
							"protocol": "HTTP",
						},
						"hosts": []string{
							"*",
						},
					},
					{
						"port": map[string]interface{}{
							"number":   443,
							"name":     "https",
							"protocol": "HTTPS",
						},
						"tls": map[string]interface{}{
							"mode":              "SIMPLE",
							"privateKey":        privateKeyPath,
							"serverCertificate": serverCertificatePath,
						},
						"hosts": []string{
							host,
						},
					},
				},
			},
		},
	}
}

// IstioVirtualServiceForKnativeServiceWithCustomDomain creates an Istio VirtualService to
// rewrite the host header of a custom domain with the ksvc's svc hostname
func IstioVirtualServiceForKnativeServiceWithCustomDomain(service *servingv1.Service, gatewayName string, customHostname string) *unstructured.Unstructured {
	serviceHostname := fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace)
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1alpha3",
			"kind":       "VirtualService",
			"metadata": map[string]interface{}{
				"name":      service.Name,
				"namespace": service.Namespace,
			},
			"spec": map[string]interface{}{
				"hosts": []string{
					customHostname,
				},
				"gateways": []string{
					gatewayName,
				},
				"http": []map[string]interface{}{
					{
						"rewrite": map[string]interface{}{
							"authority": serviceHostname,
						},
						"route": []map[string]interface{}{
							{
								"destination": map[string]interface{}{
									"host": serviceHostname,
									"port": map[string]interface{}{
										"number": 80,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// IstioServiceEntryForKnativeServiceTowardsKourier creates an Istio ServiceEntry
// for ksvc's svc hostname routing towards the knative kourier-internal gateway
func IstioServiceEntryForKnativeServiceTowardsKourier(service *servingv1.Service) *unstructured.Unstructured {
	serviceHostname := fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace)
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1alpha3",
			"kind":       "ServiceEntry",
			"metadata": map[string]interface{}{
				"namespace": service.Namespace,
				"name":      serviceHostname,
			},
			"spec": map[string]interface{}{
				"hosts": []string{
					serviceHostname,
				},
				"location": "MESH_EXTERNAL",
				"endpoints": []map[string]interface{}{
					{
						"address": "kourier-internal.knative-serving-ingress.svc",
					},
				},
				"ports": []map[string]interface{}{
					{
						"number":   80,
						"name":     "http",
						"protocol": "HTTP",
					},
				},
				"resolution": "DNS",
			},
		},
	}
}

func CreateUnstructured(ctx *Context, schema schema.GroupVersionResource, unstructured *unstructured.Unstructured) *unstructured.Unstructured {
	ret, err := ctx.Clients.Dynamic.Resource(schema).Namespace(unstructured.GetNamespace()).Create(context.Background(), unstructured, meta.CreateOptions{})
	if err != nil {
		ctx.T.Fatalf("Error creating %s %s: %v", schema.GroupResource(), unstructured.GetName(), err)
	}

	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up %s %s", schema.GroupResource(), ret.GetName())
		return ctx.Clients.Dynamic.Resource(schema).Namespace(ret.GetNamespace()).Delete(context.Background(), ret.GetName(), meta.DeleteOptions{})
	})

	return ret
}

func CreateIstioGateway(ctx *Context, gateway *unstructured.Unstructured) *unstructured.Unstructured {
	gatewayGvr := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "gateways",
	}

	return CreateUnstructured(ctx, gatewayGvr, gateway)
}

func CreateIstioServiceEntry(ctx *Context, serviceEntry *unstructured.Unstructured) *unstructured.Unstructured {
	serviceEntryGvr := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "serviceentries",
	}

	return CreateUnstructured(ctx, serviceEntryGvr, serviceEntry)
}

func CreateIstioVirtualService(ctx *Context, virtualService *unstructured.Unstructured) *unstructured.Unstructured {
	virtualServiceGvr := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "virtualservices",
	}

	return CreateUnstructured(ctx, virtualServiceGvr, virtualService)
}
