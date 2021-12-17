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
	github.com/prometheus/common v0.32.1
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.19.1
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	knative.dev/eventing v0.27.1
	knative.dev/eventing-kafka v0.27.0
	knative.dev/eventing-kafka-broker v0.0.0-00010101000000-000000000000
	knative.dev/hack v0.0.0-20211210083629-92d8a0a00cb6
	knative.dev/networking v0.0.0-20211101215640-8c71a2708e7d
	knative.dev/operator v0.27.1
	knative.dev/pkg v0.0.0-20211210132429-e86584fd3c69
	knative.dev/serving v0.27.1
	sigs.k8s.io/controller-runtime v0.9.7
	sigs.k8s.io/yaml v1.3.0
)

replace (
	// Kubernetes v1.21.4
	k8s.io/api => k8s.io/api v0.21.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.4
	k8s.io/client-go => k8s.io/client-go v0.21.4
	k8s.io/code-generator => k8s.io/code-generator v0.21.4

	// Knative forks.
	knative.dev/eventing => github.com/openshift/knative-eventing v0.99.1-0.20211209091929-0757c1f1246f
	knative.dev/eventing-kafka => github.com/openshift-knative/eventing-kafka v0.19.1-0.20211201212309-356702dbe01b
	knative.dev/eventing-kafka-broker => github.com/openshift-knative/eventing-kafka-broker v0.25.1-0.20211216103949-764b046ac45d
	knative.dev/serving => github.com/openshift/knative-serving v0.10.1-0.20211216123441-d433138b15d4
)
