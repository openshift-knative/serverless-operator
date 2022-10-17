package kourier

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
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"knative.dev/serving/pkg/apis/autoscaling"

	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/networking/pkg/apis/networking"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/helpers"
	"knative.dev/pkg/test/spoof"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingTest "knative.dev/serving/test"
)

type testCase struct {
	name               string
	labels             map[string]string // Ksvc labels
	annotations        map[string]string // Revision template annotations
	expectIstioSidecar bool              // Whether it is expected for the istio-proxy sidecar to be injected into the pod
}

const (
	serviceMeshTestNamespaceName = "serverless-tests-mesh"
	httpProxyImage               = "registry.ci.openshift.org/openshift/knative-v0.17.3:knative-serving-test-httpproxy"
	istioInjectKey               = "sidecar.istio.io/inject"
)

// Following https://docs.openshift.com/container-platform/4.9/serverless/admin_guide/serverless-ossm-setup.html
func setupNamespaceForServiceMesh(ctx *test.Context, serviceMeshNamespace, testNamespace string) {
	test.CreateServiceMeshMemberRollV1(ctx, test.ServiceMeshMemberRollV1("default", serviceMeshNamespace, testNamespace))

	test.CreateNetworkPolicy(ctx, test.AllowFromServingSystemNamespaceNetworkPolicy(testNamespace))
}

func runTestForAllServiceMeshVersions(t *testing.T, testFunc func(ctx *test.Context)) {
	const smcpName = "basic"

	type serviceMeshVersion struct {
		name             string
		smcpCreationFunc func(ctx *test.Context)
	}

	versions := []serviceMeshVersion{
		{
			name: "v1",
			smcpCreationFunc: func(ctx *test.Context) {
				smcp := test.ServiceMeshControlPlaneV1(smcpName, serviceMeshTestNamespaceName)
				test.CreateServiceMeshControlPlaneV1(ctx, smcp)
			},
		},
		{
			name: "v2",
			smcpCreationFunc: func(ctx *test.Context) {
				smcp := test.ServiceMeshControlPlaneV2(smcpName, serviceMeshTestNamespaceName)
				test.CreateServiceMeshControlPlaneV2(ctx, smcp)
			},
		},
	}

	for _, version := range versions {
		t.Run(version.name, func(t *testing.T) {
			ctx := test.SetupClusterAdmin(t)
			test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, ctx) })
			defer test.CleanupAll(t, ctx)

			if !test.IsServiceMeshInstalled(ctx) {
				t.Skip("ServiceMeshControlPlane CRD not found, use \"make install-mesh\" to install ServiceMesh")
			}

			// Create SMCP (version-specific)
			version.smcpCreationFunc(ctx)

			// Follow documented steps to add a namespace to ServiceMesh (including NetworkPolicy setup and namespace labels)
			setupNamespaceForServiceMesh(ctx, serviceMeshTestNamespaceName, test.Namespace)

			test.WaitForServiceMeshControlPlaneReady(ctx, smcpName, serviceMeshTestNamespaceName)

			// Run actual tests
			testFunc(ctx)
		})
	}
}

// A knative service acting as an "http proxy", redirects requests towards a given "host". Used to test cluster-local services
func httpProxyService(name, namespace, host string) *servingv1.Service {
	proxy := test.Service(name, namespace, httpProxyImage, nil)
	proxy.Spec.Template.Spec.Containers[0].Env = append(proxy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "TARGET_HOST",
		Value: host,
	})

	return proxy
}

// Skipped unless ServiceMesh has been installed via "make install-mesh"
func TestKsvcWithServiceMeshSidecar(t *testing.T) {
	runTestForAllServiceMeshVersions(t, func(ctx *test.Context) {
		tests := []testCase{{
			// Requests go via gateway -> activator -> pod , by default
			// Verifies the activator can connect to the pod
			name: "sidecar-via-activator",
			annotations: map[string]string{
				istioInjectKey:                     "true",
				autoscaling.TargetBurstCapacityKey: "-1",
			},
			expectIstioSidecar: true,
		}, {
			// Requests go via gateway -> pod ( activator should be skipped if burst capacity is disabled and there is at least 1 replica)
			// Verifies the gateway can connect to the pod directly
			name: "sidecar-without-activator",
			annotations: map[string]string{
				istioInjectKey:                     "true",
				autoscaling.TargetBurstCapacityKey: "0",
				autoscaling.MinScaleAnnotationKey:  "1",
			},
			expectIstioSidecar: true,
		}, {
			// Verifies the "sidecar.istio.io/inject" annotation is really what decides the istio-proxy presence
			name: "no-sidecar",
			annotations: map[string]string{
				istioInjectKey: "false",
			},
			expectIstioSidecar: false,
		}, {
			// A cluster-local variant of the "sidecar-via-activator" scenario
			name: "local-sidecar-via-activator",
			labels: map[string]string{
				networking.VisibilityLabelKey: serving.VisibilityClusterLocal,
			},
			annotations: map[string]string{
				istioInjectKey: "true",
			},
			expectIstioSidecar: true,
		}, {
			// A cluster-local variant of the "sidecar-without-activator" scenario
			name: "local-sidecar-without-activator",
			labels: map[string]string{
				networking.VisibilityLabelKey: serving.VisibilityClusterLocal,
			},
			annotations: map[string]string{
				istioInjectKey:                     "true",
				autoscaling.TargetBurstCapacityKey: "0",
				autoscaling.MinScaleAnnotationKey:  "1",
			},
			expectIstioSidecar: true,
		}}

		t := ctx.T
		for _, scenario := range tests {
			scenario := scenario
			t.Run(scenario.name, func(t *testing.T) {
				testServiceToService(t, ctx, test.Namespace, scenario)
			})
		}
	})
}

