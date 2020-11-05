package telemetry

import (
	"os"

	"go.uber.org/atomic"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
)

type Component string

const (
	EventingC           Component = "Eventing"
	ServingC            Component = "Serving"
	KnativeKafkaC       Component = "KnativeKafka"
	SkipTelemetryEnvVar           = "SKIP_TELEMETRY"
)

var (
	log                      = common.Log.WithName("telemetry")
	shouldEnableTelemetryFor = map[Component]*atomic.Bool{
		ServingC:      atomic.NewBool(false),
		EventingC:     atomic.NewBool(false),
		KnativeKafkaC: atomic.NewBool(false),
	}
	shouldDisableTelemetryFor = map[Component]*atomic.Bool{
		ServingC:      atomic.NewBool(false),
		EventingC:     atomic.NewBool(false),
		KnativeKafkaC: atomic.NewBool(false),
	}
	StopChannels = map[Component]chan struct{}{
		ServingC:      make(chan struct{}),
		EventingC:     make(chan struct{}),
		KnativeKafkaC: make(chan struct{}),
	}
)

// Telemetry per component should be setup once while the operator instance
// is running
func TryStartTelemetry(mgr manager.Manager, component Component) error {
	// if false skip telemetry setup
	if check := checkIfShouldEnableTelemetry(component); !check {
		return nil
	}
	log.Info("Starting telemetry for:", "component", component)
	c, err := createTelemetryController(mgr, component)
	if err != nil {
		return err
	}
	// Start our controller in a goroutine so that we do not block.
	go func() {
		// Block until our controller manager is elected leader. We presume our
		// entire process will terminate if we lose leadership, so we don't need
		// to handle that.
		<-mgr.Elected()
		// Start our controller. This will block until it is stopped
		// or the controller returns an error.
		if err := (*c).Start(StopChannels[component]); err != nil {
			log.Error(err, "cannot start telemetry controller for ", "component", component)
		}
	}()
	return nil
}

func TryStopTelemetry(component Component) {
	// if false skip telemetry cleanup
	if check := checkIfShouldDisableTelemetry(component); !check {
		return
	}
	log.Info("Stopping telemetry for:", "component", component)
	// stop the telemetry controller
	// the lock above makes sure we close once and not panic
	close(StopChannels[component])
	// allow future setups
	shouldEnableTelemetryFor[component].Swap(false)
	// can't use a closed channel
	StopChannels[component] = make(chan struct{})
}

// checkIfShouldEnableTelemetry returns true if we manage to swap first from false to true.
// Serves like a handy mutex lock.
func checkIfShouldEnableTelemetry(component Component) bool {
	return os.Getenv(SkipTelemetryEnvVar) == "" && !shouldEnableTelemetryFor[component].Swap(true)
}

// checkIfShouldDisableTelemetry returns true if we manage to swap first.
func checkIfShouldDisableTelemetry(component Component) bool {
	return os.Getenv(SkipTelemetryEnvVar) == "" && !shouldDisableTelemetryFor[component].Swap(true)
}
