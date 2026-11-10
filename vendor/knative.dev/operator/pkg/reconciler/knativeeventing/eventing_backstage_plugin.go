/*
Copyright 2023 The Knative Authors

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

package knativeeventing

import (
	"context"
	"fmt"

	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/eventing/pkg/apis/feature"

	"knative.dev/operator/pkg/apis/operator/base"
	"knative.dev/operator/pkg/apis/operator/v1beta1"
)

var (
	backstageResourcesPred = byLabel("app.kubernetes.io/component", "eventmesh-backend")
)

const (
	backstageBackendFeatureFlag = "backstage-backend"
)

func (r *Reconciler) handleBackstageResources(ctx context.Context, manifests *mf.Manifest, comp base.KComponent) error {
	instance := comp.(*v1beta1.KnativeEventing)

	if isBackstageEnabled(instance) {
		return nil
	}

	// Delete Backstage resources (if present)
	toBeDeleted := manifests.Filter(backstageResourcesPred)
	if err := toBeDeleted.Delete(mf.IgnoreNotFound(true)); err != nil && !meta.IsNoMatchError(err) {
		return fmt.Errorf("failed to delete backstage resources: %v", err)
	}

	// Filter out Backstage resources from the final list of manifests
	*manifests = manifests.Filter(mf.Not(backstageResourcesPred))

	return nil
}

func byLabel(key, value string) mf.Predicate {
	return func(u *unstructured.Unstructured) bool {
		v, ok := u.GetLabels()[key]
		return ok && v == value
	}
}

func isBackstageEnabled(instance *v1beta1.KnativeEventing) bool {
	cmData, ok := getFeaturesConfig(instance)
	if !ok {
		return false
	}

	f, err := feature.NewFlagsConfigFromConfigMap(&corev1.ConfigMap{Data: cmData})
	if err != nil {
		return false
	}

	return f.IsEnabled(backstageBackendFeatureFlag)
}
