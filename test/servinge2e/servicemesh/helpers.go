package servicemesh

import (
	"github.com/openshift-knative/serverless-operator/test"
	corev1 "k8s.io/api/core/v1"
	pkgTest "knative.dev/pkg/test"
	"knative.dev/pkg/test/spoof"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	IstioInjectKey              = "sidecar.istio.io/inject"
	IstioRewriteProbersKey      = "sidecar.istio.io/rewriteAppHTTPProbers"
	ServingEnablePassthroughKey = "serving.knative.openshift.io/enablePassthrough"
	helloWorldText              = "Hello World!"
)

type testCase struct {
	name              string
	labels            map[string]string     // Ksvc Labels
	annotations       map[string]string     // Revision template Annotations
	sourceNamespace   string                // Namespace for the source service (http proxy)
	targetNamespace   string                // Namespace for the target service
	gateway           string                // Value for gateway that's called by http proxy
	targetHost        string                // Value for host that's called by http proxy
	usePrivateService bool                  // Whether http proxy should call target's service private service
	checkResponseFunc spoof.ResponseChecker // Function to be used to check response
}

// HttpProxyService returns a knative service acting as "http proxy", redirects requests towards a given "host". Used to test cluster-local services
func HttpProxyService(name, namespace, gateway, target string, serviceAnnotations, templateAnnotations map[string]string) *servingv1.Service {
	proxy := test.Service(name, namespace, pkgTest.ImagePath(test.HTTPProxyImg), serviceAnnotations, templateAnnotations)
	if gateway != "" {
		proxy.Spec.Template.Spec.Containers[0].Env = append(proxy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  "GATEWAY_HOST",
			Value: gateway,
		})
	}
	proxy.Spec.Template.Spec.Containers[0].Env = append(proxy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "TARGET_HOST",
		Value: target,
	})

	return proxy
}
