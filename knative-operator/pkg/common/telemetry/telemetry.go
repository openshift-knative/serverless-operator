package telemetry

import (
	"os"
	"sync"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
	mu                       = sync.Mutex{}
	skipTelemetryEnablingFor = map[Component]bool{
		ServingC:      false,
		EventingC:     false,
		KnativeKafkaC: false,
	}
	skipTelemetryStoppingFor = map[Component]bool{
		ServingC:      true,
		EventingC:     true,
		KnativeKafkaC: true,
	}
	stopChannels = map[Component]chan struct{}{
		ServingC:      make(chan struct{}),
		EventingC:     make(chan struct{}),
		KnativeKafkaC: make(chan struct{}),
	}
)

// Telemetry per component should be setup once while the operator instance
// is running
func TryStartTelemetry(mgr manager.Manager, component Component) error {
	if os.Getenv(SkipTelemetryEnvVar) != "" {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	if skipTelemetryEnablingFor[component] {
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
		if err := (*c).Start(stopChannels[component]); err != nil {
			log.Error(err, "cannot start telemetry controller for ", "component", component)
		}
	}()
	// allow stopping telemetry if we stopped before
	skipTelemetryEnablingFor[component] = true
	skipTelemetryStoppingFor[component] = false
	return nil
}

func TryStopTelemetry(component Component) {
	if os.Getenv(SkipTelemetryEnvVar) != "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if skipTelemetryStoppingFor[component] {
		return
	}
	log.Info("Stopping telemetry for:", "component", component)
	// stop the telemetry controller
	// the lock above makes sure we close once and not panic
	close(stopChannels[component])
	// can't use a closed channel
	stopChannels[component] = make(chan struct{})
	// allow future setups
	skipTelemetryEnablingFor[component] = false
	skipTelemetryStoppingFor[component] = true
}
