#!/usr/bin/env bash

# This is due to anonymous token request issue created by the Helm image pulling bellow.
# Ideally we want to login properly if this comes up elsewhere.
if [[ "${SKIP_MESH_AUTH_POLICY_GENERATION}" == "true" ]]; then
  exit 0
fi


set -Eeuo pipefail

tenants="${1:?Provide tenants as comma-delimited as arg[1]}"

# exit if helm is not installed
helm > /dev/null || exit 127

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

policies_path="$(dirname "${BASH_SOURCE[0]}")/../lib/mesh_resources/authorization-policies/helm"
chart_version="$(metadata.get project.version | grep -Eo '[0-9]+\.[0-9]+')" # grep removes the patch version in semver

# Pull image template from Github
template_cache=$(mktemp -d)

if ! git clone -b "release-${chart_version}" --depth 1 https://github.com/openshift-knative/knative-istio-authz-chart.git "$template_cache"; then
   # branch might not yet be there, then we fallback to using `main`
   echo "Failed to clone knative-istio-authz-chart with branch release-${chart_version}. Falling back to using main."
   git clone --depth 1 https://github.com/openshift-knative/knative-istio-authz-chart.git "$template_cache"
fi

echo "Cleaning up old resources in $policies_path"

rm -rf "$policies_path"
mkdir -p "$policies_path"

for tenant in ${tenants//,/ }; do
  echo "Generating AuthorizationPolicies for tenant $tenant"

  helm template "$template_cache" --version "$chart_version" --set "name=$tenant" --set "namespaces={$tenant}" > "$policies_path/$tenant.yaml"
done

echo "Istio AuthorizationPolicies successfully updated for version $chart_version"