// formats an RSA public key as JWKS
func rsaPublicKeyAsJwks(key rsa.PublicKey, keyID string) (string, error) {
	eString := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
	nString := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	keyIDString := base64.RawURLEncoding.EncodeToString([]byte(keyID))

	// Generate JWKS
	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"e":   eString,
				"n":   nString,
				"kty": "RSA",
				"kid": keyIDString,
			},
		},
	}

	jwksBytes, err := json.Marshal(jwks)
	if err != nil {
		return "", fmt.Errorf("error marshalling jwks: %w", err)
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
		return "", fmt.Errorf("error marshalling jwt header into JSON: %w", err)
	}

	jwtPayloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling jwt payload into JSON: %w", err)
	}

	jwtHeaderBase64 := base64.RawURLEncoding.EncodeToString(jwtHeaderBytes)
	jwtPayloadBase64 := base64.RawURLEncoding.EncodeToString(jwtPayloadBytes)

	jwtSigningInput := jwtHeaderBase64 + "." + jwtPayloadBase64
	hashFunc := crypto.SHA256.New()
	hashFunc.Write([]byte(jwtSigningInput))
	hash := hashFunc.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hash)
	if err != nil {
		return "", fmt.Errorf("error signing JWT token with PKSC1v15: %w", err)
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
		return "", fmt.Errorf("error marshalling jwt header into JSON: %w", err)
	}

	jwtPayloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshalling jwt payload into JSON: %w", err)
	}

	jwtHeaderBase64 := base64.RawURLEncoding.EncodeToString(jwtHeaderBytes)
	jwtPayloadBase64 := base64.RawURLEncoding.EncodeToString(jwtPayloadBytes)

	jwtToken := jwtHeaderBase64 + "." + jwtPayloadBase64 + "."

	return jwtToken, nil
}

// Convenience method to test requests with tokens, reads the response and returns a closed response and the body bits
// token can be nil, in which case no Authorization header will be sent
func jwtHTTPGetRequestBytes(ctx *test.Context, url *url.URL, token *string) (*spoof.Response, error) {
	tlsConfig := servingTest.TLSClientConfig(context.Background(), ctx.T.Logf, &servingTest.Clients{KubeClient: ctx.Clients.Kube})
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	sc := &spoof.SpoofingClient{
		Client: &http.Client{Transport: transport},
		Logf:   ctx.T.Logf,
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP GET request: %w", err)
	}
	if token != nil {
		req.Header.Add("Authorization", "Bearer "+*token)
	}

	// Poll, as it is expected OpenShift Router will return 503s until it reconciles the Route.
	resp, err := sc.Poll(req, spoof.IsOneOfStatusCodes(http.StatusOK, http.StatusUnauthorized, http.StatusForbidden))
	if err != nil {
		return nil, fmt.Errorf("error polling: %w", err)
	}

	return resp, err
}

