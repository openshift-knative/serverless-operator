package telemetry

import (
	"context"
	"sync"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kafkasourcev1beta1 "knative.dev/eventing-contrib/kafka/source/pkg/apis/sources/v1beta1"
	eventingsourcesv1beta1 "knative.dev/eventing/pkg/apis/sources/v1beta1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = common.Log.WithName("telemetry")

type Telemetry struct {
	name            string
	stop            chan struct{}
	metricsSnapshot sync.Map
	tc              *controller.Controller
}

func NewTelemetry(name string, mgr manager.Manager, objects []runtime.Object) (*Telemetry, error) {
	t := &Telemetry{}
	t.name = name
	t.stop = make(chan struct{})
	tc, err := newTelemetryController(name, objects, mgr, t)
	if err != nil {
		return nil, err
	}
	t.tc = tc
	return t, nil
}

// TryStartTelemetry setups telemetry per component either Eventing, KnativeKafka or Serving.
// When called it assumes that the component has status ready.
func (t *Telemetry) TryStartTelemetry(api client.Client, mgr manager.Manager) error {
	log.Info("starting telemetry for:", "component", t.name)
	// Initialize metrics before we start the corresponding controller.
	// There is a tiny window to miss events here, but should be ok for telemetry purposes.
	t.InitializeAndTakeMetricsSnapshot(api)
	// Start our controller in a goroutine so that we do not block.
	if t.tc != nil { // meant to allow nop objects in tests
		go func() {
			// Block until our controller manager is elected leader. We presume our
			// entire process will terminate if we lose leadership, so we don'telemetry need
			// to handle that.
			<-mgr.Elected()
			// Start our controller. This will block until it is stopped
			// or the controller returns an error.
			if err := (*t.tc).Start(t.stop); err != nil {
				log.Error(err, "cannot start telemetry controller for", "component", t.name)
			}
		}()
	}
	return nil
}

// TryStopTelemetry stops telemetry per component either Eventing, KnativeKafka or Serving
// When called it assumes that we are reconciling a deletion event.
func (t *Telemetry) TryStopTelemetry() {
	log.Info("stopping telemetry for:", "component", t.name)
	// Stop the telemetry controller
	// the lock above makes sure we close once and not panic
	close(t.stop)
	// Can'telemetry use a closed channel
	t.stop = make(chan struct{})
	// Remove snapshot entries for the components so that snapshot is does not get
	// unbounded since the telemetry controller can be restarted multiple times.
	t.metricsSnapshot.Range(func(key interface{}, value interface{}) bool {
		t.metricsSnapshot.Delete(key)
		return true
	})
}

// InitializeAndTakeMetricsSnapshot is used for taking a global snapshot of metrics
// before we start any telemetry controller. If the operator is restarted
// client metrics will not reflect the state in the cluster so there is need to restore current state.
// It should run each time operator is initialized and for each trial to start a Telemetry controller.
// Snapshot is used for skipping events already counted but also received later after
// telemetry controllers are started. Metrics are initialized here and kept around as long as the
// Serverless Operator is running. No metric label is removed when components are removed so that
// if any component eg. Eventing is removed and CRDs exist, it will show existing CRs correctly.
// Telemetry should be used in the background so no error is returned here.
func (t *Telemetry) InitializeAndTakeMetricsSnapshot(api client.Client) {
	switch t.name {
	case "eventing":
		sourcesG = serverlessTelemetryG.WithLabelValues("source")
		pingSourceList := &eventingsourcesv1beta1.PingSourceList{}
		if err := api.List(context.TODO(), pingSourceList); err == nil {
			sourcesG.Set(float64(len(pingSourceList.Items)))
			for _, obj := range pingSourceList.Items {
				t.addToSnaphost(obj.GetObjectMeta())
			}
		}

		sinkBindingList := &eventingsourcesv1beta1.SinkBindingList{}
		if err := api.List(context.TODO(), sinkBindingList); err == nil {
			sourcesG.Add(float64(len(sinkBindingList.Items)))
			for _, obj := range sinkBindingList.Items {
				t.addToSnaphost(obj.GetObjectMeta())
			}
		}

		apiServerSourceList := &eventingsourcesv1beta1.ApiServerSourceList{}
		if err := api.List(context.TODO(), apiServerSourceList); err == nil {
			sourcesG.Add(float64(len(apiServerSourceList.Items)))
			for _, obj := range pingSourceList.Items {
				t.addToSnaphost(obj.GetObjectMeta())
			}
		}

	case "knativeKafka":
		knativeKafkaList := &kafkasourcev1beta1.KafkaSourceList{}
		if err := api.List(context.TODO(), knativeKafkaList); err == nil {
			sourcesG.Add(float64(len(knativeKafkaList.Items)))
			for _, obj := range knativeKafkaList.Items {
				t.addToSnaphost(obj.GetObjectMeta())
			}
		}

	case "serving":
		servicesG = serverlessTelemetryG.WithLabelValues("service")
		serviceList := &servingv1.ServiceList{}
		if err := api.List(context.TODO(), serviceList); err == nil {
			servicesG.Set(float64(len(serviceList.Items)))
			for _, obj := range serviceList.Items {
				t.addToSnaphost(obj.GetObjectMeta())
			}
		}

		revisionsG = serverlessTelemetryG.WithLabelValues("revision")
		revisionList := &servingv1.RevisionList{}
		if err := api.List(context.TODO(), revisionList); err != nil {
			revisionsG.Set(float64(len(revisionList.Items)))
			for _, obj := range revisionList.Items {
				t.addToSnaphost(obj.GetObjectMeta())
			}
		}

		routesG = serverlessTelemetryG.WithLabelValues("route")
		routeList := &servingv1.RouteList{}
		if err := api.List(context.TODO(), routeList); err == nil {
			routesG.Set(float64(len(routeList.Items)))
			for _, obj := range routeList.Items {
				t.addToSnaphost(obj.GetObjectMeta())
			}
		}
	}
}

func (t *Telemetry) addToSnaphost(obj metav1.Object) {
	if obj == nil {
		return
	}
	t.metricsSnapshot.Store(types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}, true)
}

func (t *Telemetry) deleteFromSnaphost(obj metav1.Object) {
	if obj == nil {
		return
	}
	t.metricsSnapshot.Delete(types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	})
}

func (t *Telemetry) inSnapshot(obj metav1.Object) (ok bool) {
	if obj == nil {
		return false
	}
	_, ok = t.metricsSnapshot.Load(types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	})
	return
}
