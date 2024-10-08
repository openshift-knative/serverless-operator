#!/usr/bin/env bash

set -Eeuo pipefail

#target="${1:?Provide a target index yaml file as arg[1]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function generate_catalog {
  local root_dir index_dir catalog_template

#  if [[ -n "${REGISTRY_REDHAT_IO_USERNAME:-}" ]] || [[ -n "${REGISTRY_REDHAT_IO_PASSWORD:-}" ]]; then
#    skopeo login registry.redhat.io -u "${REGISTRY_REDHAT_IO_USERNAME}" -p "${REGISTRY_REDHAT_IO_PASSWORD}"
#  fi

  root_dir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  index_dir="${root_dir}/olm-catalog/serverless-operator/index"

  # TODO: Remove this
  catalog_tmp_dir=./catalog-migrate

  while IFS=$'\n' read -r ocp_version; do
    logger.info "Generating catalog for OCP ${ocp_version}"

    #catalog_tmp_dir=$(mktemp -d)
    mkdir -p "${index_dir}/v${ocp_version}/catalog/serverless-operator"

    catalog_template="${index_dir}/v${ocp_version}/catalog-template.json"
    # TODO: Use only if it differs from last one?
    #skopeo inspect --no-tags=true "docker://registry.redhat.io/redhat/redhat-operator-index:v${ocp_version}" | jq -r '.Digest'

    #opm migrate "registry.redhat.io/redhat/redhat-operator-index:v${ocp_version}" "${catalog_tmp_dir}"

    # Generate simplified template
    opm alpha convert-template basic "${catalog_tmp_dir}/serverless-operator/catalog.json" | jq . \
      > "${catalog_template}"

    while IFS=$'\n' read -r channel; do
      add_channel "${catalog_template}" "$channel"
      # Also add previous version for cases when it was not released yet
      add_channel "${catalog_template}" "$channel" "$(metadata.get 'olm.replaces')"
    done < <(metadata.get 'olm.channels.list[*]')

    add_latest_bundle "${catalog_template}"

    # TODO: Add previous channel if necessary (1.34.0)

    # Generate full catalog
    #opm alpha render-template basic "${catalog_template}" \
    #  > "${index_dir}/v${ocp_version}/catalog/serverless-operator/catalog.json"

    rm -rf "${catalog_tmp_dir}"
  done < <(metadata.get 'requirements.ocpVersion.list[*]')

}

function add_channel {
  local channel catalog_template catalog current_version current_csv major minor micro previous_version channel_entry version
  catalog_template=${1?Pass catalog template path as arg[1]}
  channel=${2:?Pass channel name as arg[2]}

  current_version=$(metadata.get 'project.version')
  version="${3:-$current_version}"

  current_csv="serverless-operator.v${version}"
  major=$(versions.major "${version}")
  minor=$(versions.minor "${version}")
  micro=$(versions.micro "${version}")

  # Handle the first entry specifically as it might be a z-stream release.
  if [[ "$micro" == "0" ]]; then
    previous_version="${major}.$(( minor-1 )).${micro}"
  else
    previous_version="${major}.${minor}.0"
  fi

  catalog=$(mktemp catalog-XXX.json)
  channel_entry=$(jq '.entries[] | select(.schema=="olm.channel" and .name=="'"${channel}"'").entries[]' "${catalog_template}")

  # Add channel if necessary
  if [[ "${channel_entry}" == "" ]]; then
    jq '.entries += [{
          "name": "'"${channel}"'",
          "package": "serverless-operator",
          "schema": "olm.channel"
    }]' "${catalog_template}" > "${catalog}"
  else
    cp "${catalog_template}" "${catalog}"
  fi
  mv "${catalog}" "${catalog_template}"

  entry=$(jq '.entries[] | select(.schema=="olm.channel" and .name=="'"${channel}"'").entries[]? | select(.name=="'"${current_csv}"'")' "${catalog_template}")

  # Add entry to the channel
  if [[ "${entry}" == "" ]]; then
    jq '{
      schema: .schema,
      entries: [ .entries[] | select(.schema=="olm.channel" and .name=="'"${channel}"'").entries += [{
        "name": "serverless-operator.v'"${version}"'",
        "replaces": "serverless-operator.v'"${previous_version}"'",
        "skipRange": "\u003e='"${previous_version}"' \u003c'"${version}"'"
    }]]}' "${catalog_template}" > "${catalog}"
  fi
  mv "${catalog}" "${catalog_template}"
}

function add_latest_bundle {
  local catalog_template entry catalog
  catalog_template=${1?Pass catalog template path as arg[1]}
  catalog=$(mktemp catalog-XXX.json)

  #default_serverless_operator_images
  # TODO: Remove this
  export SERVERLESS_BUNDLE=quay.io/redhat-user-workloads/ocp-serverless-tenant/serverless-operator-135/serverless-bundle@sha256:251f4734eb923eeea8fb1b49996d1c5d52e6285819162c90a4f445f644ba4754

  entry=$(jq '.entries[] | select(.schema=="olm.bundle") | select(.image|test("'${registry_quay}'"))' "${catalog_template}")
  # Add bundle itself
  if [[ "$entry" == "" ]]; then
    jq '.entries += [{
          "schema": "olm.bundle",
          "image": "'"${SERVERLESS_BUNDLE}"'",
    }]' "${catalog_template}" > "${catalog}"
  else
    jq '.entries[] | select(.schema=="olm.bundle") | select(.image|test("'$registry_quay'")) | { schema, image: "'"${SERVERLESS_BUNDLE}"'" }' \
      "${catalog_template}" > "${catalog}"
  fi
  mv "${catalog}" "${catalog_template}"
}

generate_catalog

# Clear the file.
#rm -f "${target}"

#while IFS=$'\n' read -r channel; do
#  add_channel_entries "$channel" "${target}"
#done < <(metadata.get 'olm.channels.list[*]')