// Verifies access to a Ksvc with an istio-proxy can be configured
// via istio authentication Policy to allow valid JWT only.
// Skipped unless ServiceMesh has been installed via "make install-mesh"
func TestKsvcWithServiceMeshJWTDefaultPolicy(t *testing.T) {
	runTestForAllServiceMeshVersions(t, func(ctx *test.Context) {
		t := ctx.T
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Error generating private key: %v", err)
		}

		// print out the public key for debugging purposes
		publicPksvc1, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		if err != nil {
			t.Fatalf("Error marshalling public key: %v", err)
		}
		publicPem := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: publicPksvc1,
		})
		t.Logf("%s", string(publicPem))

		// Used as "kid" in JWKS
		const keyID = "test"
		const issuer = "testing-issuer@secure.serverless.openshift.io"
		const subject = "testing-subject@secure.serverless.openshift.io"
		// For testing an invalid token (with a different issuer)
		const wrongIssuer = "eve@secure.serverless.openshift.io"

		// Generate a new key for a "wrong key" scenario
		wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Error generating private key: %v", err)
		}

		jwks, err := rsaPublicKeyAsJwks(privateKey.PublicKey, keyID)
		if err != nil {
			t.Fatalf("Error encoding RSA public key as JWKS: %v", err)
		}

		// Istio 1.1 and earlier lack "jwks" option, only "jwksUri", so we need to host it on some URL
		// We'll misuse the "hello-openshift" image with the JWKS file defined as the RESPONSE env, and deploy this as a ksvc

		// istio-pilot caches the JWKS content if a new Policy has the same jwksUri as some old policy.
		// Rerunning this test would fail if we kept the jwksUri constant across invocations then,
		// hence the random suffix for the jwks ksvc.
		jwksKsvc := test.Service(helpers.AppendRandomString("jwks"), test.Namespace, pkgTest.ImagePath(test.HelloOpenshiftImg), nil)
		jwksKsvc.Spec.Template.Spec.Containers[0].Env = append(jwksKsvc.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "RESPONSE",
			Value: jwks,
		})
		jwksKsvc.ObjectMeta.Labels = map[string]string{
			networking.VisibilityLabelKey: serving.VisibilityClusterLocal,
		}
		jwksKsvc = test.WithServiceReadyOrFail(ctx, jwksKsvc)

		smcpVersion, _, _ := test.GetServiceMeshControlPlaneVersion(ctx, "basic", serviceMeshTestNamespaceName)
		// If "version" exists and is a v1, use the obsolete "Policy"
		if strings.HasPrefix(smcpVersion, "v1.") {
			// Create a Policy
			authPolicy := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "authentication.istio.io/v1alpha1",
					"kind":       "Policy",
					"metadata": map[string]interface{}{
						"name":      "default",
						"namespace": test.Namespace,
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

			test.CreateUnstructured(ctx, policyGvr, authPolicy)
		} else {
			// On SMCP v.2.x and later, Create RequestAuthentication and AuthorizationPolicies
			jwtExampleRA := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "security.istio.io/v1beta1",
					"kind":       "RequestAuthentication",
					"metadata": map[string]interface{}{
						"name":      "jwt-example",
						"namespace": test.Namespace,
					},
					"spec": map[string]interface{}{
						"jwtRules": []map[string]interface{}{
							{
								"issuer":  issuer,
								"jwksUri": jwksKsvc.Status.URL,
							},
						},
					},
				},
			}

			requestAuthenticationGvr := schema.GroupVersionResource{
				Group:    "security.istio.io",
				Version:  "v1beta1",
				Resource: "requestauthentications",
			}

			test.CreateUnstructured(ctx, requestAuthenticationGvr, jwtExampleRA)

			allowListByPathsAP := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "security.istio.io/v1beta1",
					"kind":       "AuthorizationPolicy",
					"metadata": map[string]interface{}{
						"name":      "allowlist-by-paths",
						"namespace": test.Namespace,
					},
					"spec": map[string]interface{}{
						"action": "ALLOW",
						"rules": []map[string]interface{}{
							{
								"to": []map[string]interface{}{
									{
										"operation": map[string]interface{}{
											"paths": []string{
												"/metrics",
												"/healthz",
											},
										},
									},
								},
							},
						},
					},
				},
			}

			authorizationPolicyGvr := schema.GroupVersionResource{
				Group:    "security.istio.io",
				Version:  "v1beta1",
				Resource: "authorizationpolicies",
			}

			test.CreateUnstructured(ctx, authorizationPolicyGvr, allowListByPathsAP)

			requireJwtAP := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "security.istio.io/v1beta1",
					"kind":       "AuthorizationPolicy",
					"metadata": map[string]interface{}{
						"name":      "require-jwt",
						"namespace": test.Namespace,
					},
					"spec": map[string]interface{}{
						"action": "ALLOW",
						"rules": []map[string]interface{}{
							{
								"from": []map[string]interface{}{
									{
										"source": map[string]interface{}{
											"requestPrincipals": []string{
												issuer + "/" + subject,
											},
										},
									},
								},
							},
						},
					},
				},
			}

			test.CreateUnstructured(ctx, authorizationPolicyGvr, requireJwtAP)
		}

		// Create a test ksvc, should be accessible only via proper JWT token
		testKsvc := test.Service("jwt-test", test.Namespace, pkgTest.ImagePath(image), map[string]string{
			"sidecar.istio.io/inject":                "true",
			"sidecar.istio.io/rewriteAppHTTPProbers": "true",
		})
		testKsvc = test.WithServiceReadyOrFail(ctx, testKsvc)

		// Wait until the Route is ready and also verify the route returns a 401 or 403 without a token
		if _, err := pkgTest.CheckEndpointState(
			context.Background(),
			ctx.Clients.Kube,
			t.Logf,
			testKsvc.Status.URL.URL(),
			func(resp *spoof.Response) (bool, error) {
				if resp.StatusCode != 401 && resp.StatusCode != 403 {
					// Returning (false, nil) causes SpoofingClient.Poll to retry.
					return false, nil
				}
				return true, nil
			},
			"WaitForRouteToServe401Or403",
			true,
			servingTest.AddRootCAtoTransport(context.Background(), t.Logf, &servingTest.Clients{KubeClient: ctx.Clients.Kube}, true),
		); err != nil {
			t.Fatalf("The Route at domain %s didn't serve the expected HTTP 401 or 403 status: %v", testKsvc.Status.URL.URL(), err)
		}

		tests := []struct {
			name    string
			valid   bool // Is the token expected to be valid?
			key     *rsa.PrivateKey
			payload map[string]interface{}
		}{{
			// A valid token
			name:  "valid",
			valid: true,
			key:   privateKey,
			payload: map[string]interface{}{
				"iss": issuer,
				"sub": subject,
				"foo": "bar",
				"iat": time.Now().Unix(),
				"exp": time.Now().Unix() + 3600,
			},
		}, {
			// No token (request will be done without the Authorization header)
			name:    "no_token",
			valid:   false,
			key:     nil,
			payload: nil,
		}, {
			// Unsigned token
			name:  "unsigned",
			valid: false,
			key:   nil,
			payload: map[string]interface{}{
				"iss": issuer,
				"sub": subject,
				"foo": "bar",
				"iat": time.Now().Unix(),
				"exp": time.Now().Unix() + 3600,
			},
		}, {
			// A token with "exp" time in the past
			name:  "expired",
			valid: false,
			key:   privateKey,
			payload: map[string]interface{}{
				"iss": issuer,
				"sub": subject,
				"foo": "bar",
				// as if generated before an hour, expiring 10 minutes ago
				"iat": time.Now().Unix() - 3600,
				"exp": time.Now().Unix() - 600,
			},
		}, {
			// A token signed by a different key
			name:  "bad_key",
			valid: false,
			key:   wrongKey,
			payload: map[string]interface{}{
				"iss": issuer,
				"sub": subject,
				"foo": "bar",
				"iat": time.Now().Unix(),
				"exp": time.Now().Unix() + 3600,
			},
		}, {
			// A token with an issuer set to a different principal than the one specified in the Policy
			name:  "bad_iss",
			valid: false,
			key:   privateKey,
			payload: map[string]interface{}{
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
				resp, err := jwtHTTPGetRequestBytes(ctx, testKsvc.Status.URL.URL(), tokenRef)
				if err != nil {
					t.Fatalf("Error doing HTTP GET request: %v", err)
				}

				if scenario.valid {
					// Verify the response is a proper "hello world" when the token is valid
					if resp.StatusCode != 200 || !strings.Contains(string(resp.Body), helloworldText) {
						t.Fatalf("Unexpected response with a valid token: HTTP %d: %s", resp.StatusCode, string(resp.Body))
					}
				} else {
					// Verify the response is a 401 or a 403 for an invalid token
					if resp.StatusCode != 401 && resp.StatusCode != 403 {
						t.Fatalf("Unexpected response with an invalid token, expecting 401 or 403, got %d: %s", resp.StatusCode, string(resp.Body))
					}
				}
			})
		}
	})
}

func lookupOpenShiftRouterIP(ctx *test.Context) net.IP {
	// Deploy an auxiliary ksvc accessible via an OpenShift route, so that we have a route hostname that we can resolve
	aux := test.Service("aux", test.Namespace, pkgTest.ImagePath(image), nil)
	aux = test.WithServiceReadyOrFail(ctx, aux)

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

// newSpoofClientWithTLS returns a Spoof client that always connects to the given IP address with 'customDomain' as SNI header
func newSpoofClientWithTLS(ctx *test.Context, customDomain, ip string, certPool *x509.CertPool) (*spoof.SpoofingClient, error) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// We ignore the request address, force the given <IP>:80
			return net.Dial("tcp", ip+":80")
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
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
		},
	}

	sc := &spoof.SpoofingClient{
		Client:          &http.Client{Transport: transport},
		RequestInterval: time.Second,
		RequestTimeout:  time.Minute,
		Logf:            ctx.T.Logf,
	}

	return sc, nil
}
