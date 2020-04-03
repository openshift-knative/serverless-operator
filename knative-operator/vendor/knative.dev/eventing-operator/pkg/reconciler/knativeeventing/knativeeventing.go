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

package knativeeventing

import (
	"context"
	"reflect"

	mf "github.com/manifestival/manifestival"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"

	eventingv1alpha1 "knative.dev/eventing-operator/pkg/apis/eventing/v1alpha1"
	listers "knative.dev/eventing-operator/pkg/client/listers/eventing/v1alpha1"
	"knative.dev/eventing-operator/pkg/reconciler"
	"knative.dev/eventing-operator/pkg/reconciler/knativeeventing/common"
	"knative.dev/eventing-operator/version"
	"knative.dev/pkg/controller"
)

var (
	platform common.Platforms
)

// Reconciler implements controller.Reconciler for Knativeeventing resources.
type Reconciler struct {
	*reconciler.Base
	// Listers index properties about resources
	knativeEventingLister listers.KnativeEventingLister
	config                mf.Manifest
	eventings             sets.String
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

// Reconcile compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Knativeeventing resource
// with the current status of the resource.
func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		r.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}
	// Get the KnativeEventing resource with this namespace/name.
	original, err := r.knativeEventingLister.KnativeEventings(namespace).Get(name)
	if apierrs.IsNotFound(err) {
		// The resource was deleted
		r.eventings.Delete(key)
		if r.eventings.Len() == 0 {
			r.config.Filter(mf.NotCRDs).Delete()
		}
		return nil

	} else if err != nil {
		r.Logger.Error(err, "Error getting KnativeEventings")
		return err
	}
	// Keep track of the number of Eventings in the cluster
	r.eventings.Insert(key)

	// Don't modify the informers copy.
	knativeEventing := original.DeepCopy()

	// Reconcile this copy of the KnativeEventing resource and then write back any status
	// updates regardless of whether the reconciliation errored out.
	reconcileErr := r.reconcile(ctx, knativeEventing)
	if equality.Semantic.DeepEqual(original.Status, knativeEventing.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else if _, err = r.updateStatus(knativeEventing); err != nil {
		r.Logger.Warnw("Failed to update KnativeEventing status", zap.Error(err))
		r.Recorder.Eventf(knativeEventing, corev1.EventTypeWarning, "UpdateFailed",
			"Failed to update status for KnativeEventing %q: %v", knativeEventing.Name, err)
		return err
	}
	if reconcileErr != nil {
		r.Recorder.Event(knativeEventing, corev1.EventTypeWarning, "InternalError", reconcileErr.Error())
		return reconcileErr
	}
	return nil
}

func (r *Reconciler) reconcile(ctx context.Context, ke *eventingv1alpha1.KnativeEventing) error {
	reqLogger := r.Logger.With(zap.String("Request.Namespace", ke.Namespace)).With("Request.Name", ke.Name)
	reqLogger.Infow("Reconciling KnativeEventing", "status", ke.Status)

	stages := []func(*mf.Manifest, *eventingv1alpha1.KnativeEventing) error{
		r.initStatus,
		r.install,
		r.checkDeployments,
		r.deleteObsoleteResources,
	}

	manifest, err := r.transform(ke)
	if err != nil {
		return err
	}

	for _, stage := range stages {
		if err := stage(&manifest, ke); err != nil {
			return err
		}
	}
	reqLogger.Infow("Reconcile stages complete", "status", ke.Status)
	return nil
}

func (r *Reconciler) initStatus(_ *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
	r.Logger.Debug("Initializing status")
	if len(ke.Status.Conditions) == 0 {
		ke.Status.InitializeConditions()
		if _, err := r.updateStatus(ke); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) transform(instance *eventingv1alpha1.KnativeEventing) (mf.Manifest, error) {
	r.Logger.Debug("Transforming manifest")
	transforms, err := platform.Transformers(r.KubeClientSet, instance, r.Logger)
	if err != nil {
		return mf.Manifest{}, err
	}
	return r.config.Transform(transforms...)
}

func (r *Reconciler) install(manifest *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
	r.Logger.Debug("Installing manifest")
	defer r.updateStatus(ke)
	if err := manifest.Apply(); err != nil {
		ke.Status.MarkEventingFailed("Manifest Installation", err.Error())
		return err
	}
	ke.Status.Version = version.Version
	ke.Status.MarkInstallationReady()
	return nil
}

func (r *Reconciler) checkDeployments(manifest *mf.Manifest, ke *eventingv1alpha1.KnativeEventing) error {
	r.Logger.Debug("Checking deployments")
	defer r.updateStatus(ke)
	available := func(d *appsv1.Deployment) bool {
		for _, c := range d.Status.Conditions {
			if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	}
	for _, u := range manifest.Filter(mf.ByKind("Deployment")).Resources() {
		deployment, err := r.KubeClientSet.AppsV1().Deployments(u.GetNamespace()).Get(u.GetName(), metav1.GetOptions{})
		if err != nil {
			ke.Status.MarkEventingNotReady("Deployment check", err.Error())
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		if !available(deployment) {
			ke.Status.MarkEventingNotReady("Deployment check", "The deployment is not available.")
			return nil
		}
	}
	ke.Status.MarkEventingReady()
	return nil
}

func (r *Reconciler) updateStatus(desired *eventingv1alpha1.KnativeEventing) (*eventingv1alpha1.KnativeEventing, error) {
	ke, err := r.KnativeEventingClientSet.OperatorV1alpha1().KnativeEventings(desired.Namespace).Get(desired.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// If there's nothing to update, just return.
	if reflect.DeepEqual(ke.Status, desired.Status) {
		return ke, nil
	}
	// Don't modify the informers copy
	existing := ke.DeepCopy()
	existing.Status = desired.Status
	return r.KnativeEventingClientSet.OperatorV1alpha1().KnativeEventings(desired.Namespace).UpdateStatus(existing)
}

// Delete obsolete resources from previous versions
func (r *Reconciler) deleteObsoleteResources(manifest *mf.Manifest, instance *eventingv1alpha1.KnativeEventing) error {
	resource := &unstructured.Unstructured{}
	resource.SetNamespace(instance.GetNamespace())

	// Remove old resources from 0.12
	// https://github.com/knative/eventing-operator/issues/90
	// sources and controller are merged.
	// delete removed or renamed resources.
	resource.SetAPIVersion("v1")
	resource.SetKind("ServiceAccount")
	resource.SetName("eventing-source-controller")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	resource.SetAPIVersion("rbac.authorization.k8s.io/v1")
	resource.SetKind("ClusterRole")
	resource.SetName("knative-eventing-source-controller")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	resource.SetAPIVersion("rbac.authorization.k8s.io/v1")
	resource.SetKind("ClusterRoleBinding")
	resource.SetName("eventing-source-controller")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	resource.SetAPIVersion("rbac.authorization.k8s.io/v1")
	resource.SetKind("ClusterRoleBinding")
	resource.SetName("eventing-source-controller-resolver")
	if err := manifest.Client.Delete(resource); err != nil {
		return err
	}

	return nil
}
