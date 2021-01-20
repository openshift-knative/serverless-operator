package servinge2e

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
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

	"k8s.io/apimachinery/pkg/api/errors"
	"knative.dev/serving/pkg/apis/autoscaling"

	"github.com/openshift-knative/serverless-operator/test"
	routev1 "github.com/openshift/api/route/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	network "knative.dev/networking/pkg"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/helpers"
	"knative.dev/pkg/test/spoof"
	"knative.dev/serving/pkg/apis/serving"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

type testCase struct {
	name               string
	labels             map[string]string // Ksvc labels
	annotations        map[string]string // Revision template annotations
	expectIstioSidecar bool              // Whether it is expected for the istio-proxy sidecar to be injected into the pod
}

const (
	serviceMeshTestNamespaceName = "serverless-tests-mesh"
	helloworldImage              = "gcr.io/knative-samples/helloworld-go"
	httpProxyImage               = "registry.svc.ci.openshift.org/openshift/knative-v0.17.3:knative-serving-test-httpproxy"
	istioInjectKey               = "sidecar.istio.io/inject"
)

func isServiceMeshInstalled(ctx *test.Context) bool {
	_, err := ctx.Clients.Dynamic.Resource(schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}).Get(context.Background(), "servicemeshcontrolplanes.maistra.io", meta.GetOptions{})

	if err == nil {
		return true
	}

	if !errors.IsNotFound(err) {
		ctx.T.Fatalf("Error checking if servicemeshcontrolplanes.maistra.io CRD exists: %v", err)
	}

	return false
}

func setupCustomDomainTLSSecret(ctx *test.Context, serviceMeshNamespace, customSecretName, customDomain string) *x509.CertPool {
	// Generate example.com CA
	caCertificate := &x509.Certificate{
		SerialNumber: big.NewInt(0),
		Subject: pkix.Name{
			Organization: []string{"Example Inc."},
			CommonName:   "example.com",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		ctx.T.Fatalf("Error generating CA RSA key: %v", err)
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, caCertificate, caCertificate, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		ctx.T.Fatalf("Error self-signing CA Certificate: %v", err)
	}

	caPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	customCertificate := &x509.Certificate{
		SerialNumber: big.NewInt(0),
		Subject: pkix.Name{
			Organization: []string{"Example Inc."},
		},
		DNSNames:     []string{customDomain},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{42},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	customPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		ctx.T.Fatalf("Error generating Custom RSA key: %v", err)
	}
	customCertificateBytes, err := x509.CreateCertificate(rand.Reader, customCertificate, caCertificate, &customPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		ctx.T.Fatalf("Error signing Custom Certificate by CA: %v", err)
	}
	customPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: customCertificateBytes,
	})
	customPrivateKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(customPrivateKey),
	})

	customSecret := &core.Secret{
		ObjectMeta: meta.ObjectMeta{
			Name:      customSecretName,
			Namespace: serviceMeshNamespace,
		},
		Type: core.SecretTypeTLS,
		Data: map[string][]byte{
			core.TLSCertKey:       customPem,
			core.TLSPrivateKeyKey: customPrivateKeyPem,
		},
	}

	customSecret, err = ctx.Clients.Kube.CoreV1().Secrets(customSecret.Namespace).Create(context.Background(), customSecret, meta.CreateOptions{})
	if err != nil {
		ctx.T.Fatalf("Error creating Secret %q: %v", customSecretName, err)
	}
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up Secret %s", customSecret.GetName())
		return ctx.Clients.Kube.CoreV1().Secrets(customSecret.Namespace).Delete(context.Background(), customSecret.Name, meta.DeleteOptions{})
	})

	// Return a CertPool with our example CA
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caPem)

	return certPool
}

