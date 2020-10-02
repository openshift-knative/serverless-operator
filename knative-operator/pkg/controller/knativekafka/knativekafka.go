package knativekafka

import (
	"fmt"
	"os"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	operatorv1alpha1 "github.com/openshift-knative/serverless-operator/knative-operator/pkg/apis/operator/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// rawKafkaChannelManifest returns KafkaChannel manifest without transformations
func rawKafkaChannelManifest(apiclient client.Client) (mf.Manifest, error) {
	return mfc.NewManifest(kafkaChannelManifestPath(), apiclient, mf.UseLogger(log.WithName("mf")))
}

// rawKafkaSourceManifest returns KafkaSource manifest without transformations
func rawKafkaSourceManifest(apiclient client.Client) (mf.Manifest, error) {
	return mfc.NewManifest(kafkaSourceManifestPath(), apiclient, mf.UseLogger(log.WithName("mf")))
}

func kafkaChannelManifestPath() string {
	return os.Getenv("KAFKACHANNEL_MANIFEST_PATH")
}

func kafkaSourceManifestPath() string {
	return os.Getenv("KAFKASOURCE_MANIFEST_PATH")
}

type manifestBuild int

const (
	ManifestBuildEnabledOnly manifestBuild = iota
	ManifestBuildDisabledOnly
	ManifestBuildAll
)

func buildManifest(instance *operatorv1alpha1.KnativeKafka, apiClient client.Client, build manifestBuild) (*mf.Manifest, error) {
	combinedManifest := &mf.Manifest{}

	if build == ManifestBuildAll || (instance.Spec.Channel.Enabled && build == ManifestBuildEnabledOnly) || (!instance.Spec.Channel.Enabled && build == ManifestBuildDisabledOnly) {
		manifest, err := rawKafkaSourceManifest(apiClient)
		if err != nil {
			return nil, fmt.Errorf("failed to load KafkaChannel manifest: %w", err)
		}
		combinedManifest, err = mergeManifests(manifest.Client, combinedManifest, &manifest)
		if err != nil {
			return nil, fmt.Errorf("failed to merge KafkaChannel manifest: %w", err)
		}
	}

	if build == ManifestBuildAll || (instance.Spec.Source.Enabled && build == ManifestBuildEnabledOnly) || (!instance.Spec.Source.Enabled && build == ManifestBuildDisabledOnly) {
		manifest, err := rawKafkaSourceManifest(apiClient)
		if err != nil {
			return nil, fmt.Errorf("failed to load KafkaSource manifest: %w", err)
		}
		combinedManifest, err = mergeManifests(manifest.Client, combinedManifest, &manifest)
		if err != nil {
			return nil, fmt.Errorf("failed to merge KafkaSource manifest: %w", err)
		}
	}
	return combinedManifest, nil
}

// Merges the given manifests into a new single manifest
func mergeManifests(client mf.Client, m1, m2 *mf.Manifest) (*mf.Manifest, error) {
	result, err := mf.ManifestFrom(mf.Slice(append(m1.Resources(), m2.Resources()...)))
	if err != nil {
		return nil, fmt.Errorf("failed to merge manifests: %w", err)
	}
	result.Client = client
	return &result, nil
}

// InjectOwner creates a Tranformer which adds an OwnerReference pointing to
// `owner` to namespace-scoped objects.
//
// The difference from Manifestival's Inject owner is, it only does it for
// resources that are in the same namespace as the owner.
// For the resources that are in the same namespace, it fallbacks to
// Manifestival's InjectOwner
func InjectOwner(owner mf.Owner) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetNamespace() == owner.GetNamespace() {
			return mf.InjectOwner(owner)(u)
		} else {
			return nil
		}
	}
}

func isDeploymentAvailable(d *appsv1.Deployment) bool {
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
