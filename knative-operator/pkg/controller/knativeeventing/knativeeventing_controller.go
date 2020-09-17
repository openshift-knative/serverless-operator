package knativeeventing

import (
	"context"
	"fmt"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/controller/dashboard"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
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

// This needs to remain "knative-eventing-openshift" to be compatible with earlier versions.
const finalizerName = "knative-eventing-openshift"

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
	return c.Watch(&source.Kind{Type: &eventingv1alpha1.KnativeEventing{}}, &handler.EnqueueRequestForObject{})
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

	if instance.Status.IsReady() {
		common.KnativeEventingUpG.Set(1)
	} else {
		common.KnativeEventingUpG.Set(0)
	}
	return reconcile.Result{}, reconcileErr
}

func (r *ReconcileKnativeEventing) reconcileKnativeEventing(instance *eventingv1alpha1.KnativeEventing) error {
	stages := []func(*eventingv1alpha1.KnativeEventing) error{
		r.configure,
		r.ensureFinalizers,
		r.installServiceMonitors,
		r.installDashboards,
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
	common.MutateEventing(instance)
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

// set a finalizer to clean up the dashboard when instance is deleted
func (r *ReconcileKnativeEventing) ensureFinalizers(instance *eventingv1alpha1.KnativeEventing) error {
	for _, finalizer := range instance.GetFinalizers() {
		if finalizer == finalizerName {
			return nil
		}
	}
	log.Info("Adding finalizer")
	instance.SetFinalizers(append(instance.GetFinalizers(), finalizerName))
	return r.client.Update(context.TODO(), instance)
}

// installServiceMonitors installs service monitors for eventing dashboards
func (r *ReconcileKnativeEventing) installServiceMonitors(instance *eventingv1alpha1.KnativeEventing) error {
	log.Info("Installing Eventing Service Monitors")
	if err := common.SetupMonitoringRequirements("knative-eventing", r.client); err != nil {
		return err
	}
	if err := common.SetupEventingServiceMonitors("knative-eventing", instance); err != nil {
		return err
	}
	return nil
}

// installDashboard installs dashboard for OpenShift webconsole
func (r *ReconcileKnativeEventing) installDashboards(instance *eventingv1alpha1.KnativeEventing) error {
	log.Info("Installing Eventing Dashboards")
	if err := dashboard.Apply(dashboard.EventingBrokerDashboardPath, common.SetOwnerAnnotationsEventing(instance), r.client); err != nil {
		return err
	}
	if err := dashboard.Apply(dashboard.EventingFilterDashboardPath, common.SetOwnerAnnotationsEventing(instance), r.client); err != nil {
		return err
	}
	return nil
}

// general clean-up, mostly resources in different namespaces from eventingv1alpha1.KnativeEventing.
func (r *ReconcileKnativeEventing) delete(instance *eventingv1alpha1.KnativeEventing) error {
	finalizers := sets.NewString(instance.GetFinalizers()...)

	if !finalizers.Has(finalizerName) {
		log.Info("Finalizer has already been removed, nothing to do")
		return nil
	}
	log.Info("Running cleanup logic")
	log.Info("Deleting eventing dashboards")
	if err := dashboard.Delete(dashboard.EventingBrokerDashboardPath, common.SetOwnerAnnotationsEventing(instance), r.client); err != nil {
		return fmt.Errorf("failed to delete dashboard broker configmap: %w", err)
	}
	if err := dashboard.Delete(dashboard.EventingFilterDashboardPath, common.SetOwnerAnnotationsEventing(instance), r.client); err != nil {
		return fmt.Errorf("failed to delete dashboard filter configmap: %w", err)
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
