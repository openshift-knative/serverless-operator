package upgrade

import (
	"context"
	"fmt"
	"time"

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

	for _, j := range jobs.Items {
		err := wait.Poll(5*time.Second, 4*time.Minute, func() (done bool, err error) {
			j, err := client.Kube.
				BatchV1().
				Jobs(cfg.Namespace).
				Get(ctx, j.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}

			if j.Status.Failed == *j.Spec.BackoffLimit {
				return false, fmt.Errorf("job %s/%s failed: %+v", j.Namespace, j.Name, j.Status)
			}

			return j.Status.Succeeded > 0, nil
		})
		if err != nil {
			return fmt.Errorf("job %s/%s didn't reach completion: %w", cfg.Namespace, j.GetName(), err)
		}
	}

	if len(jobs.Items) == 0 && cfg.FailOnNoJobs {
		return fmt.Errorf("no jobs found in namespace %s", cfg.Namespace)
	}

	return nil
}
