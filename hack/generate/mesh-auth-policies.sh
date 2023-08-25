#!/usr/bin/env bash

set -Eeuo pipefail

tenants="${1:?Provide tenants as comma-delimited as arg[1]}"

# exit if helm is not installed
helm > /dev/null || exit 127

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

policies_path="$(dirname "${BASH_SOURCE[0]}")/../lib/mesh_resources/authorization-policies/helm"
if [ -z "${ISTIO_CHART_VERSION:-}" ]; then
  chart_version="$(metadata.get project.version)"
else
  chart_version="${ISTIO_CHART_VERSION}"
fi

echo "Cleaning up old resources in $policies_path"

rm -rf "$policies_path"
mkdir -p "$policies_path"

for tenant in ${tenants//,/ }; do
  echo "Generating AuthorizationPolicies for tenant $tenant"

  helm template oci://quay.io/openshift-knative/knative-istio-authz-onboarding --version "$chart_version" --set "name=$tenant" --set "namespaces={$tenant}" > "$policies_path/$tenant.yaml"
done

echo "Istio AuthorizationPolicies successfully updated for version $chart_version"
