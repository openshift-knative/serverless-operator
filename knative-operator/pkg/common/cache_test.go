package common_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/openshift-knative/serverless-operator/knative-operator/pkg/common"
)

var scopedEnv = map[string]string{
	common.NamespaceEnvKey:               "openshift-serverless",
	"REQUIRED_SERVING_NAMESPACE":         "knative-serving",
	"REQUIRED_EVENTING_NAMESPACE":        "knative-eventing",
	"REQUIRED_KAFKA_NAMESPACE":           "knative-eventing",
	"REQUIRED_SERVING_INGRESS_NAMESPACE": "knative-serving-ingress",
}

var scopedNamespaces = []string{
	"knative-eventing", "knative-serving", "knative-serving-ingress", "openshift-config-managed", "openshift-serverless",
}

func TestCacheOptionsNamespaceScoping(t *testing.T) {
	cases := []struct {
		name       string
		enable     *string // nil leaves ENABLE_NAMESPACE_SCOPING unset
		unset      []string
		namespaces []string // nil means default cluster-wide cache
	}{{
		name: "opt-in not set",
	}, {
		name:   "opt-in false",
		enable: ptr("false"),
	}, {
		name:   "opt-in invalid",
		enable: ptr("yes-please"),
	}, {
		name:       "opt-in true",
		enable:     ptr("true"),
		namespaces: scopedNamespaces,
	}, {
		name:   "opt-in true, serving ingress namespace unknown",
		enable: ptr("true"),
		unset:  []string{"REQUIRED_SERVING_INGRESS_NAMESPACE"},
	}, {
		name:   "opt-in true, operator namespace unknown",
		enable: ptr("true"),
		unset:  []string{common.NamespaceEnvKey},
	}, {
		name:   "opt-in true, serving namespace not enforced",
		enable: ptr("true"),
		unset:  []string{"REQUIRED_SERVING_NAMESPACE"},
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for k, v := range scopedEnv {
				t.Setenv(k, v)
			}
			for _, k := range c.unset {
				t.Setenv(k, "")
			}
			if c.enable != nil {
				t.Setenv(common.EnableNamespaceScopingEnvKey, *c.enable)
			}

			opts := common.CacheOptions()

			if c.namespaces == nil {
				if opts.DefaultNamespaces != nil || opts.ByObject != nil || opts.DefaultTransform != nil {
					t.Errorf("expected default cache options, got DefaultNamespaces=%v ByObject=%v DefaultTransform set=%t",
						opts.DefaultNamespaces, opts.ByObject, opts.DefaultTransform != nil)
				}
				return
			}

			if opts.DefaultTransform == nil {
				t.Error("DefaultTransform must strip managedFields when scoping is enabled")
			}

			got := slices.Sorted(maps.Keys(opts.DefaultNamespaces))
			if diff := cmp.Diff(c.namespaces, got); diff != "" {
				t.Errorf("DefaultNamespaces (-want, +got):\n%s", diff)
			}

			// Source deployment discovery needs Deployments in all namespaces.
			var deploymentNs map[string]cache.Config
			for obj, byObj := range opts.ByObject {
				if _, ok := obj.(*appsv1.Deployment); ok {
					deploymentNs = byObj.Namespaces
				}
			}
			if _, ok := deploymentNs[cache.AllNamespaces]; !ok || len(deploymentNs) != 1 {
				t.Errorf("Deployments must be cached in all namespaces, got %v", deploymentNs)
			}
		})
	}
}

func TestCacheOptionsStripsManagedFields(t *testing.T) {
	for k, v := range scopedEnv {
		t.Setenv(k, v)
	}
	t.Setenv(common.EnableNamespaceScopingEnvKey, "true")

	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:          "foo",
		ManagedFields: []metav1.ManagedFieldsEntry{{Manager: "kubectl"}},
	}}

	out, err := common.CacheOptions().DefaultTransform(cm)
	if err != nil {
		t.Fatal(err)
	}
	if mf := out.(*corev1.ConfigMap).ManagedFields; mf != nil {
		t.Errorf("expected managedFields to be stripped, got %v", mf)
	}
}

func ptr(s string) *string { return &s }
