package health

import (
	"context"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = common.Log.WithName("health-controller")

// Add creates a new Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileHealthDashboard{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("health-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(source.Kind(mgr.GetCache(), client.Object(&corev1.ConfigMap{}),
		common.EnqueueRequestByOwnerAnnotations(common.ServerlessOperatorOwnerName, common.ServerlessOperatorOwnerNamespace), skipCreatePredicate{}))
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileHealthDashboard implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileHealthDashboard{}

// ReconcileHealthDashboard reconciles a HealthDashboard configmap object
type ReconcileHealthDashboard struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a HealthDashboard
func (r *ReconcileHealthDashboard) Reconcile(_ context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling HealthDashboard")
	// in any case restore the current health dashboard, since the configmap shouldn't
	// be modified, if the configmap has not changed this will not trigger a real update
	err := InstallHealthDashboard(r.client)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

type skipCreatePredicate struct {
	predicate.Funcs
}

// since operator is responsible to create the dashboard no need to process it
func (skipCreatePredicate) Create(_ event.CreateEvent) bool {
	return false
}
