module github.com/openshift-knative/serverless-operator

go 1.15

require (
	github.com/alecthomas/units v0.0.0-20201120081800-1786d5ef83d4 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/google/go-cmp v0.5.5
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/manifestival/controller-runtime-client v0.4.0
	github.com/manifestival/manifestival v0.7.0
	github.com/openshift/api v0.0.0-20210428205234-a8389931bee7
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/operator-framework/api v0.8.1
	github.com/operator-framework/operator-lifecycle-manager v0.17.1-0.20210514182438-eaf3ca9bbd84
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.47.1
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.47.1
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/common v0.25.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.16.0
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/eventing v0.22.1
	knative.dev/eventing-kafka v0.0.0-00010101000000-000000000000
	knative.dev/hack v0.0.0-20210325223819-b6ab329907d3
	knative.dev/networking v0.0.0-20210331064822-999a7708876c
	knative.dev/operator v0.22.2-0.20210512202047-38b6790875cb
	knative.dev/pkg v0.0.0-20210331065221-952fdd90dbb0
	knative.dev/serving v0.22.0
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// Kubernetes v1.19.7
	k8s.io/api => k8s.io/api v0.19.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.7
	k8s.io/client-go => k8s.io/client-go v0.19.7
	k8s.io/code-generator => k8s.io/code-generator v0.19.7
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	knative.dev/eventing => github.com/openshift/knative-eventing v0.99.1-0.20210518171156-3b8745c96673
	knative.dev/eventing-kafka => github.com/openshift-knative/eventing-kafka v0.19.1-0.20210504074514-a18051f72852
	knative.dev/serving => github.com/openshift/knative-serving v0.10.1-0.20210511123518-503694bc711c
)
