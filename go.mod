module github.com/openshift-knative/serverless-operator

go 1.14

require (
	github.com/alecthomas/units v0.0.0-20201120081800-1786d5ef83d4 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/google/go-cmp v0.5.5
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/manifestival/controller-runtime-client v0.4.0
	github.com/manifestival/manifestival v0.7.0
	github.com/openshift/api v0.0.0-20210202165416-a9e731090f5e
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/operator-framework/api v0.6.0
	github.com/operator-framework/operator-lifecycle-manager v0.17.1-0.20210415175807-cdf51cddb619
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.45.0
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.45.0
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/common v0.19.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.16.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/eventing v0.21.3
	knative.dev/eventing-kafka v0.0.0-00010101000000-000000000000
	knative.dev/hack v0.0.0-20210317214554-58edbdc42966
	knative.dev/networking v0.0.0-20210324061918-44a3b919bce1
	knative.dev/operator v0.21.2
	knative.dev/pkg v0.0.0-20210323202917-b558677ab034
	knative.dev/serving v0.21.0
	sigs.k8s.io/controller-runtime v0.8.1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.8.1
	// Kubernetes v1.19.7
	k8s.io/api => k8s.io/api v0.19.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.7
	k8s.io/client-go => k8s.io/client-go v0.19.7
	k8s.io/code-generator => k8s.io/code-generator v0.19.7
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	knative.dev/eventing => github.com/openshift/knative-eventing v0.99.1-0.20210526121953-4c434bbf8650
	knative.dev/eventing-kafka => github.com/openshift-knative/eventing-kafka v0.21.1-0.20210526142120-91eb12066b84
	knative.dev/serving => github.com/openshift/knative-serving v0.10.1-0.20210420084308-e1193e08522a
)
