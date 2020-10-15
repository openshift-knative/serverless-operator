package sources

import (
	"context"
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = common.Log.WithName("source-deployment-discovery-controller")

// Add creates a new Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSourceDeployment{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("source-deployment-discovery-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// common function to enqueue reconcile requests for resources
	enqueueRequests := handler.ToRequestsFunc(func(obj handler.MapObject) []reconcile.Request {
		dep := obj.Object.(*v1.Deployment)
		sourceLabel := dep.Spec.Selector.MatchLabels[common.SourceLabel]
		sourceNameLabel := dep.Spec.Selector.MatchLabels[common.SourceNameLabel]
		sourceRoleLabel := dep.Spec.Selector.MatchLabels[common.SourceRoleLabel]

		if (sourceLabel != "" && sourceNameLabel != "") || (sourceLabel != "" && sourceRoleLabel != "") {
			return []reconcile.Request{{
				NamespacedName: types.NamespacedName{Namespace: obj.Meta.GetNamespace(), Name: obj.Meta.GetName()},
			}}
		}
		return nil
	})
	err = c.Watch(&source.Kind{Type: &v1.Deployment{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: enqueueRequests}, skipDeletePredicate{}, skipUpdatePredicate{})
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileSourceDeployment implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSourceDeployment{}

// ReconcileSourceDeployment reconciles a source deployment object
type ReconcileSourceDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for an eventing source deployment
func (r *ReconcileSourceDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling the source deployment, setting up a service/service monitor if required")
	dep := &v1.Deployment{}
	if err := r.client.Get(context.TODO(), request.NamespacedName, dep); err != nil {
		// the deployment does not exist anymore do nothing
		return reconcile.Result{}, nil
	}
	if err := common.SetupMonitoringRequirements(r.client, dep); err != nil {
		return reconcile.Result{}, err
	}
	if err := common.SetupSourceServiceMonitor(r.client, dep); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

type skipDeletePredicate struct {
	predicate.Funcs
}

func (skipDeletePredicate) Delete(e event.DeleteEvent) bool {
	return false
}

type skipUpdatePredicate struct {
	predicate.Funcs
}

func (skipUpdatePredicate) Update(e event.UpdateEvent) bool {
	return false
}
