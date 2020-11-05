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
	servingObjects = []runtime.Object{
		&servingv1.Service{},
		&servingv1.Revision{},
		&servingv1.Route{},
	}

	eventingObjects = []runtime.Object{
		&eventingsourcesv1beta1.PingSource{},
		&eventingsourcesv1beta1.ApiServerSource{},
		&eventingsourcesv1beta1.SinkBinding{},
	}

	knativeKafkaObjects = []runtime.Object{
		&kafkasourcev1beta1.KafkaSource{},
	}
)

// creates an unmanaged controller for watching Telemetry resources
func createTelemetryController(mgr manager.Manager, component Component) (*controller.Controller, error) {
	// Create a new controller
	c, err := controller.NewUnmanaged(fmt.Sprintf("telemetry-resources-%s-controller-%s", component, uuid.New().String()), mgr, controller.Options{
		Reconciler: reconcile.Func(func(reconcile.Request) (reconcile.Result, error) { // No actual update happens here
			return reconcile.Result{}, nil
		}),
	})
	if err != nil {
		return nil, err
	}
	for _, tp := range getObjects(component) {
		if err := c.Watch(&source.Kind{Type: tp}, &handler.EnqueueRequestForObject{}, metricsPredicate{}); err != nil {
			return nil, err
		}
	}
	return &c, nil
}

func getObjects(component Component) []runtime.Object {
	switch component {
	case EventingC:
		return eventingObjects
	case ServingC:
		return servingObjects
	case KnativeKafkaC:
		return knativeKafkaObjects
	}
	return nil
}

type metricsPredicate struct {
	predicate.Funcs
}

func (metricsPredicate) Update(_ event.UpdateEvent) bool {
	return false
}

func (metricsPredicate) Create(e event.CreateEvent) bool {
	if metric := getMetricFor(e.Object); metric != nil {
		metric.Inc()
	}
	return false
}

func (metricsPredicate) Delete(e event.DeleteEvent) bool {
	if metric := getMetricFor(e.Object); metric != nil {
		metric.Dec()
	}
	return false
}

func getMetricFor(obj runtime.Object) prometheus.Gauge {
	switch obj.(type) {
	case *servingv1.Service:
		return ServicesG
	case *servingv1.Revision:
		return RevisionsG
	case *servingv1.Route:
		return RoutesG
	case *eventingsourcesv1beta1.PingSource, *eventingsourcesv1beta1.ApiServerSource, *eventingsourcesv1beta1.SinkBinding, *kafkasourcev1beta1.KafkaSource:
		return SourcesG
	}
	return nil
}
