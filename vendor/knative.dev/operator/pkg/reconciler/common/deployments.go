/*
Copyright 2020 The Knative Authors

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

package common

import (
	"context"
	"errors"

	mf "github.com/manifestival/manifestival"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"

	"knative.dev/operator/pkg/apis/operator/base"
)

type deploymentsNotReadyError struct{}

var _ error = deploymentsNotReadyError{}

// Error implements the Error() interface of error.
func (err deploymentsNotReadyError) Error() string {
	return "deployments not ready"
}

// IsDeploymentsNotReadyError returns true if the given error is a deploymentsNotReadyError.
func IsDeploymentsNotReadyError(err error) bool {
	return errors.Is(err, deploymentsNotReadyError{})
}

// CheckDeployments checks all deployments in the given manifest and updates the given
// status with the status of the deployments.
func CheckDeployments(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	status := instance.GetStatus()
	var nonReadyDeployments []string
	for _, u := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
		resource, err := manifest.Client.Get(&u)
		if err != nil {
			status.MarkDeploymentsNotReady([]string{"all"})
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		deployment := &appsv1.Deployment{}
		if err := scheme.Scheme.Convert(resource, deployment, nil); err != nil {
			return err
		}
		if !isDeploymentAvailable(deployment) {
			nonReadyDeployments = append(nonReadyDeployments, deployment.Name)
		}
	}

	if len(nonReadyDeployments) > 0 {
		status.MarkDeploymentsNotReady(nonReadyDeployments)
		return deploymentsNotReadyError{}
	}

	status.MarkDeploymentsAvailable()
	return nil
}

func isDeploymentAvailable(d *appsv1.Deployment) bool {
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
