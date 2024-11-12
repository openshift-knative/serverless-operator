package resources

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"knative.dev/networking/pkg/apis/networking"
	networkingv1alpha1 "knative.dev/networking/pkg/apis/networking/v1alpha1"
	"knative.dev/pkg/ptr"
	"knative.dev/serving/pkg/apis/serving"
)

const (
	localDomain     = "test.default.svc.cluster.local"
	externalDomain  = "public.default.domainName"
	externalDomain2 = "another.public.default.domainName"

	lbService   = "lb-service"
	lbNamespace = "lb-namespace"

	uid             = "8a7e9a9d-fbc6-11e9-a88e-0261aff8d6d8"
	routeName0      = "route-" + uid + "-323531366235"
	routeName1      = "route-" + uid + "-663738313063"
	customRouteName = "route-" + uid + "-323563643265"
)

func TestMakeRoute(t *testing.T) {
	tests := []struct {
		name    string
		ingress *networkingv1alpha1.Ingress
		want    []*routev1.Route
		wantErr error
		timeout string
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
					Labels: map[string]string{
						networking.IngressLabelKey:        "ingress",
						serving.RouteLabelKey:             "route1",
						serving.RouteNamespaceLabelKey:    "default",
						OpenShiftIngressLabelKey:          "ingress",
						OpenShiftIngressNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: DefaultTimeout,
					},
					Namespace: lbNamespace,
					Name:      routeName0,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain,
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   lbService,
						Weight: ptr.Int32(100),
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString(HTTPPort),
					},
					TLS: &routev1.TLSConfig{
						Termination:                   routev1.TLSTerminationEdge,
						InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyAllow,
					},
					WildcardPolicy: routev1.WildcardPolicyNone,
				},
			}},
		}, {
			name: "valid, default timeout modified by env var",
			ingress: ingress(withRules(
				rule(withHosts([]string{localDomain, externalDomain}))),
			),
			want: []*routev1.Route{{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						networking.IngressLabelKey:        "ingress",
						serving.RouteLabelKey:             "route1",
						serving.RouteNamespaceLabelKey:    "default",
						OpenShiftIngressLabelKey:          "ingress",
						OpenShiftIngressNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: "900s",
					},
					Namespace: lbNamespace,
					Name:      routeName0,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain,
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   lbService,
						Weight: ptr.Int32(100),
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString(HTTPPort),
					},
					TLS: &routev1.TLSConfig{
						Termination:                   routev1.TLSTerminationEdge,
						InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyAllow,
					},
					WildcardPolicy: routev1.WildcardPolicyNone,
				},
			}},
			timeout: "900",
		},
		{
			name: "valid but disabled",
			ingress: ingress(withDisabledAnnotation, withRules(
				rule(withHosts([]string{localDomain, externalDomain}))),
			),
			want: []*routev1.Route{},
		},
		{
			name:    "valid but cluster-local",
			ingress: ingress(withRules(rule(withHosts([]string{localDomain, externalDomain}), withLocalVisibilityRule))),
			want:    []*routev1.Route{},
		},
		{
			name: "valid, multiple rules",
			ingress: ingress(withRules(
				rule(withHosts([]string{localDomain, externalDomain})),
				rule(withHosts([]string{localDomain, externalDomain2})),
			)),
			want: []*routev1.Route{{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						networking.IngressLabelKey:        "ingress",
						serving.RouteLabelKey:             "route1",
						serving.RouteNamespaceLabelKey:    "default",
						OpenShiftIngressLabelKey:          "ingress",
						OpenShiftIngressNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: DefaultTimeout,
					},
					Namespace: lbNamespace,
					Name:      routeName0,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain,
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   lbService,
						Weight: ptr.Int32(100),
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString(HTTPPort),
					},
					TLS: &routev1.TLSConfig{
						Termination:                   routev1.TLSTerminationEdge,
						InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyAllow,
					},
					WildcardPolicy: routev1.WildcardPolicyNone,
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						networking.IngressLabelKey:        "ingress",
						serving.RouteLabelKey:             "route1",
						serving.RouteNamespaceLabelKey:    "default",
						OpenShiftIngressLabelKey:          "ingress",
						OpenShiftIngressNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: DefaultTimeout,
					},
					Namespace: lbNamespace,
					Name:      routeName1,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain2,
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   lbService,
						Weight: ptr.Int32(100),
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString(HTTPPort),
					},

					TLS: &routev1.TLSConfig{
						Termination:                   routev1.TLSTerminationEdge,
						InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyAllow,
					},
					WildcardPolicy: routev1.WildcardPolicyNone,
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
					Labels: map[string]string{
						networking.IngressLabelKey:        "ingress",
						serving.RouteLabelKey:             "route1",
						serving.RouteNamespaceLabelKey:    "default",
						OpenShiftIngressLabelKey:          "ingress",
						OpenShiftIngressNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: DefaultTimeout,
					},
					Namespace: lbNamespace,
					Name:      routeName1,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain2,
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   lbService,
						Weight: ptr.Int32(100),
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString(HTTPPort),
					},

					TLS: &routev1.TLSConfig{
						Termination:                   routev1.TLSTerminationEdge,
						InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyAllow,
					},
					WildcardPolicy: routev1.WildcardPolicyNone,
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
			name: "valid, passthrough by annotation",
			ingress: ingress(withPassthroughAnnotation, withRules(
				rule(withHosts([]string{localDomain, externalDomain}))),
			),
			want: []*routev1.Route{{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						networking.IngressLabelKey:        "ingress",
						serving.RouteLabelKey:             "route1",
						serving.RouteNamespaceLabelKey:    "default",
						OpenShiftIngressLabelKey:          "ingress",
						OpenShiftIngressNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation:                DefaultTimeout,
						EnablePassthroughRouteAnnotation: "true",
					},
					Namespace: lbNamespace,
					Name:      routeName0,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain,
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   lbService,
						Weight: ptr.Int32(100),
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString(HTTPSPort),
					},
					TLS: &routev1.TLSConfig{
						Termination:                   routev1.TLSTerminationPassthrough,
						InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
					},
					WildcardPolicy: routev1.WildcardPolicyNone,
				},
			}},
		},
		{
			name: "valid, passthrough by BYO cert",
			ingress: ingress(
				withTLS(networkingv1alpha1.IngressTLS{Hosts: []string{"custom.example.com"}, SecretName: "someSecretName"}),
				withRules(rule(withHosts([]string{"custom.example.com"}))),
			),
			want: []*routev1.Route{{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						networking.IngressLabelKey:        "ingress",
						serving.RouteLabelKey:             "route1",
						serving.RouteNamespaceLabelKey:    "default",
						OpenShiftIngressLabelKey:          "ingress",
						OpenShiftIngressNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: DefaultTimeout,
					},
					Namespace: lbNamespace,
					Name:      customRouteName,
				},
				Spec: routev1.RouteSpec{
					Host: "custom.example.com",
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   lbService,
						Weight: ptr.Int32(100),
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString(HTTPSPort),
					},
					TLS: &routev1.TLSConfig{
						Termination:                   routev1.TLSTerminationPassthrough,
						InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
					},
					WildcardPolicy: routev1.WildcardPolicyNone,
				},
			}},
		},
		{
			name: "valid, http redirect option",
			ingress: ingress(
				withRules(rule(withHosts([]string{localDomain, externalDomain}))),
				withRedirect(),
			),
			want: []*routev1.Route{{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						networking.IngressLabelKey:        "ingress",
						serving.RouteLabelKey:             "route1",
						serving.RouteNamespaceLabelKey:    "default",
						OpenShiftIngressLabelKey:          "ingress",
						OpenShiftIngressNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: DefaultTimeout,
					},
					Namespace: lbNamespace,
					Name:      routeName0,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain,
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   lbService,
						Weight: ptr.Int32(100),
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString(HTTPPort),
					},
					TLS: &routev1.TLSConfig{
						Termination:                   routev1.TLSTerminationEdge,
						InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
					},
					WildcardPolicy: routev1.WildcardPolicyNone,
				},
			}},
		},
		{
			name: "system-internal-tls is enabled",
			ingress: ingress(
				withRules(rule(withHosts([]string{localDomain, externalDomain}), withHTTPSBackendService())),
			),
			want: []*routev1.Route{{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						networking.IngressLabelKey:        "ingress",
						serving.RouteLabelKey:             "route1",
						serving.RouteNamespaceLabelKey:    "default",
						OpenShiftIngressLabelKey:          "ingress",
						OpenShiftIngressNamespaceLabelKey: "default",
					},
					Annotations: map[string]string{
						TimeoutAnnotation: DefaultTimeout,
					},
					Namespace: lbNamespace,
					Name:      routeName0,
				},
				Spec: routev1.RouteSpec{
					Host: externalDomain,
					To: routev1.RouteTargetReference{
						Kind:   "Service",
						Name:   lbService,
						Weight: ptr.Int32(100),
					},
					Port: &routev1.RoutePort{
						TargetPort: intstr.FromString(HTTPSPort),
					},
					TLS: &routev1.TLSConfig{
						Termination:                   routev1.TLSTerminationPassthrough,
						InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
					},
					WildcardPolicy: routev1.WildcardPolicyNone,
				},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.timeout != "" {
				t.Setenv(HAProxyTimeoutEnv, test.timeout)
				DefaultTimeout = getDefaultHAProxyTimeout()
				defer func() {
					t.Setenv(HAProxyTimeoutEnv, "")
					DefaultTimeout = getDefaultHAProxyTimeout()
				}()
			}
			routes, err := MakeRoutes(test.ingress)
			if test.want != nil && !cmp.Equal(routes, test.want) {
				t.Errorf("got = %v, want: %v, diff: %s", routes, test.want, cmp.Diff(routes, test.want))
			}
			if !errors.Is(err, test.wantErr) {
				t.Errorf("got = %v, want: %v", err, test.wantErr)
			}

		})
	}
}

