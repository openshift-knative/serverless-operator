#!/usr/bin/env bash

set -Eeuo pipefail

target="${1:?Provide a target index yaml file as arg[1]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function add_previous_bundles {
  if [[ -n "${REGISTRY_REDHAT_IO_USERNAME:-}" ]] || [[ -n "${REGISTRY_REDHAT_IO_PASSWORD:-}" ]]; then
    skopeo login registry.redhat.io -u "${REGISTRY_REDHAT_IO_USERNAME}" -p "${REGISTRY_REDHAT_IO_PASSWORD}"
  fi

  # We're adding just the "previous bundles"
  num_csvs=$(( INDEX_IMAGE_NUM_CSVS-1 ))

  current_version=$(metadata.get 'project.version')
  major=$(versions.major "$current_version")
  minor=$(versions.minor "$current_version")
  micro=$(versions.micro "$current_version")

  for i in $(seq $num_csvs); do
    current_minor=$(( minor ))
    current_micro=$(( micro-i ))
    if [[ "$current_micro" -le 0 ]]; then
      current_minor=$((minor + current_micro))
      current_micro=0
    fi

    current_version="${major}.${current_minor}.${current_micro}"

    opm render --skip-tls-verify -o yaml "registry.redhat.io/openshift-serverless-1/serverless-operator-bundle:${current_version}" >> "$target"
  done
}
# Clear the file.
rm -f "${target}"

add_previous_bundles "${target}"
