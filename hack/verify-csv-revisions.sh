#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

# verify that the revisions (git commit) for components from the same repo match
function verify_image_revisions {
  local root_dir csv_file repo_revision rc
  root_dir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"
  csv_file="${root_dir}/olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml"
  declare -A repo_revision=()
  rc=0

  while IFS= read -r image_ref; do

    if  [[ $image_ref =~ $registry_redhat_io ]]; then
      quay_image_ref="$(get_quay_image_ref "$image_ref")"
      parameters="$(cosign download attestation "${quay_image_ref}" | jq -r '.payload' | base64 -d | jq -c '.predicate.invocation.parameters')"
      repo="$(echo "${parameters}" | jq -r '."git-url"')"
      revision="$(echo "${parameters}" | jq -r ".revision")"
      repo=${repo%".git"} # remove optional .git suffix from repo name

      if [[ ! -v repo_revision[$repo]  ]]; then
          # no revision for repo so far --> add it to map
          repo_revision[$repo]=$revision
      else
        if [[ "${repo_revision[$repo]}" != "$revision" ]]; then
          # revisions don't match
          image=${image_ref##*/} # Get image name after last slash

          echo "Revision for ${image} didn't match. Expected revision ${repo_revision[$repo]} for repo ${repo}, but got ${revision}"
          rc=1
        fi
      fi
    fi

  done <<< "$(yq read "${csv_file}" 'spec.relatedImages[*].image' | sort | uniq)"

  if [[ "$rc" == "0" ]]; then
    echo "All revisions matched correctly"
  fi

  return $rc
}

verify_image_revisions
