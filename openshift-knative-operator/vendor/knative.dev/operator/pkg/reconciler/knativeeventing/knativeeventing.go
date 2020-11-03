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
	"fmt"

	mf "github.com/manifestival/manifestival"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientset "knative.dev/operator/pkg/client/clientset/versioned"

	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	knereconciler "knative.dev/operator/pkg/client/injection/reconciler/operator/v1alpha1/knativeeventing"
	"knative.dev/operator/pkg/reconciler/common"
	kec "knative.dev/operator/pkg/reconciler/knativeeventing/common"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

// Reconciler implements controller.Reconciler for KnativeEventing resources.
type Reconciler struct {
	// kubeClientSet allows us to talk to the k8s for core APIs
	kubeClientSet kubernetes.Interface
	// kubeClientSet allows us to talk to the k8s for operator APIs
	operatorClientSet clientset.Interface
	// manifest is empty, but with a valid client and logger. all
	// manifests are immutable, and any created during reconcile are
	// expected to be appended to this one, obviating the passing of
	// client & logger
	manifest mf.Manifest
	// Platform-specific behavior to affect the transform
	extension common.Extension
}

// Check that our Reconciler implements controller.Reconciler
var _ knereconciler.Interface = (*Reconciler)(nil)
var _ knereconciler.Finalizer = (*Reconciler)(nil)

// FinalizeKind removes all resources after deletion of a KnativeEventing.
func (r *Reconciler) FinalizeKind(ctx context.Context, original *v1alpha1.KnativeEventing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)

	// List all KnativeEventings to determine if cluster-scoped resources should be deleted.
	kes, err := r.operatorClientSet.OperatorV1alpha1().KnativeEventings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all KnativeEventings: %w", err)
	}

	for _, ke := range kes.Items {
		if ke.GetDeletionTimestamp().IsZero() {
			// Not deleting all KnativeEventings. Nothing to do here.
			return nil
		}
	}

	if err := r.extension.Finalize(ctx, original); err != nil {
		logger.Error("Failed to finalize platform resources", err)
	}
	logger.Info("Deleting cluster-scoped resources")
	manifest, err := r.installed(ctx, original)
	if err != nil {
		logger.Error("Unable to fetch installed manifest; no cluster-scoped resources will be finalized", err)
		return nil
	}
	return common.Uninstall(manifest)
}

// ReconcileKind compares the actual state with the desired, and attempts to
// converge the two.
func (r *Reconciler) ReconcileKind(ctx context.Context, ke *v1alpha1.KnativeEventing) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	ke.Status.InitializeConditions()
	ke.Status.ObservedGeneration = ke.Generation

	logger.Infow("Reconciling KnativeEventing", "status", ke.Status)

	if !common.IsUpDowngradeEligible(ke) {
		msg := fmt.Errorf("It is not supported to upgrade or downgrade across multiple MINOR versions. The "+
			"installed KnativeEventing version is %v.", ke.Status.Version)
		ke.Status.MarkVersionMigrationNotEligible(msg.Error())
		return nil
	} else {
		ke.Status.MarkVersionMigrationEligible()
	}

	if err := r.extension.Reconcile(ctx, ke); err != nil {
		return err
	}
	stages := common.Stages{
		common.AppendTarget,
		r.transform,
		common.Install,
		common.CheckDeployments,
		common.DeleteObsoleteResources(ctx, ke, r.installed),
	}
	manifest := r.manifest.Append()
	return stages.Execute(ctx, &manifest, ke)
}

// transform mutates the passed manifest to one with common, component
// and platform transformations applied
func (r *Reconciler) transform(ctx context.Context, manifest *mf.Manifest, comp v1alpha1.KComponent) error {
	logger := logging.FromContext(ctx)
	instance := comp.(*v1alpha1.KnativeEventing)
	extra := []mf.Transformer{
		kec.DefaultBrokerConfigMapTransform(instance, logger),
		kec.SinkBindingSelectionModeTransform(instance, logger),
	}
	extra = append(extra, r.extension.Transformers(instance)...)
	return common.Transform(ctx, manifest, instance, extra...)
}

func (r *Reconciler) installed(ctx context.Context, instance v1alpha1.KComponent) (*mf.Manifest, error) {
	// Create new, empty manifest with valid client and logger
	installed := r.manifest.Append()
	stages := common.Stages{common.AppendInstalled, r.transform}
	err := stages.Execute(ctx, &installed, instance)
	return &installed, err
}
