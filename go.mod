module github.com/openshift-knative/serverless-operator

go 1.14

require (
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/google/go-cmp v0.5.4
	github.com/manifestival/controller-runtime-client v0.3.0
	github.com/manifestival/manifestival v0.6.1
	github.com/openshift/api v0.0.0-20210202165416-a9e731090f5e
	github.com/openshift/client-go v0.0.0-20200929181438-91d71ef2122c
	github.com/operator-framework/api v0.3.16
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20210203215341-9dde58210568
	github.com/prometheus-operator/prometheus-operator v0.44.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.44.1
	github.com/prometheus/client_golang v1.8.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.16.0
	google.golang.org/genproto v0.0.0-20200914193844-75d14daec038 // indirect
	k8s.io/api v0.20.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/eventing v0.19.4
	knative.dev/eventing-kafka v0.19.3
	knative.dev/hack v0.0.0-20201103151104-3d5abc3a0075
	knative.dev/networking v0.0.0-20201103163404-b9f80f4537af
	knative.dev/operator v0.19.4
	knative.dev/pkg v0.0.0-20201215150143-89a9cc3e03a5
	knative.dev/serving v0.19.0
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// Kubernetes v1.18.8
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/code-generator => k8s.io/code-generator v0.18.8
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
)
