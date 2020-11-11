package telemetry

import (
	"context"
	"fmt"
	"testing"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis"
	"github.com/prometheus/client_golang/prometheus"
	ioprometheusclient "github.com/prometheus/client_model/go"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	kafkasourcev1beta1 "knative.dev/eventing-contrib/kafka/source/pkg/apis/sources/v1beta1"
	eventingsourcesv1beta1 "knative.dev/eventing/pkg/apis/sources/v1beta1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var (
	service = &servingv1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name: "service",
		}}
	route = &servingv1.Route{
		ObjectMeta: v1.ObjectMeta{
			Name: "route",
		}}
	revision = &servingv1.Revision{
		ObjectMeta: v1.ObjectMeta{
			Name: "revision",
		}}
	configuration = &servingv1.Configuration{
		ObjectMeta: v1.ObjectMeta{
			Name: "configuration",
		}}
	pingSource = &eventingsourcesv1beta1.PingSource{
		ObjectMeta: v1.ObjectMeta{
			Name: "pingsource",
		}}
	apiServerSource = &eventingsourcesv1beta1.ApiServerSource{
		ObjectMeta: v1.ObjectMeta{
			Name: "apiserversource",
		}}
	sinkBindingSource = &eventingsourcesv1beta1.SinkBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: "sinkbinding",
		}}
	kafkaSource = &kafkasourcev1beta1.KafkaSource{
		ObjectMeta: v1.ObjectMeta{
			Name: "kafkasource",
		}}
)

type metricCase struct {
	name                string
	event               interface{}
	expectedMetricValue float64
}

func init() {
	apis.AddToScheme(scheme.Scheme)
}

func TestTelemetryMetricsUpdates(t *testing.T) {
	// run this in a serialized manner so that values are predictable for sources
	metricSteps := generateMetricUpdateSteps()
	cl := fake.NewFakeClient()
	for _, tc := range metricSteps {
		mp := metricsPredicate{api: cl}
		dto := ioprometheusclient.Metric{}
		var metric prometheus.Gauge
		if create, ok := tc.event.(event.CreateEvent); ok {
			err := cl.Create(context.TODO(), create.Object)
			if err != nil {
				t.Fatal("failed to create object", err)
			}
			mp.Create(create)
			metric = getMetricFor(create.Object)
		} else if delete, ok := tc.event.(event.DeleteEvent); ok {
			err := cl.Delete(context.TODO(), delete.Object)
			if err != nil {
				t.Fatal("failed to delete object", err)
			}
			mp.Delete(delete)
			metric = getMetricFor(delete.Object)
		} else if update, ok := tc.event.(event.UpdateEvent); ok {
			err := cl.Update(context.TODO(), update.ObjectOld)
			if err != nil {
				t.Fatal("failed to update object", err)
			}
			mp.Update(update)
			metric = getMetricFor(update.ObjectOld)
		}
		if metric == nil {
			t.Fatal("Cannot get metric")
		}
		err := metric.Write(&dto)
		if err != nil {
			t.Fatal("Cannot write metric:", err)
		}
		if *dto.Gauge.Value != tc.expectedMetricValue {
			t.Errorf("Got = %v, want: %v for event: %v", *dto.Gauge.Value, tc.expectedMetricValue, tc.name)
		}
	}
}

func generateMetricUpdateSteps() (ret []metricCase) {
	objects := []struct {
		name string
		obj  runtime.Object
	}{{
		name: "service",
		obj:  service,
	}, {
		name: "route",
		obj:  route,
	}, {
		name: "revision",
		obj:  revision,
	}, {
		name: "configuration",
		obj:  configuration,
	}, {
		name: "pingsource",
		obj:  pingSource,
	}, {
		name: "apiserversource",
		obj:  apiServerSource,
	}, {
		name: "sinkbindingsource",
		obj:  sinkBindingSource,
	}, {
		name: "kafkasource",
		obj:  kafkaSource,
	}}
	ret = []metricCase{}
	for _, v := range objects {
		ret = append(ret, metricCase{
			name: fmt.Sprintf("create a %s", v.name),
			event: event.CreateEvent{
				Object: v.obj,
			},
			expectedMetricValue: 1,
		}, metricCase{
			name:                fmt.Sprintf("update a %s", v.name),
			event:               event.UpdateEvent{ObjectOld: v.obj},
			expectedMetricValue: 1,
		}, metricCase{
			name: fmt.Sprintf("delete a %s", v.name),
			event: event.DeleteEvent{
				Object: v.obj,
			},
			expectedMetricValue: 0,
		})
	}
	return
}

func getMetricFor(obj runtime.Object) prometheus.Gauge {
	switch obj.(type) {
	case *servingv1.Service:
		return serviceG
	case *servingv1.Revision:
		return revisionG
	case *servingv1.Route:
		return routeG
	case *servingv1.Configuration:
		return configurationG
	case *eventingsourcesv1beta1.PingSource:
		return pingSourceG
	case *eventingsourcesv1beta1.ApiServerSource:
		return apiServerSourceG
	case *eventingsourcesv1beta1.SinkBinding:
		return sinkBindingSourceG
	case *kafkasourcev1beta1.KafkaSource:
		return kafkaSourceG
	}
	return nil
}
