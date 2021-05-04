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

# This excludes the gateways and peerauthentication settings as we want customers to do
# manipulate those.
istio_files=(200-clusterrole 500-mutating-webhook 500-validating-webhook config controller webhook-deployment webhook-service)

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

function download_ingress {
  component=$1
  version=$2
  shift
  shift

  files=("$@")

  ingress_dir="$root/openshift-knative-operator/cmd/operator/kodata/ingress/$(versions.major_minor "${KNATIVE_SERVING_VERSION}")"
  rm -r "$ingress_dir"
  mkdir -p "$ingress_dir"

  for (( i=0; i<${#files[@]}; i++ ));
  do
    index=$(( i+1 ))
    file="${files[$i]}.yaml"
    ingress_target_file="$ingress_dir/$index-$file"
    url="https://raw.githubusercontent.com/openshift-knative/${component}/${version}/config/${file}"
    wget --no-check-certificate "$url" -O "$ingress_target_file"
  done
}

download serving "$KNATIVE_SERVING_VERSION" "${serving_files[@]}"

download_ingress net-istio "v$(metadata.get dependencies.net_istio)" "${istio_files[@]}"

# TODO: Remove this once upstream fixed https://github.com/knative/operator/issues/376.
# See also https://issues.redhat.com/browse/SRVKS-670.
git apply "$root/openshift-knative-operator/hack/003-serving-pdb.patch"

download eventing "$KNATIVE_EVENTING_VERSION" "${eventing_files[@]}"
# Extra ClusterRole for downstream, so that users can get the CMs of knative-eventing
# TODO: propose to upstream
git apply "$root/openshift-knative-operator/hack/002-openshift-eventing-role.patch"
# For SRVKE-629 we disable HPA:
git apply "$root/openshift-knative-operator/hack/005-disable-hpa.patch"
