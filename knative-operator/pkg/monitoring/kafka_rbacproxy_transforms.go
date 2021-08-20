package monitoring

import (
	"context"
	"errors"

	mf "github.com/manifestival/manifestival"
	operatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/monitoring"
	"k8s.io/apimachinery/pkg/util/sets"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	KafkaChannelComponents = []string{"kafka-ch-controller", "kafka-webhook"}
	KafkaSourceComponents  = []string{"kafka-controller-manager"}
)

func AddRBACProxySupportToManifest(instance *operatorv1alpha1.KnativeKafka, components []string) (*mf.Manifest, error) {
	proxyManifest := mf.Manifest{}
	// Only create the roles needed for the deployment service accounts as Prometheus has already
	// the rights needed due to eventing that is assumed to be installed.
	for _, c := range components {
		crbM, err := monitoring.CreateClusterRoleBindingManifest(c, instance.GetNamespace())
		if err != nil {
			return nil, err
		}
		proxyManifest = proxyManifest.Append(*crbM)
		if err = monitoring.AppendManifestsForComponent(c, instance.GetNamespace(), &proxyManifest); err != nil {
			return nil, err
		}
	}
	return &proxyManifest, nil
}

func GetRBACProxyInjectTransformer(apiClient client.Client) (mf.Transformer, error) {
	eventingList := &eventingv1alpha1.KnativeEventingList{}
	err := apiClient.List(context.Background(), eventingList)
	if err != nil {
		return nil, err
	}
	if len(eventingList.Items) == 0 {
		return nil, errors.New("eventing instance not found")
	}
	if monitoring.ShouldEnableMonitoring(eventingList.Items[0].GetSpec().GetConfig()) {
		return monitoring.InjectRbacProxyContainerToDeployments(sets.NewString(append(KafkaChannelComponents, KafkaSourceComponents...)...)), nil
	}
	return nil, nil
}
