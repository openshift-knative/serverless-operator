#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
# shellcheck disable=SC1091,SC1090
source "$root/hack/lib/__sources__.bash"

# These files could in theory change from release to release, though their names should
# be fairly stable.
serving_files=(serving-crds serving-core serving-hpa serving-domainmapping-crds serving-domainmapping serving-post-install-jobs)
eventing_files=(eventing-crds eventing-core in-memory-channel mt-channel-broker eventing-sugar-controller)
kafka_files=(channel-consolidated source)

function download {
  component=$1
  version=$2
  org=$3
  shift
  shift
  shift

  files=("$@")
  local component_dir=""
  local target_dir=""
  if [[ $component == "eventing-kafka" ]]; then
      component_dir="$root/knative-operator/deploy/resources/knativekafka/"
      target_dir="${component_dir}/"
  else
      component_dir="$root/openshift-knative-operator/cmd/operator/kodata/knative-${component}"
      target_dir="${component_dir}/${version:1}"
      rm -r "$component_dir"
      mkdir -p "$target_dir"
  fi

  for (( i=0; i<${#files[@]}; i++ ));
  do
    index=$(( i+1 ))
    file="${files[$i]}.yaml"
    target_file="$target_dir/$index-$file"
    url="https://github.com/$org/$component/releases/download/$version/$file"

    wget --no-check-certificate "$url" -O "$target_file"
  done
}

download serving "$KNATIVE_SERVING_VERSION" "knative" "${serving_files[@]}"

# Create an empty ingress directory.
# TODO: Investigate moving Kourier into here rather than "manually" installing it via
#       knative-openshift.
ingress_dir="$root/openshift-knative-operator/cmd/operator/kodata/ingress/$(versions.major_minor "${KNATIVE_SERVING_VERSION}")"
mkdir -p "$ingress_dir"
touch "$ingress_dir/.gitkeep"

# TODO: Remove this once upstream fixed https://github.com/knative/operator/issues/376.
# See also https://issues.redhat.com/browse/SRVKS-670.
git apply "$root/openshift-knative-operator/hack/003-serving-pdb.patch"

download eventing "$KNATIVE_EVENTING_VERSION" "knative" "${eventing_files[@]}"
# Extra ClusterRole for downstream, so that users can get the CMs of knative-eventing
# TODO: propose to upstream
git apply "$root/openshift-knative-operator/hack/002-openshift-eventing-role.patch"
# For SRVKE-629 we disable HPA:
git apply "$root/openshift-knative-operator/hack/005-disable-hpa.patch"

download eventing-kafka "$KNATIVE_EVENTING_KAFKA_VERSION" "knative-sandbox" "${kafka_files[@]}"
