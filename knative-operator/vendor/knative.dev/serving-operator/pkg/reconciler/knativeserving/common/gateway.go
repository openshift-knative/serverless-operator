package common

import (
	mf "github.com/jcrossley3/manifestival"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	servingv1alpha1 "knative.dev/serving-operator/pkg/apis/serving/v1alpha1"
)

func GatewayTransform(instance *servingv1alpha1.KnativeServing, log *zap.SugaredLogger) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		// Update the deployment with the new registry and tag
		if u.GetAPIVersion() == "networking.istio.io/v1alpha3" && u.GetKind() == "Gateway" {
			if u.GetName() == "knative-ingress-gateway" {
				return updateKnativeIngressGateway(instance.Spec.KnativeIngressGateway, u)
			}
			if u.GetName() == "cluster-local-gateway" {
				return updateKnativeIngressGateway(instance.Spec.ClusterLocalGateway, u)
			}
		}
		return nil
	}
}

func updateKnativeIngressGateway(gatewayOverrides servingv1alpha1.IstioGatewayOverride, u *unstructured.Unstructured) error {
	if len(gatewayOverrides.Selector) > 0 {
		log.Debugw("Updating Gateway", "name", u.GetName(), "gatewayOverrides", gatewayOverrides)
		unstructured.SetNestedStringMap(u.Object, gatewayOverrides.Selector, "spec", "selector")
		log.Debugw("Finished conversion", "name", u.GetName(), "unstructured", u.Object)
	}
	return nil
}
