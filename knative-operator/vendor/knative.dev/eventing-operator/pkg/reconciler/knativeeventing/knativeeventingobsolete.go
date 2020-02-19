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

package knativeeventing

import (
	"context"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
	listers "knative.dev/eventing-operator/pkg/client/listers/eventing/v1alpha1"
	"knative.dev/eventing-operator/pkg/reconciler"
	"knative.dev/pkg/controller"
)

// ReconcilerObsolete implements controller.Reconciler for Eventing resources of the version 0.11.0 or prior.
type ReconcilerObsolete struct {
	*reconciler.Base
	// Listers index properties about resources
	knativeEventingObsoleteLister listers.EventingLister
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*ReconcilerObsolete)(nil)

// Reconcile compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Eventing resource
// with the current status of the resource.
func (r *ReconcilerObsolete) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name.
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		r.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}
	// Get the obsolete Eventing resource with this namespace/name.
	original, err := r.knativeEventingObsoleteLister.Eventings(namespace).Get(name)
	if apierrs.IsNotFound(err) {
		r.Logger.Info("No need to convert the CR of the old version, since the old CR does not exist")
		return nil

	} else if err != nil {
		r.Logger.Error(err, "Error getting the obsolete CR")
		return err
	}

	// Check whether the current KnativeEventing resource exists
	if knativeEventings, errorKE := r.KnativeEventingClientSet.OperatorV1alpha1().KnativeEventings(namespace).List(metav1.ListOptions{}); !apierrs.IsNotFound(errorKE) && len(knativeEventings.Items) != 0 {
		// We already have a converted CR or a new CR, so abort it.
		return nil
	} else {
		// There is CR of an old version, and we convert it into the new CR.
		knativeEventingObsolete := original.DeepCopy()

		// Remove finalizers to prevent deadlock.
		if len(knativeEventingObsolete.GetFinalizers()) > 0 {
			r.Logger.Info("Removing finalizers for old CR")
			knativeEventingObsolete.SetFinalizers(nil)
			if _, err := r.KnativeEventingClientSet.OperatorV1alpha1().Eventings(namespace).Update(knativeEventingObsolete); err != nil {
				return err
			}
		}

		// Create the latest CR from the current (previous) CR. Since the old spec in the CR is empty, there is nothing
		// we need to copy from the old CR.
		latest := &v1alpha1.KnativeEventing{
			ObjectMeta: metav1.ObjectMeta{
				Name:      knativeEventingObsolete.Name,
				Namespace: knativeEventingObsolete.Namespace,
			},
		}

		// Create the CR of the new version in the cluster.
		if _, err = r.KnativeEventingClientSet.OperatorV1alpha1().KnativeEventings(namespace).Create(latest); err != nil {
			return err
		}
		return nil
	}
	return nil
}
