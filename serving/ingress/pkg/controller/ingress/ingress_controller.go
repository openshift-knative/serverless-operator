package ingress

import (
	"context"
	"fmt"
	"reflect"

	routev1 "github.com/openshift/api/route/v1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"knative.dev/pkg/logging"
	"knative.dev/pkg/logging/logkey"
	"knative.dev/serving/pkg/apis/networking"
	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
	"knative.dev/serving/pkg/apis/serving"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/controller/ingress/resources"
)

var baseLogger *zap.SugaredLogger

func init() {
	loggingConfig, _ := logging.NewConfigFromMap(nil) // force the default values
	baseLogger, _ = logging.NewLoggerFromConfig(loggingConfig, "knative-openshift-ingress")
}

// Add creates a new Ingress Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	client := mgr.GetClient()
	return &ReconcileIngress{
		client:   client,
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetRecorder("knative-openshift-ingress"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("ingress-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Ingress
	err = c.Watch(&source.Kind{Type: &networkingv1alpha1.Ingress{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Routes and requeue the owner Ingress
	err = c.Watch(&source.Kind{Type: &routev1.Route{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []reconcile.Request {
			labels := obj.Meta.GetLabels()

			// These labels are already present on the routes so using them. The route
			// namespace is guaranteed to be equal to the ingress namespace.
			namespace := labels[serving.RouteNamespaceLabelKey]
			name := labels[networking.IngressLabelKey]

			if namespace == "" || name == "" {
				return nil
			}

			return []reconcile.Request{{
				NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      name,
				},
			}}
		}),
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileIngress{}

// ReconcileIngress reconciles an Ingress object
type ReconcileIngress struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a Ingress
// object and makes changes based on the state read and what is in the
// Ingress.Spec
//
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileIngress) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := baseLogger.With(logkey.Key, request.NamespacedName.String())
	ctx := logging.WithLogger(context.Background(), logger)

	// Fetch the Ingress instance
	original := &networkingv1alpha1.Ingress{}
	err := r.client.Get(context.TODO(), request.NamespacedName, original)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	// Don't modify the informer's copy
	ing := original.DeepCopy()
	if newFinalizer, change := appendIfAbsent(ing.Finalizers, "ocp-ingress"); change {
		ing.Finalizers = newFinalizer
		if err := r.client.Update(context.TODO(), ing); err != nil {
			return reconcile.Result{}, nil
		}
	}
	reconcileErr := r.ReconcileIngress(ctx, ing)
	if equality.Semantic.DeepEqual(original.Status, ing.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else if _, err := r.updateStatus(ctx, ing); err != nil {
		logger.Errorf("Failed to update ingress status %v", err)
		r.recorder.Event(ing, corev1.EventTypeWarning, "SyncError", err.Error())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, reconcileErr
}

func (r *ReconcileIngress) ReconcileIngress(ctx context.Context, ing *networkingv1alpha1.Ingress) error {
	logger := logging.FromContext(ctx)

	if ing.GetDeletionTimestamp() != nil {
		return r.reconcileDeletion(ctx, ing)
	}

	logger.Infof("Reconciling ingress :%v", ing)

	exposed := ing.Spec.Visibility == networkingv1alpha1.IngressVisibilityExternalIP
	if exposed {
		existing, err := r.routeList(ctx, ing)
		if err != nil {
			logger.Errorf("Failed to list openshift routes %v", err)
			return err
		}
		existingMap := make(map[string]routev1.Route, len(existing.Items))
		for _, route := range existing.Items {
			existingMap[route.Name] = route
		}

		routes, err := resources.MakeRoutes(ing)
		if err != nil {
			logger.Warnf("Failed to generate routes from ingress %v", err)
			// Returning nil aborts the reconcilation. It will be retriggered once the status of the ingress changes.
			return nil
		}
		for _, route := range routes {
			logger.Infof("Creating/Updating OpenShift Route for host %s", route.Spec.Host)
			if err := r.reconcileRoute(ctx, ing, route); err != nil {
				return fmt.Errorf("failed to create route for host %s: %v", route.Spec.Host, err)
			}
			delete(existingMap, route.Name)
		}
		// If routes remains in existingMap, it must be obsoleted routes. Clean them up.
		for _, rt := range existingMap {
			logger.Infof("Deleting obsoleted route for host: %s", rt.Spec.Host)
			if err := r.deleteRoute(ctx, &rt); err != nil {
				return err
			}
		}
	} else {
		if err := r.deleteRoutes(ctx, ing); err != nil {
			return err
		}
	}

	logger.Info("Ingress successfully synced")
	return nil
}

func (r *ReconcileIngress) deleteRoute(ctx context.Context, route *routev1.Route) error {
	logger := logging.FromContext(ctx)
	logger.Infof("Deleting OpenShift Route for host %s", route.Spec.Host)
	if err := r.client.Delete(ctx, route); err != nil {
		return fmt.Errorf("failed to delete obsoleted route for host %s: %v", route.Spec.Host, err)
	}
	logger.Infof("Deleted OpenShift Route %q in namespace %q", route.Name, route.Namespace)
	return nil
}

func (r *ReconcileIngress) deleteRoutes(ctx context.Context, ing *networkingv1alpha1.Ingress) error {
	routeList, err := r.routeList(ctx, ing)
	if err != nil {
		return fmt.Errorf("failed to list routes for deletion: %w", err)
	}

	for _, route := range routeList.Items {
		if err := r.deleteRoute(ctx, &route); err != nil {
			return fmt.Errorf("failed to delete routes: %w", err)
		}
	}
	return nil
}

func (r *ReconcileIngress) reconcileRoute(ctx context.Context, ci *networkingv1alpha1.Ingress, desired *routev1.Route) error {
	logger := logging.FromContext(ctx)

	// Check if this Route already exists
	route := &routev1.Route{}
	err := r.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, route)
	if err != nil && errors.IsNotFound(err) {
		err = r.client.Create(ctx, desired)
		if err != nil {
			logger.Errorf("Failed to create OpenShift Route %q in namespace %q: %v", desired.Name, desired.Namespace, err)
			return err
		}
		logger.Infof("Created OpenShift Route %q in namespace %q", desired.Name, desired.Namespace)
	} else if err != nil {
		return err
	} else if !equality.Semantic.DeepEqual(route.Spec, desired.Spec) {
		// Don't modify the informers copy
		existing := route.DeepCopy()
		existing.Spec = desired.Spec
		existing.Annotations = desired.Annotations
		err = r.client.Update(ctx, existing)
		if err != nil {
			logger.Errorf("Failed to update OpenShift Route %q in namespace %q: %v", desired.Name, desired.Namespace, err)
			return err
		}
	}

	return nil
}

func (r *ReconcileIngress) reconcileDeletion(ctx context.Context, ing *networkingv1alpha1.Ingress) error {
	logger := logging.FromContext(ctx)

	if len(ing.GetFinalizers()) == 0 || ing.GetFinalizers()[0] != "ocp-ingress" {
		return nil
	}

	if err := r.deleteRoutes(ctx, ing); err != nil {
		return err
	}

	logger.Infof("Removing finalizer for ingress %q", ing.GetName())
	ing.SetFinalizers(ing.GetFinalizers()[1:])
	return r.client.Update(ctx, ing)
}

func (r *ReconcileIngress) routeList(ctx context.Context, ing *networkingv1alpha1.Ingress) (routev1.RouteList, error) {
	ingressLabels := ing.GetLabels()
	listOpts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			networking.IngressLabelKey:     ing.GetName(),
			serving.RouteLabelKey:          ingressLabels[serving.RouteLabelKey],
			serving.RouteNamespaceLabelKey: ingressLabels[serving.RouteNamespaceLabelKey],
		}),
	}
	var routeList routev1.RouteList
	return routeList, r.client.List(ctx, listOpts, &routeList)
}

// Update the Status of the Ingress.  Caller is responsible for checking
// for semantic differences before calling.
func (r *ReconcileIngress) updateStatus(ctx context.Context, desired *networkingv1alpha1.Ingress) (*networkingv1alpha1.Ingress, error) {
	ing := &networkingv1alpha1.Ingress{}
	err := r.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, ing)
	if err != nil {
		return nil, err
	}

	// If there's nothing to update, just return.
	if reflect.DeepEqual(ing.Status, desired.Status) {
		return ing, nil
	}
	// Don't modify the informers copy
	existing := ing.DeepCopy()
	existing.Status = desired.Status
	err = r.client.Status().Update(ctx, existing)
	return existing, err
}

// appendIfAbsent append namespace to member if its not exist
func appendIfAbsent(members []string, routeNamespace string) ([]string, bool) {
	for _, val := range members {
		if val == routeNamespace {
			return members, false
		}
	}
	return append(members, routeNamespace), true
}
