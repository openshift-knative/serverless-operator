package common

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	util "knative.dev/operator/pkg/reconciler/common/testing"
)

const (
	CurrentVersion = "1.22.0"
)

func TestJobGeneratedNameTransform(t *testing.T) {

	os.Setenv("CURRENT_VERSION", CurrentVersion)

	tests := []struct {
		name     string
		job      batchv1.Job
		expected string
	}{{
		name:     "Change generated name to versioned name",
		job:      createJob("", "gen"),
		expected: "gen-" + CurrentVersion,
	}, {
		name:     "Change name to versioned name",
		job:      createJob("name", ""),
		expected: "name-" + CurrentVersion,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			u := util.MakeUnstructured(t, &test.job)
			if err := VersionedJobNameTransform()(&u); err != nil {
				t.Fatal("Unexpected error from transformer", err)
			}

			if !cmp.Equal(u.GetName(), test.expected) {
				t.Errorf("Unexpected label: Got = %q, want = %q", u.GetName(), test.expected)
			}
		})
	}

}

func TestJobsRemoveTTLSecondsAfterFinished(t *testing.T) {
	got := createJob("", "gen")
	got.Spec.TTLSecondsAfterFinished = ptr.To[int32](32)

	expected := createJob("", "gen")
	expected.Spec.TTLSecondsAfterFinished = nil

	u := util.MakeUnstructured(t, &got)
	if err := JobsRemoveTTLSecondsAfterFinished()(&u); err != nil {
		t.Fatal("Unexpected error from transformer", err)
	}

	expectedU := util.MakeUnstructured(t, &expected)
	if diff := cmp.Diff(u, expectedU); diff != "" {
		t.Errorf("Got = %#v, want = %#v\n%s", u, expectedU, diff)
	}
}

func createJob(name, gen string) batchv1.Job {
	return batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         name,
			GenerateName: gen + "-",
		},
	}
}
