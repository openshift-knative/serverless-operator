package common

import (
	"fmt"
	"os"

	mf "github.com/manifestival/manifestival"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

// VersionedJobNameTransform ensures that only a name is used from the job
// which contains the Serverless Version as part of its name.
func VersionedJobNameTransform() mf.Transformer {
	version := os.Getenv("CURRENT_VERSION")
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Job" {
			job := &batchv1.Job{}
			if err := scheme.Scheme.Convert(u, job, nil); err != nil {
				return err
			}
			if job.GetName() == "" && job.GetGenerateName() != "" {
				job.SetName(fmt.Sprintf("%s%s", job.GetGenerateName(), version))
				job.SetGenerateName("")
			} else {
				job.SetName(fmt.Sprintf("%s-%s", job.GetName(), version))
			}
			return scheme.Scheme.Convert(job, u, nil)
		}
		return nil
	}
}

func JobsRemoveTTLSecondsAfterFinished() mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() == "Job" {
			job := &batchv1.Job{}
			if err := scheme.Scheme.Convert(u, job, nil); err != nil {
				return err
			}
			if job.Spec.TTLSecondsAfterFinished != nil {
				job.Spec.TTLSecondsAfterFinished = nil
			}
			return scheme.Scheme.Convert(job, u, nil)
		}
		return nil
	}
}
