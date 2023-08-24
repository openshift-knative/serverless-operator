package serving

import (
	"strings"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/operator/pkg/apis/operator/base"
)

const (
	providerLabel           = "networking.knative.dev/ingress-provider"
	kourierIngressClassName = "kourier.ingress.networking.knative.dev"
	networkCMName           = "network"

	// TODO: Use "knative.dev/networking/pkg/config" once the repo pulled Knative 1.6.
	// Backport messes up the dependencies.
	InternalEncryptionKey = "internal-encryption"

	// IngressDefaultCertificateKey is the OpenShift Ingress default certificate name.
	// The default cert name is different when users changed the default ingress certificate name via IngressController CR (SRVKS-955).
	IngressDefaultCertificateKey = "openshift-ingress-default-certificate"

	// ingressDefaultCertificateNameSpace is the namespace where the default ingress certificate is deployed.
	ingressDefaultCertificateNameSpace = "openshift-ingress"

	// ingressDefaultCertificateName is the name of the default ingress certificate.
	ingressDefaultCertificateName = "router-certs-default"
)

// overrideKourierNamespace overrides the namespace of all Kourier related resources to
// the -ingress suffix to be backwards compatible.
func overrideKourierNamespace(ks base.KComponent) mf.Transformer {
	nsInjector := mf.InjectNamespace(kourierNamespace(ks.GetNamespace()))
	return func(u *unstructured.Unstructured) error {
		provider := u.GetLabels()[providerLabel]
		if provider != "kourier" {
			return nil
		}

		labels := u.GetLabels()
		if labels == nil {
			labels = make(map[string]string, 2)
		}
		labels[socommon.ServingOwnerNamespace] = ks.GetNamespace()
		labels[socommon.ServingOwnerName] = ks.GetName()
		u.SetLabels(labels)

		// We need to unset OwnerReferences so Openshift doesn't delete Kourier ressources.
		u.SetOwnerReferences(nil)
		return nsInjector(u)
	}
}

// kourierNamespace returns the namespace Kourier was installed into for backwards
// compatibility.
func kourierNamespace(servingNs string) string {
	return servingNs + "-ingress"
}

func addKourierEnvValues(ks base.KComponent) mf.Transformer {
	envVars := []corev1.EnvVar{
		{Name: "KOURIER_HTTPOPTION_DISABLED", Value: "true"},
		{Name: "SERVING_NAMESPACE", Value: "knative-serving"},
	}

	networkCM := ks.GetSpec().GetConfig()[networkCMName]
	if encrypt := networkCM[InternalEncryptionKey]; strings.ToLower(encrypt) == "true" {
		if certName := networkCM[IngressDefaultCertificateKey]; certName != "" {
			envVars = append(envVars,
				corev1.EnvVar{Name: "CERTS_SECRET_NAMESPACE", Value: ingressDefaultCertificateNameSpace},
				corev1.EnvVar{Name: "CERTS_SECRET_NAME", Value: certName})
		} else {
			envVars = append(envVars,
				corev1.EnvVar{Name: "CERTS_SECRET_NAMESPACE", Value: ingressDefaultCertificateNameSpace},
				corev1.EnvVar{Name: "CERTS_SECRET_NAME", Value: ingressDefaultCertificateName})
		}
	}
	return common.InjectEnvironmentIntoDeployment("net-kourier-controller", "controller", envVars...)
}

// addKourierAppProtocol adds appProtocol name to the Kourier service.
// OpenShift Ingress needs to have it to handle gRPC/H2C.
func addKourierAppProtocol(ks base.KComponent) mf.Transformer {
	// TODO: revisit after OCP 4.13 (HAProxy 2.4) is available.
	// As current h2c protocol name breaks websocket on OCP Route, change the port name only when the annotation is added.
	//
	// see - https://docs.openshift.com/container-platform/4.11/networking/ingress-operator.html#nw-http2-haproxy_configuring-ingress
	// > Consequently, if you have an application that is intended to accept WebSocket connections,
	// > it must not allow negotiating the HTTP/2 protocol or else clients will fail to upgrade to the WebSocket protocol.
	if _, ok := ks.GetAnnotations()["serverless.openshift.io/default-enable-http2"]; !ok {
		return nil
	}
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Service" || u.GetName() != "kourier" {
			return nil
		}

		service := &corev1.Service{}
		if err := scheme.Scheme.Convert(u, service, nil); err != nil {
			return err
		}
		appProtocolName := "h2c"
		for i := range service.Spec.Ports {
			port := &service.Spec.Ports[i]
			if port.Name != "http2" {
				continue
			}
			port.AppProtocol = &appProtocolName
		}

		return scheme.Scheme.Convert(service, u, nil)
	}
}
