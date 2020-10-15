module github.com/openshift-knative/serverless-operator/serving/ingress

go 1.14

require (
	github.com/google/go-cmp v0.5.2
	github.com/openshift/api v0.0.0-20200917102736-0a191b5b9bb0
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.1
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.19.0 // indirect
	k8s.io/klog/v2 v2.2.0 // indirect
	knative.dev/networking v0.0.0-20200831172815-5f2e0ad6215f
	knative.dev/pkg v0.0.0-20200831162708-14fb2347fb77
	knative.dev/serving v0.17.3
	knative.dev/test-infra v0.0.0-20200915193842-f4d4232c1f04
)

replace (
	k8s.io/api => k8s.io/api v0.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/code-generator => k8s.io/code-generator v0.17.6
)
