module github.com/openshift-knative/serverless-operator

go 1.14

require (
	github.com/Jeffail/gabs v1.4.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/go-logr/logr v0.3.0
	github.com/google/go-cmp v0.5.4
	github.com/manifestival/controller-runtime-client v0.3.0
	github.com/manifestival/manifestival v0.6.1
	github.com/openshift/api v0.0.0-20210202165416-a9e731090f5e
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/operator-framework/api v0.5.1
	github.com/operator-framework/operator-lifecycle-manager v0.17.1-0.20210204051820-4b67acc560a7
	github.com/prometheus-operator/prometheus-operator v0.44.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.44.1
	github.com/prometheus/client_golang v1.8.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.16.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/eventing v0.20.1
	knative.dev/eventing-kafka v0.20.1-0.20210202112232-900179eb4a86
	knative.dev/hack v0.0.0-20201214230143-4ed1ecb8db24
	knative.dev/networking v0.0.0-20210107024535-ecb89ced52d9
	knative.dev/operator v0.20.1
	knative.dev/pkg v0.0.0-20210107022335-51c72e24c179
	knative.dev/serving v0.20.0
	sigs.k8s.io/controller-runtime v0.8.1
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
	knative.dev/eventing => github.com/openshift/knative-eventing v0.99.1-0.20210112194111-24e8ffe5a3a5
	knative.dev/eventing-kafka => github.com/openshift-knative/eventing-kafka v0.19.1-0.20210202142951-dd8077f58870
	knative.dev/serving => github.com/openshift/knative-serving v0.10.1-0.20210127081219-6330eae520d2
)
