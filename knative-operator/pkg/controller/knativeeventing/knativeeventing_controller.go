package knativeeventing

import (
	"context"
	"fmt"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/operator-framework/operator-sdk/pkg/predicate"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	eventingv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This needs to remain "knative-eventing-openshift" to be compatible with earlier versions.
	finalizerName = "knative-eventing-openshift"
)

var log = common.Log.WithName("controller")

// Add creates a new KnativeEventing Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKnativeEventing{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("knativeeventing-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KnativeEventing
	return c.Watch(&source.Kind{Type: &eventingv1alpha1.KnativeEventing{}}, &handler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{})
}

// blank assignment to verify that ReconcileKnativeEventing implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKnativeEventing{}

// ReconcileKnativeEventing reconciles a KnativeEventing object
type ReconcileKnativeEventing struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a KnativeEventing
func (r *ReconcileKnativeEventing) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KnativeEventing")

	// Fetch the KnativeEventing instance
	original := &eventingv1alpha1.KnativeEventing{}
	err := r.client.Get(context.TODO(), request.NamespacedName, original)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if original.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, r.delete(original)
	}

	instance := original.DeepCopy()
	reconcileErr := r.reconcileKnativeEventing(instance)

	if !equality.Semantic.DeepEqual(original.Status, instance.Status) {
		if err := r.client.Status().Update(context.TODO(), instance); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update status: %w", err)
		}
	}
	return reconcile.Result{}, reconcileErr
}

func (r *ReconcileKnativeEventing) reconcileKnativeEventing(instance *eventingv1alpha1.KnativeEventing) error {
	stages := []func(*eventingv1alpha1.KnativeEventing) error{
		r.configure,
		r.ensureFinalizers,
	}
	for _, stage := range stages {
		if err := stage(instance); err != nil {
			return err
		}
	}
	return nil
}

// configure default settings for OpenShift
func (r *ReconcileKnativeEventing) configure(instance *eventingv1alpha1.KnativeEventing) error {
	before := instance.DeepCopy()
	if err := common.MutateEventing(instance, r.client); err != nil {
		return err
	}
	if equality.Semantic.DeepEqual(before.Spec, instance.Spec) {
		return nil
	}

	// Only apply the update if something changed.
	log.Info("Updating KnativeEventing with mutated state for Openshift")
	if err := r.client.Update(context.TODO(), instance); err != nil {
		return fmt.Errorf("failed to update KnativeEventing with mutated state: %w", err)
	}
	return nil
}

// set a finalizer to clean up hanging resources when instance is deleted
func (r *ReconcileKnativeEventing) ensureFinalizers(instance *eventingv1alpha1.KnativeEventing) error {
	for _, finalizer := range instance.GetFinalizers() {
		if finalizer == finalizerName {
			return nil
		}
	}
	log.Info("Adding finalizer to KnativeEventing")
	instance.SetFinalizers(append(instance.GetFinalizers(), finalizerName))
	return r.client.Update(context.TODO(), instance)
}

// general clean-up, like deletion of hanging resources
func (r *ReconcileKnativeEventing) delete(instance *eventingv1alpha1.KnativeEventing) error {
	finalizers := sets.NewString(instance.GetFinalizers()...)

	if !finalizers.Has(finalizerName) {
		log.Info("Finalizer has already been removed from KnativeEventing, nothing to do")
		return nil
	}

	log.Info("Running cleanup logic for KnativeEventing")
	if err := r.deleteHangingResources(instance); err != nil {
		return fmt.Errorf("failed to delete hanging resources for KnativeEventing: %w", err)
	}

	// The above might take a while, so we refetch the resource again in case it has changed.
	refetched := &eventingv1alpha1.KnativeEventing{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}, refetched); err != nil {
		return fmt.Errorf("failed to refetch KnativeEventing: %w", err)
	}

	// Update the refetched finalizer list.
	finalizers = sets.NewString(refetched.GetFinalizers()...)
	finalizers.Delete(finalizerName)
	refetched.SetFinalizers(finalizers.List())

	if err := r.client.Update(context.TODO(), refetched); err != nil {
		return fmt.Errorf("failed to update KnativeEventing with removed finalizer: %w", err)
	}
	return nil
}

func (r *ReconcileKnativeEventing) deleteHangingResources(instance *eventingv1alpha1.KnativeEventing) error {
	log.Info("Deleting hanging resources for KnativeEventing")

	deployment := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: instance.Namespace, Name: "pingsource-jobrunner"}, deployment)
	if apierrors.IsNotFound(err) {
		// We can safely ignore this. There is nothing to do for us.
		log.Info("Pingsource jobrunner deployment not found, nothing to do") // TODO
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to fetch pingsource-jobrunner deployment: %w", err)
	}

	log.Info("Pingsource jobrunner deployment found, deleting it") // TODO

	if err := r.client.Delete(context.TODO(), deployment); err != nil {
		return fmt.Errorf("failed to remove pingsource-jobrunner deployment: %w", err)
	}

	return nil
}
