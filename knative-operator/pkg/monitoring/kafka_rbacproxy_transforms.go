package monitoring

import (
	"context"
	"errors"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/util/sets"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	serverlessoperatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/monitoring"
)

var (
	KafkaChannelComponents         = []string{"kafka-ch-controller", "kafka-webhook"}
	KafkaSourceComponents          = []string{"kafka-controller-manager"}
	KafkaBrokerDataPlaneComponents = []string{"kafka-broker-dispatcher", "kafka-broker-receiver"}
	KafkaSinkDataPlaneComponents   = []string{"kafka-sink-receiver"}
	KafkaControllerComponents      = []string{"kafka-controller"}
)

func AddRBACProxySupportToManifest(instance *serverlessoperatorv1alpha1.KnativeKafka, components []string) (*mf.Manifest, error) {
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
	eventingList := &operatorv1alpha1.KnativeEventingList{}
	err := apiClient.List(context.Background(), eventingList)
	if err != nil {
		return nil, err
	}
	if len(eventingList.Items) == 0 {
		return nil, errors.New("eventing instance not found")
	}
	if monitoring.ShouldEnableMonitoring(eventingList.Items[0].GetSpec().GetConfig()) {
		components := make([]string,
			len(KafkaChannelComponents)+
				len(KafkaSourceComponents)+
				len(KafkaControllerComponents)+
				len(KafkaBrokerDataPlaneComponents)+
				len(KafkaSinkDataPlaneComponents),
		)
		components = append(components, KafkaChannelComponents...)
		components = append(components, KafkaSourceComponents...)
		components = append(components, KafkaBrokerDataPlaneComponents...)
		components = append(components, KafkaControllerComponents...)
		components = append(components, KafkaSinkDataPlaneComponents...)
		return monitoring.InjectRbacProxyContainerToDeployments(sets.NewString(components...)), nil
	}
	return nil, nil
}
