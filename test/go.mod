module github.com/openshift-knative/serverless-operator/test

go 1.14

require (
	github.com/coreos/prometheus-operator v0.38.1
	github.com/evanphx/json-patch v4.9.0+incompatible // indirect
	github.com/openshift-knative/serverless-operator v1.3.1-0.20201015133617-f1541b896646
	github.com/openshift/api v3.9.1-0.20190924102528-32369d4db2ad+incompatible
	github.com/openshift/client-go v0.0.0-20200116152001-92a2713fa240
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200911191357-6307c54ea472
	github.com/pkg/errors v0.9.1
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.19.1
	k8s.io/client-go v12.0.0+incompatible
	knative.dev/eventing v0.17.6-0.20201006114020-60caedca1325
	knative.dev/operator v0.17.2
	knative.dev/pkg v0.0.0-20200831162708-14fb2347fb77
	knative.dev/serving v0.17.3
	knative.dev/test-infra v0.0.0-20200915193842-f4d4232c1f04
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm
	// Pick our local version of knative-operator to be able to change both codebases at once.
	github.com/openshift-knative/serverless-operator/knative-operator => ../knative-operator

	// Hardcoded for now, see comment int hack/update-deps.sh.
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200618202633-7192180f496a

	k8s.io/api => k8s.io/api v0.17.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/apiserver => k8s.io/apiserver v0.17.6
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.6
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.6
	k8s.io/code-generator => k8s.io/code-generator v0.17.6
	k8s.io/component-base => k8s.io/component-base v0.17.6
	k8s.io/cri-api => k8s.io/cri-api v0.17.6
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.6
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.6
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.6
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.6
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.6
	k8s.io/kubectl => k8s.io/kubectl v0.17.6
	k8s.io/kubelet => k8s.io/kubelet v0.17.6
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.6
	k8s.io/metrics => k8s.io/metrics v0.17.6
	k8s.io/node-api => k8s.io/node-api v0.17.6
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.6
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.6
	k8s.io/sample-controller => k8s.io/sample-controller v0.17.6
)
