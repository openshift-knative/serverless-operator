package resources

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"knative.dev/pkg/kmeta"
	"knative.dev/serving/pkg/apis/networking"
	networkingv1alpha1 "knative.dev/serving/pkg/apis/networking/v1alpha1"
	"knative.dev/serving/pkg/apis/serving"
)

const (
	localDomain     = "test.default.svc.cluster.local"
	externalDomain  = "public.default.domainName"
	externalDomain2 = "another.public.default.domainName"

	lbService   = "lb-service"
	lbNamespace = "lb-namespace"

	uid        = "8a7e9a9d-fbc6-11e9-a88e-0261aff8d6d8"
	routeName0 = "route-" + uid + "-323531366235"
	routeName1 = "route-" + uid + "-663738313063"
)

var ownerRef = *kmeta.NewControllerRef(ingress())

func TestMakeRoute(t *testing.T) {
	tests := []struct {
		name    string
		ingress networkingv1alpha1.IngressAccessor
		want    []*routev1.Route
		wantErr error
	}{
		{
			name:    "no rules",
			ingress: ingress(),
			want:    []*routev1.Route{},
		},
		{
			name: "skip internal host name",
			ingress: ingress(withRules(
				rule(withHosts([]string{localDomain}))),
			),
			want: []*routev1.Route{},
		},
		{
			name: "valid, default timeout",
			ingress: ingress(withRules(
				rule(withHosts([]string{localDomain, externalDomain}))),
			),
			want: []*routev1.Route{{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{ownerRef},
					Labels: map[string]string{
						networking.IngressLabelKey:     "ingress",
						serving.RouteLabelKey:          "route1",
						serving.RouteNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: "600s",
					},
					Namespace: lbNamespace,
					Name:      routeName0,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain,
					To: routev1.RouteTargetReference{
						Kind: "Service",
						Name: lbService,
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString("http2"),
					},
				},
			}},
		},
		{
			name: "valid but disabled",
			ingress: ingress(withDisabledAnnotation, withRules(
				rule(withHosts([]string{localDomain, externalDomain}))),
			),
			want: []*routev1.Route{},
		},
		{
			name: "valid but cluster-local",
			ingress: ingress(withLocalVisibility, withRules(
				rule(withHosts([]string{localDomain, externalDomain}))),
			),
			want: []*routev1.Route{},
		},
		{
			name: "valid, with timeout",
			ingress: ingress(withRules(
				rule(withHosts([]string{localDomain, externalDomain}), withTimeout(1*time.Hour))),
			),
			want: []*routev1.Route{{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{ownerRef},
					Labels: map[string]string{
						networking.IngressLabelKey:     "ingress",
						serving.RouteLabelKey:          "route1",
						serving.RouteNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: "3600s",
					},
					Namespace: lbNamespace,
					Name:      routeName0,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain,
					To: routev1.RouteTargetReference{
						Kind: "Service",
						Name: lbService,
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString("http2"),
					},
				},
			}},
		},
		{
			name: "valid, multiple rules",
			ingress: ingress(withRules(
				rule(withHosts([]string{localDomain, externalDomain})),
				rule(withHosts([]string{localDomain, externalDomain2})),
			)),
			want: []*routev1.Route{{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{ownerRef},
					Labels: map[string]string{
						networking.IngressLabelKey:     "ingress",
						serving.RouteLabelKey:          "route1",
						serving.RouteNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: "600s",
					},
					Namespace: lbNamespace,
					Name:      routeName0,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain,
					To: routev1.RouteTargetReference{
						Kind: "Service",
						Name: lbService,
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString("http2"),
					},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{ownerRef},
					Labels: map[string]string{
						networking.IngressLabelKey:     "ingress",
						serving.RouteLabelKey:          "route1",
						serving.RouteNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: "600s",
					},
					Namespace: lbNamespace,
					Name:      routeName1,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain2,
					To: routev1.RouteTargetReference{
						Kind: "Service",
						Name: lbService,
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString("http2"),
					},
				},
			}},
		},
		{
			name: "valid, multiple rules, one local",
			ingress: ingress(withRules(
				rule(withHosts([]string{localDomain, externalDomain}), withLocalVisibilityRule),
				rule(withHosts([]string{localDomain, externalDomain2})),
			)),
			want: []*routev1.Route{{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{ownerRef},
					Labels: map[string]string{
						networking.IngressLabelKey:     "ingress",
						serving.RouteLabelKey:          "route1",
						serving.RouteNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: "600s",
					},
					Namespace: lbNamespace,
					Name:      routeName1,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain2,
					To: routev1.RouteTargetReference{
						Kind: "Service",
						Name: lbService,
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString("http2"),
					},
				},
			}},
		},
		{
			name: "invalid LB domain",
			ingress: ingress(withLBInternalDomain("not.a.private.name"), withRules(
				rule(withHosts([]string{localDomain, externalDomain}))),
			),
			wantErr: ErrNoValidLoadbalancerDomain,
		},
		{
			name: "tls: passthrough termination",
			ingress: ingress(withTLSTerminationAnnotation("passthrough"), withRules(
				rule(withHosts([]string{localDomain, externalDomain}))),
			),
			want: []*routev1.Route{{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{ownerRef},
					Labels: map[string]string{
						networking.IngressLabelKey:     "ingress",
						serving.RouteLabelKey:          "route1",
						serving.RouteNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation:        "600s",
						TLSTerminationAnnotation: "passthrough",
					},
					Namespace: lbNamespace,
					Name:      routeName0,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain,
					To: routev1.RouteTargetReference{
						Kind: "Service",
						Name: lbService,
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString("https"),
					},
					TLS: &routev1.TLSConfig{
						Termination: routev1.TLSTerminationPassthrough,
					},
				},
			}},
		},
		{
			name: "tls: unsupported termination",
			ingress: ingress(withTLSTerminationAnnotation("edge"), withRules(
				rule(withHosts([]string{localDomain, externalDomain}))),
			),
			wantErr: ErrNotSupportedTLSTermination,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			routes, err := MakeRoutes(test.ingress)
			if test.want != nil && !cmp.Equal(routes, test.want) {
				t.Errorf("got = %v, want: %v, diff: %s", routes, test.want, cmp.Diff(routes, test.want))
			}
			if err != test.wantErr {
				t.Errorf("got = %v, want: %v", err, test.wantErr)
			}
		})
	}
}

