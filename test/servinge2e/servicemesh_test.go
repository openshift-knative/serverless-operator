package servinge2e

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/openshift-knative/serverless-operator/test"
	routev1 "github.com/openshift/api/route/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/helpers"
	"knative.dev/pkg/test/spoof"
	"knative.dev/serving/pkg/apis/autoscaling"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	// A test namespace that is part of the ServiceMesh (setup by "make install-mesh")
	serviceMeshTestNamespaceName = "default"
	serviceMeshTestImage         = "gcr.io/knative-samples/helloworld-go"
	serviceMeshTestProxyImage    = "registry.svc.ci.openshift.org/openshift/knative-v0.17.3:knative-serving-test-httpproxy"
)

func getServiceMeshNamespace(ctx *test.Context) string {
	namespace, err := ctx.Clients.Kube.CoreV1().Namespaces().Get(serviceMeshTestNamespaceName, meta.GetOptions{})
	if err != nil {
		ctx.T.Fatalf("Failed to verify %q namespace labels: %v", serviceMeshTestNamespaceName, err)
	}

	return namespace.Labels["maistra.io/member-of"]
}

func isServiceMeshInstalled(ctx *test.Context) bool {
	return getServiceMeshNamespace(ctx) != ""
}

// A knative service acting as an "http proxy", redirects requests towards a given "host". Used to test cluster-local services
func httpProxyService(name, host string) *servingv1.Service {
	proxy := test.Service(name, serviceMeshTestNamespaceName, serviceMeshTestProxyImage, nil)
	proxy.Spec.Template.Spec.Containers[0].Env = append(proxy.Spec.Template.Spec.Containers[0].Env, core.EnvVar{
		Name:  "TARGET_HOST",
		Value: host,
	})

	return proxy
}

func withServiceReadyOrFail(ctx *test.Context, service *servingv1.Service) *servingv1.Service {
	service, err := ctx.Clients.Serving.ServingV1().Services(service.Namespace).Create(service)
	if err != nil {
		ctx.T.Fatalf("Error creating ksvc: %v", err)
	}

	// Let the ksvc be deleted after test
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up Knative Service '%s/%s'", service.Namespace, service.Name)
		return ctx.Clients.Serving.ServingV1().Services(service.Namespace).Delete(service.Name, &meta.DeleteOptions{})
	})

	service, err = test.WaitForServiceState(ctx, service.Name, service.Namespace, test.IsServiceReady)
	if err != nil {
		ctx.T.Fatalf("Error waiting for ksvc readiness: %v", err)
	}

	return service
}