// Following https://docs.openshift.com/container-platform/4.6/serverless/networking/serverless-ossm.html
func setupNamespaceForServiceMesh(ctx *test.Context, serviceMeshNamespace, testNamespace string) {
	test.CreateServiceMeshMemberRollV1(ctx, test.ServiceMeshMemberRollV1("default", serviceMeshNamespace, testNamespace))

	test.CreateNetworkPolicy(ctx, test.AllowFromServingSystemNamespaceNetworkPolicy(testNamespace))
	test.LabelNamespace(ctx, "knative-serving", test.KnativeSystemNamespaceKey, "true")
	test.LabelNamespace(ctx, "knative-serving-ingress", test.KnativeSystemNamespaceKey, "true")
}

func runCustomDomainTLSTestForAllServiceMeshVersions(t *testing.T, customSecretName, secretVolumeName, secretVolumeMountPath string, testFunc func(ctx *test.Context)) {
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
				if customSecretName != "" {
					test.AddServiceMeshControlPlaneV1IngressGatewaySecretVolume(smcp, secretVolumeName, customSecretName, secretVolumeMountPath)
				}
				test.CreateServiceMeshControlPlaneV1(ctx, smcp)
			},
		},
		{
			name: "v2",
			smcpCreationFunc: func(ctx *test.Context) {
				smcp := test.ServiceMeshControlPlaneV2(smcpName, serviceMeshTestNamespaceName)
				if customSecretName != "" {
					test.AddServiceMeshControlPlaneV2IngressGatewaySecretVolume(smcp, secretVolumeName, customSecretName, secretVolumeMountPath)
				}
				test.CreateServiceMeshControlPlaneV2(ctx, smcp)
			},
		},
	}

	for _, version := range versions {
		t.Run(version.name, func(t *testing.T) {
			ctx := test.SetupClusterAdmin(t)
			test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, ctx) })
			defer test.CleanupAll(t, ctx)

			if !isServiceMeshInstalled(ctx) {
				t.Skip("ServiceMeshControlPlane CRD not found, use \"make install-mesh\" to install ServiceMesh")
			}

			// Create SMCP (version-specific)
			version.smcpCreationFunc(ctx)

			// Follow documented steps to add a namespace to ServiceMesh (including NetworkPolicy setup and namespace labels)
			setupNamespaceForServiceMesh(ctx, serviceMeshTestNamespaceName, testNamespace)

			test.WaitForServiceMeshControlPlaneReady(ctx, smcpName, serviceMeshTestNamespaceName)

			// Run actual tests
			testFunc(ctx)
		})
	}
}

func runTestForAllServiceMeshVersions(t *testing.T, testFunc func(ctx *test.Context)) {
	runCustomDomainTLSTestForAllServiceMeshVersions(t, "", "", "", testFunc)
}

