package serving

import (
	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func enableSecretInformerFiltering(ks operatorv1alpha1.KComponent) mf.Transformer {
	if v, ok := ks.GetAnnotations()["serverless.openshift.io/enable-secret-informer-filtering"]; ok {
		return common.InjectEnvironmentIntoDeployment("net-istio-controller", "controller",
			corev1.EnvVar{Name: "ENABLE_SECRET_INFORMER_FILTERING_BY_CERT_UID", Value: v})
	}
	return nil
}
