package eventing

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"
	"knative.dev/eventing/pkg/apis/eventing"
	"knative.dev/eventing/pkg/client/informers/externalversions"
	"knative.dev/eventing/pkg/client/injection/client"
	eventingv1 "knative.dev/eventing/pkg/client/listers/eventing/v1"
	messagingv1 "knative.dev/eventing/pkg/client/listers/messaging/v1"
	"knative.dev/operator/pkg/apis/operator/base"
	operatorv1beta1 "knative.dev/operator/pkg/apis/operator/v1beta1"
	knativeeventinginformer "knative.dev/operator/pkg/client/injection/informers/operator/v1beta1/knativeeventing"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/ptr"
)

type coreScaler struct {
	eventingv1.BrokerLister

	messagingv1.InMemoryChannelLister

	apiExtensionClient *apiextension.Clientset

	cacheSynced      sync.WaitGroup
	hasCRDsInstalled atomic.Bool
	cancel           context.CancelFunc
	factory          externalversions.SharedInformerFactory

	logger *zap.Logger
}

type CoreScalerWrapper struct {
	scaler                  func() *coreScaler
	scalerMu                sync.Mutex
	scalerCtx               context.Context
	knativeEventingInformer cache.SharedIndexInformer
	impl                    *controller.Impl
}

func NewScaler(ctx context.Context, impl *controller.Impl) *CoreScalerWrapper {
	s := &CoreScalerWrapper{
		scalerCtx:               ctx,
		scalerMu:                sync.Mutex{},
		knativeEventingInformer: knativeeventinginformer.Get(ctx).Informer(),
		impl:                    impl,
	}
	s.resetScaler()

	return s
}

func (w *CoreScalerWrapper) Scale(ke *operatorv1beta1.KnativeEventing) error {
	w.scalerMu.Lock()
	defer w.scalerMu.Unlock()

	return w.scaler().scale(ke)
}

func (w *CoreScalerWrapper) Finalize() error {
	w.scalerMu.Lock()
	defer w.scalerMu.Unlock()

	w.scaler().finalize()
	w.resetScaler()

	return nil
}

func (w *CoreScalerWrapper) resetScaler() {
	w.scalerMu.Lock()
	defer w.scalerMu.Unlock()

	w.scaler = sync.OnceValue(func() *coreScaler {
		return newInternalScaler(w.scalerCtx, controller.HandleAll(func(i interface{}) {
			w.impl.GlobalResync(w.knativeEventingInformer)
		}))
	})
}

func newInternalScaler(ctx context.Context, resync cache.ResourceEventHandler) *coreScaler {

	c := client.Get(ctx)
	f := externalversions.NewSharedInformerFactoryWithOptions(c, controller.GetResyncPeriod(ctx))

	ctx, cancel := context.WithCancel(ctx)

	logger := logging.FromContext(ctx).With(zap.String("component", "scaler"))

	apiExtensionClient, _ := apiextension.NewForConfig(injection.GetConfig(ctx))

	s := &coreScaler{
		BrokerLister: f.Eventing().V1().Brokers().Lister(),

		InMemoryChannelLister: f.Messaging().V1().InMemoryChannels().Lister(),

		apiExtensionClient: apiExtensionClient,

		cacheSynced:      sync.WaitGroup{},
		hasCRDsInstalled: atomic.Bool{},

		cancel:  cancel,
		factory: f,

		logger: logger.Desugar(),
	}
	_, _ = f.Eventing().V1().Brokers().Informer().AddEventHandler(resync)

	_, _ = f.Messaging().V1().InMemoryChannels().Informer().AddEventHandler(resync)

	s.cacheSynced.Add(1)
	go func() {
		err := wait.PollUntilContextCancel(ctx, time.Second, false, func(ctx context.Context) (done bool, err error) {
			hasCRDsInstalled, err := s.verifyCRDsInstalled(ctx)
			logger.Debugw("Waiting for CRDs to be installed", zap.Bool("hasCRDsInstalled", hasCRDsInstalled))
			if err != nil {
				logger.Debugw("Failed to wait for CRDs to be installed", zap.Error(err))
				return false, nil
			}
			return hasCRDsInstalled, nil
		})
		if err != nil {
			return
		}

		logger.Debugw("Starting scaler informer factory and waiting for cache sync")

		f.Start(ctx.Done())
		f.WaitForCacheSync(ctx.Done())
		s.cacheSynced.Done()
	}()

	return s
}

