module github.com/openshift-knative/serverless-operator/openshift-knative-operator

go 1.14

require (
	contrib.go.opencensus.io/exporter/ocagent v0.7.1-0.20200907061046-05415f1de66d // indirect
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.14.8 // indirect
	k8s.io/api v0.18.8 // indirect
	k8s.io/code-generator v0.18.8 // indirect
	knative.dev/operator v0.17.2
	knative.dev/pkg v0.0.0-20200831162708-14fb2347fb77
	knative.dev/test-infra v0.0.0-20200915193842-f4d4232c1f04
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309

	github.com/go-logr/logr => github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr => github.com/go-logr/zapr v0.1.1

	k8s.io/api => k8s.io/api v0.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/code-generator => k8s.io/code-generator v0.17.6
)
