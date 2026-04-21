package installation

import (
	"strings"
	"time"

	"github.com/openshift-knative/serverless-operator/test"
)

type clusterExtensionLifecycle struct{}

func (l *clusterExtensionLifecycle) UpgradeTo(ctx *test.Context, version string, timeout time.Duration) error {
	semver := strings.TrimPrefix(version, "serverless-operator.v")

	if err := test.PatchClusterExtensionVersion(ctx, test.ClusterExtensionName, semver); err != nil {
		return err
	}

	if err := test.WaitForClusterExtensionReady(ctx, test.ClusterExtensionName, semver, timeout); err != nil {
		return err
	}

	return WaitForKnativeComponentsReady(ctx,
		test.Flags.ServingVersion, test.Flags.EventingVersion, test.Flags.KafkaVersion)
}

func (l *clusterExtensionLifecycle) Upgrade(ctx *test.Context) error {
	return l.UpgradeTo(ctx, test.Flags.CSV, DefaultInstallPlanTimeout)
}

func (l *clusterExtensionLifecycle) Downgrade(ctx *test.Context) error {
	version := strings.TrimPrefix(test.Flags.CSVPrevious, "serverless-operator.v")

	if err := test.PatchClusterExtensionVersionWithPolicy(
		ctx, test.ClusterExtensionName, version, "SelfCertified"); err != nil {
		return err
	}

	if err := test.WaitForClusterExtensionReady(ctx, test.ClusterExtensionName, version, DefaultInstallPlanTimeout); err != nil {
		return err
	}

	return WaitForKnativeComponentsReady(ctx,
		test.Flags.ServingVersionPrevious, test.Flags.EventingVersionPrevious, test.Flags.KafkaVersionPrevious)
}
