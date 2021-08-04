module github.com/openshift-knative/serverless-operator

go 1.15

require (
	github.com/alecthomas/units v0.0.0-20201120081800-1786d5ef83d4 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/google/go-cmp v0.5.6
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/manifestival/controller-runtime-client v0.4.0
	github.com/manifestival/manifestival v0.7.0
	github.com/openshift/api v0.0.0-20210428205234-a8389931bee7
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/operator-framework/api v0.8.1
	github.com/operator-framework/operator-lifecycle-manager v0.17.1-0.20210607005641-f05ea078ab46
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.49.0
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.49.0
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.26.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.18.1
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/eventing v0.23.2
	knative.dev/eventing-kafka v0.0.0-00010101000000-000000000000
	knative.dev/hack v0.0.0-20210602212444-509255f29a24
	knative.dev/networking v0.0.0-20210608114541-4b1712c029b7
	knative.dev/operator v0.23.2
	knative.dev/pkg v0.0.0-20210510175900-4564797bf3b7
	knative.dev/serving v0.23.1
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
	knative.dev/eventing => github.com/openshift/knative-eventing v0.99.1-0.20210629103904-1be9f5f98cec
	knative.dev/eventing-kafka => github.com/openshift-knative/eventing-kafka v0.19.1-0.20210706095154-bc9b27b4771b
	knative.dev/serving => github.com/openshift/knative-serving v0.10.1-0.20210701102323-75ffe62956f2
)
