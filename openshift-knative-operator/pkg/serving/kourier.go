package serving

import (
	"strings"

	mf "github.com/manifestival/manifestival"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/networking/pkg/config"
	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/pkg/network"
)

const (
	providerLabel           = "networking.knative.dev/ingress-provider"
	kourierIngressClassName = "kourier.ingress.networking.knative.dev"
	networkCMName           = "network"

	// IngressDefaultCertificateKey is the OpenShift Ingress default certificate name.
	// The default cert name is different when users changed the default ingress certificate name via IngressController CR (SRVKS-955).
	IngressDefaultCertificateKey = "openshift-ingress-default-certificate"

	// ingressDefaultCertificateNameSpace is the namespace where the default ingress certificate is deployed.
	ingressDefaultCertificateNameSpace = "openshift-ingress"

	// ingressDefaultCertificateName is the name of the default ingress certificate.
	ingressDefaultCertificateName = "router-certs-default"

	// bootStrapConfigKey is the key of kourier-bootstrap configmap data.
	bootStrapConfigKey = "envoy-bootstrap.yaml"

	// defaultControllerAddress is the address of net-kourier-controller defined in kourier-bootstrap configmap by default.
	defaultControllerAddress = "net-kourier-controller.knative-serving"
)

// overrideKourierBootstrap overrides the address of kourier controller address.
func overrideKourierBootstrap(ks base.KComponent) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "ConfigMap" || u.GetName() != "kourier-bootstrap" {
			return nil
		}

		clusterLocalDomain := network.GetClusterDomainName()

		cm := &corev1.ConfigMap{}
		if err := scheme.Scheme.Convert(u, cm, nil); err != nil {
			return err
		}

		controllerAddress := "net-kourier-controller." + kourierNamespace(ks.GetNamespace()) + ".svc." + clusterLocalDomain + "."
		data := cm.Data[bootStrapConfigKey]

		// Replace defaultControllerAddress with the complete kourier controller address.
		// i.e. "net-kourier-controller.knative-serving" to "net-kourier-controller.knative-serving-ingress.svc.cluster.local."
		cm.Data[bootStrapConfigKey] = strings.Replace(data, defaultControllerAddress, controllerAddress, 1)

		return scheme.Scheme.Convert(cm, u, nil)
	}
}

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

		// We need to unset OwnerReferences so Openshift doesn't delete Kourier resources.
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
	if encrypt := networkCM[config.SystemInternalTLSKey]; strings.ToLower(encrypt) == string(config.EncryptionEnabled) {
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
