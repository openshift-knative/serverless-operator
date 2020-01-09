package ingress

import (
	"context"
	"reflect"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/controller/common"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/logging"
	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new Ingress Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client := mgr.GetClient()
	return &ReconcileIngress{
		base: &common.BaseIngressReconciler{
			Client: client,
		},
		client:   client,
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetRecorder("knative-openshift-ingress"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("ingress-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Ingress
	err = c.Watch(&source.Kind{Type: &networkingv1alpha1.Ingress{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Routes and requeue the
	// owner Ingress
	err = c.Watch(&source.Kind{Type: &routev1.Route{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &networkingv1alpha1.Ingress{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileIngress{}

// ReconcileIngress reconciles an Ingress object
type ReconcileIngress struct {
	base *common.BaseIngressReconciler
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Ingress
// object and makes changes based on the state read and what is in the
// Ingress.Spec
//
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileIngress) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()
	logger := logging.FromContext(ctx)

	// Fetch the Ingress instance
	original := &networkingv1alpha1.Ingress{}
	err := r.client.Get(context.TODO(), request.NamespacedName, original)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	// Don't modify the informer's copy
	ci := original.DeepCopy()
	if newFinalizer, change := common.AppendIfAbsent(ci.Finalizers, "ocp-ingress"); change {
		ci.Finalizers = newFinalizer
		if err := r.client.Update(context.TODO(), ci); err != nil {
			return reconcile.Result{}, nil
		}
	}
	reconcileErr := r.base.ReconcileIngress(ctx, ci)
	if equality.Semantic.DeepEqual(original.Status, ci.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else if _, err := r.updateStatus(ctx, ci); err != nil {
		logger.Errorf("Failed to update ingress status %v", err)
		r.recorder.Event(ci, corev1.EventTypeWarning, "SyncError", err.Error())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, reconcileErr
}

// Update the Status of the Ingress.  Caller is responsible for checking
// for semantic differences before calling.
func (r *ReconcileIngress) updateStatus(ctx context.Context, desired *networkingv1alpha1.Ingress) (*networkingv1alpha1.Ingress, error) {
	ci := &networkingv1alpha1.Ingress{}
	err := r.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, ci)
	if err != nil {
		return nil, err
	}

	// If there's nothing to update, just return.
	if reflect.DeepEqual(ci.Status, desired.Status) {
		return ci, nil
	}
	// Don't modify the informers copy
	existing := ci.DeepCopy()
	existing.Status = desired.Status
	err = r.client.Status().Update(ctx, existing)
	return existing, err
}
