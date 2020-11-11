package telemetry

import (
	"context"
	"fmt"

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

// newTelemetryController creates an unmanaged controller for watching Telemetry resources
func newTelemetryController(name string, objects []runtime.Object, mgr manager.Manager) (controller.Controller, error) {
	// Create a new controller
	c, err := controller.NewUnmanaged(fmt.Sprintf("telemetry-resources-%s-controller", name), mgr, controller.Options{
		Reconciler: reconcile.Func(func(reconcile.Request) (reconcile.Result, error) { // No actual update happens here
			return reconcile.Result{}, nil
		}),
	})
	if err != nil {
		return nil, err
	}
	for _, tp := range objects {
		if err := c.Watch(&source.Kind{Type: tp}, &handler.EnqueueRequestForObject{}, metricsPredicate{
			api: mgr.GetClient(),
		}); err != nil {
			return nil, err
		}
	}
	return c, nil
}

type metricsPredicate struct {
	predicate.Funcs
	api client.Client
}

func (metricsPredicate) Update(_ event.UpdateEvent) bool {
	return false
}

func (mp metricsPredicate) Create(e event.CreateEvent) bool {
	updateMetricFor(e.Object, mp.api)
	return false
}

func (mp metricsPredicate) Delete(e event.DeleteEvent) bool {
	updateMetricFor(e.Object, mp.api)
	return false
}

func updateMetricFor(obj runtime.Object, api client.Client) {
	switch obj.(type) {
	case *servingv1.Service:
		serviceList := &servingv1.ServiceList{}
		if err := api.List(context.TODO(), serviceList); err == nil {
			serviceG.Set(float64(len(serviceList.Items)))
		}
	case *servingv1.Revision:
		revisionList := &servingv1.RevisionList{}
		if err := api.List(context.TODO(), revisionList); err == nil {
			revisionG.Set(float64(len(revisionList.Items)))
		}
	case *servingv1.Route:
		routeList := &servingv1.RouteList{}
		if err := api.List(context.TODO(), routeList); err == nil {
			routeG.Set(float64(len(routeList.Items)))
		}
	case *servingv1.Configuration:
		configurationList := &servingv1.ConfigurationList{}
		if err := api.List(context.TODO(), configurationList); err == nil {
			configurationG.Set(float64(len(configurationList.Items)))
		}
	case *eventingsourcesv1beta1.PingSource:
		pingSourceList := &eventingsourcesv1beta1.PingSourceList{}
		if err := api.List(context.TODO(), pingSourceList); err == nil {
			pingSourceG.Set(float64(len(pingSourceList.Items)))
		}
	case *eventingsourcesv1beta1.ApiServerSource:
		apiServerSourceList := &eventingsourcesv1beta1.ApiServerSourceList{}
		if err := api.List(context.TODO(), apiServerSourceList); err == nil {
			apiServerSourceG.Set(float64(len(apiServerSourceList.Items)))
		}
	case *eventingsourcesv1beta1.SinkBinding:
		sinkBindingList := &eventingsourcesv1beta1.SinkBindingList{}
		if err := api.List(context.TODO(), sinkBindingList); err == nil {
			sinkBindingSourceG.Set(float64(len(sinkBindingList.Items)))
		}
	case *kafkasourcev1beta1.KafkaSource:
		knativeKafkaList := &kafkasourcev1beta1.KafkaSourceList{}
		if err := api.List(context.TODO(), knativeKafkaList); err == nil {
			kafkaSourceG.Set(float64(len(knativeKafkaList.Items)))
		}
	}
}
