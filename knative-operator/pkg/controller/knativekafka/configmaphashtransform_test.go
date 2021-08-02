package knativekafka

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	util "knative.dev/operator/pkg/reconciler/common/testing"

	appsv1 "k8s.io/api/apps/v1"
)

func TestConfigMapHashTransform(t *testing.T) {
	configMapHashTransformCases := []struct {
		name                  string
		currentAnnotations    map[string]string
		annotationsInManifest map[string]string
		expectedAnnotations   map[string]string
	}{{
		name: "copy over",
		currentAnnotations: map[string]string{
			"kafka.eventing.knative.dev/configmap-hash": "deadbeef",
		},
		annotationsInManifest: map[string]string{
			"kafka.eventing.knative.dev/configmap-hash": "",
		},
		expectedAnnotations: map[string]string{
			"kafka.eventing.knative.dev/configmap-hash": "deadbeef",
		},
	}, {
		name: "overwrite",
		currentAnnotations: map[string]string{
			"kafka.eventing.knative.dev/configmap-hash": "deadbeef",
		},
		annotationsInManifest: map[string]string{
			"kafka.eventing.knative.dev/configmap-hash": "feeddead",
		},
		expectedAnnotations: map[string]string{
			"kafka.eventing.knative.dev/configmap-hash": "deadbeef",
		},
	}, {
		name: "overwrite if key exists even without value",
		currentAnnotations: map[string]string{
			"kafka.eventing.knative.dev/configmap-hash": "",
		},
		annotationsInManifest: map[string]string{
			"kafka.eventing.knative.dev/configmap-hash": "feeddead",
		},
		expectedAnnotations: map[string]string{
			"kafka.eventing.knative.dev/configmap-hash": "",
		},
	}, {
		name:               "do not clear",
		currentAnnotations: map[string]string{},
		annotationsInManifest: map[string]string{
			"kafka.eventing.knative.dev/configmap-hash": "feeddead",
		},
		expectedAnnotations: map[string]string{
			"kafka.eventing.knative.dev/configmap-hash": "feeddead",
		},
	}}

	for _, tt := range configMapHashTransformCases {
		t.Run(tt.name, func(t *testing.T) {

			currentDeployment := makeDeploymentWithPodAnnotations(tt.name, tt.currentAnnotations)
			deploymentInManifest := makeDeploymentWithPodAnnotations(tt.name, tt.annotationsInManifest)

			doConfigMapHashTransform(currentDeployment, deploymentInManifest)

			if diff := cmp.Diff(tt.expectedAnnotations, deploymentInManifest.Spec.Template.ObjectMeta.Annotations); diff != "" {
				t.Fatal("Unexpected defaults (-want, +got):", diff)
			}
		})
	}

}

func makeDeploymentWithPodAnnotations(name string, podAnnotations map[string]string) *appsv1.Deployment {
	d := util.MakeDeployment(name, corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "container1",
				Image: "gcr.io/cmd/queue:test",
			},
		},
	})

	d.Spec.Template.ObjectMeta.Annotations = podAnnotations

	return d
}
