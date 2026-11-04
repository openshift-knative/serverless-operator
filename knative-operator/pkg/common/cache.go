package common

import (
	"maps"
	"os"
	"slices"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// EnableNamespaceScopingEnvKey opts in to scoping the manager cache to the
	// namespaces the operator manages when set to "true". Off by default.
	EnableNamespaceScopingEnvKey = "ENABLE_NAMESPACE_SCOPING"

	// ConfigManagedNamespace hosts the console dashboards ConfigMaps.
	ConfigManagedNamespace = "openshift-config-managed"
)

// scopedNamespaceEnvKeys hold the namespaces in which the operator reads
// namespaced objects through the cache: its own namespace and the namespaces
// the KnativeServing, KnativeEventing and KnativeKafka CRs are pinned to.
var scopedNamespaceEnvKeys = []string{
	NamespaceEnvKey,
	"REQUIRED_SERVING_NAMESPACE",
	"REQUIRED_EVENTING_NAMESPACE",
	"REQUIRED_KAFKA_NAMESPACE",
	// Added defensively: the operator does not currently read objects in the
	// serving ingress namespace through the cache, but future reads there
	// would otherwise fail with scoping enabled.
	"REQUIRED_SERVING_INGRESS_NAMESPACE",
}

// CacheOptions returns the manager cache configuration.
//
// By default every watched kind (ConfigMaps, Secrets, Services, ...) is listed
// and cached cluster-wide, which makes memory usage grow with the size of the
// cluster rather than with the number of objects the operator manages.
//
// With ENABLE_NAMESPACE_SCOPING=true, namespaced kinds are cached only in the
// namespaces the operator manages, and managedFields are stripped. Exceptions:
//   - Deployments stay cluster-wide, as the source deployment discovery
//     controller watches source adapters in user namespaces.
//   - Cluster-scoped kinds are unaffected by namespace scoping.
//   - Unstructured reads (manifestival) bypass the cache entirely.
//
// Scoping is only applied when all namespaces are known; otherwise the cache
// falls back to the default cluster-wide behavior.
func CacheOptions() cache.Options {
	raw, set := os.LookupEnv(EnableNamespaceScopingEnvKey)
	if !set {
		return cache.Options{}
	}
	enabled, err := strconv.ParseBool(raw)
	if err != nil {
		Log.Info("Invalid value, namespace scoping disabled", "env", EnableNamespaceScopingEnvKey, "value", raw)
		return cache.Options{}
	}
	if !enabled {
		return cache.Options{}
	}

	namespaces := map[string]cache.Config{ConfigManagedNamespace: {}}
	for _, key := range scopedNamespaceEnvKeys {
		ns := os.Getenv(key)
		if ns == "" {
			Log.Info("Namespace env var not set, namespace scoping disabled", "env", key)
			return cache.Options{}
		}
		namespaces[ns] = cache.Config{}
	}

	Log.Info("Namespace scoping enabled for the cache", "namespaces", slices.Sorted(maps.Keys(namespaces)))
	return cache.Options{
		DefaultTransform:  cache.TransformStripManagedFields(),
		DefaultNamespaces: namespaces,
		ByObject: map[client.Object]cache.ByObject{
			&appsv1.Deployment{}: {
				Namespaces: map[string]cache.Config{cache.AllNamespaces: {}},
			},
		},
	}
}
