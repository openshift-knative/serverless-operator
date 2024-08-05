package servicemesh

import (
	"knative.dev/pkg/test/spoof"
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
	usePrivateService bool                  // Whether http proxy should call target's service private service
	checkResponseFunc spoof.ResponseChecker // Function to be used to check response
}