func (s *coreScaler) scale(ke *operatorv1beta1.KnativeEventing) error {
	// If CRDs are not installed, it means that this is the first time we're reconciling Eventing,
	// and so we need to install the resources first and then try to scale down components.
	if !s.hasCRDsInstalled.Load() {
		return nil
	}

	if ke.Spec.Workloads == nil {
		ke.Spec.Workloads = make([]base.WorkloadOverride, 0)
	}

	hasMTChannelBrokers, err := s.hasMTChannelBrokers()
	if err != nil {
		s.logger.Warn("failed to verify if there are MT Channel Based Brokers", zap.Error(err))
		return err
	}
	if hasMTChannelBrokers {
		s.ensureAtLeastOneReplica(ke, "mt-broker-controller")
		s.ensureAtLeastOneReplica(ke, "mt-broker-ingress")
		s.ensureAtLeastOneReplica(ke, "mt-broker-filter")
	} else {
		s.scaleToZero(ke, "mt-broker-controller")
		s.scaleToZero(ke, "mt-broker-ingress")
		s.scaleToZero(ke, "mt-broker-filter")
	}

	hasInMemoryChannels, err := s.hasInMemoryChannels()
	if err != nil {
		s.logger.Warn("failed to verify if there are in memory channels", zap.Error(err))
		return err
	}
	if hasInMemoryChannels {
		s.ensureAtLeastOneReplica(ke, "imc-controller")
		s.ensureAtLeastOneReplica(ke, "imc-dispatcher")
	} else {
		s.scaleToZero(ke, "imc-controller")
		s.scaleToZero(ke, "imc-dispatcher")
	}

	return nil
}

func (s *coreScaler) finalize() {
	s.cancel()
	s.factory.Shutdown()
}

func (s *coreScaler) hasMTChannelBrokers() (bool, error) {
	brokers, err := s.BrokerLister.List(labels.Everything())
	if err != nil {
		return false, fmt.Errorf("failed to list brokers: %w", err)
	}
	for _, b := range brokers {
		if v, ok := b.Annotations[eventing.BrokerClassKey]; ok && v == eventing.MTChannelBrokerClassValue {
			return true, nil
		}
	}
	return false, nil
}

func (s *coreScaler) hasInMemoryChannels() (bool, error) {
	imcs, err := s.InMemoryChannelLister.List(labels.Everything())
	if err != nil {
		return false, fmt.Errorf("failed to list inmemorychannels: %w", err)
	}
	return len(imcs) > 0, nil
}

func (s *coreScaler) ensureAtLeastOneReplica(ke *operatorv1beta1.KnativeEventing, name string) {
	replicas := ptr.Int32(1)
	if ke.Spec.HighAvailability != nil && ke.Spec.HighAvailability.Replicas != nil {
		replicas = ke.Spec.HighAvailability.Replicas
	}

	s.logger.Info("Scaling up component", zap.String("name", name), zap.Int32("replicas", *replicas))

	for i, w := range ke.Spec.Workloads {
		if w.Name == name {
			if w.Replicas == nil {
				ke.Spec.Workloads[i].Replicas = replicas
			}
			return
		}
	}

	ke.Spec.Workloads = append(ke.Spec.Workloads, base.WorkloadOverride{
		Name:     name,
		Replicas: replicas,
	})
}

func (s *coreScaler) scaleToZero(ke *operatorv1beta1.KnativeEventing, name string) {
	s.logger.Info("Scaling down component", zap.String("name", name))

	replicas := pointer.Int32(0)
	for i, w := range ke.Spec.Workloads {
		if w.Name == name {
			// Important: Only set this when replicas is unset
			if w.Replicas == nil {
				ke.Spec.Workloads[i].Replicas = replicas
			}
			return
		}
	}

	ke.Spec.Workloads = append(ke.Spec.Workloads, base.WorkloadOverride{
		Name:     name,
		Replicas: replicas,
	})
}

func (s *coreScaler) verifyCRDsInstalled(ctx context.Context) (bool, error) {
	if s.hasCRDsInstalled.Load() {
		return true, nil
	}

	_, err := s.apiExtensionClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(ctx, "brokers.eventing.knative.dev", metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get broker CRD: %w", err)
	}
	_, err = s.apiExtensionClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(ctx, "inmemorychannels.messaging.knative.dev", metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get inmemorychannel CRD: %w", err)
	}

	s.hasCRDsInstalled.Store(true)
	return true, nil
}
