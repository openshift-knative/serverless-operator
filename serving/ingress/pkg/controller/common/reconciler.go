package common

import (
	"context"
	"fmt"

	"github.com/openshift-knative/serverless-operator/serving/ingress/pkg/controller/resources"
	routev1 "github.com/openshift/api/route/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/logging"
	"knative.dev/serving/pkg/apis/networking"
	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
	"knative.dev/serving/pkg/apis/serving"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const NetworkPolicyAllowAllName = "knative-serving-allow-all"

type BaseIngressReconciler struct {
	Client client.Client
}

func (r *BaseIngressReconciler) ReconcileIngress(ctx context.Context, ci networkingv1alpha1.IngressAccessor) error {
	logger := logging.FromContext(ctx)

	// Delete obsoleted NeworkPolicy which required for ServiceMesh.
	// TODO: This should be removed in the future version.
	if err := r.deleteNetworkPolicy(ctx, ci); err != nil {
		return err
	}

	if ci.GetDeletionTimestamp() != nil {
		return r.reconcileDeletion(ctx, ci)
	}

	logger.Infof("Reconciling ingress :%v", ci)

	exposed := ci.GetSpec().Visibility == networkingv1alpha1.IngressVisibilityExternalIP
	if exposed {
		selector, existing, err := r.routeList(ctx, ci)
		if err != nil {
			logger.Errorf("Failed to list openshift routes %v", err)
			return err
		}
		existingMap := routeMap(existing, selector)

		routes, err := resources.MakeRoutes(ci)
		if err != nil {
			logger.Warnf("Failed to generate routes from ingress %v", err)
			// Returning nil aborts the reconcilation. It will be retriggered once the status of the ingress changes.
			return nil
		}
		for _, route := range routes {
			logger.Infof("Creating/Updating OpenShift Route for host %s", route.Spec.Host)
			if err := r.reconcileRoute(ctx, ci, route); err != nil {
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
		if err := r.deleteRoutes(ctx, ci); err != nil {
			return err
		}
	}

	logger.Info("Ingress successfully synced")
	return nil
}

func (r *BaseIngressReconciler) deleteNetworkPolicy(ctx context.Context, ci networkingv1alpha1.IngressAccessor) error {
	logger := logging.FromContext(ctx)

	networkPolicy := &networkingv1.NetworkPolicy{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: NetworkPolicyAllowAllName, Namespace: ci.GetNamespace()}, networkPolicy)

	if errors.IsNotFound(err) {
		// Doesn't exist, so no need to try to delete it
		return nil
	} else if err != nil {
		return err
	} else {
		logger.Infof("Deleting NetworkPolicy %q in namespace %q", NetworkPolicyAllowAllName, ci.GetNamespace())
		if err := r.Client.Delete(ctx, networkPolicy); err != nil {
			logger.Errorf("Failed to delete NetworkPolicy %q in namespace %q: %v", NetworkPolicyAllowAllName, ci.GetNamespace(), err)
			return err
		}
		logger.Infof("Deleted NetworkPolicy %q in namespace %q", NetworkPolicyAllowAllName, ci.GetNamespace())
	}
	return nil
}

func routeMap(routes routev1.RouteList, selector map[string]string) map[string]routev1.Route {
	mp := make(map[string]routev1.Route, len(routes.Items))
	for _, route := range routes.Items {
		// TODO: This routeFilter is used only for testing as fake client does not support list option
		// and we can't bump the osdk version quickly. ref:
		// https://github.com/openshift-knative/serverless-operator/serving/ingress/pull/24#discussion_r341804021
		if routeLabelFilter(route, selector) {
			mp[route.Name] = route
		}
	}
	return mp
}

// routeLabelFilter verifies if the route has required labels.
func routeLabelFilter(route routev1.Route, selector map[string]string) bool {
	labels := route.GetLabels()
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

func (r *BaseIngressReconciler) deleteRoute(ctx context.Context, route *routev1.Route) error {
	logger := logging.FromContext(ctx)
	logger.Infof("Deleting OpenShift Route for host %s", route.Spec.Host)
	if err := r.Client.Delete(ctx, route); err != nil {
		return fmt.Errorf("failed to delete obsoleted route for host %s: %v", route.Spec.Host, err)
	}
	logger.Infof("Deleted OpenShift Route %q in namespace %q", route.Name, route.Namespace)
	return nil
}

func (r *BaseIngressReconciler) deleteRoutes(ctx context.Context, ci networkingv1alpha1.IngressAccessor) error {
	listOpts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			networking.IngressLabelKey: ci.GetName(),
		}),
	}
	var routeList routev1.RouteList
	if err := r.Client.List(ctx, listOpts, &routeList); err != nil {
		return err
	}

	for _, route := range routeList.Items {
		if err := r.deleteRoute(ctx, &route); err != nil {
			return err
		}
	}
	return nil
}

func (r *BaseIngressReconciler) reconcileRoute(ctx context.Context, ci networkingv1alpha1.IngressAccessor, desired *routev1.Route) error {
	logger := logging.FromContext(ctx)

	// Check if this Route already exists
	route := &routev1.Route{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, route)
	if err != nil && errors.IsNotFound(err) {
		err = r.Client.Create(ctx, desired)
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
		err = r.Client.Update(ctx, existing)
		if err != nil {
			logger.Errorf("Failed to update OpenShift Route %q in namespace %q: %v", desired.Name, desired.Namespace, err)
			return err
		}
	}

	return nil
}

func (r *BaseIngressReconciler) reconcileDeletion(ctx context.Context, ci networkingv1alpha1.IngressAccessor) error {
	logger := logging.FromContext(ctx)

	if len(ci.GetFinalizers()) == 0 || ci.GetFinalizers()[0] != "ocp-ingress" {
		return nil
	}
	// get list of ingress object for a namespace
	ingressList := networkingv1alpha1.IngressList{}
	if err := r.Client.List(ctx, &client.ListOptions{
		Namespace: ci.GetNamespace(),
	}, &ingressList); err != nil {
		return err
	}

	_, list, err := r.routeList(ctx, ci)
	if err != nil {
		logger.Errorf("Failed to list openshift routes %v", err)
		return err
	}
	for i := range list.Items {
		if err := r.deleteRoute(ctx, &list.Items[i]); err != nil {
			logger.Errorf("Failed to delete openshift route %q with error %v", list.Items[i].Name, err)
			return err
		}
	}

	logger.Infof("Removing finalizer for ingress %q", ci.GetName())
	ci.SetFinalizers(ci.GetFinalizers()[1:])
	return r.Client.Update(ctx, ci)
}

// AppendIfAbsent append namespace to member if its not exist
func AppendIfAbsent(members []string, routeNamespace string) ([]string, bool) {
	for _, val := range members {
		if val == routeNamespace {
			return members, false
		}
	}
	return append(members, routeNamespace), true
}

func (r *BaseIngressReconciler) routeList(ctx context.Context, ci networkingv1alpha1.IngressAccessor) (map[string]string, routev1.RouteList, error) {
	ingressLabels := ci.GetLabels()
	selector := map[string]string{
		networking.IngressLabelKey:     ci.GetName(),
		serving.RouteLabelKey:          ingressLabels[serving.RouteLabelKey],
		serving.RouteNamespaceLabelKey: ingressLabels[serving.RouteNamespaceLabelKey],
	}
	listOpts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(selector),
	}
	var routeList routev1.RouteList
	return selector, routeList, r.Client.List(ctx, listOpts, &routeList)
}