// A knative service acting as an "http proxy", redirects requests towards a given "host". Used to test cluster-local services
func httpProxyService(name, namespace, host string) *servingv1.Service {
	proxy := test.Service(name, namespace, httpProxyImage, nil)
	proxy.Spec.Template.Spec.Containers[0].Env = append(proxy.Spec.Template.Spec.Containers[0].Env, core.EnvVar{
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
				network.VisibilityLabelKey: serving.VisibilityClusterLocal,
			},
			annotations: map[string]string{
				istioInjectKey: "true",
			},
			expectIstioSidecar: true,
		}, {
			// A cluster-local variant of the "sidecar-without-activator" scenario
			name: "local-sidecar-without-activator",
			labels: map[string]string{
				network.VisibilityLabelKey: serving.VisibilityClusterLocal,
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
				testServiceToService(t, ctx, testNamespace, scenario)
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
func jwtHTTPGetRequestBytes(url string, token *string) (*http.Response, []byte, error) {
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
		jwksKsvc := test.Service(helpers.AppendRandomString("jwks"), testNamespace, "openshift/hello-openshift", nil)
		jwksKsvc.Spec.Template.Spec.Containers[0].Env = append(jwksKsvc.Spec.Template.Spec.Containers[0].Env, core.EnvVar{
			Name:  "RESPONSE",
			Value: jwks,
		})
		jwksKsvc.ObjectMeta.Labels = map[string]string{
			network.VisibilityLabelKey: serving.VisibilityClusterLocal,
		}
		jwksKsvc = withServiceReadyOrFail(ctx, jwksKsvc)

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
						"namespace": testNamespace,
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
						"namespace": testNamespace,
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
						"namespace": testNamespace,
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
						"namespace": testNamespace,
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
		testKsvc := test.Service("jwt-test", testNamespace, image, map[string]string{
			"sidecar.istio.io/inject": "true",
		})
		testKsvc = withServiceReadyOrFail(ctx, testKsvc)

		// Wait until the Route is ready and also verify the route returns a 401 or 403 without a token
		if _, err := pkgTest.WaitForEndpointState(
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
			true); err != nil {
			t.Fatalf("The Route at domain %s didn't serve the expected HTTP 401 or 403 status: %v", testKsvc.Status.URL.URL(), err)
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
				resp, body, err := jwtHTTPGetRequestBytes(testKsvc.Status.URL.String(), tokenRef)
				if err != nil {
					t.Fatalf("Error doing HTTP GET request: %v", err)
				}

				if scenario.valid {
					// Verify the response is a proper "hello world" when the token is valid
					if resp.StatusCode != 200 || !strings.Contains(string(body), helloworldText) {
						t.Fatalf("Unexpected response with a valid token: HTTP %d: %s", resp.StatusCode, string(body))
					}
				} else {
					// Verify the response is a 401 or a 403 for an invalid token
					if resp.StatusCode != 401 && resp.StatusCode != 403 {
						t.Fatalf("Unexpected response with an invalid token, expecting 401 or 403, got %d: %s", resp.StatusCode, string(body))
					}
				}
			})
		}
	})
}

