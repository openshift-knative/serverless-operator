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
  local rootdir snapshot_file so_version serving_version
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  snapshot_file="${1}/override-snapshot.yaml"

  serving_version="$(metadata.get dependencies.serving)"
  serving_version="${serving_version/knative-v/}" # -> 1.15
  serving_version=${serving_version/./}
  so_version=$(get_app_version_from_tag "$(metadata.get dependencies.serving)")

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

  while IFS= read -r image_ref; do
    # shellcheck disable=SC2053

    if  [[ $image_ref =~ $registry_redhat_io ]]; then
      image=${image_ref##*/} # Get image name after last slash
      image_sha=${image_ref##*@} # Get SHA of image
      image=${image%@*} # Remove sha
      image=${image/-rhel[0-9]/} # Remove -rhelX part

      if [[ $image =~ serverless ]]; then
        component_name="${image}-${so_version}"
      else
        component_name="${image}-${serving_version}"
      fi

      component_image_ref="${registry_quay}/${image}@${image_sha}"

      add_component "${snapshot_file}" "${component_name}" "${component_image_ref}"
    fi
  done <<< "$(yq read "${rootdir}/olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml" 'spec.relatedImages[*].image' | sort | uniq)"

  # Add bundle, as this is not referenced in the CSV
  bundle_repo="${registry_quay}/serverless-bundle"
  bundle_image="${registry_quay}/serverless-bundle:$(metadata.get project.version)"
  bundle_digest=$(skopeo inspect --no-tags=true "docker://${bundle_image}" | jq -r '.Digest')
  add_component "${snapshot_file}" "serverless-bundle-${so_version}" "${bundle_repo}@${bundle_digest}"
}

function create_fbc_snapshots {
  local snapshot_dir so_version
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
create_fbc_snapshots "${target_dir}"
