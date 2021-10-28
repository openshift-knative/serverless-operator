module github.com/openshift-knative/serverless-operator

go 1.16

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/go-logr/logr v0.4.0
	github.com/google/go-cmp v0.5.6
	github.com/manifestival/controller-runtime-client v0.4.0
	github.com/manifestival/manifestival v0.7.0
	github.com/openshift/api v0.0.0-20210428205234-a8389931bee7
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/operator-framework/api v0.8.1
	github.com/operator-framework/operator-lifecycle-manager v0.17.1-0.20210607005641-f05ea078ab46
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.49.0
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.49.0
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.30.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.19.0
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/eventing v0.25.1
	knative.dev/eventing-kafka v0.0.0-00010101000000-000000000000
	knative.dev/hack v0.0.0-20210622141627-e28525d8d260
	knative.dev/networking v0.0.0-20210903132258-9d8ab8618e5f
	knative.dev/operator v0.25.3-0.20211014142443-7d5a16050119
	knative.dev/pkg v0.0.0-20210902173607-844a6bc45596
	knative.dev/serving v0.25.1
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	// TODO: Remove this after Knative is bumped to 0.26.
	contrib.go.opencensus.io/exporter/prometheus => contrib.go.opencensus.io/exporter/prometheus v0.3.1-0.20210621165811-f3a7283b3002

	// Kubernetes v1.20.7
	k8s.io/api => k8s.io/api v0.20.7
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.7
	k8s.io/client-go => k8s.io/client-go v0.20.7
	k8s.io/code-generator => k8s.io/code-generator v0.20.7

	// Knative forks.
	knative.dev/eventing => github.com/openshift/knative-eventing v0.99.1-0.20211005104650-2127d7fc1400
	knative.dev/eventing-kafka => github.com/openshift-knative/eventing-kafka v0.19.1-0.20211011070646-46b07bfde414
	knative.dev/serving => github.com/openshift/knative-serving v0.10.1-0.20210924121646-ec4ba4824474
)
