package common

import (
	"slices"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
)

const (
	istioSidecarInjectionLabel          = "sidecar.istio.io/inject"
	istioSidecarRewriteProbesAnnotation = "sidecar.istio.io/rewriteAppHTTPProbers"
)

var deploymentsWithSidecarInjection = []string{
	// Serving
	"activator", "autoscaler",
	// Eventing
	"pingsource-mt-adapter", "mt-broker-ingress", "mt-broker-filter", "imc-dispatcher",
}

func AddIstioSidecarInjectLabels(kcomp base.KComponent) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Deployment" {
			return nil
		}

		deploy := &appsv1.Deployment{}
		if err := scheme.Scheme.Convert(u, deploy, nil); err != nil {
			return err
		}

		if deploy.Spec.Template.ObjectMeta.Labels == nil {
			deploy.Spec.Template.ObjectMeta.Labels = map[string]string{}
		}
		if deploy.Spec.Template.ObjectMeta.Annotations == nil {
			deploy.Spec.Template.ObjectMeta.Annotations = map[string]string{}
		}

		if slices.Contains(deploymentsWithSidecarInjection, deploy.Name) {
			deploy.Spec.Template.ObjectMeta.Labels[istioSidecarInjectionLabel] = "true"
			deploy.Spec.Template.ObjectMeta.Annotations[istioSidecarRewriteProbesAnnotation] = "true"
		} else {
			deploy.Spec.Template.ObjectMeta.Labels[istioSidecarInjectionLabel] = "false"
		}

		return scheme.Scheme.Convert(deploy, u, nil)
	}
}
