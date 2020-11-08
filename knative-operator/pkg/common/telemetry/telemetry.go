package telemetry

import (
	"context"
	"os"
	"sync"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kafkasourcev1beta1 "knative.dev/eventing-contrib/kafka/source/pkg/apis/sources/v1beta1"
	eventingsourcesv1beta1 "knative.dev/eventing/pkg/apis/sources/v1beta1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const SkipTelemetryEnvVar = "SKIP_TELEMETRY"

var (
	log             = common.Log.WithName("telemetry")
	metricsSnapshot = map[string]*sync.Map{
		"eventing":     {},
		"serving":      {},
		"knativeKafka": {},
	}
)

// TryStartTelemetry setups telemetry per component either Eventing, KnativeKafka or Serving.
// When called it assumes that the component has status ready.
func TryStartTelemetry(c controller.Controller, mgr manager.Manager, stop chan struct{}, component string, api client.Client) error {
	if os.Getenv(SkipTelemetryEnvVar) != "" {
		return nil
	}
	log.Info("starting telemetry for:", "component", component)
	// Initialize metrics before we start the corresponding controller.
	// There is a tiny window to miss events here, but should be ok for telemetry purposes.
	InitializeAndTakeMetricsSnapshot(component, api)
	// Start our controller in a goroutine so that we do not block.
	go func() {
		// Block until our controller manager is elected leader. We presume our
		// entire process will terminate if we lose leadership, so we don't need
		// to handle that.
		<-mgr.Elected()
		// Start our controller. This will block until it is stopped
		// or the controller returns an error.
		if err := c.Start(stop); err != nil {
			log.Error(err, "cannot start telemetry controller for", "component", component)
		}
	}()
	return nil
}

// TryStopTelemetry stops telemetry per component either Eventing, KnativeKafka or Serving
// When called it assumes that we are reconciling a deletion event.
func TryStopTelemetry(stop *chan struct{}, component string) {
	if os.Getenv(SkipTelemetryEnvVar) != "" {
		return
	}
	log.Info("stopping telemetry for:", "component", component)
	// Stop the telemetry controller
	// the lock above makes sure we close once and not panic
	close(*stop)
	// Can't use a closed channel
	*stop = make(chan struct{})
	// Remove snapshot entries for the components so that snapshot is does not get
	// unbounded since the telemetry controller can be restarted multiple times.
	metricsSnapshot[component].Range(func(key interface{}, value interface{}) bool {
		metricsSnapshot[component].Delete(key)
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
func InitializeAndTakeMetricsSnapshot(component string, api client.Client) {
	switch component {
	case "eventing":
		sourcesG = serverlessTelemetryG.WithLabelValues("source")
		pingSourceList := &eventingsourcesv1beta1.PingSourceList{}
		if err := api.List(context.TODO(), pingSourceList); err == nil {
			sourcesG.Set(float64(len(pingSourceList.Items)))
			for _, obj := range pingSourceList.Items {
				addToSnaphost(obj.GetObjectMeta(), component)
			}
		}

		sinkBindingList := &eventingsourcesv1beta1.SinkBindingList{}
		if err := api.List(context.TODO(), sinkBindingList); err == nil {
			sourcesG.Add(float64(len(sinkBindingList.Items)))
			for _, obj := range sinkBindingList.Items {
				addToSnaphost(obj.GetObjectMeta(), component)
			}
		}

		apiServerSourceList := &eventingsourcesv1beta1.ApiServerSourceList{}
		if err := api.List(context.TODO(), apiServerSourceList); err == nil {
			sourcesG.Add(float64(len(apiServerSourceList.Items)))
			for _, obj := range pingSourceList.Items {
				addToSnaphost(obj.GetObjectMeta(), component)
			}
		}

	case "knativeKafka":
		knativeKafkaList := &kafkasourcev1beta1.KafkaSourceList{}
		if err := api.List(context.TODO(), knativeKafkaList); err == nil {
			sourcesG.Add(float64(len(knativeKafkaList.Items)))
			for _, obj := range knativeKafkaList.Items {
				addToSnaphost(obj.GetObjectMeta(), component)
			}
		}

	case "serving":
		servicesG = serverlessTelemetryG.WithLabelValues("service")
		serviceList := &servingv1.ServiceList{}
		if err := api.List(context.TODO(), serviceList); err == nil {
			servicesG.Set(float64(len(serviceList.Items)))
			for _, obj := range serviceList.Items {
				addToSnaphost(obj.GetObjectMeta(), component)
			}
		}

		revisionsG = serverlessTelemetryG.WithLabelValues("revision")
		revisionList := &servingv1.RevisionList{}
		if err := api.List(context.TODO(), revisionList); err != nil {
			revisionsG.Set(float64(len(revisionList.Items)))
			for _, obj := range revisionList.Items {
				addToSnaphost(obj.GetObjectMeta(), component)
			}
		}

		routesG = serverlessTelemetryG.WithLabelValues("route")
		routeList := &servingv1.RouteList{}
		if err := api.List(context.TODO(), routeList); err == nil {
			routesG.Set(float64(len(routeList.Items)))
			for _, obj := range routeList.Items {
				addToSnaphost(obj.GetObjectMeta(), component)
			}
		}
	}
}

func addToSnaphost(obj metav1.Object, component string) {
	if obj == nil {
		return
	}
	metricsSnapshot[component].Store(types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}, true)
}

func deleteFromSnaphost(obj metav1.Object, component string) {
	if obj == nil {
		return
	}
	metricsSnapshot[component].Delete(types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	})
}

func inSnapshot(obj metav1.Object, component string) (ok bool) {
	if obj == nil {
		return false
	}
	_, ok = metricsSnapshot[component].Load(types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	})
	return
}
