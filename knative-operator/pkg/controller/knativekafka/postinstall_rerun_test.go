package knativekafka

import (
	"context"
	"testing"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	socommon "github.com/openshift-knative/serverless-operator/openshift-knative-operator/pkg/common"
)

// SRVKE-1658: after upgrade -> downgrade -> upgrade the post-install Job of the
// target version already exists (Complete) and is never re-run, so its
// migrations (deleting the legacy dispatcher Deployment, storage version
// migration) are skipped. The downgrade is visible in the live Deployments,
// which then run the images of the other version. The controller must
// re-create the Jobs of the current version in that case.
func TestPostInstallJobRerunsAfterOtherVersionRan(t *testing.T) {
	const currentVersion = "1.34.1"
	t.Setenv("TEST_DEPRECATED_APIS_K8S_VERSION", "v1.24.0")
	t.Setenv("CURRENT_VERSION", currentVersion)
	t.Setenv("KNATIVE_EVENTING_KAFKA_BROKER_VERSION", "1.14")
	// The monitoring transform injects this sidecar into every Deployment.
	t.Setenv("IMAGE_KUBE_RBAC_PROXY", "kube-rbac-proxy:injected")

	// VersionedJobNameTransform appends CURRENT_VERSION to the manifest name.
	jobName := types.NamespacedName{Namespace: "knative-eventing", Name: "kafka-controller-post-install-" + currentVersion}
	const seedLabel = "from-previous-install"
	// Image of the fixture testdata/controller/post-install-job.yaml.
	const manifestImage = "kafka-controller:current"
	const otherImage = "kafka-controller:other-version"

	tests := []struct {
		name          string
		liveImage     string // image of the live kafka-controller Deployment, "" for none
		seedJob       *batchv1.Job
		wantRecreated bool
	}{{
		name:          "another version was installed after ours, completed job is re-run",
		liveImage:     otherImage,
		seedJob:       completedJob(jobName, seedLabel),
		wantRecreated: true,
	}, {
		name:          "deployment already runs our images, completed job is left alone",
		liveImage:     manifestImage,
		seedJob:       completedJob(jobName, seedLabel),
		wantRecreated: false,
	}, {
		name:          "another version was installed but the job is still running, it is left alone",
		liveImage:     otherImage,
		seedJob:       activeJob(jobName, seedLabel),
		wantRecreated: false,
	}, {
		name:          "plain upgrade from another version has no job of ours yet, job is created",
		liveImage:     otherImage,
		seedJob:       nil,
		wantRecreated: true,
	}, {
		name:          "fresh install has no deployment and no job, job is created",
		liveImage:     "",
		seedJob:       nil,
		wantRecreated: true,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := []client.Object{
				makeCr(withChannelEnabled),
				&operatorv1beta1.KnativeEventing{},
				&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "knative-eventing", Name: "config-features"}},
			}
			if tc.liveImage != "" {
				objs = append(objs, kafkaControllerDeployment(tc.liveImage))
			}
			if tc.seedJob != nil {
				objs = append(objs, tc.seedJob)
			}
			cl := fake.NewClientBuilder().
				WithObjects(objs...).
				WithStatusSubresource(&v1alpha1.KnativeKafka{}).
				Build()

			controllerManifest, err := mf.ManifestFrom(mf.Path("testdata/controller/post-install-job.yaml"))
			if err != nil {
				t.Fatalf("load manifest: %v", err)
			}
			r := &ReconcileKnativeKafka{client: cl, scheme: scheme.Scheme, rawKafkaControllerManifest: controllerManifest}

			if _, err := r.Reconcile(context.Background(), defaultRequest); err != nil {
				t.Fatalf("reconcile: %v", err)
			}

			got := &batchv1.Job{}
			if err := cl.Get(context.Background(), jobName, got); err != nil {
				t.Fatalf("job %s must exist after reconcile: %v", jobName, err)
			}
			_, isSeeded := got.Labels[seedLabel]
			if isSeeded == tc.wantRecreated {
				t.Errorf("job recreated = %v, want %v (labels %v)", !isSeeded, tc.wantRecreated, got.Labels)
			}
		})
	}
}

// The Job deletion queues another reconcile that can read the old Job from a
// stale cache. The delete carries the resourceVersion the stage looked at, so
// a stale pass cannot delete the Job that apply re-created under the same name.
func TestRerunStageDoesNotDeleteJobItDidNotInspect(t *testing.T) {
	const currentVersion = "1.34.1"
	t.Setenv("CURRENT_VERSION", currentVersion)
	jobName := types.NamespacedName{Namespace: "knative-eventing", Name: "kafka-controller-post-install-" + currentVersion}

	recreated := completedJob(jobName, "recreated-by-apply")
	cl := fake.NewClientBuilder().WithObjects(recreated, kafkaControllerDeployment("kafka-controller:other-version")).Build()
	manifest, err := mf.ManifestFrom(mf.Path("testdata/controller/post-install-job.yaml"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	manifest, err = manifest.Transform(socommon.VersionedJobNameTransform())
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	// A client whose Get returns the stale, already deleted Job.
	stale := recreated.DeepCopy()
	stale.ResourceVersion = "stale"
	r := &ReconcileKnativeKafka{client: &staleJobClient{Client: cl, job: stale}, scheme: scheme.Scheme}

	if err := r.rerunJobsAfterVersionSwitch(&manifest, nil); err != nil {
		t.Fatalf("stage: %v", err)
	}

	if err := cl.Get(context.Background(), jobName, &batchv1.Job{}); apierrors.IsNotFound(err) {
		t.Fatalf("the re-created job was deleted from a stale read")
	} else if err != nil {
		t.Fatalf("get job: %v", err)
	}
}

type staleJobClient struct {
	client.Client
	job *batchv1.Job
}

func (c *staleJobClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if j, ok := obj.(*batchv1.Job); ok {
		c.job.DeepCopyInto(j)
		return nil
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

func kafkaControllerDeployment(image string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "kafka-controller", Namespace: "knative-eventing"},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{
			{Name: "controller", Image: image},
			{Name: "kube-rbac-proxy", Image: "kube-rbac-proxy:injected"},
		}}}},
	}
}

func completedJob(name types.NamespacedName, label string) *batchv1.Job {
	j := activeJob(name, label)
	j.Status = batchv1.JobStatus{Succeeded: 1}
	return j
}

func activeJob(name types.NamespacedName, label string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: name.Name, Namespace: name.Namespace, Labels: map[string]string{label: "true"}},
		Status:     batchv1.JobStatus{Active: 1},
	}
}