func ingress(options ...ingressOption) networkingv1alpha1.IngressAccessor {
	ing := &networkingv1alpha1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				serving.RouteLabelKey:          "route1",
				serving.RouteNamespaceLabelKey: "default",
			},
			Namespace: "default",
			Name:      "ingress",
			UID:       uid,
		},
		Spec: networkingv1alpha1.IngressSpec{
			Visibility: networkingv1alpha1.IngressVisibilityExternalIP,
		},
		Status: networkingv1alpha1.IngressStatus{
			LoadBalancer: &networkingv1alpha1.LoadBalancerStatus{
				Ingress: []networkingv1alpha1.LoadBalancerIngressStatus{{
					DomainInternal: fmt.Sprintf("%s.%s.svc.cluster.local", lbService, lbNamespace),
				}},
			},
		},
	}

	for _, opt := range options {
		opt(ing)
	}

	return ing
}

func rule(options ...ruleOption) networkingv1alpha1.IngressRule {
	rule := networkingv1alpha1.IngressRule{
		HTTP: &networkingv1alpha1.HTTPIngressRuleValue{
			Paths: []networkingv1alpha1.HTTPIngressPath{{}},
		},
	}

	for _, opt := range options {
		opt(&rule)
	}

	return rule
}

type ingressOption func(networkingv1alpha1.IngressAccessor)

func withRules(rules ...networkingv1alpha1.IngressRule) ingressOption {
	return func(ing networkingv1alpha1.IngressAccessor) {
		spec := ing.GetSpec()
		spec.Rules = rules
	}
}

func withDisabledAnnotation(ing networkingv1alpha1.IngressAccessor) {
	annos := ing.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
	}
	annos[DisableRouteAnnotation] = ""
	ing.SetAnnotations(annos)
}

func withTLSTerminationAnnotation(value string) ingressOption {
	return func(ing networkingv1alpha1.IngressAccessor) {
		annos := ing.GetAnnotations()
		if annos == nil {
			annos = map[string]string{}
		}
		annos[TLSTerminationAnnotation] = value
		ing.SetAnnotations(annos)
	}
}

func withLocalVisibility(ing networkingv1alpha1.IngressAccessor) {
	ing.GetSpec().Visibility = networkingv1alpha1.IngressVisibilityClusterLocal
}

func withLBInternalDomain(domain string) ingressOption {
	return func(ing networkingv1alpha1.IngressAccessor) {
		status := ing.GetStatus()
		status.LoadBalancer.Ingress[0].DomainInternal = domain
	}
}

type ruleOption func(*networkingv1alpha1.IngressRule)

func withLocalVisibilityRule(rule *networkingv1alpha1.IngressRule) {
	rule.Visibility = networkingv1alpha1.IngressVisibilityClusterLocal
}

func withHosts(hosts []string) ruleOption {
	return func(rule *networkingv1alpha1.IngressRule) {
		rule.Hosts = hosts
	}
}

func withTimeout(timeout time.Duration) ruleOption {
	return func(rule *networkingv1alpha1.IngressRule) {
		rule.HTTP = &networkingv1alpha1.HTTPIngressRuleValue{
			Paths: []networkingv1alpha1.HTTPIngressPath{{
				Timeout: &metav1.Duration{Duration: timeout},
			}},
		}
	}
}
