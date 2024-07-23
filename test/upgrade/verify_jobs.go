package upgrade

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/pkg/test/upgrade"

	"github.com/openshift-knative/serverless-operator/test"
)

type VerifyPostJobsConfig struct {
	Namespace    string
	FailOnNoJobs bool
	ValidateJob  func(j batchv1.Job) error
}

func VerifyPostInstallJobs(ctx *test.Context, cfg VerifyPostJobsConfig) upgrade.Operation {
	return upgrade.NewOperation("Verify jobs in "+cfg.Namespace, func(c upgrade.Context) {
		if err := verifyPostInstallJobs(context.Background(), ctx, c, cfg); err != nil {
			c.T.Error(err)
		}
	})
}

func verifyPostInstallJobs(ctx context.Context, testCtx *test.Context, c upgrade.Context, cfg VerifyPostJobsConfig) error {
	jobs, err := testCtx.Clients.Kube.
		BatchV1().
		Jobs(cfg.Namespace).
		List(ctx, metav1.ListOptions{Limit: 500 /* Use a very large number to avoid handling pagination */})
	if err != nil {
		return fmt.Errorf("failed to list jobs in namespace %s: %w", cfg.Namespace, err)
	}

	if len(jobs.Items) == 0 && cfg.FailOnNoJobs {
		return fmt.Errorf("no jobs found in namespace %s", cfg.Namespace)
	}
	kubeClient := testCtx.Clients.Kube

	eg, ctx := errgroup.WithContext(ctx)
	for _, j := range jobs.Items {
		j := j

		if cfg.ValidateJob != nil {
			if err := cfg.ValidateJob(j); err != nil {
				return fmt.Errorf("failed to validate job %s: %w", j.Name, err)
			}
		}

		if j.Status.Succeeded > 0 {
			// We don't need to wait for a job that is already succeeded.
			// In addition, an already succeeded job might go away due to the job's TTL.
			continue
		}

		eg.Go(func() error {
			err := wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(_ context.Context) (bool, error) {
				j, err := kubeClient.
					BatchV1().
					Jobs(cfg.Namespace).
					Get(ctx, j.Name, metav1.GetOptions{})
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				if err != nil {
					return false, err
				}

				if j.Status.Failed > 0 {
					c.T.Logf("Job %s/%s failed %d times", j.Namespace, j.Name, j.Status.Failed)
				}

				if j.Status.Failed == *j.Spec.BackoffLimit {
					return false, fmt.Errorf("job %s/%s failed: %+v", j.Namespace, j.Name, j.Status)
				}

				return j.Status.Succeeded > 0, nil
			})
			if err != nil {
				return fmt.Errorf("%w, job:\n%+v", err, j)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("jobs in %s didn't run successfully: %w", cfg.Namespace, err)
	}

	return nil
}
