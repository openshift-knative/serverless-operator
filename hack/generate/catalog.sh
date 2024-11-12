#!/usr/bin/env bash

set -Eeuo pipefail

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function generate_catalog {
  local root_dir index_dir catalog_template

  if [[ -n "${REGISTRY_REDHAT_IO_USERNAME:-}" ]] || [[ -n "${REGISTRY_REDHAT_IO_PASSWORD:-}" ]]; then
    skopeo login registry.redhat.io -u "${REGISTRY_REDHAT_IO_USERNAME}" -p "${REGISTRY_REDHAT_IO_PASSWORD}"
  fi

  default_serverless_operator_images

  root_dir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  index_dir="${root_dir}/olm-catalog/serverless-operator-index"

  while IFS=$'\n' read -r ocp_version; do
    logger.info "Generating catalog for OCP ${ocp_version}"

    catalog_tmp_dir=$(mktemp -d)

    mkdir -p "${index_dir}/v${ocp_version}/catalog/serverless-operator"

    catalog_template="${index_dir}/v${ocp_version}/catalog-template.yaml"

    opm migrate "registry.redhat.io/redhat/redhat-operator-index:v${ocp_version}" "${catalog_tmp_dir}" -oyaml

    # Generate simplified template
    opm alpha convert-template basic "${catalog_tmp_dir}/serverless-operator/catalog.yaml" -oyaml \
      > "${catalog_template}"

    while IFS=$'\n' read -r channel; do
      add_channel "${catalog_template}" "$channel"
      # Also add previous version for cases when it was not released yet
      add_channel "${catalog_template}" "$channel" "$(metadata.get 'olm.replaces')"
    done < <(metadata.get 'olm.channels.list[*]')

    level=none
    # For 4.17 and newer the olm.bundle.object must be converted to olm.csv.metadata
    # See https://github.com/konflux-ci/build-definitions/blob/main/task/fbc-validation/0.1/USAGE.md#bundle-metadata-in-the-appropriate-format.
    if versions.ge "$ocp_version" "4.17" ; then
      level="bundle-object-to-csv-metadata"
    fi

    # Generate full catalog
    opm alpha render-template basic --migrate-level="$level" "${catalog_template}" -oyaml \
      > "${index_dir}/v${ocp_version}/catalog/serverless-operator/catalog.yaml"

    # Replace quay.io with registry.redhat.io for bundle image.
    sed -ri "s#(.*)(${SERVERLESS_BUNDLE})(.*)#\1${SERVERLESS_BUNDLE_REDHAT_IO}\3#" "${index_dir}/v${ocp_version}/catalog/serverless-operator/catalog.yaml"

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
  channel_entry=$(yq read "${catalog_template}" "entries[name==${channel}]")
  # Add channel if necessary
  if [[ "${channel_entry}" == "" ]]; then
    copy_of_stable=$(yq read "${catalog_template}" "entries[name==stable]")
    versioned_channel=$(echo "${copy_of_stable}" | yq write - name "${channel}")
    versioned_channel_json=$(echo "${versioned_channel}" | yq read - --tojson)

    yq read "${catalog_template}" --tojson --prettyPrint | \
      jq '.entries += ['"${versioned_channel_json}"']' | \
      yq read - --prettyPrint > "${catalog}"

    mv "${catalog}" "${catalog_template}"
  fi

  current_csv_entry=$(yq read "${catalog_template}" "entries[name==${channel}].entries[name==${current_csv}]")

  should_add=0
  # Add entry to the channel if doesn't exist yet
  if [[ "${current_csv_entry}" == "" ]]; then
    replaces="serverless-operator.v${previous_version}"
    entry_with_same_replaces=$(yq read "${catalog_template}" "entries[name==${channel}].entries[replaces==${replaces}].name")
    if [[ "${entry_with_same_replaces}" == "" ]]; then
      should_add=1
      cp "${catalog_template}" "${catalog}"
    else
      # Only replace the entry if the version is higher. We should not replace e.g. 1.34.0 with 1.33.3
      # even if 1.33.3 is released later.
      if versions.ge "${current_csv}" "${entry_with_same_replaces}"; then
        should_add=1
        # Get the channel and remove the entry with the same "replaces"
        yq delete "${catalog_template}" "entries[name==${channel}].entries[replaces==${replaces}]" > "${catalog}"
      fi
    fi

    if (( should_add )); then
      cat << EOF | yq write --inplace --script - "$catalog"
      - command: update
        path: entries[name==${channel}].entries[+]
        value:
          name: "serverless-operator.v${version}"
          replaces: "${replaces}"
          skipRange: "\u003e=${previous_version} \u003c${version}"
EOF
      mv "${catalog}" "${catalog_template}"

      # If entry was added, add also the bundle
      add_bundle "${catalog_template}" "$(get_bundle_for_version "${version}")"
    fi
  fi
  rm -f "${catalog}"
}

function add_bundle {
  local bundle catalog_template sha
  catalog_template=${1?Pass catalog template path as arg[1]}
  bundle="${2:?Pass bundle as arg[2]}"

  sha=${bundle##*:} # Get sha
  entry=$(yq read "${catalog_template}" --tojson --prettyPrint | jq '.entries[] | select(.schema=="olm.bundle") | select(.image|test("'${sha}'"))')
  if [[ "${entry}" == "" ]]; then
    # Add bundle itself
    cat << EOF | yq write --inplace --script - "$catalog_template"
    - command: update
      path: entries[+]
      value:
        schema: "olm.bundle"
        image: "${bundle}"
EOF
  fi
}

function upgrade_service_mesh_proxy_image() {
  sm_proxy_image=$(yq r olm-catalog/serverless-operator/project.yaml 'dependencies.service_mesh_proxy')
  sm_proxy_image_stream=$(skopeo inspect --no-tags=true "docker://${sm_proxy_image}" | jq -r '.Labels.version')
  sm_proxy_image_stream=${sm_proxy_image_stream%.*}
  sm_proxy_image=$(latest_konflux_image_sha "${sm_proxy_image}" "${sm_proxy_image_stream}")
  yq w --inplace olm-catalog/serverless-operator/project.yaml 'dependencies.service_mesh_proxy' "${sm_proxy_image}"
}

function upgrade_kube_rbac_proxy_image() {
  local image image_stream
  image=$(metadata.get 'dependencies.kube_rbac_proxy')
  image_stream=$(metadata.get 'requirements.ocpVersion.list[-1]')
  image=$(latest_konflux_image_sha "${image}" "v${image_stream}")
  yq w --inplace olm-catalog/serverless-operator/project.yaml 'dependencies.kube_rbac_proxy' "${image}"
}

function upgrade_dependencies_images {
  if [[ -n "${REGISTRY_REDHAT_IO_USERNAME:-}" ]] || [[ -n "${REGISTRY_REDHAT_IO_PASSWORD:-}" ]]; then
    skopeo login registry.redhat.io -u "${REGISTRY_REDHAT_IO_USERNAME}" -p "${REGISTRY_REDHAT_IO_PASSWORD}"
  fi

  upgrade_service_mesh_proxy_image

  upgrade_kube_rbac_proxy_image
}

logger.info "Upgrading registry.redhat.io images"
upgrade_dependencies_images

logger.info "Generating catalog"
generate_catalog

logger.info "Generating ImageContextSourcePolicy"
create_image_content_source_policy "${INDEX_IMAGE}" "$registry_redhat_io" "$registry_quay" "olm-catalog/serverless-operator-index/image_content_source_policy.yaml"
