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
	service           = servingv1.Service{}
	route             = servingv1.Route{}
	revision          = servingv1.Revision{}
	pingSource        = eventingsourcesv1beta1.PingSource{}
	apiServerSource   = eventingsourcesv1beta1.ApiServerSource{}
	SinkBindingSource = eventingsourcesv1beta1.SinkBinding{}
	kafkaSource       = kafkasourcev1beta1.KafkaSource{}
)

type metricCase struct {
	name                string
	event               interface{}
	expectedMetricValue float64
}

func TestTelemetryMetricsUpdates(t *testing.T) {
	// run this in a serialized manner so that values are predictable for sources
	metricSteps := generateMetricUpdateSteps()
	for n, tc := range metricSteps {
		mp := metricsPredicate{}
		dto := ioprometheusclient.Metric{}
		var metric prometheus.Gauge
		switch tc.event.(type) {
		case event.CreateEvent:
			mp.Create(tc.event.(event.CreateEvent))
			metric = getMetricFor(tc.event.(event.CreateEvent).Object)
		case event.DeleteEvent:
			mp.Delete(tc.event.(event.DeleteEvent))
			metric = getMetricFor(tc.event.(event.DeleteEvent).Object)
		case event.UpdateEvent:
			mp.Update(tc.event.(event.UpdateEvent))
			metric = getMetricFor(tc.event.(event.UpdateEvent).ObjectOld)
		}
		if metric == nil {
			t.Fatal("Cannot get metric")
		}
		err := metric.Write(&dto)
		if err != nil {
			t.Fatal("Cannot write metric:", err)
		}
		if *dto.Gauge.Value != tc.expectedMetricValue {
			t.Errorf("Got = %v, want: %v for event: %v", *dto.Gauge.Value, tc.expectedMetricValue, n)
		}
	}
}

func generateMetricUpdateSteps() (ret []metricCase) {
	objects := []struct {
		name string
		obj  runtime.Object
	}{{
		name: "service",
		obj:  service.DeepCopyObject(),
	}, {
		name: "route",
		obj:  route.DeepCopyObject(),
	}, {
		name: "revision",
		obj:  revision.DeepCopyObject(),
	}, {
		name: "pingsource",
		obj:  pingSource.DeepCopyObject(),
	}, {
		name: "apiserversource",
		obj:  apiServerSource.DeepCopyObject(),
	}, {
		name: "sinkbindingsource",
		obj:  SinkBindingSource.DeepCopyObject(),
	}, {
		name: "kafkasource",
		obj:  kafkaSource.DeepCopyObject(),
	}}
	ret = []metricCase{}
	for _, v := range objects {
		ret = append(ret, metricCase{
			name:                fmt.Sprintf("create a %s", v.name),
			event:               event.CreateEvent{Object: v.obj},
			expectedMetricValue: 1,
		})
		ret = append(ret, metricCase{
			name:                fmt.Sprintf("delete a %s", v.name),
			event:               event.DeleteEvent{Object: v.obj},
			expectedMetricValue: 0,
		})
		ret = append(ret, metricCase{
			name:                fmt.Sprintf("update a %s", v.name),
			event:               event.UpdateEvent{ObjectOld: v.obj},
			expectedMetricValue: 0,
		})
	}
	return
}
