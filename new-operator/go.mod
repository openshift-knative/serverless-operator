module github.com/openshift-knative/serverless-operator/new-operator

go 1.14

require (
	github.com/manifestival/manifestival v0.6.1
	github.com/openshift/api v0.0.0-20200930075302-db52bc4ef99f
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/eventing v0.14.1-0.20200428210242-f355830c4d70 // indirect
	knative.dev/operator v0.17.1-0.20200925150344-ce82f1b08943
	knative.dev/pkg v0.0.0-20201002052829-735a38c03260
	knative.dev/test-infra v0.0.0-20201001200229-a6988e3b3b38
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309

	github.com/go-logr/logr => github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr => github.com/go-logr/zapr v0.1.1

	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/code-generator => k8s.io/code-generator v0.18.8
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.0.0
)
