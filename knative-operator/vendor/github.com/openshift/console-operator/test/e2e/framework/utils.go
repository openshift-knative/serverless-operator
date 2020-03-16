package framework

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	v1 "github.com/openshift/api/operator/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
)

// func that ensures a clean slate before a test runs.
// setup is more aggressive than cleanup as the request for
// a clean slate on setup is assertive, not courtesy
func StandardSetup(t *testing.T) (*ClientSet, *v1.Console) {
	t.Helper()
	client := MustNewClientset(t, nil)
	operatorConfig := &v1.Console{}

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		conf, err := Pristine(t, client)
		operatorConfig = conf // fix shadowing
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	WaitForSettledState(t, client, "setup")

	return client, operatorConfig
}

// courtesy func to return state to something reasonable before
// the next test runs.
func StandardCleanup(t *testing.T, client *ClientSet) {
	t.Helper()
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		_, err := Pristine(t, client)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	WaitForSettledState(t, client, "cleanup")
}

func CheckEnvVars(want []corev1.EnvVar, have []corev1.EnvVar, includes bool) []error {
	var errs []error

	for _, val := range want {
		found := false
		for _, v := range have {
			if v.Name == val.Name {
				found = true
				if includes {
					if !strings.Contains(v.Value, val.Value) {
						errs = append(errs, fmt.Errorf("environment variable does not contain the expected value: expected %#v, got %#v", val, v))
					}
				} else {
					if !reflect.DeepEqual(v, val) {
						errs = append(errs, fmt.Errorf("environment variable does not equal the expected value: expected %#v, got %#v", val, v))
					}
				}
			}
		}
		if !found {
			errs = append(errs, fmt.Errorf("unable to find environment variable: wanted %s", val.Name))
		}
	}

	return errs
}
