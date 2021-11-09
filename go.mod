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
	github.com/prometheus/common v0.31.1
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.19.1
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	knative.dev/eventing v0.26.1
	knative.dev/eventing-kafka v0.0.0-00010101000000-000000000000
	knative.dev/hack v0.0.0-20210806075220-815cd312d65c
	knative.dev/networking v0.0.0-20210914225408-69ad45454096
	knative.dev/operator v0.26.3
	knative.dev/pkg v0.0.0-20210919202233-5ae482141474
	knative.dev/serving v0.26.0
	sigs.k8s.io/controller-runtime v0.9.7
)

replace (
	// Kubernetes v1.21.4
	k8s.io/api => k8s.io/api v0.21.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.4
	k8s.io/client-go => k8s.io/client-go v0.21.4
	k8s.io/code-generator => k8s.io/code-generator v0.21.4

	// Knative forks.
	knative.dev/eventing => github.com/openshift/knative-eventing v0.99.1-0.20211104153203-bb8a2e690a22
	knative.dev/eventing-kafka => github.com/openshift-knative/eventing-kafka v0.19.1-0.20211109065725-8b5a70060273
	knative.dev/serving => github.com/openshift/knative-serving v0.10.1-0.20211104123023-a70b6bb3b54c
)
