package ingress

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"knative.dev/networking/pkg/apis/networking/v1alpha1"
	ingressreconciler "knative.dev/networking/pkg/client/injection/reconciler/networking/v1alpha1/ingress"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/reconciler/ingress/resources"
	routev1 "github.com/openshift/api/route/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	routev1lister "github.com/openshift/client-go/route/listers/route/v1"
)

// Reconciler implements controller.Reconciler for Ingress resources.
type Reconciler struct {
	routeLister routev1lister.RouteLister
	routeClient routev1client.RouteV1Interface
}

var _ ingressreconciler.Interface = (*Reconciler)(nil)
var _ ingressreconciler.Finalizer = (*Reconciler)(nil)

// FinalizeKind finalizes ingress resource.
func (r *Reconciler) FinalizeKind(ctx context.Context, ing *v1alpha1.Ingress) reconciler.Event {
	routes, err := r.routeList(ing)
	if err != nil {
		return fmt.Errorf("failed to list routes for deletion: %w", err)
	}

	for _, route := range routes {
		if err := r.deleteRoute(ctx, route); err != nil {
			return fmt.Errorf("failed to delete routes: %w", err)
		}
	}
	return nil
}

// ReconcileKind reconciles ingress resource.
func (r *Reconciler) ReconcileKind(ctx context.Context, ing *v1alpha1.Ingress) reconciler.Event {
	logger := logging.FromContext(ctx)

	existingMap, err := r.routeList(ing)
	if err != nil {
		return fmt.Errorf("failed to list routes: %w", err)
	}

	routes, err := resources.MakeRoutes(ing)
	if err != nil {
		logger.Warnf("Failed to generate routes from ingress %v", err)
		// Returning nil aborts the reconciliation. It will be retriggered once the status of the ingress changes.
		return nil
	}
	for _, route := range routes {
		if err := r.reconcileRoute(ctx, route); err != nil {
			return err
		}
		delete(existingMap, route.Name)
	}
	// If routes remains in existingMap, it must be obsoleted routes. Clean them up.
	for _, rt := range existingMap {
		if err := r.deleteRoute(ctx, rt); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) deleteRoute(ctx context.Context, route *routev1.Route) error {
	logger := logging.FromContext(ctx)
	logger.Infof("Deleting route %s(%s)", route.Name, route.Spec.Host)
	if err := r.routeClient.Routes(route.Namespace).Delete(ctx, route.Name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	return nil
}

func (r *Reconciler) reconcileRoute(ctx context.Context, desired *routev1.Route) error {
	logger := logging.FromContext(ctx)

	// Check if this Route already exists
	route, err := r.routeLister.Routes(desired.Namespace).Get(desired.Name)
	if errors.IsNotFound(err) {
		logger.Infof("Creating route %s(%s)", desired.Name, desired.Spec.Host)
		if _, err := r.routeClient.Routes(desired.Namespace).Create(ctx, desired, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create route :%w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get route: %w", err)
	} else if !equality.Semantic.DeepEqual(route.Spec, desired.Spec) ||
		!equality.Semantic.DeepEqual(route.Annotations, desired.Annotations) ||
		!equality.Semantic.DeepEqual(route.Labels, desired.Labels) {
		// Don't modify the informers copy
		existing := route.DeepCopy()
		existing.Spec = desired.Spec
		existing.Annotations = desired.Annotations
		existing.Labels = desired.Labels

		if _, err := r.routeClient.Routes(existing.Namespace).Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("failed to update route :%w", err)
		}
	}

	return nil
}

func (r *Reconciler) routeList(ing *v1alpha1.Ingress) (map[string]*routev1.Route, error) {
	routes := make(map[string]*routev1.Route)

	// List routes by the downstream label.
	rs, err := r.routeLister.List(labels.SelectorFromSet(map[string]string{
		resources.OpenShiftIngressLabelKey:          ing.GetName(),
		resources.OpenShiftIngressNamespaceLabelKey: ing.GetNamespace(),
	}))
	if err != nil {
		return nil, err
	}

	for _, r := range rs {
		routes[r.Name] = r
	}
	return routes, nil
}
