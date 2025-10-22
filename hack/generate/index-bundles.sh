#!/usr/bin/env bash

set -Eeuo pipefail

target="${1:?Provide a target index yaml file as arg[1]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function add_previous_bundles {
  if [[ -n "${REGISTRY_REDHAT_IO_USERNAME:-}" ]] || [[ -n "${REGISTRY_REDHAT_IO_PASSWORD:-}" ]]; then
    skopeo login registry.redhat.io -u "${REGISTRY_REDHAT_IO_USERNAME}" -p "${REGISTRY_REDHAT_IO_PASSWORD}"
  fi

  for version in $(metadata.get 'olm.previousBundles' | yq read - '[*].version')
  do
    opm render --skip-tls-verify -o yaml "registry.redhat.io/openshift-serverless-1/serverless-operator-bundle:${version}" >> "$target"
  done
}
# Clear the file.
rm -f "${target}"

add_previous_bundles "${target}"
