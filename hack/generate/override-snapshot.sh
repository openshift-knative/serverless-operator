#!/usr/bin/env bash

set -Eeuo pipefail

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function add_component {
  local component image_ref revision
  component=${2}
  image_ref=${3}
  revision="$(skopeo inspect --no-tags=true "docker://${image_ref}" | jq -r '.Labels["vcs-ref"]')"
  git_repo="$(kubectl get component ${component} -ojsonpath='{.spec.source.git.url}')"

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
EOF
}

function create_snapshot {
  local snapshot_file rootdir so_version serving_version
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  snapshot_file="$(mktemp override-snapshot-XXXXX.json)"

  serving_version="$(metadata.get dependencies.serving)"
  serving_version="${serving_version/knative-v/}" # -> 1.15
  serving_version=${serving_version/./}
  so_version=$(get_app_version_from_tag "$(metadata.get dependencies.serving)")

  cat > "${snapshot_file}" <<EOF
apiVersion: appstudio.redhat.com/v1alpha1
kind: Snapshot
metadata:
  name: override-snapshot
  labels:
    test.appstudio.openshift.io/type: override
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
  add_component ${snapshot_file} "serverless-bundle-${so_version}" "${bundle_repo}@${bundle_digest}"

  cat "${snapshot_file}"
  rm -f "${snapshot_file}"
}

create_snapshot
