package serving

import (
	"strconv"

	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/networking/pkg/apis/networking"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
)

const (
	// TODO: remove when available in knative.dev/networking/config
	ServingInternalCertName = "knative-serving-certs"
	// TODO: Maybe decide to fetch from net-kourier deps instead
	EnableSecretInformerFilteringByCertUIDEnv = "ENABLE_SECRET_INFORMER_FILTERING_BY_CERT_UID"
)

func enableSecretInformerFiltering(ks base.KComponent) mf.Transformer {
	shouldInject := false
	for _, dep := range []string{"net-istio-controller", "net-kourier-controller"} {
		if configIfUnsetAndCheckIfShouldInject(ks, dep, "controller") {
			shouldInject = true
		}
	}
	if shouldInject {
		return injectLabelIntoInternalEncryptionSecret()
	}
	return nil
}

func injectLabelIntoInternalEncryptionSecret() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Secret" && u.GetName() == ServingInternalCertName {
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
// metadata to other resources eg. label to secrets
func configIfUnsetAndCheckIfShouldInject(ks base.KComponent, deployment string, container string) bool {
	comp := ks.(*operatorv1beta1.KnativeServing)
	for _, o := range comp.Spec.GetWorkloadOverrides() {
		if o.Name == deployment {
			for _, env := range o.Env {
				if env.Container == container {
					for _, envVar := range env.EnvVars {
						if envVar.Name == EnableSecretInformerFilteringByCertUIDEnv {
							if b, err := strconv.ParseBool(envVar.Value); err == nil {
								return b
							}
							return false
						}
					}
				}
			}
		}
	}
	comp.Spec.DeploymentOverride = append(comp.Spec.DeploymentOverride, base.WorkloadOverride{
		Name: deployment,
		Env: []base.EnvRequirementsOverride{
			{
				Container: container,
				EnvVars: []corev1.EnvVar{{
					Name:  EnableSecretInformerFilteringByCertUIDEnv,
					Value: "true",
				}},
			}},
	})
	return true
}
