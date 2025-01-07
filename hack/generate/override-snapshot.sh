#!/usr/bin/env bash

set -Eeuo pipefail

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function add_component {
  local component image_ref parameters git_repo revision dockerfile
  component=${2}
  image_ref=${3}

  parameters="$(cosign download attestation "${image_ref}" | jq -r '.payload' | base64 -d | jq -c '.predicate.invocation.parameters')"
  git_repo="$(echo "${parameters}" | jq -r '."git-url"')"
  revision="$(echo "${parameters}" | jq -r ".revision")"
  dockerfile="$(echo "${parameters}" | jq -r ".dockerfile")"

  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.components[+]
  value:
    name: "${component}"
    containerImage: "${image_ref}"
    source:
      git:
        url: "${git_repo}"
        revision: "${revision}"
        dockerfileUrl: "${dockerfile}"
EOF
}

function create_component_snapshot {
  local snapshot_file so_version so_semversion serving_version tmp_catalog_dir max_ocp_version latest_index_image
  snapshot_file="${1}/override-snapshot.yaml"

  serving_version="$(metadata.get dependencies.serving)"
  serving_version="${serving_version/knative-v/}" # -> 1.15
  serving_version="${serving_version/./}"
  so_version="$(get_app_version_from_tag "$(metadata.get dependencies.serving)")"
  so_semversion="$(metadata.get project.version)"

  cat > "${snapshot_file}" <<EOF
apiVersion: appstudio.redhat.com/v1alpha1
kind: Snapshot
metadata:
  generateName: serverless-operator-${so_version}-override-snapshot-
  labels:
    test.appstudio.openshift.io/type: override
    application: serverless-operator-${so_version}
spec:
  application: serverless-operator-${so_version}
EOF

  tmp_catalog_dir=$(mktemp -d)
  max_ocp_version="$(metadata.get requirements.ocpVersion.max)"
  max_ocp_version=${max_ocp_version/./}
  latest_index_image="${registry_quay}-fbc-${max_ocp_version}/serverless-index-${so_version}-fbc-${max_ocp_version}:latest"

  # get catalog from latest index, so we can get the referenced images from there
  opm migrate "${latest_index_image}" "${tmp_catalog_dir}" -o json

  while IFS= read -r image_ref; do
    # shellcheck disable=SC2053

    if  [[ $image_ref =~ $registry_redhat_io ]]; then
      image=${image_ref##*/} # Get image name after last slash
      image_sha=${image_ref##*@} # Get SHA of image
      image=${image%@*} # Remove sha
      image=${image/-rhel[0-9]/} # Remove -rhelX part

      component_image_ref="${registry_quay}/${image}@${image_sha}"

      if [[ $image == "serverless-operator-bundle" ]]; then
        # bundle component is named in konflux serverless-bundle-<version>

        component_name="serverless-bundle-${so_version}"
        component_image_ref="${registry_quay}/serverless-bundle@${image_sha}"
      elif [[ $image =~ serverless ]]; then
        component_name="${image}-${so_version}"
      else
        component_name="${image}-${serving_version}"
      fi

      add_component "${snapshot_file}" "${component_name}" "${component_image_ref}"
    fi
  done <<< "$(jq -r '. | select(.name == "serverless-operator.v'${so_semversion}'") | .relatedImages[].image' "${tmp_catalog_dir}/serverless-operator/catalog.json" | sort | uniq)"
  # ^ we take the images from the catalogs relatedImages section for the given SO version. We could also extract the bundle image from the catalog (jq -r '. | select(.name == "serverless-operator.v'${so_semversion}'") | .image')
  # and extract the CSV from there and use the CSVs relatedImages section.

  rm -rf "${tmp_catalog_dir}"
}

function verify_component_snapshot {
  local snapshot_file repo revision component repo_revision failed
  snapshot_file="${1}/override-snapshot.yaml"
  declare -A repo_revision=()

  while IFS= read -r json; do
    repo="$(echo "$json" | jq -r .source.git.url)"
    repo=${repo%".git"} # remove optional .git suffix from repo name
    revision="$(echo "$json" | jq -r .source.git.revision)"
    component="$(echo "$json" | jq -r .name)"

    if [[ ! -v repo_revision[$repo]  ]]; then
      # no revision for repo so far --> add it to map
      repo_revision[$repo]=$revision
    else
      if [[ "${repo_revision[$repo]}" != "$revision" ]]; then
        # revisions don't match
        if [[ $component =~ "serverless-bundle" ]]; then
          #ignore serverless bundle
          continue
        fi

        echo "Revision for ${component} didn't match. Expected revision ${repo_revision[$repo]} for repo ${repo}, but got ${revision}"
        failed="true"
      fi
    fi

  done <<< "$(yq read --tojson "${snapshot_file}" "spec.components[*]")"

  if [[ "$failed" == "true" ]]; then
    exit 1
  fi
}

function create_fbc_snapshots {
  local rootdir snapshot_dir so_version
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  snapshot_dir="${1}"

  so_version=$(get_app_version_from_tag "$(metadata.get dependencies.serving)")

  while IFS= read -r ocp_version; do
    ocp_version=${ocp_version/./}
    snapshot_file="${snapshot_dir}/override-snapshot-fbc-${ocp_version}.yaml"

    cat > "${snapshot_file}" <<EOF
apiVersion: appstudio.redhat.com/v1alpha1
kind: Snapshot
metadata:
  generateName: serverless-operator-${so_version}-fbc-${ocp_version}-override-snapshot-
  labels:
    test.appstudio.openshift.io/type: override
    application: serverless-operator-${so_version}-fbc-${ocp_version}
spec:
  application: serverless-operator-${so_version}-fbc-${ocp_version}
EOF

  index_image="${registry_quay}-fbc-${ocp_version}/serverless-index-${so_version}-fbc-${ocp_version}"
  index_image_digest="$(skopeo inspect --no-tags docker://"${index_image}:latest" | jq -r .Digest)"
  add_component "${snapshot_file}" "serverless-index-${so_version}-fbc-${ocp_version}" "${index_image}@${index_image_digest}"

  done <<< "$(yq read "${rootdir}/olm-catalog/serverless-operator/project.yaml" 'requirements.ocpVersion.list[*]')"
}

target_dir="${1:?Provide a target directory for the override snapshots as arg[1]}"
create_component_snapshot "${target_dir}"
verify_component_snapshot "${target_dir}"
create_fbc_snapshots "${target_dir}"
