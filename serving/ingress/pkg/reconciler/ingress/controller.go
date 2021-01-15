package ingress

import (
	"context"

	"k8s.io/client-go/tools/cache"
	"knative.dev/networking/pkg/apis/networking"
	ingressinformer "knative.dev/networking/pkg/client/injection/informers/networking/v1alpha1/ingress"
	ingressreconciler "knative.dev/networking/pkg/client/injection/reconciler/networking/v1alpha1/ingress"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/serving/pkg/apis/serving"

	routeclient "github.com/openshift-knative/serverless-operator/serving/ingress/pkg/client/injection/client"
	routeinformer "github.com/openshift-knative/serverless-operator/serving/ingress/pkg/client/injection/informers/route/v1/route"
	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/reconciler/ingress/resources"
)

const kourierIngressClassName = "kourier.ingress.networking.knative.dev"

// NewController returns a new Ingress controller for Ingress on Openshift.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	logger := logging.FromContext(ctx)

	ingressInformer := ingressinformer.Get(ctx)
	routeInformer := routeinformer.Get(ctx)

	c := &Reconciler{
		routeLister: routeInformer.Lister(),
		routeClient: routeclient.Get(ctx).RouteV1(),
	}

	impl := ingressreconciler.NewImpl(ctx, c, kourierIngressClassName, func(impl *controller.Impl) controller.Options {
		return controller.Options{
			SkipStatusUpdates: true,
			FinalizerName:     "ocp-ingress",
		}
	})

	logger.Info("Setting up event handlers")

	ingressInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: reconciler.AnnotationFilterFunc(networking.IngressClassAnnotationKey, kourierIngressClassName, false),
		Handler:    controller.HandleAll(impl.Enqueue),
	})

	// We started using OpenShiftIngressLabelKey labels below but still handle resources with this labels for safety.
	routeInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: reconciler.LabelExistsFilterFunc(networking.IngressLabelKey),
		Handler: controller.HandleAll(impl.EnqueueLabelOfNamespaceScopedResource(
			serving.RouteNamespaceLabelKey,
			networking.IngressLabelKey,
		)),
	})

	routeInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: reconciler.LabelExistsFilterFunc(networking.IngressLabelKey),
		Handler: controller.HandleAll(impl.EnqueueLabelOfNamespaceScopedResource(
			resources.OpenShiftIngressNamespaceLabelKey,
			resources.OpenShiftIngressLabelKey,
		)),
	})

	return impl
}
