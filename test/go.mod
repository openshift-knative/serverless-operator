module github.com/openshift-knative/serverless-operator/test

go 1.14

require (
	github.com/google/uuid v1.1.2 // indirect
	github.com/openshift-knative/serverless-operator v1.3.1-0.20201015133617-f1541b896646
	github.com/openshift/api v0.0.0-20200930075302-db52bc4ef99f
	github.com/openshift/client-go v0.0.0-20200929181438-91d71ef2122c
	github.com/operator-framework/api v0.3.16
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20201013052701-d7b2aff003ad
	github.com/prometheus-operator/prometheus-operator v0.43.0
	google.golang.org/genproto v0.0.0-20200914193844-75d14daec038 // indirect
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/eventing v0.18.4-0.20201026103441-891cb08ff28d
	knative.dev/eventing-contrib v0.18.2
	knative.dev/networking v0.0.0-20200922180040-a71b40c69b15
	knative.dev/operator v0.18.1
	knative.dev/pkg v0.0.0-20201026165741-2f75016c1368
	knative.dev/serving v0.18.1
	knative.dev/test-infra v0.0.0-20200921012245-37f1a12adbd3
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm
	// Pick our local version of knative-operator to be able to change both codebases at once.
	github.com/openshift-knative/serverless-operator/knative-operator => ../knative-operator

	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
)
