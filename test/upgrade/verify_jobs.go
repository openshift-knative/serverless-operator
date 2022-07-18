package upgrade

import (
	"context"
	"fmt"
	"time"

	"github.com/openshift-knative/serverless-operator/test"
	"golang.org/x/sync/errgroup"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	testlib "knative.dev/eventing/test/lib"
	"knative.dev/pkg/test/upgrade"
)

type VerifyPostJobsConfig struct {
	Namespace    string
	FailOnNoJobs bool
}

func VerifyPostInstallJobs(cfg VerifyPostJobsConfig) upgrade.Operation {
	return upgrade.NewOperation("Verify jobs in "+cfg.Namespace, func(c upgrade.Context) {
		if err := verifyPostInstallJobs(context.Background(), c, cfg); err != nil {
			c.T.Error(err)
		}
	})
}

func VerifyPostInstallServingJobs(ctx *test.Context, cfg VerifyPostJobsConfig) upgrade.Operation {
	return upgrade.NewOperation("Verify jobs in "+cfg.Namespace, func(c upgrade.Context) {
		if err := verifyPostInstallServingJobs(context.Background(), ctx, c, cfg); err != nil {
			c.T.Error(err)
		}
	})
}

func verifyPostInstallServingJobs(ctx context.Context, testCtx *test.Context, c upgrade.Context, cfg VerifyPostJobsConfig) error {
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

		if j.Status.Succeeded > 0 {
			// We don't need to wait for a job that is already succeeded.
			// In addition, an already succeeded job might go away due to the job's TTL.
			continue
		}
		eg.Go(func() error {
			err := wait.PollUntil(5*time.Second, func() (bool, error) {
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
			}, ctx.Done())
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

func verifyPostInstallJobs(ctx context.Context, c upgrade.Context, cfg VerifyPostJobsConfig) error {
	client := testlib.Setup(c.T, false)
	defer testlib.TearDown(client)

	jobs, err := client.Kube.
		BatchV1().
		Jobs(cfg.Namespace).
		List(ctx, metav1.ListOptions{Limit: 500 /* Use a very large number to avoid handling pagination */})
	if err != nil {
		return fmt.Errorf("failed to list jobs in namespace %s: %w", cfg.Namespace, err)
	}

	if len(jobs.Items) == 0 && cfg.FailOnNoJobs {
		return fmt.Errorf("no jobs found in namespace %s", cfg.Namespace)
	}

	kubeClient := client.Kube

	eg, ctx := errgroup.WithContext(ctx)
	for _, j := range jobs.Items {
		j := j

		if j.Status.Succeeded > 0 {
			// We don't need to wait for a job that is already succeeded.
			// In addition, an already succeeded job might go away due to the job's TTL.
			continue
		}

		eg.Go(func() error {
			err := wait.PollUntil(5*time.Second, func() (bool, error) {
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
			}, ctx.Done())
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