// Skipped unless ServiceMesh has been installed via "make install-mesh"
func TestKsvcWithServiceMeshSidecar(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	// Skip test if ServiceMesh not installed
	if !isServiceMeshInstalled(caCtx) {
		t.Skipf("Test namespace %q not a mesh member, use \"make install-mesh\" for ServiceMesh setup", serviceMeshTestNamespaceName)
	}

	tests := []struct {
		name               string
		labels             map[string]string // Ksvc labels
		annotations        map[string]string // Revision template annotations
		expectIstioSidecar bool              // Whether it is expected for the istio-proxy sidecar to be injected into the pod
	}{{
		// Requests go via gateway -> activator -> pod , by default
		// Verifies the activator can connect to the pod
		name: "sidecar-via-activator",
		annotations: map[string]string{
			"sidecar.istio.io/inject": "true",
		},
		expectIstioSidecar: true,
	}, {
		// Requests go via gateway -> pod ( activator should be skipped if burst capacity is disabled and there is at least 1 replica)
		// Verifies the gateway can connect to the pod directly
		name: "sidecar-without-activator",
		annotations: map[string]string{
			"sidecar.istio.io/inject":          "true",
			autoscaling.TargetBurstCapacityKey: "0",
			autoscaling.MinScaleAnnotationKey:  "1",
		},
		expectIstioSidecar: true,
	}, {
		// Verifies the "sidecar.istio.io/inject" annotation is really what decides the istio-proxy presence
		name: "no-sidecar",
		annotations: map[string]string{
			"sidecar.istio.io/inject": "false",
		},
		expectIstioSidecar: false,
	}, {
		// A cluster-local variant of the "sidecar-via-activator" scenario
		name: "local-sidecar-via-activator",
		labels: map[string]string{
			serving.VisibilityLabelKey: serving.VisibilityClusterLocal,
		},
		annotations: map[string]string{
			"sidecar.istio.io/inject": "true",
		},
		expectIstioSidecar: true,
	}, {
		// A cluster-local variant of the "sidecar-without-activator" scenario
		name: "local-sidecar-without-activator",
		labels: map[string]string{
			serving.VisibilityLabelKey: serving.VisibilityClusterLocal,
		},
		annotations: map[string]string{
			"sidecar.istio.io/inject":          "true",
			autoscaling.TargetBurstCapacityKey: "0",
			autoscaling.MinScaleAnnotationKey:  "1",
		},
		expectIstioSidecar: true,
	}}

	for _, scenario := range tests {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			// Create a ksvc with the specified annotations and labels
			service := test.Service(scenario.name, serviceMeshTestNamespaceName, serviceMeshTestImage, scenario.annotations)
			service.ObjectMeta.Labels = scenario.labels
			service, err := caCtx.Clients.Serving.ServingV1().Services(serviceMeshTestNamespaceName).Create(service)
			if err != nil {
				t.Errorf("error creating ksvc: %v", err)
				return
			}

			// Let the ksvc be deleted after test
			caCtx.AddToCleanup(func() error {
				t.Logf("Cleaning up Knative Service '%s/%s'", service.Namespace, service.Name)
				return caCtx.Clients.Serving.ServingV1().Services(serviceMeshTestNamespaceName).Delete(service.Name, &meta.DeleteOptions{})
			})

			// Wait until the Ksvc is ready.
			service, err = test.WaitForServiceState(caCtx, service.Name, service.Namespace, test.IsServiceReady)
			if err != nil {
				t.Errorf("error waiting for ksvc readiness: %v", err)
				return
			}

			serviceURL := service.Status.URL.URL()

			// For cluster-local ksvc, we deploy an "HTTP proxy" service, and request that one instead
			if service.GetLabels()[serving.VisibilityLabelKey] == serving.VisibilityClusterLocal {
				// Deploy an "HTTP proxy" towards the ksvc (using an httpproxy image from knative-serving testsuite)
				httpProxy, err := caCtx.Clients.Serving.ServingV1().Services(serviceMeshTestNamespaceName).Create(
					httpProxyService(scenario.name+"-proxy", service.Status.URL.Host))
				if err != nil {
					t.Errorf("error creating ksvc: %v", err)
					return
				}

				// Let the ksvc be deleted after test
				caCtx.AddToCleanup(func() error {
					t.Logf("Cleaning up Knative Service '%s/%s'", httpProxy.Namespace, httpProxy.Name)
					return caCtx.Clients.Serving.ServingV1().Services(serviceMeshTestNamespaceName).Delete(httpProxy.Name, &meta.DeleteOptions{})
				})

				// Wait until the Proxy is ready.
				httpProxy, err = test.WaitForServiceState(caCtx, httpProxy.Name, httpProxy.Namespace, test.IsServiceReady)
				if err != nil {
					t.Errorf("error waiting for ksvc readiness: %v", err)
					return
				}

				serviceURL = httpProxy.Status.URL.URL()
			}

			// Verify the service is actually accessible from the outside
			if _, err := pkgTest.WaitForEndpointState(
				&pkgTest.KubeClient{Kube: caCtx.Clients.Kube},
				t.Logf,
				serviceURL,
				pkgTest.EventuallyMatchesBody(helloworldText),
				"WaitForRouteToServeText",
				true); err != nil {
				t.Errorf("the Route at domain %s didn't serve the expected text %q: %v", service.Status.URL.URL(), helloworldText, err)
			}

			// Verify the expected istio-proxy is really there
			podList, err := caCtx.Clients.Kube.CoreV1().Pods(serviceMeshTestNamespaceName).List(meta.ListOptions{LabelSelector: "serving.knative.dev/service=" + service.Name})
			if err != nil {
				t.Errorf("error listing pods: %v", err)
				return
			}

			for _, pod := range podList.Items {
				istioProxyFound := false
				for _, container := range pod.Spec.Containers {
					if container.Name == "istio-proxy" {
						istioProxyFound = true
					}
				}

				if scenario.expectIstioSidecar != istioProxyFound {
					if scenario.expectIstioSidecar {
						t.Errorf("scenario %s expects istio-proxy to be present, but no such container exists in %s", scenario.name, pod.Name)
					} else {
						t.Errorf("scenario %s does not expect istio-proxy to be present in pod %s, but it has one", scenario.name, pod.Name)
					}
				}
			}
		})
	}
}

