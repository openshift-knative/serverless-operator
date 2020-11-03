module github.com/openshift-knative/serverless-operator/knative-operator

go 1.14

require (
	github.com/coreos/prometheus-operator v0.38.1-0.20200424145508-7e176fda06cc
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/google/go-cmp v0.5.2
	github.com/manifestival/controller-runtime-client v0.3.0
	github.com/manifestival/manifestival v0.6.1
	github.com/openshift/api v0.0.0-20200930075302-db52bc4ef99f
	github.com/openzipkin/zipkin-go v0.2.5 // indirect
	github.com/operator-framework/operator-sdk v0.19.4
	github.com/prometheus/client_golang v1.6.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.15.0
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/operator v0.18.1-0.20201023205637-2c391292de60
	knative.dev/pkg v0.0.0-20201023161837-a45780482d2d
	knative.dev/serving v0.18.1
	knative.dev/test-infra v0.0.0-20200921012245-37f1a12adbd3
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// Kubernetes v1.18.8
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
)
