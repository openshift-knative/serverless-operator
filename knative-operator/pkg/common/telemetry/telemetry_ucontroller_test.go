package telemetry

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	ioprometheusclient "github.com/prometheus/client_model/go"
	"k8s.io/apimachinery/pkg/runtime"
	kafkasourcev1beta1 "knative.dev/eventing-contrib/kafka/source/pkg/apis/sources/v1beta1"
	eventingsourcesv1beta1 "knative.dev/eventing/pkg/apis/sources/v1beta1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var (
	service           = &servingv1.Service{}
	route             = &servingv1.Route{}
	revision          = &servingv1.Revision{}
	pingSource        = &eventingsourcesv1beta1.PingSource{}
	apiServerSource   = &eventingsourcesv1beta1.ApiServerSource{}
	sinkBindingSource = &eventingsourcesv1beta1.SinkBinding{}
	kafkaSource       = &kafkasourcev1beta1.KafkaSource{}
)

type metricCase struct {
	name                string
	event               interface{}
	expectedMetricValue float64
}

func TestTelemetryMetricsUpdates(t *testing.T) {
	// run this in a serialized manner so that values are predictable for sources
	metricSteps := generateMetricUpdateSteps()
	for _, tc := range metricSteps {
		mp := metricsPredicate{}
		dto := ioprometheusclient.Metric{}
		var metric prometheus.Gauge
		if create, ok := tc.event.(event.CreateEvent); ok {
			mp.Create(create)
			metric = getMetricFor(create.Object)
		} else if delete, ok := tc.event.(event.DeleteEvent); ok {
			mp.Delete(delete)
			metric = getMetricFor(delete.Object)
		} else if update, ok := tc.event.(event.UpdateEvent); ok {
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
			name:                fmt.Sprintf("create a %s", v.name),
			event:               event.CreateEvent{Object: v.obj},
			expectedMetricValue: 1,
		}, metricCase{
			name:                fmt.Sprintf("delete a %s", v.name),
			event:               event.DeleteEvent{Object: v.obj},
			expectedMetricValue: 0,
		}, metricCase{
			name:                fmt.Sprintf("update a %s", v.name),
			event:               event.UpdateEvent{ObjectOld: v.obj},
			expectedMetricValue: 0,
		})
	}
	return
}
