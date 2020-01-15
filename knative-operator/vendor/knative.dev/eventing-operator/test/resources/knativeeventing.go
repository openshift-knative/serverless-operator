/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// knativeserving.go provides methods to perform actions on the KnativeEventing resource.

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
	eventingv1alpha1 "knative.dev/eventing-operator/pkg/client/clientset/versioned/typed/eventing/v1alpha1"
	"knative.dev/eventing-operator/test"
	"knative.dev/pkg/test/logging"
)

const (
	// Interval specifies the time between two polls.
	Interval = 10 * time.Second
	// Timeout specifies the timeout for the function PollImmediate to reach a certain status.
	Timeout = 5 * time.Minute
)

// WaitForKnativeEventingState polls the status of the KnativeEventing called name
// from client every `interval` until `inState` returns `true` indicating it
// is done, returns an error or timeout.
func WaitForKnativeEventingState(clients eventingv1alpha1.KnativeEventingInterface, name string,
	inState func(s *v1alpha1.KnativeEventing, err error) (bool, error)) (*v1alpha1.KnativeEventing, error) {
	span := logging.GetEmitableSpan(context.Background(), fmt.Sprintf("WaitForKnativeEventingState/%s/%s", name, "KnativeEventingIsReady"))
	defer span.End()

	var lastState *v1alpha1.KnativeEventing
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err := clients.Get(name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "KnativeEventing %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

// CreateKnativeEventing creates a KnativeEventingServing with the name names.KnativeEventing under the namespace names.Namespace.
func CreateKnativeEventing(clients eventingv1alpha1.KnativeEventingInterface, names test.ResourceNames) (*v1alpha1.KnativeEventing, error) {
	ks := &v1alpha1.KnativeEventing{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.KnativeEventing,
			Namespace: names.Namespace,
		},
	}
	svc, err := clients.Create(ks)
	return svc, err
}

// IsKnativeEventingReady will check the status conditions of the KnativeEventing and return true if the KnativeEventing is ready.
func IsKnativeEventingReady(s *v1alpha1.KnativeEventing, err error) (bool, error) {
	return s.Status.IsReady(), err
}

// IsDeploymentAvailable will check the status conditions of the deployment and return true if the deployment is available.
func IsDeploymentAvailable(d *v1.Deployment) (bool, error) {
	return getDeploymentStatus(d) == "True", nil
}

func getDeploymentStatus(d *v1.Deployment) corev1.ConditionStatus {
	for _, dc := range d.Status.Conditions {
		if dc.Type == "Available" {
			return dc.Status
		}
	}
	return "unknown"
}
