#!/usr/bin/env bash

set -Eeuo pipefail

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function generate_catalog {
  local root_dir index_dir catalog_template

  if [[ -n "${REGISTRY_REDHAT_IO_USERNAME:-}" ]] || [[ -n "${REGISTRY_REDHAT_IO_PASSWORD:-}" ]]; then
    skopeo login registry.redhat.io -u "${REGISTRY_REDHAT_IO_USERNAME}" -p "${REGISTRY_REDHAT_IO_PASSWORD}"
  fi

  root_dir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  index_dir="${root_dir}/olm-catalog/serverless-operator-index"

  while IFS=$'\n' read -r ocp_version; do
    logger.info "Generating catalog for OCP ${ocp_version}"

    catalog_tmp_dir=$(mktemp -d)
    mkdir -p "${index_dir}/v${ocp_version}/catalog/serverless-operator"

    catalog_template="${index_dir}/v${ocp_version}/catalog-template.json"

    opm migrate "registry.redhat.io/redhat/redhat-operator-index:v${ocp_version}" "${catalog_tmp_dir}"

    # Generate simplified template
    opm alpha convert-template basic "${catalog_tmp_dir}/serverless-operator/catalog.json" \
      > "${catalog_template}"

    while IFS=$'\n' read -r channel; do
      add_channel "${catalog_template}" "$channel"
      # Also add previous version for cases when it was not released yet
      add_channel "${catalog_template}" "$channel" "$(metadata.get 'olm.replaces')"
    done < <(metadata.get 'olm.channels.list[*]')

    # Generate full catalog
    opm alpha render-template basic "${catalog_template}" \
      > "${index_dir}/v${ocp_version}/catalog/serverless-operator/catalog.json"

    rm -rf "${catalog_tmp_dir}"
  done < <(metadata.get 'requirements.ocpVersion.list[*]')
}

function add_channel {
  local channel catalog_template catalog current_version current_csv major \
    minor micro previous_version channel_entry version
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
  channel_entry=$(jq '.entries[] | select(.schema=="olm.channel" and .name=="'"${channel}"'")' "${catalog_template}")

  # Add channel if necessary
  if [[ "${channel_entry}" == "" ]]; then
    copy_of_stable=$(jq '.entries[] | select(.schema=="olm.channel" and .name=="stable")' "${catalog_template}")
    versioned_channel=$(echo "${copy_of_stable}" | \
      jq '{ name: "'"${channel}"'", package: .package, schema: .schema, entries: .entries }')
    jq '.entries += ['"${versioned_channel}"']' "${catalog_template}" > "${catalog}"
    mv "${catalog}" "${catalog_template}"
  fi

  current_csv_entry=$(jq '.entries[] | select(.schema=="olm.channel" and .name=="'"${channel}"'").entries[]? | select(.name=="'"${current_csv}"'")' "${catalog_template}")

  should_add=0
  # Add entry to the channel if doesn't exist yet
  if [[ "${current_csv_entry}" == "" ]]; then
    replaces="serverless-operator.v${previous_version}"
    entry_with_same_replaces=$(jq -r '.entries[] | select(.schema=="olm.channel" and .name=="'"${channel}"'").entries[]? | select(.replaces=="'"${replaces}"'").name' "${catalog_template}")
    if [[ "${entry_with_same_replaces}" == "" ]]; then
      should_add=1
      clean_catalog=$(jq . "${catalog_template}")
    else
      # Only replace the entry if the version is higher. We should not replace e.g. 1.34.0 with 1.33.3
      # even if 1.33.3 is released later.
      if versions.ge "${current_csv}" "${entry_with_same_replaces}"; then
        should_add=1
        # Get the channel and remove the entry with the same "replaces"
        pruned_channel=$(jq '.entries[] | select(.schema=="olm.channel" and .name=="'"${channel}"'") | del(.entries[] | select(.replaces=="'"${replaces}"'"))' "${catalog_template}")
        # Remove the outdated channel
        without_channel=$(jq 'del(.entries[] | select(.schema=="olm.channel" and .name=="'"${channel}"'"))' "${catalog_template}")
        # Create catalog with the new pruned channel
        clean_catalog=$(echo "${without_channel}" | jq '.entries += ['"${pruned_channel}"']')
      fi
    fi

    if (( should_add )); then
      # Add the new channel entry for the latest bundle
      echo "${clean_catalog}" | jq '{
        schema: .schema,
        entries: [ .entries[] | select(.schema=="olm.channel" and .name=="'"${channel}"'").entries += [{
          "name": "serverless-operator.v'"${version}"'",
          "replaces": "'"${replaces}"'",
          "skipRange": "\u003e='"${previous_version}"' \u003c'"${version}"'"
      }]]}' > "${catalog}"
      mv "${catalog}" "${catalog_template}"

      # If entry was added, add also the bundle
      add_latest_bundle "${catalog_template}"
      add_previous_bundle "${catalog_template}"
    fi
  fi
}

function add_latest_bundle {
  local catalog_template
  catalog_template=${1?Pass catalog template path as arg[1]}
  default_serverless_operator_images

  add_bundle "${catalog_template}" "${SERVERLESS_BUNDLE}"
}

function add_previous_bundle {
  local catalog_template
  catalog_template=${1?Pass catalog template path as arg[1]}
  default_serverless_operator_images

  add_bundle "${catalog_template}" "${SERVERLESS_BUNDLE_PREVIOUS}"
}

function add_bundle {
  local bundle catalog_template sha
  catalog_template=${1?Pass catalog template path as arg[1]}
  bundle="${2:?Pass bundle as arg[2]}"
  catalog=$(mktemp catalog-XXX.json)

  cp "${catalog_template}" "${catalog}"

  sha=${bundle##*:} # Get sha
  entry=$(jq '.entries[] | select(.schema=="olm.bundle") | select(.image|test("'${sha}'"))' "${catalog_template}")
  if [[ "${entry}" == "" ]]; then
    # Add bundle itself
    jq '.entries += [{
          "schema": "olm.bundle",
          "image": "'"${bundle}"'",
    }]' "${catalog_template}" > "${catalog}"
  fi

  mv "${catalog}" "${catalog_template}"
}

generate_catalog
