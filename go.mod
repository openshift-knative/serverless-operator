module github.com/openshift-knative/serverless-operator

go 1.14

require (
	github.com/coreos/prometheus-operator v0.38.1-0.20200424145508-7e176fda06cc
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/google/go-cmp v0.5.2
	github.com/manifestival/controller-runtime-client v0.3.0
	github.com/manifestival/manifestival v0.6.1
	github.com/openshift/api v0.0.0-20200930075302-db52bc4ef99f
	github.com/openshift/client-go v0.0.0-20200929181438-91d71ef2122c
	github.com/operator-framework/api v0.3.16
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20201214005626-fe32277830b3
	github.com/operator-framework/operator-sdk v0.19.4
	github.com/prometheus-operator/prometheus-operator v0.43.0
	github.com/prometheus/client_golang v1.8.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.16.0
	google.golang.org/genproto v0.0.0-20200914193844-75d14daec038 // indirect
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/eventing v0.19.3
	knative.dev/eventing-kafka v0.19.3-0.20201209141141-d86b0a2b533b
	knative.dev/hack v0.0.0-20201103151104-3d5abc3a0075
	knative.dev/networking v0.0.0-20201103163404-b9f80f4537af
	knative.dev/operator v0.19.2-0.20201214144543-1930abb4fcc7
	knative.dev/pkg v0.0.0-20201103163404-5514ab0c1fdf
	knative.dev/serving v0.19.0
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm
	github.com/go-logr/zapr => github.com/go-logr/zapr v0.2.0 // To make klog happy

	// Kubernetes v1.18.8
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/code-generator => k8s.io/code-generator v0.18.8
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
)