func ingress(options ...ingressOption) *networkingv1alpha1.Ingress {
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
		Status: networkingv1alpha1.IngressStatus{
			PublicLoadBalancer: &networkingv1alpha1.LoadBalancerStatus{
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
		Visibility: networkingv1alpha1.IngressVisibilityExternalIP,
		HTTP: &networkingv1alpha1.HTTPIngressRuleValue{
			Paths: []networkingv1alpha1.HTTPIngressPath{{}},
		},
	}

	for _, opt := range options {
		opt(&rule)
	}

	return rule
}

type ingressOption func(*networkingv1alpha1.Ingress)

func withTLS(tls ...networkingv1alpha1.IngressTLS) ingressOption {
	return func(ing *networkingv1alpha1.Ingress) {
		ing.Spec.TLS = tls
	}
}

func withRules(rules ...networkingv1alpha1.IngressRule) ingressOption {
	return func(ing *networkingv1alpha1.Ingress) {
		ing.Spec.Rules = rules
	}
}

func withRedirect() ingressOption {
	return func(ing *networkingv1alpha1.Ingress) {
		ing.Spec.HTTPOption = networkingv1alpha1.HTTPOptionRedirected
	}
}

func withDisabledAnnotation(ing *networkingv1alpha1.Ingress) {
	annos := ing.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
	}
	annos[DisableRouteAnnotation] = ""
	ing.SetAnnotations(annos)
}

func withPassthroughAnnotation(ing *networkingv1alpha1.Ingress) {
	annos := ing.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
	}
	annos[EnablePassthroughRouteAnnotation] = "true"
	ing.SetAnnotations(annos)
}

func withLBInternalDomain(domain string) ingressOption {
	return func(ing *networkingv1alpha1.Ingress) {
		ing.Status.PublicLoadBalancer.Ingress[0].DomainInternal = domain
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

func withHTTPSBackendService() ruleOption {
	return func(rule *networkingv1alpha1.IngressRule) {
		rule.HTTP.Paths = []networkingv1alpha1.HTTPIngressPath{{
			Splits: []networkingv1alpha1.IngressBackendSplit{{
				IngressBackend: networkingv1alpha1.IngressBackend{
					ServiceNamespace: "ns",
					ServiceName:      "something",
					ServicePort:      intstr.FromInt(networking.ServiceHTTPSPort),
				},
			}},
		}}
	}
}
