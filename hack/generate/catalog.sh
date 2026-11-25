#!/usr/bin/env bash

set -Eeuo pipefail

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

# Add platform override to podman cmd, in case of MacOS add OS/arch overrides
podman_platform_flags=""
if [[ "$(uname -s)" == "Darwin" ]]; then
  podman_platform_flags="--platform linux/amd64"
fi

# Extracts the serverless-operator package catalog of the given OCP version's
# published index image into the catalog_tmp_dir and echoes its canonical form
# (sorted keys and entries; the raw files can differ in ordering between index
# builds) for comparisons.
function extract_catalog {
  local ocp_version="${1:?Pass OCP version as arg[1]}"

  # The index images are file-based catalogs (label
  # operators.operatorframework.io/index.configs.v1=/configs) with one pre-rendered
  # declcfg per operator. Cat our package's file straight out of the image instead of
  # `opm migrate`, which pulls and re-renders every operator in the index (~2min per
  # OCP version). Registry auth is shared: skopeo login above writes the common
  # containers auth.json, which podman reads as well.
  # shellcheck disable=SC2086
  podman run --pull always --rm ${podman_platform_flags} --entrypoint cat \
    "registry.redhat.io/redhat/redhat-operator-index:v${ocp_version}" \
    /configs/serverless-operator/catalog.json > "${catalog_tmp_dir}/catalog-${ocp_version}.json"

  jq -S -s 'map(if .schema == "olm.channel" then (.entries |= sort_by(.name)) else . end)
    | sort_by(.schema // "", .name // "")' \
    "${catalog_tmp_dir}/catalog-${ocp_version}.json"
}

function generate_catalog {
  local root_dir index_dir catalog_template catalog_tmp_dir
  local -a ocp_versions
  local i ocp_version matched_version

  if [[ -n "${REGISTRY_REDHAT_IO_USERNAME:-}" ]] || [[ -n "${REGISTRY_REDHAT_IO_PASSWORD:-}" ]]; then
    skopeo login registry.redhat.io -u "${REGISTRY_REDHAT_IO_USERNAME}" -p "${REGISTRY_REDHAT_IO_PASSWORD}"
  fi

  default_serverless_operator_images

  root_dir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  index_dir="${root_dir}/olm-catalog/serverless-operator-index"

  # The serverless-operator package content is identical in the published index
  # of every GA OCP version, so a single shared catalog template is generated
  # instead of one per OCP version. The last version in the OCP version list is
  # the pre-release GA version, whose published index can temporarily lag
  # behind during release rollouts, so it is skipped as a source. The catalogs
  # of the versions before it are extracted pairwise, newest pair first
  # (last-1 vs last-2, then last-2 vs last-3, ...), and the first pair whose
  # catalogs match settles the result: their common content is the latest GA
  # catalog, used for the template. The per-OCP-version Dockerfiles
  # (olm-catalog/serverless-operator-index/v*/) are kept: they pin the
  # operator-registry base image per OCP version.
  catalog_template="${index_dir}/catalog-template.yaml"
  catalog_tmp_dir=$(mktemp -d)

  # All supported OCP versions except the last (pre-release GA) one
  mapfile -t ocp_versions < <(metadata.get 'requirements.ocpVersion.list[*]' | sed '$d')
  # OCP 5.0 is built from the same code stream as OCP 4.23, so their published
  # indexes are identical by construction and comparing them would not be an
  # independent confirmation. Skip 4.23 in the comparison chain (compare 5.0
  # with 4.22 directly). Later 5.x versions have no 4.x twin, so the chain
  # (5.1 vs 5.0, 5.0 vs 4.22, ...) keeps working as the version list grows.
  mapfile -t ocp_versions < <(printf '%s\n' "${ocp_versions[@]}" | sed '/^4\.23$/d')
  if ((${#ocp_versions[@]} < 2)); then
    logger.error "Need at least 2 GA OCP versions to generate the catalog template"
    exit 1
  fi

  matched_version=""
  # Walk the pairs from the newest one downwards until the extracted catalogs match
  for ((i = ${#ocp_versions[@]} - 1; i >= 1; i--)); do
    if [[ ! -f "${catalog_tmp_dir}/canonical-${ocp_versions[i]}.json" ]]; then
      logger.info "Extracting catalog for OCP ${ocp_versions[i]}"
      extract_catalog "${ocp_versions[i]}" > "${catalog_tmp_dir}/canonical-${ocp_versions[i]}.json"
    fi
    if [[ ! -f "${catalog_tmp_dir}/canonical-${ocp_versions[i - 1]}.json" ]]; then
      logger.info "Extracting catalog for OCP ${ocp_versions[i - 1]}"
      extract_catalog "${ocp_versions[i - 1]}" > "${catalog_tmp_dir}/canonical-${ocp_versions[i - 1]}.json"
    fi

    if cmp -s "${catalog_tmp_dir}/canonical-${ocp_versions[i]}.json" \
      "${catalog_tmp_dir}/canonical-${ocp_versions[i - 1]}.json"; then
      matched_version="${ocp_versions[i]}"
      logger.info "Catalogs of OCP ${ocp_versions[i]} and ${ocp_versions[i - 1]} match, using OCP ${matched_version} as source"
      break
    fi
    logger.warn "Catalogs of OCP ${ocp_versions[i]} and ${ocp_versions[i - 1]} differ, trying older versions"
  done

  if [[ -z "${matched_version}" ]]; then
    logger.error "No two consecutive OCP versions have a matching serverless-operator catalog, aborting"
    exit 1
  fi

  # Generate simplified template
  opm alpha convert-template basic "${catalog_tmp_dir}/catalog-${matched_version}.json" -oyaml \
    > "${catalog_template}"

  while IFS=$'\n' read -r channel; do
    add_channel "${catalog_template}" "$channel"
    # Also add previous version for cases when it was not released yet

    add_channel "${catalog_template}" "$channel" "$(metadata.get 'olm.replaces')"
  done < <(metadata.get 'olm.channels.list[*]')

  rm -rf "${catalog_tmp_dir}"
}

function add_channel {
  local channel catalog_template catalog current_version current_csv \
    replaces_version skip_range channel_entry version
  catalog_template=${1?Pass catalog template path as arg[1]}
  channel=${2:?Pass channel name as arg[2]}

  current_version=$(metadata.get 'project.version')
  version="${3:-$current_version}"

  current_csv="serverless-operator.v${version}"
  replaces_version=$(metadata.get 'olm.replaces')
  skip_range=$(metadata.get 'olm.skipRange')

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
    replaces="serverless-operator.v${replaces_version}"
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

    # Check if the bundle image for this version is really for the version. Because on branch cuts, the bundle from
    # the old version and new version are the same (contain same CSV), as the bundle is not updated yet. This would
    # result for example in a `package "serverless-operator" has duplicate bundle "serverless-operator.v1.36.0"`.
    # See for example https://github.com/openshift-knative/serverless-operator/pull/3608#issuecomment-2846440232
    # Therefor we only add the catalog, if the corresponding bundle has the correct version (and is "new")
    bundle_image_for_version=$(get_bundle_for_version "${version}")
    # shellcheck disable=SC2086
    version_in_bundle_image=$(skopeo inspect --no-tags docker://"$bundle_image_for_version" ${SKOPEO_EXTRA_FLAGS} | jq -r .Labels.version)
    if [[ "$version_in_bundle_image" != "$version" ]]; then
      logger.debug "Bundle image (${bundle_image_for_version}) for version ${version} not updated yet and still contains old version (${version_in_bundle_image}). Thus skipping to add it to the catalog"
      should_add=0
    fi

    if (( should_add )); then
      cat << EOF | yq write --inplace --script - "$catalog"
      - command: update
        path: entries[name==${channel}].entries[+]
        value:
          name: "serverless-operator.v${version}"
          replaces: "${replaces}"
          skipRange: "${skip_range}"
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

logger.info "Generating catalog"
generate_catalog

logger.info "Generating ImageContextSourcePolicy"

default_serverless_operator_images
# shellcheck disable=SC2154
create_image_content_source_policy "${INDEX_IMAGE}" "$registry_redhat_io" "$registry_quay" "$registry_quay_previous" "$registry_quay_next" "olm-catalog/serverless-operator-index/image_content_source_policy.yaml" ".tekton/images-mirror-set.yaml"
