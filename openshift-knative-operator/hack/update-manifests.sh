#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
source "$root/hack/lib/__sources__.bash"

# These files could in theory change from release to release, though their names should
# be fairly stable.
serving_files=(serving-crds serving-core serving-hpa serving-post-install-jobs)
eventing_files=(eventing-crds eventing-core in-memory-channel mt-channel-broker eventing-sugar-controller eventing-pre-install-jobs)

function download {
  component=$1
  version=$2
  shift
  shift

  files=("$@")

  component_dir="$root/openshift-knative-operator/cmd/operator/kodata/knative-${component}"
  target_dir="${component_dir}/${version:1}"
  rm -r "$component_dir"
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
# TODO: Remove this patch once 0.18.5 of serving or newer is available.
git apply "$root/openshift-knative-operator/hack/001-liveness.patch"

# TODO: Remove this once upstream fixed https://github.com/knative/operator/issues/376.
# See also https://issues.redhat.com/browse/SRVKS-670.
git apply "$root/openshift-knative-operator/hack/003-activator-pdb.patch"

download eventing $KNATIVE_EVENTING_VERSION "${eventing_files[@]}"
# Extra ClusterRole for downstream, so that users can get the CMs of knative-eventing
# TODO: propose to upstream
git apply "$root/openshift-knative-operator/hack/002-openshift-eventing-role.patch"

# SRVKE-654: relax the MT adapter replica
git apply "$root/openshift-knative-operator/hack/004-eventing-pingsource-one-replica.patch"

# Apply port of 4640 to Serverless Operator
git apply "$root/openshift-knative-operator/hack/005-imc-pod_anti_affinity.patch"
