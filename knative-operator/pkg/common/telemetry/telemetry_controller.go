package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/runtime"
	kafkasourcev1beta1 "knative.dev/eventing-contrib/kafka/source/pkg/apis/sources/v1beta1"
	eventingsourcesv1beta1 "knative.dev/eventing/pkg/apis/sources/v1beta1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type op int

const (
	inc op = iota
	dec
)

var typesAndMetrics = []runtime.Object{
	&servingv1.Service{},
	&servingv1.Revision{},
	&servingv1.Route{},
	&eventingsourcesv1beta1.PingSource{},
	&eventingsourcesv1beta1.ApiServerSource{},
}

// Add creates a new Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileResourcesForTelemetry{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("telemetry-resources-discovery-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	for _, tp := range typesAndMetrics {
		if err := c.Watch(&source.Kind{Type: tp}, &handler.EnqueueRequestForObject{}, skipUpdatePredicate{}, updateMetricsDeletePredicate{}, updateMetricsCreatePredicate{}); err != nil {
			return err
		}
	}
	return nil
}

// blank assignment to verify that ReconcileResourcesForTelemetry implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileResourcesForTelemetry{}

// ReconcileResourcesForTelemetry reconciles a source deployment object
type ReconcileResourcesForTelemetry struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for different resources, no actual update happens here
func (r *ReconcileResourcesForTelemetry) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

type skipUpdatePredicate struct {
	predicate.Funcs
}

func (skipUpdatePredicate) Update(_ event.UpdateEvent) bool {
	return false
}

type updateMetricsCreatePredicate struct {
	predicate.Funcs
}

func (updateMetricsCreatePredicate) Create(e event.CreateEvent) bool {
	matchAndUpdateMetric(e.Object, inc)
	return true
}

type updateMetricsDeletePredicate struct {
	predicate.Funcs
}

func (updateMetricsDeletePredicate) Delete(e event.DeleteEvent) bool {
	matchAndUpdateMetric(e.Object, dec)
	return true
}

func matchAndUpdateMetric(obj runtime.Object, update op) {
	switch obj.(type) {
	case *servingv1.Service:
		updateMetric(ServicesG, update)
	case *servingv1.Revision:
		updateMetric(RevisionsG, update)
	case *servingv1.Route:
		updateMetric(RoutesG, update)
	case *eventingsourcesv1beta1.PingSource, *eventingsourcesv1beta1.ApiServerSource, *eventingsourcesv1beta1.SinkBinding, *kafkasourcev1beta1.KafkaSource:
		updateMetric(SourcesG, update)
	}
}

func updateMetric(metric prometheus.Gauge, update op) {
	switch update {
	case inc:
		metric.Inc()
	case dec:
		metric.Dec()
	}
}
