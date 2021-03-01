module github.com/openshift-knative/serverless-operator

go 1.14

require (
	github.com/alecthomas/units v0.0.0-20201120081800-1786d5ef83d4 // indirect
	github.com/aws/aws-sdk-go v1.36.15 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/spec v0.19.14 // indirect
	github.com/go-openapi/swag v0.19.12 // indirect
	github.com/google/go-cmp v0.5.4
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/manifestival/controller-runtime-client v0.4.0
	github.com/manifestival/manifestival v0.7.0
	github.com/openshift/api v0.0.0-20210202165416-a9e731090f5e
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/operator-framework/api v0.5.1
	github.com/operator-framework/operator-lifecycle-manager v0.17.1-0.20210204051820-4b67acc560a7
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.45.0
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.45.0
	github.com/prometheus/client_golang v1.9.0
	github.com/prometheus/common v0.15.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201208171446-5f87f3452ae9 // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	golang.org/x/sys v0.0.0-20201223074533-0d417f636930 // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	golang.org/x/tools v0.0.0-20201228162255-34cd474b9958 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/eventing v0.20.1
	knative.dev/eventing-kafka v0.20.1-0.20210202112232-900179eb4a86
	knative.dev/hack v0.0.0-20201214230143-4ed1ecb8db24
	knative.dev/networking v0.0.0-20210107024535-ecb89ced52d9
	knative.dev/operator v0.20.2
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
	knative.dev/eventing => github.com/openshift/knative-eventing v0.99.1-0.20210212083459-1f8c3e444cf5
	knative.dev/eventing-kafka => github.com/openshift-knative/eventing-kafka v0.19.1-0.20210202142951-dd8077f58870
	knative.dev/serving => github.com/openshift/knative-serving v0.10.1-0.20210301102056-4e0f782522f1
)
