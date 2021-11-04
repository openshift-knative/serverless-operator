package knativekafka

import (
	"testing"

	mf "github.com/manifestival/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var notAllowed = mf.Any(
	mf.All(mf.ByKind("ConfigMap"), mf.ByName("config-tracing")),
	mf.All(mf.ByKind("ConfigMap"), mf.ByName("config-observability")),
	mf.All(mf.ByKind("ConfigMap"), mf.ByName("config-logging")),
	byNamespace("knative-sources"),
)

func TestUnallowedResourcesInManifest(t *testing.T) {
	tests := []struct {
		path  string
		fails bool
	}{{
		path:  "./channel/1-channel-consolidated.yaml",
		fails: false,
	}, {
		path:  "./channel/2-channel-post-install.yaml",
		fails: false,
	}, {
		path:  "./source/1-source.yaml",
		fails: false,
	}, {
		path:  "./testdata/config-logging.yaml",
		fails: true,
	}, {
		path:  "./testdata/config-observability.yaml",
		fails: true,
	}, {
		path:  "./testdata/config-tracing.yaml",
		fails: true,
	}, {
		path:  "./testdata/knative-sources-namespace.yaml",
		fails: true,
	}}

	for _, test := range tests {
		manifest, err := mf.ManifestFrom(mf.Path(test.path))
		if err != nil {
			t.Fatalf("Unable to load manifest at path '%s' for testing: %v", test.path, err)
		}
		manifest = manifest.Filter(notAllowed)
		if len(manifest.Resources()) > 0 && !test.fails {
			t.Fatalf("Manifest at path '%s' has unallowed resources: %+v", test.path, manifest.Resources())
		}
		if len(manifest.Resources()) == 0 && test.fails {
			t.Fatalf("Manifest at path '%s' should have unallowed resources, but it does not. Perhaps the check for unallowed resources is not working?", test.path)
		}
	}
}

func byNamespace(namespace string) mf.Predicate {
	return func(u *unstructured.Unstructured) bool {
		return u.GetNamespace() == namespace
	}
}
