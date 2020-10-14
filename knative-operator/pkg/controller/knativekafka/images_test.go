package knativekafka

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	util "knative.dev/operator/pkg/reconciler/common/testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
)

type updateImageTest struct {
	name        string
	containers  []corev1.Container
	overrideMap map[string]string
	expected    []corev1.Container
}

var updateImageTests = []updateImageTest{
	{
		name: "UsesContainerNamePerContainer",
		containers: []corev1.Container{
			{
				Name:  "container1",
				Image: "gcr.io/cmd/queue:test",
			},
			{
				Name:  "container2",
				Image: "gcr.io/cmd/queue:test",
			},
		},
		overrideMap: map[string]string{
			"container1": "new-registry.io/test/path/new-container-1:new-tag",
			"container2": "new-registry.io/test/path/new-container-2:new-tag",
		},
		expected: []corev1.Container{
			{
				Name:  "container1",
				Image: "new-registry.io/test/path/new-container-1:new-tag",
			},
			{
				Name:  "container2",
				Image: "new-registry.io/test/path/new-container-2:new-tag",
			},
		},
	},
	{
		name: "UsesOverrideFromDefault",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/serving/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
		overrideMap: map[string]string{
			"queue": "new-registry.io/test/path/new-value:new-override-tag",
		},
		expected: []corev1.Container{{
			Name:  "queue",
			Image: "new-registry.io/test/path/new-value:new-override-tag"},
		},
	},
	{
		name: "NoChangeOverrideWithDifferentName",
		containers: []corev1.Container{{
			Name:  "image",
			Image: "docker.io/name/image:tag2"},
		},
		overrideMap: map[string]string{
			"Unused": "new-registry.io/test/path",
		},
		expected: []corev1.Container{{
			Name:  "image",
			Image: "docker.io/name/image:tag2"},
		},
	},
	{
		name: "NoChange",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/eventing/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
		overrideMap: map[string]string{},
		expected: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/eventing/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45"},
		},
	},
	{
		name: "OverrideEnvVarImage",
		containers: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
		overrideMap: map[string]string{
			"SOME_IMAGE": "docker.io/my/overridden-image",
		},
		expected: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "docker.io/my/overridden-image"}},
		}},
	},
	{
		name: "NoOverrideEnvVarImage",
		containers: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
		overrideMap: map[string]string{
			"OTHER_IMAGE": "docker.io/my/overridden-image",
		},
		expected: []corev1.Container{{
			Env: []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
	},
	{
		name: "NoOverrideEnvVarImageAndContainerImageBoth",
		containers: []corev1.Container{{
			Name:  "queue",
			Image: "gcr.io/knative-releases/github.com/knative/eventing/cmd/queue@sha256:1e40c99ff5977daa2d69873fff604c6d09651af1f9ff15aadf8849b3ee77ab45",
			Env:   []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "gcr.io/foo/bar"}},
		}},
		overrideMap: map[string]string{
			"queue":      "new-registry.io/test/path/new-value:new-override-tag",
			"SOME_IMAGE": "docker.io/my/overridden-image",
		},
		expected: []corev1.Container{{
			Name:  "queue",
			Image: "new-registry.io/test/path/new-value:new-override-tag",
			Env:   []corev1.EnvVar{{Name: "SOME_IMAGE", Value: "docker.io/my/overridden-image"}},
		}},
	},
	{
		name: "OverrideWithDeploymentContainer",
		containers: []corev1.Container{
			{
				Name:  "container1",
				Image: "gcr.io/cmd/queue:test",
			},
			{
				Name:  "container2",
				Image: "gcr.io/cmd/queue:test",
			},
		},
		overrideMap: map[string]string{
			"container1": "new-registry.io/test/path/new-container-1:new-tag",
			"container2": "new-registry.io/test/path/new-container-2:new-tag",
			"OverrideWithDeploymentContainer/container1": "new-registry.io/test/path/OverrideWithDeploymentContainer/container-1:new-tag",
			"OverrideWithDeploymentContainer/container2": "new-registry.io/test/path/OverrideWithDeploymentContainer/container-2:new-tag",
		},
		expected: []corev1.Container{
			{
				Name:  "container1",
				Image: "new-registry.io/test/path/OverrideWithDeploymentContainer/container-1:new-tag",
			},
			{
				Name:  "container2",
				Image: "new-registry.io/test/path/OverrideWithDeploymentContainer/container-2:new-tag",
			},
		},
	},
	{
		name: "OverridePartialWithDeploymentContainer",
		containers: []corev1.Container{
			{
				Name:  "container1",
				Image: "gcr.io/cmd/queue:test",
			},
			{
				Name:  "container2",
				Image: "gcr.io/cmd/queue:test",
			},
		},
		overrideMap: map[string]string{
			"container1": "new-registry.io/test/path/new-container-1:new-tag",
			"container2": "new-registry.io/test/path/new-container-2:new-tag",
			"OverridePartialWithDeploymentContainer/container1": "new-registry.io/test/path/OverridePartialWithDeploymentContainer/container-1:new-tag",
		},
		expected: []corev1.Container{
			{
				Name:  "container1",
				Image: "new-registry.io/test/path/OverridePartialWithDeploymentContainer/container-1:new-tag",
			},
			{
				Name:  "container2",
				Image: "new-registry.io/test/path/new-container-2:new-tag",
			},
		},
	},
	{
		name: "OverrideWithDeploymentName",
		containers: []corev1.Container{
			{
				Name:  "container1",
				Image: "gcr.io/cmd/queue:test",
			},
			{
				Name:  "container2",
				Image: "gcr.io/cmd/queue:test",
			},
		},
		overrideMap: map[string]string{
			"OverrideWithDeploymentName/container1": "new-registry.io/test/path/OverrideWithDeploymentName/container-1:new-tag",
			"OverrideWithDeploymentName/container2": "new-registry.io/test/path/OverrideWithDeploymentName/container-2:new-tag",
		},
		expected: []corev1.Container{
			{
				Name:  "container1",
				Image: "new-registry.io/test/path/OverrideWithDeploymentName/container-1:new-tag",
			},
			{
				Name:  "container2",
				Image: "new-registry.io/test/path/OverrideWithDeploymentName/container-2:new-tag",
			},
		},
	},
}

