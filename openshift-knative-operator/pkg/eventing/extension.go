package eventing

import (
	"context"
	"fmt"
	"os"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	operator "knative.dev/operator/pkg/reconciler/common"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/ptr"

	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/monitoring"
	"github.com/openshift-knative/serverless-operator/pkg/istio"
)

const requiredNsEnvName = "REQUIRED_EVENTING_NAMESPACE"

// NewExtension creates a new extension for a Knative Eventing controller.
func NewExtension(ctx context.Context, _ *controller.Impl) operator.Extension {
	return &extension{
		kubeclient: kubeclient.Get(ctx),
	}
}

type extension struct {
	kubeclient kubernetes.Interface
}

func (e *extension) Manifests(ke base.KComponent) ([]mf.Manifest, error) {
	m, err := monitoring.GetEventingMonitoringPlatformManifests(ke)
	if err != nil {
		return m, err
	}
	p, err := istio.GetServiceMeshNetworkPolicy()
	if err != nil {
		return nil, err
	}
	if enabled, err := istio.IsEnabled(e.kubeclient, os.Getenv(requiredNsEnvName)); err == nil && enabled {
		m = append(m, p)
	}
	return m, nil
}

func (e *extension) Transformers(ke base.KComponent) []mf.Transformer {
	tf := []mf.Transformer{
		common.InjectCommonLabelIntoNamespace(),
		common.VersionedJobNameTransform(),
		common.InjectCommonEnvironment(),
	}
	tf = append(tf, monitoring.GetEventingTransformers(ke)...)
	return append(tf, common.DeprecatedAPIsTranformers(e.kubeclient.Discovery())...)
}

func (e *extension) Reconcile(ctx context.Context, comp base.KComponent) error {
	ke := comp.(*operatorv1beta1.KnativeEventing)

	requiredNs := os.Getenv(requiredNsEnvName)
	if requiredNs != "" && ke.Namespace != requiredNs {
		ke.Status.MarkInstallFailed(fmt.Sprintf("Knative Eventing must be installed into the namespace %q", requiredNs))
		return controller.NewPermanentError(fmt.Errorf("deployed Knative Eventing into unsupported namespace %q", ke.Namespace))
	}

	// Override images.
	// TODO(SRVCOM-1069): Rethink overriding behavior and/or error surfacing.
	images := common.ImageMapFromEnvironment(os.Environ())
	ke.Spec.Registry.Override = images
	ke.Spec.Registry.Default = images["default"]

	// Ensure webhook has 1G of memory.
	common.EnsureContainerMemoryLimit(&ke.Spec.CommonSpec, "eventing-webhook", resource.MustParse("1024Mi"))

	// SRVKE-500: Ensure we set the SinkBindingSelectionMode to inclusion
	if ke.Spec.SinkBindingSelectionMode == "" {
		ke.Spec.SinkBindingSelectionMode = "inclusion"
	}

	// Default to 2 replicas.
	if ke.Spec.HighAvailability == nil {
		ke.Spec.HighAvailability = &base.HighAvailability{
			Replicas: ptr.Int32(2),
		}
	}

	return monitoring.ReconcileMonitoringForEventing(ctx, e.kubeclient, ke)
}

func (e *extension) Finalize(context.Context, base.KComponent) error {
	return nil
}
