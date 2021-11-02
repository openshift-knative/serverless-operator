package common

import (
	serverlessoperatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
)

func MutateKafka(ke *serverlessoperatorv1alpha1.KnativeKafka) {
	defaultToKafkaHa(ke)
}

func defaultToKafkaHa(ke *serverlessoperatorv1alpha1.KnativeKafka) {
	if ke.Spec.HighAvailability == nil {
		ke.Spec.HighAvailability = &operatorv1alpha1.HighAvailability{
			Replicas: 1,
		}
	}
}
