package telemetry

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/runtime"
	kafkasourcev1beta1 "knative.dev/eventing-contrib/kafka/source/pkg/apis/sources/v1beta1"
	eventingsourcesv1beta1 "knative.dev/eventing/pkg/apis/sources/v1beta1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	ServingObjects = []runtime.Object{
		&servingv1.Service{},
		&servingv1.Revision{},
		&servingv1.Route{},
	}

	EventingObjects = []runtime.Object{
		&eventingsourcesv1beta1.PingSource{},
		&eventingsourcesv1beta1.ApiServerSource{},
		&eventingsourcesv1beta1.SinkBinding{},
	}

	KnativeKafkaObjects = []runtime.Object{
		&kafkasourcev1beta1.KafkaSource{},
	}
)

// newTelemetryController creates an unmanaged controller for watching Telemetry resources
func newTelemetryController(name string, objects []runtime.Object, mgr manager.Manager, telemetry *Telemetry) (*controller.Controller, error) {
	// Create a new controller
	c, err := controller.NewUnmanaged(fmt.Sprintf("telemetry-resources-%s-controller-%s", name, uuid.New().String()), mgr, controller.Options{
		Reconciler: reconcile.Func(func(reconcile.Request) (reconcile.Result, error) { // No actual update happens here
			return reconcile.Result{}, nil
		}),
	})
	if err != nil {
		return nil, err
	}
	for _, tp := range objects {
		if err := c.Watch(&source.Kind{Type: tp}, &handler.EnqueueRequestForObject{}, metricsPredicate{
			telemetry: telemetry,
		}); err != nil {
			return nil, err
		}
	}
	return &c, nil
}

type metricsPredicate struct {
	predicate.Funcs
	telemetry *Telemetry
}

func (metricsPredicate) Update(_ event.UpdateEvent) bool {
	return false
}

func (mp metricsPredicate) Create(e event.CreateEvent) bool {
	if metric := getMetricFor(e.Object); metric != nil {
		if !mp.telemetry.inSnapshot(e.Meta) { // skip if we have seen this at controller creation time, so we don'telemetry count it twice
			metric.Inc()
		}
	}
	return false
}

func (mp metricsPredicate) Delete(e event.DeleteEvent) bool {
	if metric := getMetricFor(e.Object); metric != nil {
		if mp.telemetry.inSnapshot(e.Meta) {
			mp.telemetry.deleteFromSnaphost(e.Meta)
		}
		metric.Dec()
	}
	return false
}

func getMetricFor(obj runtime.Object) prometheus.Gauge {
	switch obj.(type) {
	case *servingv1.Service:
		return servicesG
	case *servingv1.Revision:
		return revisionsG
	case *servingv1.Route:
		return routesG
	case *eventingsourcesv1beta1.PingSource, *eventingsourcesv1beta1.ApiServerSource, *eventingsourcesv1beta1.SinkBinding, *kafkasourcev1beta1.KafkaSource:
		return sourcesG
	}
	return nil
}