func TestResourceTransform(t *testing.T) {
	for _, tt := range updateImageTests {
		t.Run(tt.name, func(t *testing.T) {
			runResourceTransformTest(t, &tt)
		})
	}
}

func runResourceTransformTest(t *testing.T, tt *updateImageTest) {
	// test for deployment
	unstructuredDeployment := util.MakeUnstructured(t, util.MakeDeployment(tt.name, corev1.PodSpec{Containers: tt.containers}))
	deploymentTransform := ImageTransform(tt.overrideMap, log)
	deploymentTransform(&unstructuredDeployment)
	validateUnstructuredDeploymentChanged(t, tt, &unstructuredDeployment)

	// test for daemonSet
	unstructuredDaemonSet := util.MakeUnstructured(t, makeDaemonSet(tt.name, corev1.PodSpec{Containers: tt.containers}))
	daemonSetTransform := ImageTransform(tt.overrideMap, log)
	daemonSetTransform(&unstructuredDaemonSet)
	validateUnstructuredDaemonSetChanged(t, tt, &unstructuredDaemonSet)

	// test for job
	unstructuredJob := util.MakeUnstructured(t, makeJob(tt.name, corev1.PodSpec{Containers: tt.containers}))
	jobTransform := ImageTransform(tt.overrideMap, log)
	jobTransform(&unstructuredJob)
	validateUnstructuredJobChanged(t, tt, &unstructuredJob)
}

func validateUnstructuredDeploymentChanged(t *testing.T, tt *updateImageTest, u *unstructured.Unstructured) {
	var deployment = &appsv1.Deployment{}
	err := scheme.Scheme.Convert(u, deployment, nil)
	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, deployment.Spec.Template.Spec.Containers, tt.expected)
}

func validateUnstructuredDaemonSetChanged(t *testing.T, tt *updateImageTest, u *unstructured.Unstructured) {
	var daemonSet = &appsv1.DaemonSet{}
	err := scheme.Scheme.Convert(u, daemonSet, nil)
	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, daemonSet.Spec.Template.Spec.Containers, tt.expected)
}

func validateUnstructuredJobChanged(t *testing.T, tt *updateImageTest, u *unstructured.Unstructured) {
	var job = &batchv1.Job{}
	err := scheme.Scheme.Convert(u, job, nil)
	util.AssertEqual(t, err, nil)
	util.AssertDeepEqual(t, job.Spec.Template.Spec.Containers, tt.expected)
}

func makeDaemonSet(name string, podSpec corev1.PodSpec) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}
}

func makeJob(name string, podSpec corev1.PodSpec) *batchv1.Job {
	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind: "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}
}