// formats an RSA public key as JWKS
func rsaPublicKeyAsJwks(key rsa.PublicKey, keyId string) (string, error) {
	eString := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
	nString := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	keyIdString := base64.RawURLEncoding.EncodeToString([]byte(keyId))

	// Generate JWKS
	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"e":   eString,
				"n":   nString,
				"kty": "RSA",
				"kid": keyIdString,
			},
		},
	}

	jwksBytes, err := json.Marshal(jwks)
	if err != nil {
		return "", fmt.Errorf("error marshalling jwks: %v", err)
	}

	return string(jwksBytes), nil
}

// jwtRs256Token generates a valid JWT RS256 token with the given payload and signs it with rsaKey
func jwtRs256Token(rsaKey *rsa.PrivateKey, payload map[string]interface{}) (string, error) {
	jwtHeader := map[string]interface{}{
		"alg": "RS256",
		"typ": "JWT",
	}
	jwtHeaderBytes, err := json.Marshal(jwtHeader)
	if err != nil {
		return "", fmt.Errorf("error marshalling jwt header into JSON: %v", err)
	}

	jwtPayloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling jwt payload into JSON: %v", err)
	}

	jwtHeaderBase64 := base64.RawURLEncoding.EncodeToString(jwtHeaderBytes)
	jwtPayloadBase64 := base64.RawURLEncoding.EncodeToString(jwtPayloadBytes)

	jwtSigningInput := jwtHeaderBase64 + "." + jwtPayloadBase64
	hashFunc := crypto.SHA256.New()
	hashFunc.Write([]byte(jwtSigningInput))
	hash := hashFunc.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hash)
	if err != nil {
		return "", fmt.Errorf("error signing JWT token with PKSC1v15: %v", err)
	}

	jwtSignatureBase64 := base64.RawURLEncoding.EncodeToString(signature)
	jwtToken := jwtSigningInput + "." + jwtSignatureBase64

	return jwtToken, nil
}

// jwtUnsignedToken generates a valid unsigned JWT token with the given payload
func jwtUnsignedToken(payload map[string]interface{}) (string, error) {
	jwtHeader := map[string]interface{}{
		"alg": "none",
	}
	jwtHeaderBytes, err := json.Marshal(jwtHeader)
	if err != nil {
		return "", fmt.Errorf("error marshalling jwt header into JSON: %v", err)
	}

	jwtPayloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling jwt payload into JSON: %v", err)
	}

	jwtHeaderBase64 := base64.RawURLEncoding.EncodeToString(jwtHeaderBytes)
	jwtPayloadBase64 := base64.RawURLEncoding.EncodeToString(jwtPayloadBytes)

	jwtToken := jwtHeaderBase64 + "." + jwtPayloadBase64 + "."

	return jwtToken, nil
}