func lookupOpenShiftRouterIP(ctx *test.Context) net.IP {
	// Deploy an auxiliary ksvc accessible via an OpenShift route, so that we have a route hostname that we can resolve
	aux := test.Service("aux", testNamespace, "openshift/hello-openshift", nil)
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

	runTestForAllServiceMeshVersions(t, func(ctx *test.Context) {
		t := ctx.T

		// Deploy a cluster-local ksvc "hello"
		ksvc := test.Service("hello", testNamespace, "openshift/hello-openshift", nil)
		ksvc.ObjectMeta.Labels = map[string]string{
			network.VisibilityLabelKey: serving.VisibilityClusterLocal,
		}
		ksvc = withServiceReadyOrFail(ctx, ksvc)

		// Create the Istio Gateway for traffic via istio-ingressgateway
		defaultGateway := test.IstioGateway("default-gateway", testNamespace)
		test.CreateIstioGateway(ctx, defaultGateway)

		// Create the Istio VirtualService to rewrite the host header of a custom domain with the ksvc's svc hostname
		virtualService := test.IstioVirtualServiceForKnativeServiceWithCustomDomain(ksvc, defaultGateway.GetName(), customDomain)
		test.CreateIstioVirtualService(ctx, virtualService)

		// Create the Istio ServiceEntry for ksvc's svc hostname routing towards the knative kourier-internal gateway
		serviceEntry := test.IstioServiceEntryForKnativeServiceTowardsKourier(ksvc)
		test.CreateIstioServiceEntry(ctx, serviceEntry)

		// Create the OpenShift Route for the custom domain pointing to the istio-ingressgateway
		// Note, this one is created in the service mesh namespace ("istio-system"), not the test namespace
		route := &routev1.Route{
			ObjectMeta: meta.ObjectMeta{
				Name:      "hello",
				Namespace: serviceMeshTestNamespaceName,
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
		route, err := ctx.Clients.Route.Routes(serviceMeshTestNamespaceName).Create(context.Background(), route, meta.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating OpenShift Route: %v", err)
		}

		ctx.AddToCleanup(func() error {
			t.Logf("Cleaning up OpenShift Route %s", route.GetName())
			return ctx.Clients.Route.Routes(route.Namespace).Delete(context.Background(), route.Name, meta.DeleteOptions{})
		})

		// Do a spoofed HTTP request via the OpenShiftRouter
		// Note, here we go via the OpenShift Router IP address, not kourier, as usual with the "spoof" client.
		routerIP := lookupOpenShiftRouterIP(ctx)
		sc, err := newSpoofClientWithTLS(ctx, customDomain, routerIP.String(), nil)
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
	})
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

func TestKsvcWithServiceMeshCustomTlsDomain(t *testing.T) {

	const customDomain = "custom-ksvc-domain.example.com"
	const customSecretName = "custom.example.com"
	const secretVolumeName = "custom-example-com"
	const secretVolumeMountPath = "/custom.example.com"

	ctx := test.SetupClusterAdmin(t)
	test.CleanupOnInterrupt(t, func() { test.CleanupAll(t, ctx) })
	defer test.CleanupAll(t, ctx)

	certPool := setupCustomDomainTLSSecret(ctx, serviceMeshTestNamespaceName, customSecretName, customDomain)

	runCustomDomainTLSTestForAllServiceMeshVersions(t, customSecretName, secretVolumeName, secretVolumeMountPath, func(ctx *test.Context) {
		t := ctx.T

		// Deploy a cluster-local ksvc "hello"
		ksvc := test.Service("hello", testNamespace, "openshift/hello-openshift", nil)
		ksvc.ObjectMeta.Labels = map[string]string{
			network.VisibilityLabelKey: serving.VisibilityClusterLocal,
		}
		ksvc = withServiceReadyOrFail(ctx, ksvc)

		// Create the Istio Gateway for traffic via istio-ingressgateway
		// The secret and its mounts are specified in the example SMCP in hack/lib/mesh.bash
		//
		//       istio-ingressgateway:
		//        secretVolumes:
		//        - mountPath: /custom.example.com
		//          name: custom-example-com
		//          secretName: custom.example.com
		//
		defaultGateway := test.IstioGatewayWithTLS("default-gateway",
			testNamespace,
			customDomain,
			secretVolumeMountPath+"/tls.key",
			secretVolumeMountPath+"/tls.crt",
		)
		defaultGateway = test.CreateIstioGateway(ctx, defaultGateway)

		// Create the Istio VirtualService to rewrite the host header of a custom domain with the ksvc's svc hostname
		virtualService := test.IstioVirtualServiceForKnativeServiceWithCustomDomain(ksvc, defaultGateway.GetName(), customDomain)
		test.CreateIstioVirtualService(ctx, virtualService)

		// Create the Istio ServiceEntry for ksvc's svc hostname routing towards the knative kourier-internal gateway
		serviceEntry := test.IstioServiceEntryForKnativeServiceTowardsKourier(ksvc)
		test.CreateIstioServiceEntry(ctx, serviceEntry)

		// Create the OpenShift Route for the custom domain pointing to the istio-ingressgateway
		// Note, this one is created in the service mesh namespace ("istio-system"), not the test namespace
		route := &routev1.Route{
			ObjectMeta: meta.ObjectMeta{
				Name:      "hello",
				Namespace: serviceMeshTestNamespaceName,
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
		route, err := ctx.Clients.Route.Routes(serviceMeshTestNamespaceName).Create(context.Background(), route, meta.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating OpenShift Route: %v", err)
		}

		ctx.AddToCleanup(func() error {
			t.Logf("Cleaning up OpenShift Route %s", route.GetName())
			return ctx.Clients.Route.Routes(route.Namespace).Delete(context.Background(), route.Name, meta.DeleteOptions{})
		})

		// Do a spoofed HTTP request.
		// Note, here we go via the OpenShift Router IP address, not kourier as usual with the "spoof" client.
		routerIP := lookupOpenShiftRouterIP(ctx)
		sc, err := newSpoofClientWithTLS(ctx, customDomain, routerIP.String(), certPool)
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
	})
}
