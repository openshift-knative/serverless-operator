package telemetry

import (
	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = common.Log.WithName("telemetry")

type Telemetry struct {
	name string
	stop chan struct{}
	tc   *controller.Controller
	// Protects from processing order, if true we should install telemetry
	// if it is false we need to uninstall in the next delete stage.
	// We start by assuming no telemetry is available.
	shouldInstallTelemetry bool
	objects                []runtime.Object
}

func NewTelemetry(name string, mgr manager.Manager, objects []runtime.Object, api client.Client) (*Telemetry, error) {
	t := &Telemetry{}
	t.name = name
	t.stop = make(chan struct{})
	t.objects = objects
	t.shouldInstallTelemetry = true
	tc, err := newTelemetryController(name, objects, mgr, t, api)
	if err != nil {
		return nil, err
	}
	t.tc = tc
	return t, nil
}

// TryStartTelemetry setups telemetry per component either Eventing, KnativeKafka or Serving.
// When called it assumes that the component has status ready.
func (t *Telemetry) TryStartTelemetry(api client.Client, mgr manager.Manager) error {
	if t == nil {
		return nil
	}
	if t.shouldInstallTelemetry {
		log.Info("starting telemetry for:", "component", t.name)
		// Initialize metrics before we start the corresponding controller.
		// There is a tiny window to miss events here, but should be ok for telemetry purposes.
		t.initializeAndUpdateMetrics(api)
		// Start our controller in a goroutine so that we do not block.
		go func() {
			// Block until our controller manager is elected leader. We presume our
			// entire process will terminate if we lose leadership, so we don'telemetry need
			// to handle that.
			<-mgr.Elected()
			// Start our controller. This will block until it is stopped
			// or the controller returns a starting error.
			if err := (*t.tc).Start(t.stop); err != nil {
				log.Error(err, "cannot start telemetry controller for", "component", t.name)
			}
		}()
		t.shouldInstallTelemetry = false
	}
	return nil
}

// TryStopTelemetry stops telemetry per component either Eventing, KnativeKafka or Serving
// When called it assumes that we are reconciling a deletion event.
func (t *Telemetry) TryStopTelemetry() {
	if t == nil {
		return
	}
	if !t.shouldInstallTelemetry {
		log.Info("stopping telemetry for:", "component", t.name)
		// Stop the telemetry controller
		close(t.stop)
		// Can't use a closed channel
		t.stop = make(chan struct{})
		t.shouldInstallTelemetry = true
	}
}

// initializeAndUpdateMetrics is used for taking a global snapshot of metrics
// before we start a telemetry controller. Cost should be low since we are fetching from cache.
func (t *Telemetry) initializeAndUpdateMetrics(api client.Client) {
	if t == nil {
		return
	}
	switch t.name {
	case "eventing":
		pingSourceG = serverlessTelemetryG.WithLabelValues("source_ping")
		apiServerSourceG = serverlessTelemetryG.WithLabelValues("source_apiserver")
		sinkBindingSourceG = serverlessTelemetryG.WithLabelValues("source_sinkbinding")
	case "knativeKafka":
		kafkaSourceG = serverlessTelemetryG.WithLabelValues("source_kafka")
	case "serving":
		serviceG = serverlessTelemetryG.WithLabelValues("service")
		routeG = serverlessTelemetryG.WithLabelValues("route")
		revisionG = serverlessTelemetryG.WithLabelValues("revision")
		configurationG = serverlessTelemetryG.WithLabelValues("configuration")
	}
	for _, obj := range t.objects {
		updateMetricFor(obj, api)
	}
}
