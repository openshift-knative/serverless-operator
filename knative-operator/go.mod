module github.com/openshift-knative/serverless-operator/knative-operator

go 1.14

require (
	github.com/appscode/jsonpatch v0.0.0-20190108182946-7c0e3b262f30
	github.com/coreos/prometheus-operator v0.29.0
	github.com/go-logr/logr v0.1.0
	github.com/google/go-cmp v0.5.2
	github.com/gophercloud/gophercloud v0.6.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/manifestival/controller-runtime-client v0.2.0-0.1.12
	github.com/manifestival/manifestival v0.6.1
	github.com/openshift/api v0.0.0-20190927182313-d4a64ec2cbd8
	github.com/openzipkin/zipkin-go v0.2.5 // indirect
	github.com/operator-framework/operator-sdk v0.10.1
	github.com/prometheus/client_golang v1.6.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.18.7-rc.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/operator v0.17.2
	knative.dev/pkg v0.0.0-20200831162708-14fb2347fb77
	knative.dev/serving v0.15.3
	sigs.k8s.io/controller-runtime v0.6.1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// Because the bitbucket version is broken
	bitbucket.org/ww/goautoneg => github.com/adjust/goautoneg v0.0.0-20150426214442-d788f35a0315

	// Kubernetes v1.13.4
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4

	// controller-runtime
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.1.12
)