// Convenience method to test requests with tokens, reads the response and returns a closed response and the body bits
// token can be nil, in which case no Authorization header will be sent
func jwtHttpGetRequestBytes(url string, token *string) (*http.Response, []byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating HTTP GET request: %v", err)
	}
	if token != nil {
		req.Header.Add("Authorization", "Bearer "+*token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("error doing HTTP GET request: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return resp, body, err
}

// Verifies access to a Ksvc with an istio-proxy can be configured
// via istio authentication Policy to allow valid JWT only.
// Skipped unless ServiceMesh has been installed via "make install-mesh"
func TestKsvcWithServiceMeshJWTDefaultPolicy(t *testing.T) {

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	// Skip test if ServiceMesh not installed
	if !isServiceMeshInstalled(caCtx) {
		t.Skipf("Test namespace %q not a mesh member, use \"make install-mesh\" for ServiceMesh setup", serviceMeshTestNamespaceName)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Error generating private key: %v", err)
	}

	// print out the public key for debugging purposes
	publicPksvc1, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	publicPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicPksvc1,
	})
	t.Logf("%s", string(publicPem))

	// Used as "kid" in JWKS
	const keyId = "test"
	const issuer = "testing-issuer@secure.serverless.openshift.io"
	const subject = "testing-subject@secure.serverless.openshift.io"
	// For testing an invalid token (with a different issuer)
	const wrongIssuer = "eve@secure.serverless.openshift.io"

	// Generate a new key for a "wrong key" scenario
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Error generating private key: %v", err)
	}

	jwks, err := rsaPublicKeyAsJwks(privateKey.PublicKey, keyId)
	if err != nil {
		t.Fatalf("Error encoding RSA public key as JWKS: %v", err)
	}

	// Istio 1.1 and earlier lack "jwks" option, only "jwksUri", so we need to host it on some URL
	// We'll misuse the "hello-openshift" image with the JWKS file defined as the RESPONSE env, and deploy this as a ksvc

	// istio-pilot caches the JWKS content if a new Policy has the same jwksUri as some old policy.
	// Rerunning this test would fail if we kept the jwksUri constant across invocations then,
	// hence the random suffix for the jwks ksvc.
	jwksKsvc := test.Service(helpers.AppendRandomString("jwks"), serviceMeshTestNamespaceName, "openshift/hello-openshift", nil)
	jwksKsvc.Spec.Template.Spec.Containers[0].Env = append(jwksKsvc.Spec.Template.Spec.Containers[0].Env, core.EnvVar{
		Name:  "RESPONSE",
		Value: jwks,
	})
	jwksKsvc.ObjectMeta.Labels = map[string]string{
		serving.VisibilityLabelKey: serving.VisibilityClusterLocal,
	}
	jwksKsvc = withServiceReadyOrFail(caCtx, jwksKsvc)

	// Create a Policy
	authPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "authentication.istio.io/v1alpha1",
			"kind":       "Policy",
			"metadata": map[string]interface{}{
				"name": "default",
			},
			"spec": map[string]interface{}{
				"principalBinding": "USE_ORIGIN",
				"origins": []map[string]interface{}{
					{
						"jwt": map[string]interface{}{
							"issuer":  issuer,
							"jwksUri": jwksKsvc.Status.URL,
							"triggerRules": []map[string]interface{}{
								{
									"excludedPaths": []map[string]interface{}{
										{
											"prefix": "/metrics",
										},
										{
											"prefix": "/healthz",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	policyGvr := schema.GroupVersionResource{
		Group:    "authentication.istio.io",
		Version:  "v1alpha1",
		Resource: "policies",
	}

	authPolicy, err = caCtx.Clients.Dynamic.Resource(policyGvr).Namespace(serviceMeshTestNamespaceName).Create(authPolicy, meta.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating istio Policy: %v", err)
	}

	caCtx.AddToCleanup(func() error {
		t.Logf("Cleaning up istio Policy default")
		return caCtx.Clients.Dynamic.Resource(policyGvr).Namespace(authPolicy.GetNamespace()).Delete(authPolicy.GetName(), &meta.DeleteOptions{})
	})

	// Create a test ksvc, should be accessible only via proper JWT token
	testKsvc := test.Service("jwt-test", serviceMeshTestNamespaceName, image, map[string]string{
		"sidecar.istio.io/inject": "true",
	})
	testKsvc = withServiceReadyOrFail(caCtx, testKsvc)

	// Wait until the Route is ready and also verify the route returns a 401 without a token
	if _, err := pkgTest.WaitForEndpointState(
		&pkgTest.KubeClient{Kube: caCtx.Clients.Kube},
		t.Logf,
		testKsvc.Status.URL.URL(),
		func(resp *spoof.Response) (bool, error) {
			if resp.StatusCode != 401 {
				// Returning (false, nil) causes SpoofingClient.Poll to retry.
				return false, nil
			}
			return true, nil
		},
		"WaitForRouteToServe401",
		true); err != nil {
		t.Fatalf("The Route at domain %s didn't serve the expected HTTP 401 status: %v", testKsvc.Status.URL.URL(), err)
	}

	tests := []struct {
		name    string
		valid   bool // Is the token expected to be valid?
		key     *rsa.PrivateKey
		payload map[string]interface{}
	}{{
		// A valid token
		"valid",
		true,
		privateKey,
		map[string]interface{}{
			"iss": issuer,
			"sub": subject,
			"foo": "bar",
			"iat": time.Now().Unix(),
			"exp": time.Now().Unix() + 3600,
		},
	},
		{
			// No token (request will be done without the Authorization header)
			"no_token",
			false,
			nil,
			nil,
		},
		{
			// Unsigned token
			"unsigned",
			false,
			nil,
			map[string]interface{}{
				"iss": issuer,
				"sub": subject,
				"foo": "bar",
				"iat": time.Now().Unix(),
				"exp": time.Now().Unix() + 3600,
			},
		},
		{
			// A token with "exp" time in the past
			"expired",
			false,
			privateKey,
			map[string]interface{}{
				"iss": issuer,
				"sub": subject,
				"foo": "bar",
				// as if generated before an hour, expiring 10 seconds ago
				"iat": time.Now().Unix() - 3600,
				"exp": time.Now().Unix() - 10,
			},
		}, {
			// A token signed by a different key
			"bad_key",
			false,
			wrongKey,
			map[string]interface{}{
				"iss": issuer,
				"sub": subject,
				"foo": "bar",
				"iat": time.Now().Unix(),
				"exp": time.Now().Unix() + 3600,
			},
		}, {
			// A token with an issuer set to a different principal than the one specified in the Policy
			"bad_iss",
			false,
			privateKey,
			map[string]interface{}{
				"iss": wrongIssuer,
				"sub": subject,
				"foo": "bar",
				"iat": time.Now().Unix(),
				"exp": time.Now().Unix() + 3600,
			},
		}}

	for _, scenario := range tests {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			var tokenRef *string
			var err error

			// nil payload means no token (in which case we don't send "Authorization" header at all)
			if scenario.payload != nil {
				var token string
				if scenario.key != nil {
					// Generate a signed RS256 token
					token, err = jwtRs256Token(scenario.key, scenario.payload)
					if err != nil {
						t.Fatalf("Error generating RS256 token: %v", err)
					}
				} else {
					// Generate an unsigned token if RSA key not specified
					token, err = jwtUnsignedToken(scenario.payload)
					if err != nil {
						t.Fatalf("Error generating an unsigned token: %v", err)
					}
				}

				tokenRef = &token
			}

			// Do a request, optionally with a token
			resp, body, err := jwtHttpGetRequestBytes(testKsvc.Status.URL.String(), tokenRef)
			if err != nil {
				t.Fatalf("Error doing HTTP GET request: %v", err)
			}

			if scenario.valid {
				// Verify the response is a proper "hello world" when the token is valid
				if resp.StatusCode != 200 || !strings.Contains(string(body), helloworldText) {
					t.Fatalf("Unexpected response with a valid token: HTTP %d: %s", resp.StatusCode, string(body))
				}
			} else {
				// Verify the response is a 401 for an invalid token
				if resp.StatusCode != 401 {
					t.Fatalf("Unexpected response with an invalid token, expecting 401, got %d: %s", resp.StatusCode, string(body))
				}
			}
		})
	}
}

func lookupOpenShiftRouterIP(ctx *test.Context) net.IP {
	// Deploy an auxiliary ksvc accessible via an OpenShift route, so that we have a route hostname that we can resolve
	aux := test.Service("aux", serviceMeshTestNamespaceName, "openshift/hello-openshift", nil)
	aux = withServiceReadyOrFail(ctx, aux)

	ips, err := net.LookupIP(aux.Status.URL.Host)
	if err != nil {
		ctx.T.Fatalf("Error looking up ksvc's hostname IP address: %v", err)
	}
	if len(ips) == 0 {
		ctx.T.Fatalf("No IP address found for %s", aux.Status.URL.Host)
	}

	ctx.T.Logf("Resolved the following IPs %v as the OpenShift Router address and use %v for test", ips, ips[0])
	return ips[0]
}

func TestKsvcWithServiceMeshCustomDomain(t *testing.T) {

	const customDomain = "custom-ksvc-domain.example.com"

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	// Skip test if ServiceMesh not installed
	serviceMeshNamespace := getServiceMeshNamespace(caCtx)
	if serviceMeshNamespace == "" {
		t.Skipf("Test namespace %q not a mesh member, use \"make install-mesh\" for ServiceMesh setup", serviceMeshTestNamespaceName)
	}

	// Deploy a cluster-local ksvc "hello"
	ksvc := test.Service("hello", serviceMeshTestNamespaceName, "openshift/hello-openshift", nil)
	ksvc.ObjectMeta.Labels = map[string]string{
		serving.VisibilityLabelKey: serving.VisibilityClusterLocal,
	}
	ksvc = withServiceReadyOrFail(caCtx, ksvc)

	// Create the Istio Gateway for traffic via istio-ingressgateway
	defaultGateway := test.IstioGateway("default-gateway", serviceMeshTestNamespaceName)
	defaultGateway = test.CreateIstioGateway(caCtx, defaultGateway)

	// Create the Istio VirtualService to rewrite the host header of a custom domain with the ksvc's svc hostname
	virtualService := test.IstioVirtualServiceForKnativeServiceWithCustomDomain(ksvc, defaultGateway.GetName(), customDomain)
	virtualService = test.CreateIstioVirtualService(caCtx, virtualService)

	// Create the Istio ServiceEntry for ksvc's svc hostname routing towards the knative kourier-internal gateway
	serviceEntry := test.IstioServiceEntryForKnativeServiceTowardsKourier(ksvc)
	serviceEntry = test.CreateIstioServiceEntry(caCtx, serviceEntry)

	// Create the OpenShift Route for the custom domain pointing to the istio-ingressgateway
	// Note, this one is created in the service mesh namespace ("istio-system"), not the test namespace
	route := &routev1.Route{
		ObjectMeta: meta.ObjectMeta{
			Name:      "hello",
			Namespace: serviceMeshNamespace,
		},
		Spec: routev1.RouteSpec{
			Host: customDomain,
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt(8080),
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "istio-ingressgateway",
			},
		},
	}
	route, err := caCtx.Clients.Route.Routes(serviceMeshNamespace).Create(route)
	if err != nil {
		t.Fatalf("Error creating OpenShift Route: %v", err)
	}

	caCtx.AddToCleanup(func() error {
		t.Logf("Cleaning up OpenShift Route %s", route.GetName())
		return caCtx.Clients.Route.Routes(route.Namespace).Delete(route.Name, &meta.DeleteOptions{})
	})

	// Do a spoofed HTTP request via the OpenShiftRouter
	// Note, here we go via the OpenShift Router IP address, not kourier, as usual with the "spoof" client.
	routerIp := lookupOpenShiftRouterIP(caCtx)
	sc, err := spoof.New(caCtx.Clients.Kube, t.Logf, customDomain, false, routerIp.String(), time.Second, time.Minute)
	if err != nil {
		t.Fatalf("Error creating a Spoofing Client: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "http://"+customDomain, nil)
	if err != nil {
		t.Fatalf("Error creating an HTTP GET request: %v", err)
	}

	// Poll, as it is expected OpenShift Router will return 503s until it reconciles the Route.
	resp, err := sc.Poll(req, pkgTest.IsStatusOK)
	if err != nil {
		t.Fatalf("Error polling custom domain: %v", err)
	}

	const expectedResponse = "Hello OpenShift!"
	if resp.StatusCode != 200 || strings.TrimSpace(string(resp.Body)) != expectedResponse {
		t.Fatalf("Expecting a HTTP 200 response with %q, got %d: %s", expectedResponse, resp.StatusCode, string(resp.Body))
	}
}

// newSpoofClientWithTls returns a Spoof client that always connects to the given IP address with 'customDomain' as SNI header
func newSpoofClientWithTls(ctx *test.Context, customDomain, ip string, certPool *x509.CertPool) (*spoof.SpoofingClient, error) {
	return spoof.New(ctx.Clients.Kube, ctx.T.Logf, customDomain, false, ip, time.Second, time.Minute, func(transport *http.Transport) *http.Transport {
		// Custom DialTLSContext to specify the ingress IP address, our certPool and the SNI header for the custom domain
		transport.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// We ignore the request address, force the given <IP>:443
			conn, err := net.Dial("tcp", ip+":443")
			if err != nil {
				return nil, err
			}

			tlsConfig := &tls.Config{
				RootCAs:    certPool,
				ServerName: customDomain,
			}

			c := tls.Client(conn, tlsConfig)
			err = c.Handshake()
			if err != nil {
				_ = c.Close()
				return nil, err
			}

			return c, nil
		}
		return transport
	})
}

func TestKsvcWithServiceMeshCustomTlsDomain(t *testing.T) {

	const customDomain = "custom-ksvc-domain.example.com"
	const caSecretName = "example.com"

	caCtx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, caCtx) })
	defer test.CleanupAll(t, caCtx)

	// Skip test if ServiceMesh not installed
	serviceMeshNamespace := getServiceMeshNamespace(caCtx)
	if serviceMeshNamespace == "" {
		t.Skipf("Test namespace %q not a mesh member, use \"make install-mesh\" for ServiceMesh setup", serviceMeshTestNamespaceName)
	}

	// Read the CA certificate for "example.com" generated by "make install-mesh"

	// Certificates are generated by:

	// openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -subj '/O=Example Inc./CN=example.com' -keyout example.com.key -out example.com.crt //
	// openssl req -out custom.example.com.csr -newkey rsa:2048 -nodes -keyout custom.example.com.key -subj "/CN=custom-ksvc-domain.example.com/O=Example Inc."
	// openssl x509 -req -days 365 -CA example.com.crt -CAkey example.com.key -set_serial 0 -in custom.example.com.csr -out custom.example.com.crt

	// The script stores the cert in a secret called "example.com":
	exampleSecret, err := caCtx.Clients.Kube.CoreV1().Secrets(serviceMeshNamespace).Get(caSecretName, meta.GetOptions{})
	if errors.IsNotFound(err) {
		t.Skipf("Secret %q in %q doesn't exist. Use \"make install-mesh\" for ServiceMesh setup.", caSecretName, serviceMeshNamespace)
	}
	if err != nil {
		t.Fatalf("Error reading Secret %s in %s: %v", caSecretName, serviceMeshNamespace, err)
	}

	// Extract the certificate from the secret and create a CertPool
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(exampleSecret.Data["tls.crt"])

	// Deploy a cluster-local ksvc "hello"
	ksvc := test.Service("hello", serviceMeshTestNamespaceName, "openshift/hello-openshift", nil)
	ksvc.ObjectMeta.Labels = map[string]string{
		serving.VisibilityLabelKey: serving.VisibilityClusterLocal,
	}
	ksvc = withServiceReadyOrFail(caCtx, ksvc)

	// Create the Istio Gateway for traffic via istio-ingressgateway
	// The secret and its mounts are specified in the example SMCP in hack/lib/mesh.bash
	//
	//       istio-ingressgateway:
	//        secretVolumes:
	//        - mountPath: /custom.example.com
	//          name: custom-example-com
	//          secretName: custom.example.com
	//
	defaultGateway := test.IstioGatewayWithTls("default-gateway",
		serviceMeshTestNamespaceName,
		customDomain,
		"/custom.example.com/tls.key",
		"/custom.example.com/tls.crt",
	)
	defaultGateway = test.CreateIstioGateway(caCtx, defaultGateway)

	// Create the Istio VirtualService to rewrite the host header of a custom domain with the ksvc's svc hostname
	virtualService := test.IstioVirtualServiceForKnativeServiceWithCustomDomain(ksvc, defaultGateway.GetName(), customDomain)
	virtualService = test.CreateIstioVirtualService(caCtx, virtualService)

	// Create the Istio ServiceEntry for ksvc's svc hostname routing towards the knative kourier-internal gateway
	serviceEntry := test.IstioServiceEntryForKnativeServiceTowardsKourier(ksvc)
	serviceEntry = test.CreateIstioServiceEntry(caCtx, serviceEntry)

	// Create the OpenShift Route for the custom domain pointing to the istio-ingressgateway
	// Note, this one is created in the service mesh namespace ("istio-system"), not the test namespace
	route := &routev1.Route{
		ObjectMeta: meta.ObjectMeta{
			Name:      "hello",
			Namespace: serviceMeshNamespace,
		},
		Spec: routev1.RouteSpec{
			Host: customDomain,
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt(8443),
			},
			TLS: &routev1.TLSConfig{
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyNone,
				Termination:                   routev1.TLSTerminationPassthrough,
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "istio-ingressgateway",
			},
		},
	}
	route, err = caCtx.Clients.Route.Routes(serviceMeshNamespace).Create(route)
	if err != nil {
		t.Fatalf("Error creating OpenShift Route: %v", err)
	}

	caCtx.AddToCleanup(func() error {
		t.Logf("Cleaning up OpenShift Route %s", route.GetName())
		return caCtx.Clients.Route.Routes(route.Namespace).Delete(route.Name, &meta.DeleteOptions{})
	})

	// Do a spoofed HTTP request.
	// Note, here we go via the OpenShift Router IP address, not kourier as usual with the "spoof" client.
	routerIp := lookupOpenShiftRouterIP(caCtx)
	sc, err := newSpoofClientWithTls(caCtx, customDomain, routerIp.String(), certPool)
	if err != nil {
		t.Fatalf("Error creating a Spoofing Client: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, "https://"+customDomain, nil)
	if err != nil {
		t.Fatalf("Error creating an HTTPS GET request: %v", err)
	}

	// Poll, as it is expected OpenShift Router will return 503s until it reconciles the Route.
	resp, err := sc.Poll(req, pkgTest.IsStatusOK)
	if err != nil {
		t.Fatalf("Error polling custom domain: %v", err)
	}

	const expectedResponse = "Hello OpenShift!"
	if resp.StatusCode != 200 || strings.TrimSpace(string(resp.Body)) != expectedResponse {
		t.Fatalf("Expecting an HTTP 200 response with %q, got %d: %s", expectedResponse, resp.StatusCode, string(resp.Body))
	}

	// Verify we cannot connect via plain HTTP (as the Route has InsecureEdgeTerminationPolicyNone)
	// In this case we expect a 503 response from the OpenShift Router.

	// As the router already returned an OK response for the HTTPS request, we assume the route is already
	// reconciled and its 503 response really means it won't serve insecure HTTP ever.
	req, err = http.NewRequest(http.MethodGet, "http://"+customDomain, nil)
	if err != nil {
		t.Fatalf("Error creating an HTTP GET request: %v", err)
	}
	resp, err = sc.Do(req)
	if err != nil {
		t.Fatalf("Error doing HTTP request: %v", err)
	}

	if resp.StatusCode != 503 {
		t.Fatalf("Expecting an HTTP 503 response for an insecure HTTP request, got %d: %s", resp.StatusCode, string(resp.Body))
	}
}
