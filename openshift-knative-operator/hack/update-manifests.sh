#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
source "$root/hack/lib/__sources__.bash"

# These files could in theory change from release to release, though their names should
# be fairly stable.
serving_files=(serving-crds serving-core serving-hpa serving-post-install-jobs)
eventing_files=(eventing-crds eventing-core in-memory-channel mt-channel-broker eventing-sugar-controller eventing-post-install-jobs)

function download {
  component=$1
  version=$2
  shift
  shift

  files=("$@")

  target_dir="$root/openshift-knative-operator/cmd/operator/kodata/knative-${component}/${version:1}"
  rm -r "$target_dir"
  mkdir -p "$target_dir"
  
  for (( i=0; i<${#files[@]}; i++ ));
  do
    index=$(( i+1 ))
    file="${files[$i]}.yaml"
    target_file="$target_dir/$index-$file"
    url="https://github.com/knative/$component/releases/download/$version/$file"

    wget --no-check-certificate "$url" -O "$target_file"
  done
}

download serving $KNATIVE_SERVING_VERSION "${serving_files[@]}"
download eventing $KNATIVE_EVENTING_VERSION "${eventing_files[@]}"