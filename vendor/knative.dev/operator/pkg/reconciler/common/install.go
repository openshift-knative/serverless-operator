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
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/api/errors"

	mf "github.com/manifestival/manifestival"
	"knative.dev/pkg/logging"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"

)

var (
	role            mf.Predicate = mf.Any(mf.ByKind("ClusterRole"), mf.ByKind("Role"))
	rolebinding     mf.Predicate = mf.Any(mf.ByKind("ClusterRoleBinding"), mf.ByKind("RoleBinding"))
	webhook         mf.Predicate = mf.Any(mf.ByKind("MutatingWebhookConfiguration"), mf.ByKind("ValidatingWebhookConfiguration"))
	gatewayNotMatch              = "no matches for kind \"Gateway\""
	Interval = 10 * time.Second
	// Timeout specifies the timeout for the function PollImmediate to reach a certain status.
	Timeout                   = 5 * time.Minute
)

// Install applies the manifest resources for the given version and updates the given
// status accordingly.
func Install(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) error {
	logger := logging.FromContext(ctx)
	logger.Debug("Installing manifest")
	status := instance.GetStatus()
	// The Operator needs a higher level of permissions if it 'bind's non-existent roles.
	// To avoid this, we strictly order the manifest application as (Cluster)Roles, then
	// (Cluster)RoleBindings, then the rest of the manifest.
	if err := manifest.Filter(role).Apply(); err != nil {
		status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply (cluster)roles: %w", err)
	}
	if err := manifest.Filter(rolebinding).Apply(); err != nil {
		status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply (cluster)rolebindings: %w", err)
	}
	if err := manifest.Filter(mf.Not(mf.Any(role, rolebinding, webhook))).Apply(); err != nil {
		status.MarkInstallFailed(err.Error())
		if ks, ok := instance.(*v1beta1.KnativeServing); ok && strings.Contains(err.Error(), gatewayNotMatch) &&
			(ks.Spec.Ingress == nil || ks.Spec.Ingress.Istio.Enabled) {
			errMessage := fmt.Errorf("please install istio or disable the istio ingress plugin: %w", err)
			status.MarkInstallFailed(errMessage.Error())
			return errMessage
		}

		return fmt.Errorf("failed to apply non rbac manifest: %w", err)
	}
	if waitErr := wait.PollUntilContextTimeout(ctx, Interval, Timeout, true, func(_ context.Context) (bool, error) {
		return checkWebhookServices(ctx, manifest, instance)
	}); waitErr != nil {
		return  fmt.Errorf("Webhook services are not ready: %w", waitErr)
	}

	if err := manifest.Filter(webhook).Apply(); err != nil {
		status.MarkInstallFailed(err.Error())
		return fmt.Errorf("failed to apply webhooks: %w", err)
	}
	status.MarkInstallSucceeded()
	status.SetVersion(TargetVersion(instance))
	return nil
}

// Uninstall removes all resources except CRDs, which are never deleted automatically.
func Uninstall(manifest *mf.Manifest) error {
	if err := manifest.Filter(mf.NoCRDs, mf.Not(mf.Any(role, rolebinding))).Delete(mf.IgnoreNotFound(true)); err != nil {
		return fmt.Errorf("failed to remove non-crd/non-rbac resources: %w", err)
	}
	// Delete Roles last, as they may be useful for human operators to clean up.
	if err := manifest.Filter(mf.Any(role, rolebinding)).Delete(mf.IgnoreNotFound(true)); err != nil {
		return fmt.Errorf("failed to remove rbac: %w", err)
	}
	return nil
}

func checkWebhookServices(ctx context.Context, manifest *mf.Manifest, instance base.KComponent) (bool, error) {
	for _, u := range manifest.Filter(mf.All(mf.ByKind("Service"), byNameSuffix("webhook"))).Resources() {
		resource, err := manifest.Client.Get(&u)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		svc := &corev1.Service{}
		if err := scheme.Scheme.Convert(resource, svc, nil); err != nil {
			return false, err
		}
		if ready, err := isSvcReady(ctx, manifest, svc); !ready {
			return false, err
		}
	}
	return true, nil
}

func isSvcReady(ctx context.Context, manifest *mf.Manifest, svc *corev1.Service) (bool, error) {
	result := unstructured.Unstructured{}
	result.SetName(svc.Name)
	svcGvk := schema.GroupVersionKind{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "Endpoints",
	}
	result.SetGroupVersionKind(svcGvk)
	result.SetNamespace(svc.Namespace)
	endpointsU, err := manifest.Client.Get(&result)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	epts := &corev1.Endpoints{}
	if err := scheme.Scheme.Convert(endpointsU, epts, nil); err != nil {
		return false, err
	}
	for _, subset := range epts.Subsets {
		if len(subset.Addresses) > 0 {
			return true, nil // At least one endpoint is available
		}
	}
	return false, nil
}

func byNameSuffix(suffix string) mf.Predicate {
	return func(u *unstructured.Unstructured) bool {
		return strings.HasSuffix(u.GetName(), suffix)
	}
}

