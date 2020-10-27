module github.com/openshift-knative/serverless-operator/openshift-knative-operator

go 1.14

require (
	github.com/google/go-cmp v0.5.2
	github.com/google/uuid v1.1.2 // indirect
	github.com/manifestival/manifestival v0.6.1
	github.com/openshift/api v0.0.0-20200930075302-db52bc4ef99f
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43 // indirect
	golang.org/x/sys v0.0.0-20200915084602-288bc346aa39 // indirect
	golang.org/x/tools v0.0.0-20200915031644-64986481280e // indirect
	google.golang.org/genproto v0.0.0-20200914193844-75d14daec038 // indirect
	google.golang.org/grpc v1.32.0 // indirect
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.1
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/eventing v0.14.1-0.20200428210242-f355830c4d70 // indirect
	knative.dev/operator v0.18.1
	knative.dev/pkg v0.0.0-20201026165741-2f75016c1368
	knative.dev/test-infra v0.0.0-20200921012245-37f1a12adbd3
)

replace (
	github.com/go-logr/zapr => github.com/go-logr/zapr v0.2.0 // To make klog happy

	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/code-generator => k8s.io/code-generator v0.18.8
)
