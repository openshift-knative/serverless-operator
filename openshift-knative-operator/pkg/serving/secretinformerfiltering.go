package serving

import (
	"strconv"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/networking/pkg/apis/networking"
	"knative.dev/networking/pkg/config"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

// TODO: Maybe decide to fetch from net-kourier deps instead
const EnableSecretInformerFilteringByCertUIDEnv = "ENABLE_SECRET_INFORMER_FILTERING_BY_CERT_UID"

func enableSecretInformerFilteringTransformers(ks base.KComponent) []mf.Transformer {
	shouldInject := false
	var tf mf.Transformer
	comp := ks.(*operatorv1beta1.KnativeServing)

	// This works because the Knative operator runs extension reconcile before the manifest transformation
	if comp.Spec.Ingress.Istio.Enabled {
		shouldInject, tf = configIfUnsetAndCheckIfShouldInject(comp, "net-istio-controller", "controller")
	}
	if comp.Spec.Ingress.Kourier.Enabled {
		shouldInject, tf = configIfUnsetAndCheckIfShouldInject(comp, "net-kourier-controller", "controller")
	}
	if shouldInject {
		return []mf.Transformer{injectLabelIntoInternalEncryptionSecret(), tf}
	}
	return nil
}

func injectLabelIntoInternalEncryptionSecret() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		//nolint:staticcheck // ignore the deprecation until internal encryption is implemented downstream
		if u.GetKind() == "Secret" && u.GetName() == config.ServingInternalCertName {
			labels := u.GetLabels()
			if labels == nil {
				labels = make(map[string]string, 1)
			}
			labels[networking.CertificateUIDLabelKey] = "data-plane"
			u.SetLabels(labels)
			return nil
		}
		return nil
	}
}

// Adds default (true) to env vars for secret informer filtering in net-* deployments and returns if we should inject
// metadata to other resources eg. label to secrets, keeps the deprecated Istio annotation
func configIfUnsetAndCheckIfShouldInject(comp *operatorv1beta1.KnativeServing, deployment string, container string) (bool, mf.Transformer) {
	for _, o := range comp.Spec.GetWorkloadOverrides() {
		if o.Name == deployment {
			for _, env := range o.Env {
				if env.Container == container {
					for _, envVar := range env.EnvVars {
						if envVar.Name == EnableSecretInformerFilteringByCertUIDEnv {
							if b, err := strconv.ParseBool(envVar.Value); err == nil {
								return b, nil
							}
							return false, nil
						}
					}
				}
			}
		}
	}

	// TODO: remove when set to true at the net-* repos upstream/midstream
	if deployment == "net-istio-controller" || deployment == "net-kourier-controller" {
		return true, common.InjectEnvironmentIntoDeployment(deployment, "controller",
			corev1.EnvVar{Name: EnableSecretInformerFilteringByCertUIDEnv, Value: "true"})
	}
	return false, nil
}
