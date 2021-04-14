/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ingress

import (
	"context"
	"os"
	"path/filepath"

	mf "github.com/manifestival/manifestival"
	"golang.org/x/mod/semver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/operator/pkg/apis/operator/v1alpha1"
	"knative.dev/operator/pkg/reconciler/common"
)

const providerLabel = "networking.knative.dev/ingress-provider"

func ingressFilter(name string) mf.Predicate {
	return func(u *unstructured.Unstructured) bool {
		provider, hasLabel := u.GetLabels()[providerLabel]
		if !hasLabel {
			return true
		}
		return provider == name
	}
}

// Filters makes sure the disabled ingress resources are removed from the manifest.
func Filters(ks *v1alpha1.KnativeServing) mf.Predicate {
	var filters []mf.Predicate
	if ks.Spec.Ingress == nil {
		return mf.Any(istioFilter)
	}
	if ks.Spec.Ingress.Istio.Enabled {
		filters = append(filters, istioFilter)
	}
	if ks.Spec.Ingress.Kourier.Enabled {
		filters = append(filters, kourierFilter)
	}
	if ks.Spec.Ingress.Contour.Enabled {
		filters = append(filters, contourFilter)
	}
	return mf.Any(filters...)
}

// Transformers returns a list of transformers based on the enabled ingresses
func Transformers(ctx context.Context, ks *v1alpha1.KnativeServing) []mf.Transformer {
	if ks.Spec.Ingress == nil {
		return istioTransformers(ctx, ks)
	}
	var transformers []mf.Transformer
	if ks.Spec.Ingress.Istio.Enabled {
		transformers = append(transformers, istioTransformers(ctx, ks)...)
	}
	if ks.Spec.Ingress.Kourier.Enabled {
		transformers = append(transformers, kourierTransformers(ctx, ks)...)
	}
	if ks.Spec.Ingress.Contour.Enabled {
		transformers = append(transformers, contourTransformers(ctx, ks)...)
	}
	return transformers
}

func getIngress(version string, manifest *mf.Manifest) error {
	// If we can not determine the version, append no ingress manifest.
	if version == "" {
		return nil
	}
	koDataDir := os.Getenv(common.KoEnvKey)
	// Ingresses are saved in the directory named major.minor. We remove the patch number.
	ingressVersion := semver.MajorMinor(common.SanitizeSemver(version))[1:]
	ingressPath := filepath.Join(koDataDir, "ingress", ingressVersion)
	m, err := common.FetchManifest(ingressPath)
	if err != nil {
		return err
	}
	*manifest = manifest.Append(m)
	return nil
}

// AppendTargetIngresses appends the manifests of ingresses to be installed
func AppendTargetIngresses(ctx context.Context, manifest *mf.Manifest, instance v1alpha1.KComponent) error {
	return getIngress(common.TargetVersion(instance), manifest)
}

// AppendInstalledIngresses appends the installed manifests of ingresses
func AppendInstalledIngresses(ctx context.Context, manifest *mf.Manifest, instance v1alpha1.KComponent) error {
	version := instance.GetStatus().GetVersion()
	if version == "" {
		version = common.TargetVersion(instance)
	}
	return getIngress(version, manifest)
}

func hasProviderLabel(u *unstructured.Unstructured) bool {
	if _, hasLabel := u.GetLabels()[providerLabel]; hasLabel {
		return true
	}
	return false
}
