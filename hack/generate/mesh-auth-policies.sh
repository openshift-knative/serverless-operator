#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

# This is due to anonymous token request issue created by the Helm image pulling bellow.
# Ideally we want to login properly if this comes up elsewhere.
if [[ "${SKIP_MESH_AUTH_POLICY_GENERATION}" == "true" ]]; then
  exit 0
fi

set -Eeuo pipefail

tenants="${1:?Provide tenants as comma-delimited as arg[1]}"

# exit if helm is not installed
helm > /dev/null || exit 127

policies_path="$(dirname "${BASH_SOURCE[0]}")/../lib/mesh_resources/authorization-policies/helm"
chart_version="$(metadata.get project.version | grep -Eo '[0-9]+\.[0-9]+')" # grep removes the patch version in semver

# Pull helm chart from Github
template_cache=$(mktemp -d)

# Flag for testing a released helm chart.
if [[ "${USE_RELEASED_HELM_CHART}" == "true" ]]; then
  helm repo add openshift-helm-charts https://charts.openshift.io/
  for tenant in ${tenants//,/ }; do
    echo "Generating AuthorizationPolicies for tenant $tenant"
    helm template openshift-helm-charts/redhat-knative-istio-authz \
      --version "$(metadata.get project.version)" \
      --set "name=$tenant" --set "namespaces={$tenant}" > "$policies_path/$tenant.yaml"
  done
  echo "Istio AuthorizationPolicies successfully updated for version $(metadata.get project.version)"
  exit 0
fi

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
  # Steps to verify the final helm chart packaged as .tgz:
  # 1. Use a URL pointing to the tgz file below. Example: helm template https://github.com/Kaustubh-pande/charts/raw/knative-istio-authz-1.31-release/charts/redhat/redhat/knative-istio-authz/1.31.0/knative-istio-authz-1.31.0.tgz --set "name=$tenant" --set "namespaces={$tenant}" > "$policies_path/$tenant.yaml"
  # 2. Send a PR against Github.
  # 3. Check if the Github action called "Validate / Generated files are committed" passes. If the
  #    action fails it means the helm chart is different from what was tested in CI.
  helm template "$template_cache" --set "name=$tenant" --set "namespaces={$tenant}" > "$policies_path/$tenant.yaml"
done

rm -rf "$template_cache"

echo "Istio AuthorizationPolicies successfully updated for version $chart_version"
