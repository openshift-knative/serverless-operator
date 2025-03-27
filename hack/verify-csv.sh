#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

root_dir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"
csv_file="${root_dir}/olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml"

# verify that the revisions (git commit) for components from the same repo match
# shellcheck disable=SC2317
function verify_image_revisions {
  local repo_revision rc
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

  return "$rc"
}

# verify that the image does not have a CVE
# shellcheck disable=SC2317
function verify_no_cve_in_image {
  # based on https://github.com/enterprise-contract/ec-cli/blob/main/hack/view-clair-reports.sh
  # migrated to parse mostly with JQ and thus being more flexible with the YQ version
  rc=0
  image="${1}"
  repo="$(echo "$image" | cut -d '@' -f 1)"

  clair_report_shas="$(
    cosign download attestation "$image" | jq -r '.payload|@base64d|fromjson|.predicate.buildConfig.tasks[]|select(.name=="clair-scan").results[]|select(.name=="REPORTS").value|fromjson|.[]'
  )"

  # For multi-arch the same report maybe associated with each of the per-arch
  # images. Use sort uniq to avoid displaying it multiple times, but still
  # support the possibility of different reports
  all_blobs=""

  for sha in $clair_report_shas; do
    blob="$(skopeo inspect --raw docker://"$repo@$sha" | jq -r '.layers[].digest')"
    all_blobs="$( (echo "$all_blobs"; echo "$blob") | sort | uniq )"
  done

  for b in $all_blobs; do
    output=$(oras blob fetch "$repo@$b" --output - | jq '.vulnerabilities[] | select((.normalized_severity=="High") or (.normalized_severity=="Critical")) | pick(.name, .description, .issued, .normalized_severity, .package_name, .fixed_in_version)' | jq -s .)
    cve_counter=$(echo "$output" | jq ". | length")

    if [ "$cve_counter" -gt "0" ]; then
      rc=1
      echo "Found $cve_counter CVEs of High/Critical in $repo@$b:"
      echo "$output" | yq r -P -
      echo
    fi
  done

  if [[ "$rc" == "0" ]]; then
    echo "No critical/high CVE found in ${img}"
  fi

  return "$rc"
}

# verifies that the component images of the CSV do not contain a CVE
# shellcheck disable=SC2317
function verify_no_cve {
  rc=0

  while IFS= read -r img; do
    quay_image_ref="$(get_quay_image_ref "$img")"
    verify_no_cve_in_image "$quay_image_ref" || { rc=$?; }
  done <<< "$(yq read "${csv_file}" 'spec.relatedImages[*].image' | sort | uniq)"

  if [[ "$rc" == "0" ]]; then
    echo "No CVEs found in component images of CSV"
  fi

  return "$rc"
}

failed=0
declare -a checks_to_run=()

while [[ $# -ne 0 ]]; do
  parameter=$1
  case ${parameter} in
    --revision) checks_to_run+=(verify_image_revisions) ;;
    --cve) checks_to_run+=(verify_no_cve) ;;
    *) abort "error: unknown option ${parameter}" ;;
  esac
  shift
done

if [[ "${#checks_to_run[@]}" == "0" ]]; then
  # in case no specific checks are given, run all checks
  checks_to_run+=(verify_image_revisions)
  checks_to_run+=(verify_no_cve)
fi

for check_to_run in "${checks_to_run[@]}"; do
  ${check_to_run} || { failed=$?; }
done

exit $failed
